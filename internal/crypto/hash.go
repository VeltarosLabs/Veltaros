package crypto

import (
	"crypto/sha256"
	"encoding/hex"
)

func Sha256(data []byte) [32]byte {
	return sha256.Sum256(data)
}

func DoubleSha256(data []byte) [32]byte {
	first := sha256.Sum256(data)
	return sha256.Sum256(first[:])
}

func Hex32(h [32]byte) string {
	return hex.EncodeToString(h[:])
}
