package p2p

import (
	"bufio"
	"context"
	"crypto/ed25519"
	"errors"
	"log/slog"
	"math/rand"
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

	BanlistPath   string
	PeerStorePath string
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

	// Known peers (for dialing / discovery)
	knownMu    sync.RWMutex
	knownPeers map[string]StoredPeer

	// Dial backoff state
	backoffMu sync.Mutex
	backoff   map[string]dialBackoff

	// Persistence
	banlist   *Banlist
	peerStore *PeerStore
}

type peerConn struct {
	conn        net.Conn
	inbound     bool
	connectedAt time.Time
	pubKey      ed25519.PublicKey
	nodeVersion string
	helloNonce  [32]byte
	lastMsgAt   time.Time
}

type dialBackoff struct {
	Attempts  int
	NextTryAt time.Time
	LastErr   string
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
	if cfg.BanlistPath == "" {
		return nil, errors.New("BanlistPath is required")
	}
	if cfg.PeerStorePath == "" {
		return nil, errors.New("PeerStorePath is required")
	}

	ctx, cancel := context.WithCancel(context.Background())

	n := &Node{
		cfg:        cfg,
		log:        log.With("component", "p2p"),
		ctx:        ctx,
		cancel:     cancel,
		peers:      make(map[string]peerConn),
		knownPeers: make(map[string]StoredPeer),
		backoff:    make(map[string]dialBackoff),
		banlist:    NewBanlist(cfg.BanlistPath),
		peerStore:  NewPeerStore(cfg.PeerStorePath),
	}

	// Load persisted state
	_ = n.banlist.Load()

	if peers, err := n.peerStore.Load(); err == nil {
		for _, p := range peers {
			n.knownPeers[p.Addr] = p
		}
	}

	// Seed with bootstrap
	now := time.Now().UTC()
	for _, a := range cfg.BootstrapPeers {
		a = sanitizeHelloString(a)
		if a == "" {
			continue
		}
		n.knownPeers[a] = StoredPeer{Addr: a, SeenAt: now, Source: "bootstrap"}
	}

	return n, nil
}

func (n *Node) Start() error {
	ln, err := net.Listen("tcp", n.cfg.ListenAddr)
	if err != nil {
		return err
	}
	n.ln = ln

	n.log.Info("p2p listening",
		"addr", n.cfg.ListenAddr,
		"external", n.cfg.ExternalAddr,
		"maxPeers", n.cfg.MaxPeers,
		"networkID", n.cfg.NetworkID,
	)

	go n.acceptLoop()
	go n.dialLoop()
	go n.discoveryLoop()
	go n.persistLoop()

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

	_ = n.persistOnce()

	n.log.Info("p2p stopped")
	return nil
}

func (n *Node) PeerCount() int {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return len(n.peers)
}

func (n *Node) KnownPeerCount() int {
	n.knownMu.RLock()
	defer n.knownMu.RUnlock()
	return len(n.knownPeers)
}

func (n *Node) BanCount() int {
	return n.banlist.CountActive()
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

		remote := conn.RemoteAddr().String()
		if banned, e := n.banlist.IsBanned(remote); banned {
			n.log.Warn("peer rejected: banned", "remote", remote, "until", e.Until, "reason", e.Reason)
			_ = conn.Close()
			continue
		}

		if !n.tryRegisterPeer(conn, true) {
			_ = conn.Close()
			continue
		}

		go n.handleConn(conn, true)
	}
}

func (n *Node) dialLoop() {
	// Dial cadence: continuously attempt to fill peer slots, respecting backoff.
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-n.ctx.Done():
			return
		case <-ticker.C:
			n.fillOutbound()
		}
	}
}

func (n *Node) fillOutbound() {
	// Keep a small outbound target; inbound can fill remaining slots.
	targetOutbound := n.cfg.MaxPeers / 3
	if targetOutbound < 4 {
		targetOutbound = 4
	}

	outbound := 0
	n.mu.RLock()
	for _, p := range n.peers {
		if !p.inbound {
			outbound++
		}
	}
	n.mu.RUnlock()

	if outbound >= targetOutbound {
		return
	}

	addrs := n.pickDialCandidates(targetOutbound - outbound)
	for _, addr := range addrs {
		addr := addr
		go n.dialPeer(addr)
	}
}

func (n *Node) discoveryLoop() {
	// Periodically request peers from connected peers.
	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()

	// small initial delay
	select {
	case <-n.ctx.Done():
		return
	case <-time.After(5 * time.Second):
	}

	for {
		select {
		case <-n.ctx.Done():
			return
		case <-ticker.C:
			n.requestPeersFromSome()
		}
	}
}

func (n *Node) requestPeersFromSome() {
	peers := n.snapshotConns()
	for i := 0; i < len(peers) && i < 8; i++ {
		conn := peers[i]
		go n.sendGetPeers(conn)
	}
}

func (n *Node) snapshotConns() []net.Conn {
	n.mu.RLock()
	defer n.mu.RUnlock()

	out := make([]net.Conn, 0, len(n.peers))
	for _, p := range n.peers {
		out = append(out, p.conn)
	}
	return out
}

func (n *Node) persistLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-n.ctx.Done():
			return
		case <-ticker.C:
			_ = n.persistOnce()
		}
	}
}

func (n *Node) persistOnce() error {
	// Persist banlist and known peers
	_ = n.banlist.Save()

	n.knownMu.RLock()
	peers := make([]StoredPeer, 0, len(n.knownPeers))
	for _, p := range n.knownPeers {
		peers = append(peers, p)
	}
	n.knownMu.RUnlock()

	return n.peerStore.Save(peers)
}

func (n *Node) pickDialCandidates(limit int) []string {
	if limit <= 0 {
		return nil
	}

	now := time.Now().UTC()
	n.knownMu.RLock()
	candidates := make([]string, 0, len(n.knownPeers))
	for addr := range n.knownPeers {
		if addr == "" {
			continue
		}
		if n.isConnectedTo(addr) {
			continue
		}
		if banned, _ := n.banlist.IsBanned(addr); banned {
			continue
		}
		if !n.canDial(addr, now) {
			continue
		}
		candidates = append(candidates, addr)
	}
	n.knownMu.RUnlock()

	// Shuffle a bit for diversity
	rand.Shuffle(len(candidates), func(i, j int) { candidates[i], candidates[j] = candidates[j], candidates[i] })

	if len(candidates) > limit {
		candidates = candidates[:limit]
	}
	return candidates
}

func (n *Node) canDial(addr string, now time.Time) bool {
	n.backoffMu.Lock()
	defer n.backoffMu.Unlock()

	b, ok := n.backoff[addr]
	if !ok {
		return true
	}
	return now.After(b.NextTryAt)
}

func (n *Node) recordDialFailure(addr string, err error) {
	n.backoffMu.Lock()
	defer n.backoffMu.Unlock()

	b := n.backoff[addr]
	b.Attempts++
	if err != nil {
		b.LastErr = err.Error()
	}

	// Exponential backoff with jitter, capped.
	base := 2 * time.Second
	max := 2 * time.Minute
	delay := base * time.Duration(1<<minInt(b.Attempts-1, 8)) // up to 256x
	if delay > max {
		delay = max
	}

	// jitter: 0.5x .. 1.5x
	j := 0.5 + rand.Float64()
	delay = time.Duration(float64(delay) * j)

	b.NextTryAt = time.Now().UTC().Add(delay)
	n.backoff[addr] = b

	// update stored peer error
	n.knownMu.Lock()
	p, ok := n.knownPeers[addr]
	if ok {
		p.LastError = b.LastErr
		p.SeenAt = time.Now().UTC()
		n.knownPeers[addr] = p
	}
	n.knownMu.Unlock()
}

func (n *Node) recordDialSuccess(addr string) {
	n.backoffMu.Lock()
	delete(n.backoff, addr)
	n.backoffMu.Unlock()

	n.knownMu.Lock()
	p, ok := n.knownPeers[addr]
	if ok {
		p.LastError = ""
		p.SeenAt = time.Now().UTC()
		n.knownPeers[addr] = p
	} else {
		n.knownPeers[addr] = StoredPeer{Addr: addr, SeenAt: time.Now().UTC(), Source: "learned"}
	}
	n.knownMu.Unlock()
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (n *Node) isConnectedTo(addr string) bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
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

	if banned, _ := n.banlist.IsBanned(addr); banned {
		return
	}

	dialer := &net.Dialer{Timeout: n.cfg.DialTimeout}
	conn, err := dialer.DialContext(n.ctx, "tcp", addr)
	if err != nil {
		n.recordDialFailure(addr, err)
		n.log.Debug("dial failed", "addr", addr, "err", err)
		return
	}

	if !n.tryRegisterPeer(conn, false) {
		_ = conn.Close()
		return
	}

	n.recordDialSuccess(addr)
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
		lastMsgAt:   time.Now().UTC(),
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
	p.lastMsgAt = time.Now().UTC()
	n.peers[key] = p

	// learn their address as a peer candidate (best effort)
	n.learnPeer(conn.RemoteAddr().String(), "learned")
}

func (n *Node) touchPeer(conn net.Conn) {
	n.mu.Lock()
	defer n.mu.Unlock()
	key := conn.RemoteAddr().String()
	p, ok := n.peers[key]
	if !ok {
		return
	}
	p.lastMsgAt = time.Now().UTC()
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

func (n *Node) learnPeer(addr, source string) {
	addr = sanitizeHelloString(addr)
	if addr == "" {
		return
	}
	if banned, _ := n.banlist.IsBanned(addr); banned {
		return
	}

	n.knownMu.Lock()
	p, ok := n.knownPeers[addr]
	if ok {
		p.SeenAt = time.Now().UTC()
		if p.Source == "" {
			p.Source = source
		}
		n.knownPeers[addr] = p
	} else {
		n.knownPeers[addr] = StoredPeer{Addr: addr, SeenAt: time.Now().UTC(), Source: source}
	}
	n.knownMu.Unlock()
}

func (n *Node) handleConn(conn net.Conn, inbound bool) {
	defer func() {
		_ = conn.Close()
		n.unregisterPeer(conn)
	}()

	br := bufio.NewReaderSize(conn, 64*1024)
	bw := bufio.NewWriterSize(conn, 64*1024)

	_ = conn.SetDeadline(time.Now().Add(n.cfg.HandshakeTimeout))

	// HELLO handshake
	if inbound {
		peerHello, err := n.readAndValidateHello(br)
		if err != nil {
			n.log.Warn("handshake failed (inbound read hello)", "remote", conn.RemoteAddr().String(), "err", err)
			n.penalize(conn.RemoteAddr().String(), err)
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
			n.penalize(conn.RemoteAddr().String(), err)
			return
		}
		n.setPeerHello(conn, peerHello)
	}

	_ = conn.SetDeadline(time.Time{})

	// After handshake, request peers once to seed discovery
	go n.sendGetPeers(conn)

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
		n.touchPeer(conn)

		switch f.Type {
		case MsgPing:
			_ = conn.SetWriteDeadline(time.Now().Add(DefaultWriteTimeout))
			if err := WriteFrame(bw, MsgPong, []byte("pong")); err != nil {
				return
			}
			if err := bw.Flush(); err != nil {
				return
			}

		case MsgGetPeers:
			// Reply with a slice of known peers (bounded).
			addrs := n.sampleKnownPeers(64)
			payload, err := EncodePeers(addrs)
			if err != nil {
				return
			}
			_ = conn.SetWriteDeadline(time.Now().Add(DefaultWriteTimeout))
			if err := WriteFrame(bw, MsgPeers, payload); err != nil {
				return
			}
			if err := bw.Flush(); err != nil {
				return
			}

		case MsgPeers:
			peers, err := DecodePeers(f.Payload)
			if err != nil {
				n.penalize(conn.RemoteAddr().String(), err)
				return
			}
			for _, a := range peers {
				n.learnPeer(a, "learned")
			}

		case MsgGoodbye:
			return

		default:
			n.log.Debug("ignored message", "remote", conn.RemoteAddr().String(), "type", f.Type)
		}
	}
}

func (n *Node) sampleKnownPeers(limit int) []string {
	if limit <= 0 {
		return nil
	}

	n.knownMu.RLock()
	addrs := make([]string, 0, len(n.knownPeers))
	for addr := range n.knownPeers {
		if addr == "" {
			continue
		}
		if n.isConnectedTo(addr) {
			continue
		}
		if banned, _ := n.banlist.IsBanned(addr); banned {
			continue
		}
		addrs = append(addrs, addr)
	}
	n.knownMu.RUnlock()

	rand.Shuffle(len(addrs), func(i, j int) { addrs[i], addrs[j] = addrs[j], addrs[i] })

	if len(addrs) > limit {
		addrs = addrs[:limit]
	}
	return addrs
}

func (n *Node) sendGetPeers(conn net.Conn) {
	bw := bufio.NewWriterSize(conn, 64*1024)
	_ = conn.SetWriteDeadline(time.Now().Add(DefaultWriteTimeout))
	// Payload is a single byte marker for future extensibility
	_ = WriteFrame(bw, MsgGetPeers, []byte{1})
	_ = bw.Flush()
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

	ourPub := n.cfg.IdentityPrivKey.Public().(ed25519.PublicKey)
	if vcrypto.ConstantTimeEqual(ourPub, h.PublicKey) {
		return Hello{}, errors.New("peer has same identity public key")
	}
	return h, nil
}

func (n *Node) penalize(addr string, err error) {
	// Conservative: ban only on protocol violations (decode/parse errors).
	// This can be expanded with a scoring system next.
	if addr == "" || err == nil {
		return
	}
	n.log.Warn("peer penalized", "addr", addr, "err", err.Error())
	n.banlist.Ban(addr, 10*time.Minute, "protocol violation")
}
