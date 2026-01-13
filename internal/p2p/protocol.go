package p2p

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"time"
)

// Protocol notes (production-minded baseline):
// - We use a length-prefixed framing format to avoid ambiguous reads and to bound memory use.
// - Handshake/protocol negotiation will be added next; for now we only provide safe framing helpers.
//
// Frame format:
//   [4 bytes length LE][1 byte msgType][N bytes payload]
// where length = 1 + len(payload), and must be <= MaxFrameSize.

const (
	// MaxFrameSize prevents memory abuse. Increase only when protocol demands it.
	MaxFrameSize = 1 << 20 // 1 MiB

	// ReadTimeout/WriteTimeout are used by callers at net.Conn level.
	DefaultReadTimeout  = 7 * time.Second
	DefaultWriteTimeout = 7 * time.Second
)

type MessageType uint8

const (
	MsgUnknown MessageType = 0

	// Reserved for next phases:
	MsgHello   MessageType = 1
	MsgPing    MessageType = 2
	MsgPong    MessageType = 3
	MsgGoodbye MessageType = 4

	MsgGetPeers MessageType = 10
	MsgPeers    MessageType = 11
)

// Frame represents a decoded message frame.
type Frame struct {
	Type    MessageType
	Payload []byte
}

// WriteFrame writes a single framed message to w.
// It does not flush; flushing is the caller's responsibility when using buffered writers.
func WriteFrame(w io.Writer, msgType MessageType, payload []byte) error {
	if w == nil {
		return errors.New("writer is nil")
	}
	if msgType == MsgUnknown {
		return errors.New("message type is unknown")
	}

	// length includes the 1 byte type field
	length := 1 + len(payload)
	if length <= 1 {
		return errors.New("payload must not be empty")
	}
	if length > MaxFrameSize {
		return fmt.Errorf("frame too large: %d > %d", length, MaxFrameSize)
	}

	// 4 bytes length prefix + 1 byte type + payload
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

// ReadFrame reads a single framed message from r.
// It validates the size bound to prevent memory abuse.
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

	// Read type + payload
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
