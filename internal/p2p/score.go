package p2p

import (
	"sync"
	"time"
)

// Production-minded scoring model:
// - Score increases on protocol violations / rate limit violations / handshake failures.
// - Score decays over time so nodes can recover.
// - Threshold triggers ban (hard ban persisted via Banlist).

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
