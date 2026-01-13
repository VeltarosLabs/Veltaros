package p2p

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type BanEntry struct {
	Addr      string    `json:"addr"`
	Until     time.Time `json:"until"`
	Reason    string    `json:"reason"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type Banlist struct {
	mu    sync.RWMutex
	path  string
	items map[string]BanEntry
}

func NewBanlist(path string) *Banlist {
	return &Banlist{
		path:  filepath.Clean(path),
		items: make(map[string]BanEntry),
	}
}

func (b *Banlist) Load() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	raw, err := os.ReadFile(b.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}

	var entries []BanEntry
	if err := json.Unmarshal(raw, &entries); err != nil {
		return err
	}

	b.items = make(map[string]BanEntry, len(entries))
	now := time.Now().UTC()
	for _, e := range entries {
		if e.Addr == "" {
			continue
		}
		// Drop expired on load
		if !e.Until.IsZero() && e.Until.After(now) {
			b.items[e.Addr] = e
		}
	}
	return nil
}

func (b *Banlist) Save() error {
	b.mu.RLock()
	entries := make([]BanEntry, 0, len(b.items))
	now := time.Now().UTC()
	for _, e := range b.items {
		if e.Addr == "" {
			continue
		}
		if e.Until.IsZero() || !e.Until.After(now) {
			continue
		}
		entries = append(entries, e)
	}
	b.mu.RUnlock()

	if err := os.MkdirAll(filepath.Dir(b.path), 0o700); err != nil {
		return err
	}

	tmp := b.path + ".tmp"
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, b.path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	_ = os.Chmod(b.path, 0o600)
	return nil
}

func (b *Banlist) IsBanned(addr string) (bool, BanEntry) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	e, ok := b.items[addr]
	if !ok {
		return false, BanEntry{}
	}
	if e.Until.IsZero() {
		return false, BanEntry{}
	}
	if time.Now().UTC().After(e.Until) {
		return false, BanEntry{}
	}
	return true, e
}

func (b *Banlist) Ban(addr string, duration time.Duration, reason string) {
	if addr == "" {
		return
	}
	now := time.Now().UTC()
	entry := BanEntry{
		Addr:      addr,
		Until:     now.Add(duration),
		Reason:    reason,
		UpdatedAt: now,
	}

	b.mu.Lock()
	b.items[addr] = entry
	b.mu.Unlock()
}

func (b *Banlist) Unban(addr string) {
	b.mu.Lock()
	delete(b.items, addr)
	b.mu.Unlock()
}

func (b *Banlist) CountActive() int {
	b.mu.RLock()
	defer b.mu.RUnlock()

	now := time.Now().UTC()
	n := 0
	for _, e := range b.items {
		if !e.Until.IsZero() && e.Until.After(now) {
			n++
		}
	}
	return n
}

func (b *Banlist) ListActive() []BanEntry {
	b.mu.RLock()
	defer b.mu.RUnlock()

	now := time.Now().UTC()
	out := make([]BanEntry, 0, len(b.items))
	for _, e := range b.items {
		if !e.Until.IsZero() && e.Until.After(now) {
			out = append(out, e)
		}
	}
	return out
}
