package p2p

import (
	"bufio"
	"context"
	"crypto/ed25519"
	"errors"
	"log/slog"
	"net"
	"sync"
	"time"

	vcrypto "github.com/VeltarosLabs/Veltaros/internal/crypto"
)

type Config struct {
	ListenAddr       string
	ExternalAddr     string
	BootstrapPeers   []string
	MaxPeers         int
	DialTimeout      time.Duration
	HandshakeTimeout time.Duration

	NetworkID       string
	IdentityPrivKey ed25519.PrivateKey
}

type PeerInfo struct {
	RemoteAddr   string `json:"remoteAddr"`
	Inbound      bool   `json:"inbound"`
	ConnectedAt  int64  `json:"connectedAt"`
	PublicKeyHex string `json:"publicKeyHex"`
	NodeVersion  string `json:"nodeVersion"`
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

	pubKey      ed25519.PublicKey
	nodeVersion string
	helloNonce  [32]byte
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
	if cfg.NetworkID == "" {
		return nil, errors.New("NetworkID is required")
	}
	if len(cfg.IdentityPrivKey) != ed25519.PrivateKeySize {
		return nil, errors.New("IdentityPrivKey is required (ed25519)")
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

	n.log.Info("p2p listening", "addr", n.cfg.ListenAddr, "external", n.cfg.ExternalAddr, "maxPeers", n.cfg.MaxPeers, "networkID", n.cfg.NetworkID)

	go n.acceptLoop()
	go n.bootstrapLoop()

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
			RemoteAddr:   p.conn.RemoteAddr().String(),
			Inbound:      p.inbound,
			ConnectedAt:  p.connectedAt.UTC().Unix(),
			PublicKeyHex: PublicKeyHex(p.pubKey),
			NodeVersion:  p.nodeVersion,
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

func (n *Node) bootstrapLoop() {
	// Best-effort dialing; safe and bounded. We’ll add backoff/jitter and peer discovery next.
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	// Initial immediate attempt
	n.tryBootstrapOnce()

	for {
		select {
		case <-n.ctx.Done():
			return
		case <-ticker.C:
			n.tryBootstrapOnce()
		}
	}
}

func (n *Node) tryBootstrapOnce() {
	for _, addr := range n.cfg.BootstrapPeers {
		addr := addr
		if addr == "" {
			continue
		}
		if n.isConnectedTo(addr) {
			continue
		}
		go n.dialPeer(addr)
	}
}

func (n *Node) isConnectedTo(addr string) bool {
	n.mu.RLock()
	defer n.mu.RUnlock()

	// addr may be "host:port". Our key is conn.RemoteAddr().String() which might differ.
	// For bootstrap we only need a coarse guard: if any peer has same remote string.
	for _, p := range n.peers {
		if p.conn.RemoteAddr().String() == addr {
			return true
		}
	}
	return false
}

func (n *Node) dialPeer(addr string) {
	select {
	case <-n.ctx.Done():
		return
	default:
	}

	dialer := &net.Dialer{Timeout: n.cfg.DialTimeout}
	conn, err := dialer.DialContext(n.ctx, "tcp", addr)
	if err != nil {
		n.log.Debug("dial failed", "addr", addr, "err", err)
		return
	}

	if !n.tryRegisterPeer(conn, false) {
		_ = conn.Close()
		return
	}

	n.handleConn(conn, false)
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

func (n *Node) setPeerHello(conn net.Conn, h Hello) {
	n.mu.Lock()
	defer n.mu.Unlock()

	key := conn.RemoteAddr().String()
	p, ok := n.peers[key]
	if !ok {
		return
	}
	p.pubKey = h.PublicKey
	p.nodeVersion = h.NodeVersion
	p.helloNonce = h.Nonce
	n.peers[key] = p
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

	br := bufio.NewReaderSize(conn, 64*1024)
	bw := bufio.NewWriterSize(conn, 64*1024)

	_ = conn.SetDeadline(time.Now().Add(n.cfg.HandshakeTimeout))

	// Handshake:
	// - Outbound: send HELLO then expect HELLO back.
	// - Inbound: expect HELLO then send HELLO.
	if inbound {
		peerHello, err := n.readAndValidateHello(br)
		if err != nil {
			n.log.Warn("handshake failed (inbound read hello)", "remote", conn.RemoteAddr().String(), "err", err)
			return
		}
		if err := n.writeHello(bw); err != nil {
			n.log.Warn("handshake failed (inbound write hello)", "remote", conn.RemoteAddr().String(), "err", err)
			return
		}
		if err := bw.Flush(); err != nil {
			n.log.Warn("handshake failed (inbound flush)", "remote", conn.RemoteAddr().String(), "err", err)
			return
		}
		n.setPeerHello(conn, peerHello)
	} else {
		if err := n.writeHello(bw); err != nil {
			n.log.Warn("handshake failed (outbound write hello)", "remote", conn.RemoteAddr().String(), "err", err)
			return
		}
		if err := bw.Flush(); err != nil {
			n.log.Warn("handshake failed (outbound flush)", "remote", conn.RemoteAddr().String(), "err", err)
			return
		}
		peerHello, err := n.readAndValidateHello(br)
		if err != nil {
			n.log.Warn("handshake failed (outbound read hello)", "remote", conn.RemoteAddr().String(), "err", err)
			return
		}
		n.setPeerHello(conn, peerHello)
	}

	// Clear handshake deadline
	_ = conn.SetDeadline(time.Time{})

	// Connection loop
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
			n.log.Debug("ignored message", "remote", conn.RemoteAddr().String(), "type", f.Type)
		}
	}
}

func (n *Node) writeHello(bw *bufio.Writer) error {
	pub := n.cfg.IdentityPrivKey.Public().(ed25519.PublicKey)
	h, err := NewHello(n.cfg.NetworkID, pub)
	if err != nil {
		return err
	}
	payload, err := h.Encode()
	if err != nil {
		return err
	}
	return WriteFrame(bw, MsgHello, payload)
}

func (n *Node) readAndValidateHello(br *bufio.Reader) (Hello, error) {
	frame, err := ReadFrame(br)
	if err != nil {
		return Hello{}, err
	}
	if frame.Type != MsgHello {
		return Hello{}, errors.New("expected HELLO")
	}
	h, err := DecodeHello(frame.Payload)
	if err != nil {
		return Hello{}, err
	}
	if err := ValidateHello(h, HelloValidation{
		NetworkID:      n.cfg.NetworkID,
		MaxClockSkew:   2 * time.Minute,
		RequireNonZero: true,
	}); err != nil {
		return Hello{}, err
	}

	// Ensure peer isn't claiming our exact identity (simple self-connection guard).
	ourPub := n.cfg.IdentityPrivKey.Public().(ed25519.PublicKey)
	if vcrypto.ConstantTimeEqual(ourPub, h.PublicKey) {
		return Hello{}, errors.New("peer has same identity public key")
	}
	return h, nil
}
