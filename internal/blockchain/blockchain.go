package blockchain

import (
	"errors"
	"sync"
)

type Chain struct {
	genesis Block
	height  uint64

	mu      sync.RWMutex
	mempool map[string]SignedTx

	nonces     *NonceTracker
	nonceStore *NonceStore
}

func New(nonceStorePath string) *Chain {
	g := NewGenesisBlock()
	c := &Chain{
		genesis:    g,
		height:     0,
		mempool:    make(map[string]SignedTx),
		nonces:     NewNonceTracker(),
		nonceStore: NewNonceStore(nonceStorePath),
	}
	return c
}

func (c *Chain) Height() uint64 { return c.height }

func (c *Chain) Genesis() Block { return c.genesis }

func (c *Chain) AddBlock(_ Block) error {
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

// MempoolDrain returns all current mempool txs and clears the mempool.
// Used by dev/testnet block production.
func (c *Chain) MempoolDrain() []SignedTx {
	c.mu.Lock()
	defer c.mu.Unlock()

	out := make([]SignedTx, 0, len(c.mempool))
	for _, tx := range c.mempool {
		out = append(out, tx)
	}
	c.mempool = make(map[string]SignedTx)
	return out
}

func (c *Chain) LastNonce(addr string) uint64 {
	return c.nonces.Get(addr)
}

func (c *Chain) ExpectedNonce(addr string) uint64 {
	return c.nonces.ExpectedNext(addr)
}

func (c *Chain) ReserveNonce(addr string, nonce uint64) bool {
	return c.nonces.CheckAndUpdate(addr, nonce)
}

func (c *Chain) LoadNonceState() error {
	if c.nonceStore == nil {
		return nil
	}
	snaps, err := c.nonceStore.Load()
	if err != nil {
		return err
	}
	c.nonces.ApplySnapshot(snaps)
	return nil
}

func (c *Chain) SaveNonceState() error {
	if c.nonceStore == nil {
		return nil
	}
	return c.nonceStore.Save(c.nonces.Snapshot())
}

var ErrInvalidBlock = errors.New("invalid block")
