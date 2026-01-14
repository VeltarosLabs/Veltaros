package blockchain

import (
	"encoding/hex"
	"errors"

	vcrypto "github.com/VeltarosLabs/Veltaros/internal/crypto"
)

const (
	AddressLenBytes = 24 // 20 hash + 4 checksum
)

// ValidateAddress checks hex(pubHash20||checksum4) where checksum4 = doubleSha256(pubHash20)[:4]
func ValidateAddress(addr string) error {
	b, err := hex.DecodeString(addr)
	if err != nil {
		return errors.New("invalid address hex")
	}
	if len(b) != AddressLenBytes {
		return errors.New("invalid address length")
	}

	pubHash20 := b[:20]
	got := b[20:24]
	want := vcrypto.DoubleSha256(pubHash20)

	if !vcrypto.ConstantTimeEqual(got, want[:4]) {
		return errors.New("invalid address checksum")
	}
	return nil
}
