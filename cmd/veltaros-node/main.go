package main

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/VeltarosLabs/Veltaros/internal/blockchain"
	"github.com/VeltarosLabs/Veltaros/internal/config"
	"github.com/VeltarosLabs/Veltaros/internal/logging"
	"github.com/VeltarosLabs/Veltaros/internal/p2p"
	"github.com/VeltarosLabs/Veltaros/internal/storage"
	"github.com/VeltarosLabs/Veltaros/pkg/version"
)

type nodeRuntime struct {
	startedAt time.Time
	chain     *blockchain.Chain
	store     *storage.Store
	p2p       *p2p.Node
	networkID string
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

	chain := blockchain.New()

	p2pNode, err := p2p.New(p2p.Config{
		ListenAddr:       cfg.Network.ListenAddr,
		ExternalAddr:     cfg.Network.ExternalAddr,
		BootstrapPeers:   cfg.Network.BootstrapPeers,
		MaxPeers:         cfg.Network.MaxPeers,
		DialTimeout:      cfg.Network.DialTimeout,
		HandshakeTimeout: cfg.Network.HandshakeTimeout,
		NetworkID:        cfg.Network.NetworkID,
		IdentityPrivKey:  identityPriv,
	}, log)
	if err != nil {
		os.Exit(exitWithError(err))
	}

	if err := p2pNode.Start(); err != nil {
		os.Exit(exitWithError(err))
	}
	defer func() { _ = p2pNode.Close() }()

	rt := &nodeRuntime{
		startedAt: time.Now().UTC(),
		chain:     chain,
		store:     store,
		p2p:       p2pNode,
		networkID: cfg.Network.NetworkID,
	}

	var apiSrv *http.Server
	if cfg.API.Enabled {
		apiSrv = startAPI(log, cfg.API.ListenAddr, rt)
		defer func() {
			ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
			defer cancel()
			_ = apiSrv.Shutdown(ctx)
		}()
	}

	waitForShutdown(log)

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
			"dataDir":   rt.store.DataDir,
		})
	})

	mux.HandleFunc("/peers", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"count": rt.p2p.PeerCount(),
			"peers": rt.p2p.Peers(),
		})
	})

	srv := &http.Server{
		Addr:              listen,
		Handler:           securityHeaders(mux),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		log.Info("api listening", "addr", listen)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("api server error", "err", err)
		}
	}()

	return srv
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
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

// Identity key file format: hex-encoded ed25519 private key (64 bytes).
// Stored at config p2p.identityKey (default: data/node/identity.key).
func loadOrCreateIdentityKey(path string) (ed25519.PrivateKey, error) {
	if b, err := os.ReadFile(path); err == nil {
		s := string(b)
		s = trimSpaceASCII(s)
		raw, err := hex.DecodeString(s)
		if err != nil {
			return nil, errors.New("invalid identity key hex")
		}
		if len(raw) != ed25519.PrivateKeySize {
			return nil, errors.New("invalid identity key size")
		}
		return ed25519.PrivateKey(raw), nil
	}

	// Create new
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}

	_, priv, err := ed25519.GenerateKey(nil)
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
