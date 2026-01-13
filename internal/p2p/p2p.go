package p2p

import (
	"bufio"
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

type PeerInfo struct {
	RemoteAddr  string `json:"remoteAddr"`
	Inbound     bool   `json:"inbound"`
	ConnectedAt int64  `json:"connectedAt"`
}

type Node struct {
	cfg    Config
	log    *slog.Logger
	ln     net.Listener
	ctx    context.Context
	cancel context.CancelFunc

	mu     sync.RWMutex
	closed bool
	peers  map[string]peerConn
}

type peerConn struct {
	conn        net.Conn
	inbound     bool
	connectedAt time.Time
}

func New(cfg Config, log *slog.Logger) (*Node, error) {
	if log == nil {
		return nil, errors.New("logger is required")
	}
	if cfg.ListenAddr == "" {
		return nil, errors.New("ListenAddr is required")
	}
	if cfg.MaxPeers <= 0 || cfg.MaxPeers > 4096 {
		return nil, errors.New("MaxPeers out of range")
	}
	if cfg.DialTimeout <= 0 {
		cfg.DialTimeout = 7 * time.Second
	}
	if cfg.HandshakeTimeout <= 0 {
		cfg.HandshakeTimeout = 7 * time.Second
	}

	ctx, cancel := context.WithCancel(context.Background())
	return &Node{
		cfg:    cfg,
		log:    log.With("component", "p2p"),
		ctx:    ctx,
		cancel: cancel,
		peers:  make(map[string]peerConn),
	}, nil
}

func (n *Node) Start() error {
	ln, err := net.Listen("tcp", n.cfg.ListenAddr)
	if err != nil {
		return err
	}
	n.ln = ln

	n.log.Info("p2p listening", "addr", n.cfg.ListenAddr, "external", n.cfg.ExternalAddr, "maxPeers", n.cfg.MaxPeers)

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

	// Close peers
	n.mu.Lock()
	for k, p := range n.peers {
		_ = p.conn.Close()
		delete(n.peers, k)
	}
	n.mu.Unlock()

	n.log.Info("p2p stopped")
	return nil
}

func (n *Node) PeerCount() int {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return len(n.peers)
}

func (n *Node) Peers() []PeerInfo {
	n.mu.RLock()
	defer n.mu.RUnlock()

	out := make([]PeerInfo, 0, len(n.peers))
	for _, p := range n.peers {
		out = append(out, PeerInfo{
			RemoteAddr:  p.conn.RemoteAddr().String(),
			Inbound:     p.inbound,
			ConnectedAt: p.connectedAt.UTC().Unix(),
		})
	}
	return out
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

		if !n.tryRegisterPeer(conn, true) {
			_ = conn.Close()
			continue
		}

		go n.handleConn(conn, true)
	}
}

func (n *Node) tryRegisterPeer(conn net.Conn, inbound bool) bool {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.closed {
		return false
	}
	if len(n.peers) >= n.cfg.MaxPeers {
		n.log.Warn("peer rejected: max peers reached", "remote", conn.RemoteAddr().String())
		return false
	}

	key := conn.RemoteAddr().String()
	n.peers[key] = peerConn{
		conn:        conn,
		inbound:     inbound,
		connectedAt: time.Now().UTC(),
	}

	n.log.Info("peer connected", "remote", key, "inbound", inbound, "peers", len(n.peers))
	return true
}

func (n *Node) unregisterPeer(conn net.Conn) {
	n.mu.Lock()
	defer n.mu.Unlock()

	key := conn.RemoteAddr().String()
	if _, ok := n.peers[key]; ok {
		delete(n.peers, key)
		n.log.Info("peer disconnected", "remote", key, "peers", len(n.peers))
	}
}

func (n *Node) handleConn(conn net.Conn, inbound bool) {
	defer func() {
		_ = conn.Close()
		n.unregisterPeer(conn)
	}()

	// Defensive timeouts
	_ = conn.SetDeadline(time.Now().Add(n.cfg.HandshakeTimeout))

	// Minimal safe handshake placeholder: require a valid framed message (MsgHello)
	// We will expand to version negotiation, network ID, and node identity next.
	br := bufio.NewReaderSize(conn, 64*1024)
	bw := bufio.NewWriterSize(conn, 64*1024)

	frame, err := ReadFrame(br)
	if err != nil {
		n.log.Warn("handshake failed (read)", "remote", conn.RemoteAddr().String(), "err", err)
		return
	}
	if frame.Type != MsgHello {
		n.log.Warn("handshake failed (type)", "remote", conn.RemoteAddr().String(), "type", frame.Type)
		return
	}

	// Reply with Pong-style ack (temporary)
	if err := WriteFrame(bw, MsgPong, []byte("ok")); err != nil {
		n.log.Warn("handshake failed (write)", "remote", conn.RemoteAddr().String(), "err", err)
		return
	}
	if err := bw.Flush(); err != nil {
		n.log.Warn("handshake failed (flush)", "remote", conn.RemoteAddr().String(), "err", err)
		return
	}

	// Clear handshake deadline; set read/write deadlines per message in future
	_ = conn.SetDeadline(time.Time{})

	// Connection loop (for now, we only support Ping and Goodbye to keep it safe)
	for {
		select {
		case <-n.ctx.Done():
			return
		default:
		}

		_ = conn.SetReadDeadline(time.Now().Add(DefaultReadTimeout))
		f, err := ReadFrame(br)
		if err != nil {
			return
		}

		switch f.Type {
		case MsgPing:
			_ = conn.SetWriteDeadline(time.Now().Add(DefaultWriteTimeout))
			if err := WriteFrame(bw, MsgPong, []byte("pong")); err != nil {
				return
			}
			if err := bw.Flush(); err != nil {
				return
			}
		case MsgGoodbye:
			return
		default:
			// Unknown messages are ignored for now (future: penalize/misbehavior scoring)
			n.log.Debug("ignored message", "remote", conn.RemoteAddr().String(), "type", f.Type)
		}
	}
}
