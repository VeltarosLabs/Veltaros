package blockchain

import "errors"

type Chain struct {
	genesis Block
	height  uint64
}

func New() *Chain {
	g := NewGenesisBlock()
	return &Chain{
		genesis: g,
		height:  0,
	}
}

func (c *Chain) Height() uint64 { return c.height }

func (c *Chain) Genesis() Block { return c.genesis }

func (c *Chain) AddBlock(_ Block) error {
	// Placeholder for now; real validation + storage comes next.
	// This function exists to provide a stable API surface.
	c.height++
	return nil
}

var ErrInvalidBlock = errors.New("invalid block")
