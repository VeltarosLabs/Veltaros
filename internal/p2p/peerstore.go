package p2p

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type StoredPeer struct {
	Addr      string    `json:"addr"`
	SeenAt    time.Time `json:"seenAt"`
	Source    string    `json:"source"` // bootstrap|learned|manual
	LastError string    `json:"lastError,omitempty"`
}

type PeerStore struct {
	path string
}

func NewPeerStore(path string) *PeerStore {
	return &PeerStore{path: filepath.Clean(path)}
}

func (ps *PeerStore) Load() ([]StoredPeer, error) {
	raw, err := os.ReadFile(ps.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []StoredPeer{}, nil
		}
		return nil, err
	}
	var peers []StoredPeer
	if err := json.Unmarshal(raw, &peers); err != nil {
		return nil, err
	}
	// Normalize
	out := make([]StoredPeer, 0, len(peers))
	seen := make(map[string]struct{}, len(peers))
	for _, p := range peers {
		if p.Addr == "" {
			continue
		}
		if _, ok := seen[p.Addr]; ok {
			continue
		}
		seen[p.Addr] = struct{}{}
		out = append(out, p)
	}
	return out, nil
}

func (ps *PeerStore) Save(peers []StoredPeer) error {
	if err := os.MkdirAll(filepath.Dir(ps.path), 0o700); err != nil {
		return err
	}

	// Stable ordering
	sort.Slice(peers, func(i, j int) bool {
		return peers[i].Addr < peers[j].Addr
	})

	data, err := json.MarshalIndent(peers, "", "  ")
	if err != nil {
		return err
	}

	tmp := ps.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, ps.path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	_ = os.Chmod(ps.path, 0o600)
	return nil
}
