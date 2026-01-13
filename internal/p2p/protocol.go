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

// Production-minded protocol baseline:
// - Length-prefixed framing with strict max size.
// - Versioned HELLO handshake with network ID and node identity.
// - Replay-resistant nonce + timestamp.
// - Ed25519 identity key (public key shared in HELLO).
// - Handshake has strict deadlines and validation.

const (
	MaxFrameSize = 1 << 20 // 1 MiB

	DefaultReadTimeout  = 7 * time.Second
	DefaultWriteTimeout = 7 * time.Second

	ProtocolVersion uint16 = 1
)

// MessageType identifies message category.
type MessageType uint8

const (
	MsgUnknown MessageType = 0

	MsgHello   MessageType = 1
	MsgPing    MessageType = 2
	MsgPong    MessageType = 3
	MsgGoodbye MessageType = 4

	MsgGetPeers MessageType = 10
	MsgPeers    MessageType = 11
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
//
// Payload fields (binary, little-endian for ints):
// [2] protocolVersion (uint16)
// [2] networkIDLen (uint16) + [N] networkID bytes (utf-8, <= 64)
// [2] nodeVersionLen (uint16) + [M] nodeVersion bytes (utf-8, <= 64)
// [8] unixTimeSec (int64)
// [32] nonce
// [32] ed25519 public key
//
// Notes:
// - No signature yet; next phase will add signed hello or challenge-response.
// - Validation: protocolVersion match, networkID match, timestamp skew bounded, nonce non-zero, pubkey length exact.

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
		// Strict parsing.
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

func sanitizeHelloString(s string) string {
	// Keep it simple and deterministic. Trim whitespace and remove NUL.
	s = string([]byte(s))
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
	s = removeNulls(s)
	return s
}

func removeNulls(s string) string {
	b := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] != 0 {
			b = append(b, s[i])
		}
	}
	return string(b)
}

func PublicKeyHex(pub ed25519.PublicKey) string {
	return hex.EncodeToString(pub)
}
