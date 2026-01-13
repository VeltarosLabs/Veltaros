package p2p

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"time"

	vcrypto "github.com/VeltarosLabs/Veltaros/internal/crypto"
	"github.com/VeltarosLabs/Veltaros/pkg/version"
)

const (
	MaxFrameSize = 1 << 20 // 1 MiB

	DefaultReadTimeout  = 7 * time.Second
	DefaultWriteTimeout = 7 * time.Second

	ProtocolVersion uint16 = 1
)

type MessageType uint8

const (
	MsgUnknown MessageType = 0

	MsgHello MessageType = 1
	MsgPing  MessageType = 2
	MsgPong  MessageType = 3

	MsgGoodbye MessageType = 4

	MsgGetPeers MessageType = 10
	MsgPeers    MessageType = 11

	// Challenge-response proof of key ownership
	MsgChallenge     MessageType = 20
	MsgChallengeResp MessageType = 21
)

type Frame struct {
	Type    MessageType
	Payload []byte
}

func WriteFrame(w io.Writer, msgType MessageType, payload []byte) error {
	if w == nil {
		return errors.New("writer is nil")
	}
	if msgType == MsgUnknown {
		return errors.New("message type is unknown")
	}

	length := 1 + len(payload)
	if length <= 1 {
		return errors.New("payload must not be empty")
	}
	if length > MaxFrameSize {
		return fmt.Errorf("frame too large: %d > %d", length, MaxFrameSize)
	}

	header := make([]byte, 5)
	binary.LittleEndian.PutUint32(header[:4], uint32(length))
	header[4] = byte(msgType)

	if _, err := w.Write(header); err != nil {
		return err
	}
	if _, err := w.Write(payload); err != nil {
		return err
	}
	return nil
}

func ReadFrame(r io.Reader) (Frame, error) {
	if r == nil {
		return Frame{}, errors.New("reader is nil")
	}

	var lenBuf [4]byte
	if _, err := io.ReadFull(r, lenBuf[:]); err != nil {
		return Frame{}, err
	}

	length := int(binary.LittleEndian.Uint32(lenBuf[:]))
	if length <= 1 {
		return Frame{}, fmt.Errorf("invalid frame length: %d", length)
	}
	if length > MaxFrameSize {
		return Frame{}, fmt.Errorf("frame length exceeds limit: %d > %d", length, MaxFrameSize)
	}

	buf := make([]byte, length)
	if _, err := io.ReadFull(r, buf); err != nil {
		return Frame{}, err
	}

	msgType := MessageType(buf[0])
	payload := buf[1:]

	if msgType == MsgUnknown {
		return Frame{}, errors.New("invalid message type: unknown")
	}
	if len(payload) == 0 {
		return Frame{}, errors.New("invalid payload: empty")
	}

	return Frame{Type: msgType, Payload: payload}, nil
}

// ---- HELLO handshake payload (v1) ----
// Payload fields (binary, little-endian for ints):
// [2] protocolVersion (uint16)
// [2] networkIDLen (uint16) + [N] networkID bytes (utf-8, <= 64)
// [2] nodeVersionLen (uint16) + [M] nodeVersion bytes (utf-8, <= 64)
// [8] unixTimeSec (int64)
// [32] nonce
// [32] ed25519 public key

const (
	maxHelloString = 64
	helloNonceSize = 32
)

type Hello struct {
	ProtocolVersion uint16
	NetworkID       string
	NodeVersion     string
	TimeUnixSec     int64
	Nonce           [helloNonceSize]byte
	PublicKey       ed25519.PublicKey
}

func NewHello(networkID string, identityPub ed25519.PublicKey) (Hello, error) {
	if len(identityPub) != ed25519.PublicKeySize {
		return Hello{}, errors.New("invalid identity public key size")
	}
	nid := sanitizeHelloString(networkID)
	if nid == "" {
		return Hello{}, errors.New("networkID must not be empty")
	}
	if len(nid) > maxHelloString {
		return Hello{}, errors.New("networkID too long")
	}

	var nonce [helloNonceSize]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return Hello{}, err
	}

	v := version.Get()
	nodeVer := sanitizeHelloString(v.Version)
	if nodeVer == "" {
		nodeVer = "dev"
	}
	if len(nodeVer) > maxHelloString {
		nodeVer = nodeVer[:maxHelloString]
	}

	return Hello{
		ProtocolVersion: ProtocolVersion,
		NetworkID:       nid,
		NodeVersion:     nodeVer,
		TimeUnixSec:     time.Now().UTC().Unix(),
		Nonce:           nonce,
		PublicKey:       identityPub,
	}, nil
}

func (h Hello) Encode() ([]byte, error) {
	if h.ProtocolVersion == 0 {
		return nil, errors.New("protocol version must be set")
	}
	if len(h.PublicKey) != ed25519.PublicKeySize {
		return nil, errors.New("invalid identity public key size")
	}
	nid := sanitizeHelloString(h.NetworkID)
	nver := sanitizeHelloString(h.NodeVersion)
	if nid == "" || nver == "" {
		return nil, errors.New("hello strings must not be empty")
	}
	if len(nid) > maxHelloString || len(nver) > maxHelloString {
		return nil, errors.New("hello string too long")
	}

	buf := make([]byte, 0, 2+2+len(nid)+2+len(nver)+8+helloNonceSize+ed25519.PublicKeySize)

	tmp2 := make([]byte, 2)
	binary.LittleEndian.PutUint16(tmp2, h.ProtocolVersion)
	buf = append(buf, tmp2...)

	binary.LittleEndian.PutUint16(tmp2, uint16(len(nid)))
	buf = append(buf, tmp2...)
	buf = append(buf, []byte(nid)...)

	binary.LittleEndian.PutUint16(tmp2, uint16(len(nver)))
	buf = append(buf, tmp2...)
	buf = append(buf, []byte(nver)...)

	tmp8 := make([]byte, 8)
	binary.LittleEndian.PutUint64(tmp8, uint64(h.TimeUnixSec))
	buf = append(buf, tmp8...)

	buf = append(buf, h.Nonce[:]...)
	buf = append(buf, h.PublicKey...)

	return buf, nil
}

func DecodeHello(b []byte) (Hello, error) {
	if len(b) < 2+2+1+2+1+8+helloNonceSize+ed25519.PublicKeySize {
		return Hello{}, errors.New("hello payload too short")
	}

	off := 0
	readU16 := func() (uint16, error) {
		if off+2 > len(b) {
			return 0, io.ErrUnexpectedEOF
		}
		v := binary.LittleEndian.Uint16(b[off : off+2])
		off += 2
		return v, nil
	}
	readBytes := func(n int) ([]byte, error) {
		if n < 0 || off+n > len(b) {
			return nil, io.ErrUnexpectedEOF
		}
		out := b[off : off+n]
		off += n
		return out, nil
	}
	readI64 := func() (int64, error) {
		if off+8 > len(b) {
			return 0, io.ErrUnexpectedEOF
		}
		u := binary.LittleEndian.Uint64(b[off : off+8])
		off += 8
		return int64(u), nil
	}

	pv, err := readU16()
	if err != nil {
		return Hello{}, err
	}

	nidLen, err := readU16()
	if err != nil {
		return Hello{}, err
	}
	if nidLen == 0 || nidLen > maxHelloString {
		return Hello{}, errors.New("invalid networkID length")
	}
	nidBytes, err := readBytes(int(nidLen))
	if err != nil {
		return Hello{}, err
	}

	nverLen, err := readU16()
	if err != nil {
		return Hello{}, err
	}
	if nverLen == 0 || nverLen > maxHelloString {
		return Hello{}, errors.New("invalid nodeVersion length")
	}
	nverBytes, err := readBytes(int(nverLen))
	if err != nil {
		return Hello{}, err
	}

	tsec, err := readI64()
	if err != nil {
		return Hello{}, err
	}

	nonceBytes, err := readBytes(helloNonceSize)
	if err != nil {
		return Hello{}, err
	}
	var nonce [helloNonceSize]byte
	copy(nonce[:], nonceBytes)

	pub, err := readBytes(ed25519.PublicKeySize)
	if err != nil {
		return Hello{}, err
	}
	pubKey := ed25519.PublicKey(make([]byte, ed25519.PublicKeySize))
	copy(pubKey, pub)

	if off != len(b) {
		return Hello{}, errors.New("hello payload has trailing bytes")
	}

	return Hello{
		ProtocolVersion: pv,
		NetworkID:       string(nidBytes),
		NodeVersion:     string(nverBytes),
		TimeUnixSec:     tsec,
		Nonce:           nonce,
		PublicKey:       pubKey,
	}, nil
}

type HelloValidation struct {
	NetworkID      string
	MaxClockSkew   time.Duration
	RequireNonZero bool
}

func ValidateHello(h Hello, rules HelloValidation) error {
	if h.ProtocolVersion != ProtocolVersion {
		return fmt.Errorf("protocol version mismatch: got %d want %d", h.ProtocolVersion, ProtocolVersion)
	}
	if sanitizeHelloString(h.NetworkID) == "" {
		return errors.New("networkID is empty")
	}
	if rules.NetworkID != "" && h.NetworkID != rules.NetworkID {
		return fmt.Errorf("networkID mismatch: got %q want %q", h.NetworkID, rules.NetworkID)
	}
	if len(h.PublicKey) != ed25519.PublicKeySize {
		return errors.New("invalid public key size")
	}
	if rules.RequireNonZero {
		zero := [helloNonceSize]byte{}
		if vcrypto.ConstantTimeEqual(h.Nonce[:], zero[:]) {
			return errors.New("nonce must be non-zero")
		}
	}
	if rules.MaxClockSkew > 0 {
		now := time.Now().UTC()
		t := time.Unix(h.TimeUnixSec, 0).UTC()
		d := now.Sub(t)
		if d < 0 {
			d = -d
		}
		if d > rules.MaxClockSkew {
			return fmt.Errorf("hello timestamp skew too large: %s", d)
		}
	}
	return nil
}

func PublicKeyHex(pub ed25519.PublicKey) string {
	return hex.EncodeToString(pub)
}

func sanitizeHelloString(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t' || s[0] == '\n' || s[0] == '\r') {
		s = s[1:]
	}
	for len(s) > 0 {
		last := s[len(s)-1]
		if last == ' ' || last == '\t' || last == '\n' || last == '\r' {
			s = s[:len(s)-1]
			continue
		}
		break
	}
	b := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] != 0 {
			b = append(b, s[i])
		}
	}
	return string(b)
}

// ---- PEERS payload ----
// [2] count (uint16)
// repeated count times: [2] addrLen (uint16) + [addrLen] addr bytes (utf-8), addrLen <= 128

const maxPeerAddrLen = 128

func EncodePeers(addrs []string) ([]byte, error) {
	if len(addrs) > 4096 {
		addrs = addrs[:4096]
	}
	buf := make([]byte, 0, 2+len(addrs)*16)
	tmp2 := make([]byte, 2)
	binary.LittleEndian.PutUint16(tmp2, uint16(len(addrs)))
	buf = append(buf, tmp2...)

	for _, a := range addrs {
		a = sanitizeHelloString(a)
		if a == "" || len(a) > maxPeerAddrLen {
			continue
		}
		binary.LittleEndian.PutUint16(tmp2, uint16(len(a)))
		buf = append(buf, tmp2...)
		buf = append(buf, []byte(a)...)
	}
	return buf, nil
}

func DecodePeers(b []byte) ([]string, error) {
	if len(b) < 2 {
		return nil, errors.New("peers payload too short")
	}
	off := 0
	count := int(binary.LittleEndian.Uint16(b[off : off+2]))
	off += 2
	if count < 0 || count > 4096 {
		return nil, errors.New("invalid peers count")
	}

	out := make([]string, 0, count)
	seen := make(map[string]struct{}, count)

	for i := 0; i < count; i++ {
		if off+2 > len(b) {
			return nil, io.ErrUnexpectedEOF
		}
		n := int(binary.LittleEndian.Uint16(b[off : off+2]))
		off += 2
		if n <= 0 || n > maxPeerAddrLen {
			return nil, errors.New("invalid peer addr length")
		}
		if off+n > len(b) {
			return nil, io.ErrUnexpectedEOF
		}
		addr := string(b[off : off+n])
		off += n
		addr = sanitizeHelloString(addr)
		if addr == "" {
			continue
		}
		if _, ok := seen[addr]; ok {
			continue
		}
		seen[addr] = struct{}{}
		out = append(out, addr)
	}

	if off != len(b) {
		return nil, errors.New("peers payload has trailing bytes")
	}

	return out, nil
}

// ---- Challenge-response signing ----
//
// Challenge payload: [32] random bytes
// Response payload: [32] challenge bytes + [64] ed25519 signature
//
// Signature message = SHA256( "veltaros-p2p-challenge" || networkID || challengeBytes )
//
// This binds proof to a specific network ID and prevents cross-network reuse.

const (
	challengeSize     = 32
	challengeSigSize  = ed25519.SignatureSize
	challengeRespSize = challengeSize + challengeSigSize
)

func NewChallenge() ([challengeSize]byte, error) {
	var c [challengeSize]byte
	_, err := rand.Read(c[:])
	return c, err
}

func ChallengeMessage(networkID string, challenge [challengeSize]byte) [32]byte {
	// domain separation + bind network ID
	domain := []byte("veltaros-p2p-challenge")
	msg := make([]byte, 0, len(domain)+len(networkID)+challengeSize)
	msg = append(msg, domain...)
	msg = append(msg, []byte(networkID)...)
	msg = append(msg, challenge[:]...)
	return vcrypto.Sha256(msg)
}

func SignChallenge(identityPriv ed25519.PrivateKey, networkID string, challenge [challengeSize]byte) ([]byte, error) {
	if len(identityPriv) != ed25519.PrivateKeySize {
		return nil, errors.New("invalid identity private key size")
	}
	h := ChallengeMessage(networkID, challenge)
	sig := ed25519.Sign(identityPriv, h[:])

	out := make([]byte, 0, challengeRespSize)
	out = append(out, challenge[:]...)
	out = append(out, sig...)
	return out, nil
}

func VerifyChallengeResp(pub ed25519.PublicKey, networkID string, resp []byte, expected [challengeSize]byte) error {
	if len(pub) != ed25519.PublicKeySize {
		return errors.New("invalid public key size")
	}
	if len(resp) != challengeRespSize {
		return errors.New("invalid challenge response size")
	}

	var gotChallenge [challengeSize]byte
	copy(gotChallenge[:], resp[:challengeSize])

	if !vcrypto.ConstantTimeEqual(gotChallenge[:], expected[:]) {
		return errors.New("challenge mismatch")
	}

	sig := resp[challengeSize:]
	h := ChallengeMessage(networkID, expected)
	if !ed25519.Verify(pub, h[:], sig) {
		return errors.New("invalid challenge signature")
	}
	return nil
}
