package consensus

import "errors"

// PoS is a placeholder for Proof-of-Stake rules.
// We will implement stake weighting, validator selection, and slashing rules next.
type PoS struct{}

func NewPoS() *PoS { return &PoS{} }

func (p *PoS) ValidateBlockHeader(_ []byte) error {
	// TODO: real PoS validation
	return errors.New("pos not implemented")
}
