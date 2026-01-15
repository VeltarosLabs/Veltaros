package ledger

import "errors"

// ApplyConfirmedTx applies a confirmed tx to the ledger:
// - subtract amount from sender
// - add (amount - fee) to recipient
//
// Note: fee accounting is a later phase (miner/validator reward).
func (l *Ledger) ApplyConfirmedTx(from string, to string, amount uint64, fee uint64) error {
	if from == "" || to == "" {
		return errors.New("from/to required")
	}
	if amount == 0 {
		return errors.New("amount must be > 0")
	}
	if fee > amount {
		return errors.New("fee must be <= amount")
	}

	receive := amount - fee

	l.mu.Lock()
	defer l.mu.Unlock()

	fromBal := l.balances[from]
	if fromBal < amount {
		return errors.New("insufficient confirmed balance")
	}

	l.balances[from] = fromBal - amount
	l.balances[to] = l.balances[to] + receive

	// Pending out will be rebuilt by mempool staging; confirm clears are handled elsewhere.
	return nil
}
