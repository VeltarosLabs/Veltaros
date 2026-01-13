package ledger

import "errors"

// Ledger will track balances/UTXO/state transitions.
// This is intentionally an interface-first design so storage can be swapped.
type Ledger interface {
	ApplyBlock(blockBytes []byte) error
	Height() uint64
}

var ErrLedger = errors.New("ledger error")
