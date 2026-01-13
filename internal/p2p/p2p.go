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
	Verified     bool   `json:"verified"`
	Score        int    `json:"score"`
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

	knownMu    sync.RWMutex
	knownPeers map[string]StoredPeer

	backoffMu sync.Mutex
	backoff   map[string]dialBackoff

	banlist   *Banlist
	peerStore *PeerStore

	scorer *Scorer
}

type peerConn struct {
	conn        net.Conn
	inbound     bool
	connectedAt time.Time

	pubKey      ed25519.PublicKey
	nodeVersion string

	verified bool
	score    int

	lastMsgAt time.Time

	lim limiter
}

type limiter struct {
	mu         sync.Mutex
	tokens     float64
	last       time.Time
	rate       float64 // tokens per second
	burst      float64
	costPerMsg float64
}

func newLimiter() limiter {
	// Default: allow ~60 messages/min with burst.
	return limiter{
		tokens:     30,
		last:       time.Now().UTC(),
		rate:       1.0,  // 1 token/sec
		burst:      60.0, // max tokens
		costPerMsg: 1.0,
	}
}

func (l *limiter) allow() bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now().UTC()
	elapsed := now.Sub(l.last).Seconds()
	if elapsed > 0 {
		l.tokens += elapsed * l.rate
		if l.tokens > l.burst {
			l.tokens = l.burst
		}
		l.last = now
	}

	if l.tokens < l.costPerMsg {
		return false
	}
	l.tokens -= l.costPerMsg
	return true
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
		scorer: NewScorer(ScoreConfig{
			DecayInterval: 1 * time.Minute,
			DecayAmount:   1,
			BanThreshold:  10,
			BanDuration:   30 * time.Minute,
		}),
	}

	_ = n.banlist.Load()

	if peers, err := n.peerStore.Load(); err == nil {
		for _, p := range peers {
			n.knownPeers[p.Addr] = p
		}
	}

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
			Verified:     p.verified,
			Score:        p.score,
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

func (n *Node) discoveryLoop() {
	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()

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
	_ = n.banlist.Save()

	n.knownMu.RLock()
	peers := make([]StoredPeer, 0, len(n.knownPeers))
	for _, p := range n.knownPeers {
		peers = append(peers, p)
	}
	n.knownMu.RUnlock()

	return n.peerStore.Save(peers)
}

func (n *Node) fillOutbound() {
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

	base := 2 * time.Second
	max := 2 * time.Minute
	delay := base * time.Duration(1<<minInt(b.Attempts-1, 8))
	if delay > max {
		delay = max
	}
	j := 0.5 + rand.Float64()
	delay = time.Duration(float64(delay) * j)

	b.NextTryAt = time.Now().UTC().Add(delay)
	n.backoff[addr] = b

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
		lim:         newLimiter(),
	}

	n.log.Info("peer connected", "remote", key, "inbound", inbound, "peers", len(n.peers))
	return true
}

func (n *Node) updatePeer(conn net.Conn, fn func(p peerConn) peerConn) {
	n.mu.Lock()
	defer n.mu.Unlock()

	key := conn.RemoteAddr().String()
	p, ok := n.peers[key]
	if !ok {
		return
	}
	p = fn(p)
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

func (n *Node) requestPeersFromSome() {
	conns := n.snapshotConns()
	for i := 0; i < len(conns) && i < 8; i++ {
		conn := conns[i]
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

func (n *Node) sendGetPeers(conn net.Conn) {
	bw := bufio.NewWriterSize(conn, 64*1024)
	_ = conn.SetWriteDeadline(time.Now().Add(DefaultWriteTimeout))
	_ = WriteFrame(bw, MsgGetPeers, []byte{1})
	_ = bw.Flush()
}

func (n *Node) handleConn(conn net.Conn, inbound bool) {
	defer func() {
		_ = conn.Close()
		n.unregisterPeer(conn)
	}()

	br := bufio.NewReaderSize(conn, 64*1024)
	bw := bufio.NewWriterSize(conn, 64*1024)

	_ = conn.SetDeadline(time.Now().Add(n.cfg.HandshakeTimeout))

	// HELLO exchange
	var peerHello Hello
	var err error
	if inbound {
		peerHello, err = n.readAndValidateHello(br)
		if err != nil {
			n.penalize(conn.RemoteAddr().String(), 3, "hello invalid: "+err.Error())
			return
		}
		if err := n.writeHello(bw); err != nil {
			return
		}
		if err := bw.Flush(); err != nil {
			return
		}
	} else {
		if err := n.writeHello(bw); err != nil {
			return
		}
		if err := bw.Flush(); err != nil {
			return
		}
		peerHello, err = n.readAndValidateHello(br)
		if err != nil {
			n.penalize(conn.RemoteAddr().String(), 3, "hello invalid: "+err.Error())
			return
		}
	}

	// Store peer hello
	n.updatePeer(conn, func(p peerConn) peerConn {
		p.pubKey = peerHello.PublicKey
		p.nodeVersion = peerHello.NodeVersion
		p.lastMsgAt = time.Now().UTC()
		p.score = n.scorer.Get(conn.RemoteAddr().String())
		return p
	})
	n.learnPeer(conn.RemoteAddr().String(), "learned")

	// Challenge-response: prove the peer controls their announced public key.
	verified, verr := n.performChallengeHandshake(conn, br, bw, peerHello.PublicKey)
	if verr != nil || !verified {
		n.penalize(conn.RemoteAddr().String(), 5, "challenge failed: "+safeErr(verr))
		return
	}
	n.updatePeer(conn, func(p peerConn) peerConn { p.verified = true; return p })

	_ = conn.SetDeadline(time.Time{})

	// Seed discovery
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

		// Rate limiting (per-connection)
		allowed := true
		n.updatePeer(conn, func(p peerConn) peerConn {
			if !p.lim.allow() {
				allowed = false
			}
			p.lastMsgAt = time.Now().UTC()
			return p
		})
		if !allowed {
			n.penalize(conn.RemoteAddr().String(), 2, "rate limit exceeded")
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

		case MsgGetPeers:
			addrs := n.sampleKnownPeers(64)
			payload, err := EncodePeers(addrs)
			if err != nil {
				n.penalize(conn.RemoteAddr().String(), 2, "encode peers failed")
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
				n.penalize(conn.RemoteAddr().String(), 3, "decode peers failed: "+err.Error())
				return
			}
			for _, a := range peers {
				n.learnPeer(a, "learned")
			}

		case MsgChallenge:
			// Respond to their challenge anytime after handshake.
			if len(f.Payload) != challengeSize {
				n.penalize(conn.RemoteAddr().String(), 3, "invalid challenge size")
				return
			}
			var c [challengeSize]byte
			copy(c[:], f.Payload)
			resp, err := SignChallenge(n.cfg.IdentityPrivKey, n.cfg.NetworkID, c)
			if err != nil {
				return
			}
			_ = conn.SetWriteDeadline(time.Now().Add(DefaultWriteTimeout))
			if err := WriteFrame(bw, MsgChallengeResp, resp); err != nil {
				return
			}
			if err := bw.Flush(); err != nil {
				return
			}

		case MsgChallengeResp:
			// Unexpected during steady-state; treat as suspicious.
			n.penalize(conn.RemoteAddr().String(), 2, "unexpected challenge response")
			return

		case MsgGoodbye:
			return

		default:
			n.log.Debug("ignored message", "remote", conn.RemoteAddr().String(), "type", f.Type)
		}
	}
}

func safeErr(err error) string {
	if err == nil {
		return "unknown"
	}
	return err.Error()
}

func (n *Node) performChallengeHandshake(conn net.Conn, br *bufio.Reader, bw *bufio.Writer, peerPub ed25519.PublicKey) (bool, error) {
	// Steps:
	// 1) Send challenge to peer.
	// 2) Read frames until:
	//    - We receive a valid ChallengeResp for our challenge (success), OR
	//    - Timeout / invalid response / too many frames.
	//
	// Also: if peer sends us a challenge in the middle, we respond.

	if len(peerPub) != ed25519.PublicKeySize {
		return false, errors.New("peer pubkey invalid")
	}

	chal, err := NewChallenge()
	if err != nil {
		return false, err
	}

	if err := WriteFrame(bw, MsgChallenge, chal[:]); err != nil {
		return false, err
	}
	if err := bw.Flush(); err != nil {
		return false, err
	}

	// Handshake deadline bound
	_ = conn.SetReadDeadline(time.Now().Add(n.cfg.HandshakeTimeout))

	maxFrames := 16
	for i := 0; i < maxFrames; i++ {
		f, err := ReadFrame(br)
		if err != nil {
			return false, err
		}

		switch f.Type {
		case MsgChallenge:
			if len(f.Payload) != challengeSize {
				return false, errors.New("invalid challenge size")
			}
			var c [challengeSize]byte
			copy(c[:], f.Payload)

			resp, err := SignChallenge(n.cfg.IdentityPrivKey, n.cfg.NetworkID, c)
			if err != nil {
				return false, err
			}
			_ = conn.SetWriteDeadline(time.Now().Add(DefaultWriteTimeout))
			if err := WriteFrame(bw, MsgChallengeResp, resp); err != nil {
				return false, err
			}
			if err := bw.Flush(); err != nil {
				return false, err
			}

		case MsgChallengeResp:
			if err := VerifyChallengeResp(peerPub, n.cfg.NetworkID, f.Payload, chal); err != nil {
				return false, err
			}
			return true, nil

		case MsgPing:
			_ = conn.SetWriteDeadline(time.Now().Add(DefaultWriteTimeout))
			_ = WriteFrame(bw, MsgPong, []byte("pong"))
			_ = bw.Flush()

		case MsgHello:
			// HELLO should not repeat after initial exchange
			return false, errors.New("unexpected hello during challenge")

		default:
			// Ignore other frames during handshake window
		}
	}

	return false, errors.New("challenge handshake exceeded frame limit")
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

func (n *Node) penalize(addr string, points int, reason string) {
	if addr == "" || points <= 0 {
		return
	}

	score, ban, banFor := n.scorer.Add(addr, points)

	// Update peer score if connected
	n.mu.Lock()
	if p, ok := n.peers[addr]; ok {
		p.score = score
		n.peers[addr] = p
	}
	n.mu.Unlock()

	n.log.Warn("peer penalized", "addr", addr, "points", points, "score", score, "reason", reason)

	if ban {
		n.banlist.Ban(addr, banFor, reason)
		_ = n.banlist.Save()
		n.log.Warn("peer banned", "addr", addr, "for", banFor.String(), "reason", reason)

		// If currently connected, close it
		n.mu.RLock()
		p, ok := n.peers[addr]
		n.mu.RUnlock()
		if ok {
			_ = p.conn.Close()
		}
	}
}
