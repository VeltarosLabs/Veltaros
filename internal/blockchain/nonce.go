package blockchain

import "sync"

// NonceTracker tracks the highest seen nonce per sender address.
// Policy: strictly increasing nonces (nonce must be > last).
type NonceTracker struct {
	mu   sync.RWMutex
	last map[string]uint64
}

func NewNonceTracker() *NonceTracker {
	return &NonceTracker{
		last: make(map[string]uint64),
	}
}

func (n *NonceTracker) Get(addr string) uint64 {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.last[addr]
}

// CheckAndUpdate validates that nonce is strictly greater than the last nonce.
// If valid, it updates the stored value and returns true.
// If invalid, it returns false and does not update.
func (n *NonceTracker) CheckAndUpdate(addr string, nonce uint64) bool {
	n.mu.Lock()
	defer n.mu.Unlock()

	prev := n.last[addr]
	if nonce <= prev {
		return false
	}
	n.last[addr] = nonce
	return true
}

func (n *NonceTracker) ExpectedNext(addr string) uint64 {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.last[addr] + 1
}
