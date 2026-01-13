package crypto

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"
)

type PrivateKey = ed25519.PrivateKey
type PublicKey = ed25519.PublicKey

func GenerateEd25519Keypair() (PublicKey, PrivateKey, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	return pub, priv, nil
}

func SignEd25519(priv PrivateKey, msg []byte) ([]byte, error) {
	if len(priv) != ed25519.PrivateKeySize {
		return nil, errors.New("invalid ed25519 private key size")
	}
	sig := ed25519.Sign(priv, msg)
	return sig, nil
}

func VerifyEd25519(pub PublicKey, msg, sig []byte) bool {
	if len(pub) != ed25519.PublicKeySize {
		return false
	}
	if len(sig) != ed25519.SignatureSize {
		return false
	}
	return ed25519.Verify(pub, msg, sig)
}
