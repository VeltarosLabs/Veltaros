package blockchain

import (
	"errors"
	"sync"
)

type Chain struct {
	genesis Block
	height  uint64

	mu      sync.RWMutex
	mempool map[string]SignedTx // txId -> tx

	nonces *NonceTracker
}

func New() *Chain {
	g := NewGenesisBlock()
	return &Chain{
		genesis: g,
		height:  0,
		mempool: make(map[string]SignedTx),
		nonces:  NewNonceTracker(),
	}
}

func (c *Chain) Height() uint64 { return c.height }

func (c *Chain) Genesis() Block { return c.genesis }

func (c *Chain) AddBlock(_ Block) error {
	// Placeholder for now; real validation + storage comes next.
	// When blocks are implemented, this is where nonce tracking should move to chain-state.
	c.height++
	return nil
}

func (c *Chain) MempoolAdd(tx SignedTx) error {
	if err := ValidateSignedTx(tx); err != nil {
		return err
	}

	c.mu.Lock()
	c.mempool[tx.TxID] = tx
	c.mu.Unlock()

	return nil
}

func (c *Chain) MempoolHas(txID string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, ok := c.mempool[txID]
	return ok
}

func (c *Chain) MempoolList() []SignedTx {
	c.mu.RLock()
	defer c.mu.RUnlock()

	out := make([]SignedTx, 0, len(c.mempool))
	for _, tx := range c.mempool {
		out = append(out, tx)
	}
	return out
}

func (c *Chain) MempoolCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.mempool)
}

func (c *Chain) LastNonce(addr string) uint64 {
	return c.nonces.Get(addr)
}

func (c *Chain) ExpectedNonce(addr string) uint64 {
	return c.nonces.ExpectedNext(addr)
}

// ReserveNonce enforces strictly increasing nonces for broadcast.
// If nonce is valid, it reserves/updates the last nonce and returns true.
func (c *Chain) ReserveNonce(addr string, nonce uint64) bool {
	return c.nonces.CheckAndUpdate(addr, nonce)
}

var ErrInvalidBlock = errors.New("invalid block")
