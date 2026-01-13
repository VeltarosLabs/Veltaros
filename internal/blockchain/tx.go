package blockchain

import "errors"

// Transaction is intentionally minimal for now.
// We will evolve this into a real signed transaction format next phase.
type Transaction struct {
	ID   [32]byte
	Data []byte
}

func (t Transaction) ValidateBasic() error {
	if len(t.Data) == 0 {
		return errors.New("transaction data must not be empty")
	}
	return nil
}
