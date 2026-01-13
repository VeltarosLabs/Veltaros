package p2p

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"sync"
	"time"
)

type Config struct {
	ListenAddr       string
	ExternalAddr     string
	BootstrapPeers   []string
	MaxPeers         int
	DialTimeout      time.Duration
	HandshakeTimeout time.Duration
}

type Node struct {
	cfg    Config
	log    *slog.Logger
	ln     net.Listener
	ctx    context.Context
	cancel context.CancelFunc

	mu     sync.Mutex
	closed bool
}

func New(cfg Config, log *slog.Logger) (*Node, error) {
	if log == nil {
		return nil, errors.New("logger is required")
	}
	if cfg.ListenAddr == "" {
		return nil, errors.New("ListenAddr is required")
	}
	if cfg.MaxPeers <= 0 {
		return nil, errors.New("MaxPeers must be > 0")
	}

	ctx, cancel := context.WithCancel(context.Background())
	return &Node{
		cfg:    cfg,
		log:    log.With("component", "p2p"),
		ctx:    ctx,
		cancel: cancel,
	}, nil
}

func (n *Node) Start() error {
	ln, err := net.Listen("tcp", n.cfg.ListenAddr)
	if err != nil {
		return err
	}
	n.ln = ln

	n.log.Info("p2p listening", "addr", n.cfg.ListenAddr, "external", n.cfg.ExternalAddr, "maxPeers", n.cfg.MaxPeers)

	// Production note:
	// We are creating a real listening socket and handling graceful shutdown correctly.
	// The peer protocol + message framing will be implemented in subsequent phases.
	go n.acceptLoop()

	return nil
}

func (n *Node) Close() error {
	n.mu.Lock()
	if n.closed {
		n.mu.Unlock()
		return nil
	}
	n.closed = true
	n.mu.Unlock()

	n.cancel()

	if n.ln != nil {
		_ = n.ln.Close()
	}

	n.log.Info("p2p stopped")
	return nil
}

func (n *Node) acceptLoop() {
	for {
		conn, err := n.ln.Accept()
		if err != nil {
			select {
			case <-n.ctx.Done():
				return
			default:
				n.log.Warn("accept error", "err", err)
				continue
			}
		}

		// For now, immediately close inbound connections until handshake/protocol is implemented.
		// This is safer than accepting unknown traffic and leaving sockets hanging.
		_ = conn.Close()
	}
}
