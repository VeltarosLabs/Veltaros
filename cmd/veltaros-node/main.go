package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
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

	chain := blockchain.New()

	p2pNode, err := p2p.New(p2p.Config{
		ListenAddr:       cfg.Network.ListenAddr,
		ExternalAddr:     cfg.Network.ExternalAddr,
		BootstrapPeers:   cfg.Network.BootstrapPeers,
		MaxPeers:         cfg.Network.MaxPeers,
		DialTimeout:      cfg.Network.DialTimeout,
		HandshakeTimeout: cfg.Network.HandshakeTimeout,
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
