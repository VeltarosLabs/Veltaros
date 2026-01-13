package consensus

import "errors"

// Engine defines the validation interface for consensus.
type Engine interface {
	ValidateBlockHeader(headerBytes []byte) error
}

var ErrInvalidConsensus = errors.New("invalid consensus")
