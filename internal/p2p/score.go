package p2p

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Production-minded scoring model:
// - Score increases on protocol violations / rate limit violations / handshake failures.
// - Score decays over time so nodes can recover.
// - Threshold triggers ban (hard ban persisted via Banlist).
// - Scores persist to disk so restarts don’t reset reputation.

type ScoreConfig struct {
	DecayInterval time.Duration // e.g. 1 minute
	DecayAmount   int           // e.g. 1 point per interval
	BanThreshold  int           // e.g. 10 points
	BanDuration   time.Duration // e.g. 30 minutes
}

type scoreEntry struct {
	Score      int
	LastUpdate time.Time
}

type ScoreSnapshot struct {
	Addr       string    `json:"addr"`
	Score      int       `json:"score"`
	LastUpdate time.Time `json:"lastUpdate"`
}

type Scorer struct {
	mu   sync.Mutex
	cfg  ScoreConfig
	data map[string]scoreEntry
}

func NewScorer(cfg ScoreConfig) *Scorer {
	if cfg.DecayInterval <= 0 {
		cfg.DecayInterval = 1 * time.Minute
	}
	if cfg.DecayAmount <= 0 {
		cfg.DecayAmount = 1
	}
	if cfg.BanThreshold <= 0 {
		cfg.BanThreshold = 10
	}
	if cfg.BanDuration <= 0 {
		cfg.BanDuration = 30 * time.Minute
	}
	return &Scorer{
		cfg:  cfg,
		data: make(map[string]scoreEntry),
	}
}

func (s *Scorer) Get(addr string) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	e := s.data[addr]
	e = s.applyDecayLocked(e, time.Now().UTC())
	s.data[addr] = e
	return e.Score
}

func (s *Scorer) Add(addr string, points int) (score int, banned bool, banFor time.Duration) {
	if addr == "" || points <= 0 {
		return 0, false, 0
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	e := s.data[addr]
	e = s.applyDecayLocked(e, now)
	e.Score += points
	e.LastUpdate = now
	s.data[addr] = e

	if e.Score >= s.cfg.BanThreshold {
		return e.Score, true, s.cfg.BanDuration
	}
	return e.Score, false, 0
}

func (s *Scorer) Snapshot() []ScoreSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	out := make([]ScoreSnapshot, 0, len(s.data))
	for addr, e := range s.data {
		if addr == "" {
			continue
		}
		e = s.applyDecayLocked(e, now)
		s.data[addr] = e
		if e.Score <= 0 {
			continue
		}
		out = append(out, ScoreSnapshot{
			Addr:       addr,
			Score:      e.Score,
			LastUpdate: e.LastUpdate,
		})
	}
	return out
}

func (s *Scorer) Load(path string) error {
	path = filepath.Clean(path)

	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}

	var snaps []ScoreSnapshot
	if err := json.Unmarshal(raw, &snaps); err != nil {
		return err
	}

	now := time.Now().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, sn := range snaps {
		if sn.Addr == "" || sn.Score <= 0 {
			continue
		}
		e := scoreEntry{Score: sn.Score, LastUpdate: sn.LastUpdate}
		e = s.applyDecayLocked(e, now)
		if e.Score <= 0 {
			continue
		}
		s.data[sn.Addr] = e
	}
	return nil
}

func (s *Scorer) Save(path string) error {
	path = filepath.Clean(path)

	snaps := s.Snapshot()

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(snaps, "", "  ")
	if err != nil {
		return err
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	_ = os.Chmod(path, 0o600)
	return nil
}

func (s *Scorer) applyDecayLocked(e scoreEntry, now time.Time) scoreEntry {
	if e.LastUpdate.IsZero() {
		e.LastUpdate = now
		return e
	}

	if e.Score <= 0 {
		e.Score = 0
		e.LastUpdate = now
		return e
	}

	interval := s.cfg.DecayInterval
	if interval <= 0 {
		return e
	}

	elapsed := now.Sub(e.LastUpdate)
	if elapsed <= 0 {
		return e
	}

	steps := int(elapsed / interval)
	if steps <= 0 {
		return e
	}

	decay := steps * s.cfg.DecayAmount
	e.Score -= decay
	if e.Score < 0 {
		e.Score = 0
	}
	e.LastUpdate = e.LastUpdate.Add(time.Duration(steps) * interval)
	return e
}
