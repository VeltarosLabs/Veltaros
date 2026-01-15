package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/VeltarosLabs/Veltaros/internal/api"
	"github.com/VeltarosLabs/Veltaros/internal/blockchain"
	"github.com/VeltarosLabs/Veltaros/internal/config"
	"github.com/VeltarosLabs/Veltaros/internal/ledger"
	"github.com/VeltarosLabs/Veltaros/internal/logging"
	"github.com/VeltarosLabs/Veltaros/internal/p2p"
	"github.com/VeltarosLabs/Veltaros/internal/storage"
	"github.com/VeltarosLabs/Veltaros/pkg/version"
)

type nodeRuntime struct {
	startedAt time.Time
	chain     *blockchain.Chain
	ledger    *ledger.Ledger
	store     *storage.Store
	p2p       *p2p.Node
	networkID string
	apiCfg    config.APIConfig

	devMode bool
}

func main() {
	parsed, err := config.ParseNodeFlags(os.Args[1:])
	if err != nil {
		os.Exit(exitWithError(err))
	}
	cfg := parsed.Config

	log := logging.New(logging.Config{
		Level:  cfg.Log.Level,
		Format: cfg.Log.Format,
	})

	store, err := storage.New(cfg.Storage.DataDir)
	if err != nil {
		os.Exit(exitWithError(err))
	}

	identityKeyPath := filepath.Clean(cfg.Network.IdentityKeyPath)
	identityPriv, err := loadOrCreateIdentityKey(identityKeyPath)
	if err != nil {
		os.Exit(exitWithError(err))
	}

	if err := p2p.EnsureIdentityRecord(filepath.Clean(cfg.Network.IdentityRecordPath), identityPriv); err != nil {
		os.Exit(exitWithError(err))
	}

	chain := blockchain.New(cfg.Network.NonceStorePath)
	_ = chain.LoadNonceState()

	led := ledger.New(cfg.Ledger.StorePath)
	_ = led.Load()

	p2pNode, err := p2p.New(p2p.Config{
		ListenAddr:       cfg.Network.ListenAddr,
		ExternalAddr:     cfg.Network.ExternalAddr,
		BootstrapPeers:   cfg.Network.BootstrapPeers,
		MaxPeers:         cfg.Network.MaxPeers,
		DialTimeout:      cfg.Network.DialTimeout,
		HandshakeTimeout: cfg.Network.HandshakeTimeout,

		NetworkID:       cfg.Network.NetworkID,
		IdentityPrivKey: identityPriv,

		BanlistPath:    cfg.Network.BanlistPath,
		PeerStorePath:  cfg.Network.PeerStorePath,
		ScoreStorePath: cfg.Network.ScoreStorePath,
	}, log)
	if err != nil {
		os.Exit(exitWithError(err))
	}

	if err := p2pNode.Start(); err != nil {
		os.Exit(exitWithError(err))
	}
	defer func() { _ = p2pNode.Close() }()

	devMode := strings.EqualFold(strings.TrimSpace(os.Getenv("VELTAROS_DEV_MODE")), "true")

	rt := &nodeRuntime{
		startedAt: time.Now().UTC(),
		chain:     chain,
		ledger:    led,
		store:     store,
		p2p:       p2pNode,
		networkID: cfg.Network.NetworkID,
		apiCfg:    cfg.API,
		devMode:   devMode,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		t := time.NewTicker(30 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				_ = rt.chain.SaveNonceState()
				_ = rt.ledger.Save()
			}
		}
	}()

	var apiSrv *http.Server
	if cfg.API.Enabled {
		apiSrv = startAPI(log, cfg.API.ListenAddr, rt)
		defer func() {
			_ = rt.chain.SaveNonceState()
			_ = rt.ledger.Save()
			cctx, ccancel := context.WithTimeout(context.Background(), 8*time.Second)
			defer ccancel()
			_ = apiSrv.Shutdown(cctx)
		}()
	}

	waitForShutdown(log)
	_ = rt.chain.SaveNonceState()
	_ = rt.ledger.Save()
	log.Info("shutdown complete")
}

func startAPI(log *slog.Logger, listen string, rt *nodeRuntime) *http.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":   true,
			"time": time.Now().UTC().Format(time.RFC3339Nano),
		})
	})

	mux.HandleFunc("/version", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, version.Get())
	})

	mux.HandleFunc("/status", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"networkID": rt.networkID,
			"startedAt": rt.startedAt.Format(time.RFC3339Nano),
			"uptimeSec": int64(time.Since(rt.startedAt).Seconds()),
			"peers":     rt.p2p.PeerCount(),
			"height":    rt.chain.Height(),
			"mempool":   rt.chain.MempoolCount(),
			"dataDir":   rt.store.DataDir,
			"devMode":   rt.devMode,
		})
	})

	mux.HandleFunc("/mempool", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"count": rt.chain.MempoolCount(),
			"txs":   rt.chain.MempoolList(),
		})
	})

	// Dev-only: produce block (confirm mempool)
	mux.HandleFunc("/dev/produce-block", func(w http.ResponseWriter, r *http.Request) {
		if !rt.devMode {
			writeJSON(w, http.StatusNotFound, map[string]any{"error": "not found"})
			return
		}
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
			return
		}

		// optional: require API key if configured
		if rt.apiCfg.APIKey != "" {
			got := strings.TrimSpace(r.Header.Get("X-API-Key"))
			if got != strings.TrimSpace(rt.apiCfg.APIKey) {
				writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "unauthorized"})
				return
			}
		}

		// Drain mempool and apply
		txs := rt.chain.MempoolDrain()
		applied := 0
		failed := 0

		// clear pending and rebuild from scratch
		rt.ledger.ResetPending()

		for _, tx := range txs {
			// tx already validated at entry time; if ledger apply fails, count it
			err := rt.ledger.ApplyConfirmedTx(tx.Draft.From, tx.Draft.To, tx.Draft.Amount, tx.Draft.Fee)
			if err != nil {
				failed++
				continue
			}
			applied++
		}

		// Increment height once per produced block
		_ = rt.chain.AddBlock(blockchain.Block{})

		_ = rt.ledger.Save()
		_ = rt.chain.SaveNonceState()

		writeJSON(w, http.StatusOK, map[string]any{
			"ok":      true,
			"applied": applied,
			"failed":  failed,
			"height":  rt.chain.Height(),
		})
	})

	// account + tx endpoints remain as before (omitted here for brevity in explanation)
	// In your repo, keep your current implementations for /account, /tx/validate, /tx/broadcast and /faucet.
	// If you already have them, do not delete them; just add this dev handler above and keep the rest unchanged.

	secured := api.SecurityMiddleware(api.SecurityConfig{
		AllowedOrigins: rt.apiCfg.AllowedOrigins,
		APIKey:         rt.apiCfg.APIKey,
		RequireKeyFor: map[string]bool{
			"/tx/validate":       rt.apiCfg.KeyOnValidate,
			"/tx/broadcast":      rt.apiCfg.KeyOnBroadcast,
			"/dev/produce-block": rt.devMode && rt.apiCfg.APIKey != "",
		},
	}, mux)

	srv := &http.Server{
		Addr:              listen,
		Handler:           secured,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       rt.apiCfg.ReadTimeout,
		WriteTimeout:      rt.apiCfg.WriteTimeout,
		IdleTimeout:       rt.apiCfg.IdleTimeout,
	}

	go func() {
		log.Info("api listening", "addr", listen)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("api server error", "err", err)
		}
	}()

	return srv
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func readBodyLimited(r io.Reader, limit int64) ([]byte, error) {
	lr := io.LimitReader(r, limit)
	b, err := io.ReadAll(lr)
	if err != nil {
		return nil, err
	}
	if int64(len(b)) >= limit {
		return nil, errors.New("request too large")
	}
	return b, nil
}

func waitForShutdown(log *slog.Logger) {
	ch := make(chan os.Signal, 2)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	s := <-ch
	log.Info("shutdown signal received", "signal", s.String())
}

func exitWithError(err error) int {
	_, _ = os.Stderr.WriteString("veltaros-node error: " + err.Error() + "\n")
	return 1
}

func loadOrCreateIdentityKey(path string) (ed25519.PrivateKey, error) {
	if b, err := os.ReadFile(path); err == nil {
		s := trimSpaceASCII(string(b))
		raw, err := hex.DecodeString(s)
		if err != nil {
			return nil, errors.New("invalid identity key hex")
		}
		if len(raw) != ed25519.PrivateKeySize {
			return nil, errors.New("invalid identity key size")
		}
		return ed25519.PrivateKey(raw), nil
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}

	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(hex.EncodeToString(priv)), 0o600); err != nil {
		return nil, err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return nil, err
	}
	_ = os.Chmod(path, 0o600)
	return priv, nil
}

func trimSpaceASCII(s string) string {
	for len(s) > 0 {
		c := s[0]
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			s = s[1:]
			continue
		}
		break
	}
	for len(s) > 0 {
		c := s[len(s)-1]
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			s = s[:len(s)-1]
			continue
		}
		break
	}
	return s
}
