package wallet

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	vcrypto "github.com/VeltarosLabs/Veltaros/internal/crypto"
)

type Keypair struct {
	PublicKey  ed25519.PublicKey
	PrivateKey ed25519.PrivateKey
}

// Address format (simple, deterministic, portable):
// addr := hex( pubKeyHash20 || checksum4 )
// pubKeyHash20 = first 20 bytes of SHA256(pubKey)
// checksum4 = first 4 bytes of double-SHA256(pubKeyHash20)
func AddressFromPublicKey(pub ed25519.PublicKey) (string, error) {
	if len(pub) != ed25519.PublicKeySize {
		return "", errors.New("invalid ed25519 public key size")
	}

	h := vcrypto.Sha256(pub)
	pubHash20 := h[:20]
	check := vcrypto.DoubleSha256(pubHash20)
	addrBytes := make([]byte, 0, 24)
	addrBytes = append(addrBytes, pubHash20...)
	addrBytes = append(addrBytes, check[:4]...)

	return hex.EncodeToString(addrBytes), nil
}

func ValidateAddress(addr string) bool {
	addr = strings.TrimSpace(addr)
	b, err := hex.DecodeString(addr)
	if err != nil {
		return false
	}
	if len(b) != 24 {
		return false
	}
	pubHash20 := b[:20]
	want := vcrypto.DoubleSha256(pubHash20)
	got := b[20:24]
	return vcrypto.ConstantTimeEqual(got, want[:4])
}

func Generate() (Keypair, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return Keypair{}, err
	}
	return Keypair{PublicKey: pub, PrivateKey: priv}, nil
}

// File format: raw ed25519 private key bytes (64 bytes) hex-encoded.
// Permissions: 0600.
// Note: On Windows, chmod behavior differs, but we still attempt to lock down perms.
func SavePrivateKeyHex(path string, priv ed25519.PrivateKey) error {
	if len(priv) != ed25519.PrivateKeySize {
		return errors.New("invalid ed25519 private key size")
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}

	tmp := path + ".tmp"
	data := []byte(hex.EncodeToString(priv))

	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}

	_ = os.Chmod(path, 0o600)
	return nil
}

func LoadPrivateKeyHex(path string) (ed25519.PrivateKey, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	s := strings.TrimSpace(string(raw))
	b, err := hex.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("invalid key file hex: %w", err)
	}
	if len(b) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("invalid private key size: got %d want %d", len(b), ed25519.PrivateKeySize)
	}
	return ed25519.PrivateKey(b), nil
}
