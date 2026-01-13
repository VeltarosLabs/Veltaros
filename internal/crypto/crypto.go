package crypto

import (
	"crypto/subtle"
	"encoding/hex"
	"errors"
)

func ConstantTimeEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	return subtle.ConstantTimeCompare(a, b) == 1
}

func DecodeHex(s string) ([]byte, error) {
	b, err := hex.DecodeString(s)
	if err != nil {
		return nil, err
	}
	if len(b) == 0 {
		return nil, errors.New("decoded hex is empty")
	}
	return b, nil
}
