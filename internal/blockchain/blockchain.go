package blockchain

import (
	"encoding/hex"
	"errors"
	"sync"
)

type Chain struct {
	mu sync.RWMutex

	genesis Block
	height  uint64
	tipHash [32]byte

	mempool map[string]SignedTx

	nonces     *NonceTracker
	nonceStore *NonceStore

	blockStorePath string
	blocks         []StoredBlock
	blocksByHash   map[string]StoredBlock
}

func New(nonceStorePath string, blockStorePath string) *Chain {
	g := NewGenesisBlock()
	genHash := g.Header.Hash()

	return &Chain{
		genesis:        g,
		height:         0,
		tipHash:        genHash,
		mempool:        make(map[string]SignedTx),
		nonces:         NewNonceTracker(),
		nonceStore:     NewNonceStore(nonceStorePath),
		blockStorePath: blockStorePath,
		blocks:         []StoredBlock{},
		blocksByHash:   make(map[string]StoredBlock),
	}
}

func (c *Chain) Height() uint64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.height
}

func (c *Chain) TipHash() [32]byte {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.tipHash
}

func (c *Chain) TipHashHex() string {
	h := c.TipHash()
	return hex.EncodeToString(h[:])
}

func (c *Chain) Genesis() Block { return c.genesis }

func (c *Chain) AddBlock(b Block) (StoredBlock, error) {
	if err := b.ValidateBasic(); err != nil {
		return StoredBlock{}, err
	}

	c.mu.Lock()
	c.height++
	c.tipHash = b.Header.Hash()

	sb := MakeStoredBlock(c.height, b)
	c.blocks = append(c.blocks, sb)
	c.blocksByHash[sb.HashHex] = sb
	c.mu.Unlock()

	return sb, nil
}

// Block store persistence
func (c *Chain) LoadBlocks() error {
	store := NewBlockStore(c.blockStorePath)
	blocks, err := store.Load()
	if err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.blocks = blocks
	c.blocksByHash = make(map[string]StoredBlock, len(blocks))
	for _, b := range blocks {
		c.blocksByHash[b.HashHex] = b
	}

	// If blocks exist, set height/tip based on last
	if len(blocks) > 0 {
		last := blocks[len(blocks)-1]
		c.height = last.Height
		if h, err := hex.DecodeString(last.HashHex); err == nil && len(h) == 32 {
			copy(c.tipHash[:], h)
		}
	}

	return nil
}

func (c *Chain) SaveBlocks() error {
	c.mu.RLock()
	blocks := make([]StoredBlock, len(c.blocks))
	copy(blocks, c.blocks)
	path := c.blockStorePath
	c.mu.RUnlock()

	store := NewBlockStore(path)
	return store.Save(blocks)
}

func (c *Chain) RecentBlocks(limit int) []StoredBlock {
	if limit <= 0 {
		limit = 25
	}
	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(c.blocks) == 0 {
		return []StoredBlock{}
	}

	if limit > len(c.blocks) {
		limit = len(c.blocks)
	}
	out := make([]StoredBlock, 0, limit)
	start := len(c.blocks) - limit
	for i := start; i < len(c.blocks); i++ {
		out = append(out, c.blocks[i])
	}
	return out
}

func (c *Chain) GetBlock(hashHex string) (StoredBlock, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	b, ok := c.blocksByHash[hashHex]
	return b, ok
}

// Mempool
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

// Nonces
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
