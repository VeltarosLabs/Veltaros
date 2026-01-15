package blockchain

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type NonceSnapshot struct {
	Addr      string    `json:"addr"`
	LastNonce uint64    `json:"lastNonce"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type NonceStore struct {
	path string
}

func NewNonceStore(path string) *NonceStore {
	return &NonceStore{path: filepath.Clean(path)}
}

func (s *NonceStore) Load() ([]NonceSnapshot, error) {
	raw, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []NonceSnapshot{}, nil
		}
		return nil, err
	}

	var snaps []NonceSnapshot
	if err := json.Unmarshal(raw, &snaps); err != nil {
		return nil, err
	}

	// Normalize and de-dup by addr (keep highest nonce)
	out := make([]NonceSnapshot, 0, len(snaps))
	seen := make(map[string]NonceSnapshot, len(snaps))

	for _, sn := range snaps {
		if sn.Addr == "" || sn.LastNonce == 0 {
			continue
		}
		if prev, ok := seen[sn.Addr]; !ok || sn.LastNonce > prev.LastNonce {
			seen[sn.Addr] = sn
		}
	}

	for _, v := range seen {
		out = append(out, v)
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Addr < out[j].Addr })
	return out, nil
}

func (s *NonceStore) Save(snaps []NonceSnapshot) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}

	sort.Slice(snaps, func(i, j int) bool { return snaps[i].Addr < snaps[j].Addr })

	data, err := json.MarshalIndent(snaps, "", "  ")
	if err != nil {
		return err
	}

	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, s.path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	_ = os.Chmod(s.path, 0o600)
	return nil
}
