package p2p

import (
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"
)

type IdentityRecord struct {
	PublicKeyHex string    `json:"publicKeyHex"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

func EnsureIdentityRecord(path string, priv ed25519.PrivateKey) error {
	if len(priv) != ed25519.PrivateKeySize {
		return errors.New("invalid identity private key size")
	}
	path = filepath.Clean(path)

	now := time.Now().UTC()
	pub := priv.Public().(ed25519.PublicKey)

	// If exists and matches, leave it.
	if raw, err := os.ReadFile(path); err == nil {
		var rec IdentityRecord
		if json.Unmarshal(raw, &rec) == nil && rec.PublicKeyHex == hex.EncodeToString(pub) {
			// Touch UpdatedAt occasionally not required; keep stable.
			return nil
		}
	}

	rec := IdentityRecord{
		PublicKeyHex: hex.EncodeToString(pub),
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return err
	}

	tmp := path + ".tmp"
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
