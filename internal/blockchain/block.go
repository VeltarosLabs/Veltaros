package blockchain

import (
	"encoding/binary"
	"errors"
	"time"

	vcrypto "github.com/VeltarosLabs/Veltaros/internal/crypto"
)

type BlockHeader struct {
	Version    uint32
	PrevHash   [32]byte
	MerkleRoot [32]byte
	Timestamp  int64
	Nonce      uint64
}

type Block struct {
	Header       BlockHeader
	Transactions []SignedTx
}

func (h BlockHeader) Hash() [32]byte {
	// Canonical header serialization (fixed-size fields, little-endian for integers).
	buf := make([]byte, 0, 4+32+32+8+8)

	tmp4 := make([]byte, 4)
	binary.LittleEndian.PutUint32(tmp4, h.Version)
	buf = append(buf, tmp4...)

	buf = append(buf, h.PrevHash[:]...)
	buf = append(buf, h.MerkleRoot[:]...)

	tmp8 := make([]byte, 8)
	binary.LittleEndian.PutUint64(tmp8, uint64(h.Timestamp))
	buf = append(buf, tmp8...)

	binary.LittleEndian.PutUint64(tmp8, h.Nonce)
	buf = append(buf, tmp8...)

	return vcrypto.DoubleSha256(buf)
}

func NewGenesisBlock() Block {
	// Minimal deterministic genesis. We’ll formalize genesis parameters later.
	now := time.Unix(0, 0).UTC()
	return Block{
		Header: BlockHeader{
			Version:   1,
			PrevHash:  [32]byte{},
			Timestamp: now.Unix(),
			Nonce:     0,
		},
		Transactions: []SignedTx{},
	}
}

func (b *Block) ValidateBasic() error {
	if b.Header.Timestamp <= 0 {
		return errors.New("block timestamp must be set")
	}

	// MerkleRoot computation will be added when we finalize tx merkle rules.
	// For now, allow zero root for empty blocks.
	if len(b.Transactions) == 0 {
		return nil
	}

	// Basic per-tx validation
	for i := range b.Transactions {
		if err := ValidateSignedTx(b.Transactions[i]); err != nil {
			return err
		}
	}
	return nil
}
