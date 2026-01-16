package blockchain

import (
	"encoding/hex"
	"errors"

	vcrypto "github.com/VeltarosLabs/Veltaros/internal/crypto"
)

// MerkleRootFromTxIDs computes a merkle root from tx IDs (hex string of 32-byte hash).
// Rules:
// - Leaves are raw 32-byte tx hashes
// - If odd number of nodes at any level, duplicate the last
// - Parent = doubleSha256(left || right)
func MerkleRootFromTxIDs(txIDs []string) ([32]byte, error) {
	if len(txIDs) == 0 {
		return [32]byte{}, nil
	}

	nodes := make([][]byte, 0, len(txIDs))
	for _, id := range txIDs {
		b, err := hex.DecodeString(id)
		if err != nil {
			return [32]byte{}, errors.New("invalid txId hex")
		}
		if len(b) != 32 {
			return [32]byte{}, errors.New("invalid txId length")
		}
		cp := make([]byte, 32)
		copy(cp, b)
		nodes = append(nodes, cp)
	}

	for len(nodes) > 1 {
		if len(nodes)%2 == 1 {
			nodes = append(nodes, nodes[len(nodes)-1])
		}

		next := make([][]byte, 0, len(nodes)/2)
		for i := 0; i < len(nodes); i += 2 {
			left := nodes[i]
			right := nodes[i+1]
			concat := make([]byte, 0, 64)
			concat = append(concat, left...)
			concat = append(concat, right...)
			h := vcrypto.DoubleSha256(concat)
			parent := make([]byte, 32)
			copy(parent, h[:])
			next = append(next, parent)
		}
		nodes = next
	}

	var root [32]byte
	copy(root[:], nodes[0])
	return root, nil
}
