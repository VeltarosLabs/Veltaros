package blockchain

import (
	"sync"
	"time"
)

// NonceTracker tracks the highest seen nonce per sender address.
// Policy: strictly increasing nonces (nonce must be > last).
type NonceTracker struct {
	mu   sync.RWMutex
	last map[string]nonceEntry
}

type nonceEntry struct {
	nonce     uint64
	updatedAt time.Time
}

func NewNonceTracker() *NonceTracker {
	return &NonceTracker{
		last: make(map[string]nonceEntry),
	}
}

func (n *NonceTracker) Get(addr string) uint64 {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.last[addr].nonce
}

func (n *NonceTracker) ExpectedNext(addr string) uint64 {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.last[addr].nonce + 1
}

// CheckAndUpdate validates that nonce is strictly greater than the last nonce.
// If valid, it updates the stored value and returns true.
// If invalid, it returns false and does not update.
func (n *NonceTracker) CheckAndUpdate(addr string, nonce uint64) bool {
	n.mu.Lock()
	defer n.mu.Unlock()

	prev := n.last[addr].nonce
	if nonce <= prev {
		return false
	}
	n.last[addr] = nonceEntry{nonce: nonce, updatedAt: time.Now().UTC()}
	return true
}

// Snapshot returns a compact view for persistence.
func (n *NonceTracker) Snapshot() []NonceSnapshot {
	n.mu.RLock()
	defer n.mu.RUnlock()

	out := make([]NonceSnapshot, 0, len(n.last))
	for addr, e := range n.last {
		if addr == "" || e.nonce == 0 {
			continue
		}
		out = append(out, NonceSnapshot{
			Addr:      addr,
			LastNonce: e.nonce,
			UpdatedAt: e.updatedAt,
		})
	}
	return out
}

// ApplySnapshot loads persisted values (keeps the highest nonce if conflicts exist).
func (n *NonceTracker) ApplySnapshot(snaps []NonceSnapshot) {
	n.mu.Lock()
	defer n.mu.Unlock()

	for _, sn := range snaps {
		if sn.Addr == "" || sn.LastNonce == 0 {
			continue
		}
		cur := n.last[sn.Addr]
		if sn.LastNonce > cur.nonce {
			n.last[sn.Addr] = nonceEntry{nonce: sn.LastNonce, updatedAt: sn.UpdatedAt}
		}
	}
}
