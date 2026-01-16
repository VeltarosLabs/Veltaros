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
	// Minimal deterministic genesis.
	now := time.Unix(0, 0).UTC()
	return Block{
		Header: BlockHeader{
			Version:   1,
			PrevHash:  [32]byte{},
			Timestamp: now.Unix(),
			Nonce:     0,
			// MerkleRoot = zero for empty tx list
		},
		Transactions: []SignedTx{},
	}
}

func BuildBlock(prevHash [32]byte, txs []SignedTx) (Block, error) {
	txIDs := make([]string, 0, len(txs))
	for _, tx := range txs {
		if err := ValidateSignedTx(tx); err != nil {
			return Block{}, err
		}
		txIDs = append(txIDs, tx.TxID)
	}

	root, err := MerkleRootFromTxIDs(txIDs)
	if err != nil {
		return Block{}, err
	}

	now := time.Now().UTC().Unix()
	return Block{
		Header: BlockHeader{
			Version:    1,
			PrevHash:   prevHash,
			MerkleRoot: root,
			Timestamp:  now,
			Nonce:      0,
		},
		Transactions: txs,
	}, nil
}

func (b *Block) ValidateBasic() error {
	if b.Header.Timestamp <= 0 {
		return errors.New("block timestamp must be set")
	}

	// Basic per-tx validation
	for i := range b.Transactions {
		if err := ValidateSignedTx(b.Transactions[i]); err != nil {
			return err
		}
	}

	// MerkleRoot consistency check
	txIDs := make([]string, 0, len(b.Transactions))
	for _, tx := range b.Transactions {
		txIDs = append(txIDs, tx.TxID)
	}
	root, err := MerkleRootFromTxIDs(txIDs)
	if err != nil {
		return err
	}
	if root != b.Header.MerkleRoot {
		return errors.New("merkle root mismatch")
	}

	return nil
}
