package blockchain

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type StoredBlock struct {
	HashHex     string `json:"hash"`
	Height      uint64 `json:"height"`
	PrevHashHex string `json:"prevHash"`
	MerkleRoot  string `json:"merkleRoot"`
	Timestamp   int64  `json:"timestamp"`
	TxCount     int    `json:"txCount"`
	Block       Block  `json:"block"`
}

type BlockStore struct {
	path string
}

func NewBlockStore(path string) *BlockStore {
	return &BlockStore{path: filepath.Clean(path)}
}

func (s *BlockStore) Load() ([]StoredBlock, error) {
	raw, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []StoredBlock{}, nil
		}
		return nil, err
	}

	var blocks []StoredBlock
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return nil, err
	}

	// Sort by height ascending for consistency
	sort.Slice(blocks, func(i, j int) bool { return blocks[i].Height < blocks[j].Height })
	return blocks, nil
}

func (s *BlockStore) Save(blocks []StoredBlock) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}

	sort.Slice(blocks, func(i, j int) bool { return blocks[i].Height < blocks[j].Height })

	data, err := json.MarshalIndent(blocks, "", "  ")
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

func MakeStoredBlock(height uint64, b Block) StoredBlock {
	h := b.Header.Hash()
	prev := b.Header.PrevHash
	return StoredBlock{
		HashHex:     hex.EncodeToString(h[:]),
		Height:      height,
		PrevHashHex: hex.EncodeToString(prev[:]),
		MerkleRoot:  hex.EncodeToString(b.Header.MerkleRoot[:]),
		Timestamp:   b.Header.Timestamp,
		TxCount:     len(b.Transactions),
		Block:       b,
	}
}

func NowUnix() int64 { return time.Now().UTC().Unix() }
