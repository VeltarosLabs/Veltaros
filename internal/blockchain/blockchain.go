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
}

func New() *Chain {
	g := NewGenesisBlock()
	return &Chain{
		genesis: g,
		height:  0,
		mempool: make(map[string]SignedTx),
	}
}

func (c *Chain) Height() uint64 { return c.height }

func (c *Chain) Genesis() Block { return c.genesis }

func (c *Chain) AddBlock(_ Block) error {
	// Placeholder for now; real validation + storage comes next.
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

var ErrInvalidBlock = errors.New("invalid block")
