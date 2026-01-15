package ledger

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Ledger struct {
	mu sync.RWMutex

	// confirmed balances (persisted)
	balances map[string]uint64

	// staged spends due to mempool txs (not persisted)
	pendingOut map[string]uint64

	storePath string
}

type Snapshot struct {
	Addr      string    `json:"addr"`
	Balance   uint64    `json:"balance"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func New(storePath string) *Ledger {
	return &Ledger{
		balances:   make(map[string]uint64),
		pendingOut: make(map[string]uint64),
		storePath:  filepath.Clean(storePath),
	}
}

func (l *Ledger) Load() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	raw, err := os.ReadFile(l.storePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}

	var snaps []Snapshot
	if err := json.Unmarshal(raw, &snaps); err != nil {
		return err
	}

	l.balances = make(map[string]uint64, len(snaps))
	for _, s := range snaps {
		if s.Addr == "" {
			continue
		}
		l.balances[s.Addr] = s.Balance
	}
	return nil
}

func (l *Ledger) Save() error {
	l.mu.RLock()
	snaps := make([]Snapshot, 0, len(l.balances))
	now := time.Now().UTC()
	for addr, bal := range l.balances {
		if addr == "" {
			continue
		}
		snaps = append(snaps, Snapshot{Addr: addr, Balance: bal, UpdatedAt: now})
	}
	l.mu.RUnlock()

	if err := os.MkdirAll(filepath.Dir(l.storePath), 0o700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(snaps, "", "  ")
	if err != nil {
		return err
	}

	tmp := l.storePath + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, l.storePath); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	_ = os.Chmod(l.storePath, 0o600)
	return nil
}

func (l *Ledger) ResetPending() {
	l.mu.Lock()
	l.pendingOut = make(map[string]uint64)
	l.mu.Unlock()
}

func (l *Ledger) ConfirmedBalance(addr string) uint64 {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.balances[addr]
}

func (l *Ledger) PendingOut(addr string) uint64 {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.pendingOut[addr]
}

func (l *Ledger) SpendableBalance(addr string) uint64 {
	l.mu.RLock()
	defer l.mu.RUnlock()

	confirmed := l.balances[addr]
	pending := l.pendingOut[addr]
	if pending >= confirmed {
		return 0
	}
	return confirmed - pending
}

// StageMempoolSpend reserves funds for a mempool tx.
// It does NOT change confirmed balances, only pending outflow.
// It enforces spendable >= required.
func (l *Ledger) StageMempoolSpend(from string, required uint64) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	confirmed := l.balances[from]
	pending := l.pendingOut[from]

	if pending > confirmed {
		return errors.New("ledger pending state invalid")
	}
	spendable := confirmed - pending
	if spendable < required {
		return errors.New("insufficient balance")
	}

	l.pendingOut[from] = pending + required
	return nil
}

// FaucetCredit increases confirmed balance. Intended for testnet/dev flows.
func (l *Ledger) FaucetCredit(addr string, amount uint64) error {
	if addr == "" {
		return errors.New("address required")
	}
	if amount == 0 {
		return errors.New("amount must be > 0")
	}

	l.mu.Lock()
	l.balances[addr] = l.balances[addr] + amount
	l.mu.Unlock()
	return nil
}
