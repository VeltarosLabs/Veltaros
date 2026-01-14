package blockchain

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"

	vcrypto "github.com/VeltarosLabs/Veltaros/internal/crypto"
)

// AddressFromEd25519PublicKeyHex derives the Veltaros address from the raw ed25519 public key hex (32 bytes):
// pubHash20 = sha256(pubKey)[:20]
// checksum4 = doubleSha256(pubHash20)[:4]
// address = hex(pubHash20||checksum4)
func AddressFromEd25519PublicKeyHex(pubKeyHex string) (string, error) {
	pub, err := hex.DecodeString(pubKeyHex)
	if err != nil {
		return "", errors.New("invalid public key hex")
	}
	if len(pub) != 32 {
		return "", errors.New("invalid public key length")
	}

	h := sha256.Sum256(pub)
	pubHash20 := h[:20]

	check := vcrypto.DoubleSha256(pubHash20)
	addrBytes := make([]byte, 0, 24)
	addrBytes = append(addrBytes, pubHash20...)
	addrBytes = append(addrBytes, check[:4]...)

	return hex.EncodeToString(addrBytes), nil
}
