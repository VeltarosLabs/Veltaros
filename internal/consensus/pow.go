package consensus

import "errors"

// PoW is a placeholder for Proof-of-Work rules.
// We will implement real difficulty targets, header work, and verification next.
type PoW struct{}

func NewPoW() *PoW { return &PoW{} }

func (p *PoW) ValidateBlockHeader(_ []byte) error {
	// TODO: real PoW validation
	return errors.New("pow not implemented")
}
