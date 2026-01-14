package api

import (
	"net"
	"net/http"
	"sync"
	"time"
)

type bucket struct {
	tokens float64
	last   time.Time
}

type Limiter struct {
	mu        sync.Mutex
	rate      float64 // tokens/sec
	burst     float64
	cost      float64
	clients   map[string]*bucket
	ttl       time.Duration
	lastPrune time.Time
}

func NewLimiter(rate float64, burst float64, cost float64) *Limiter {
	return &Limiter{
		rate:      rate,
		burst:     burst,
		cost:      cost,
		clients:   make(map[string]*bucket),
		ttl:       10 * time.Minute,
		lastPrune: time.Now().UTC(),
	}
}

func (l *Limiter) Allow(r *http.Request) bool {
	ip := clientIP(r)
	now := time.Now().UTC()

	l.mu.Lock()
	defer l.mu.Unlock()

	l.pruneLocked(now)

	b, ok := l.clients[ip]
	if !ok {
		b = &bucket{tokens: l.burst, last: now}
		l.clients[ip] = b
	}

	elapsed := now.Sub(b.last).Seconds()
	if elapsed > 0 {
		b.tokens += elapsed * l.rate
		if b.tokens > l.burst {
			b.tokens = l.burst
		}
		b.last = now
	}

	if b.tokens < l.cost {
		return false
	}
	b.tokens -= l.cost
	return true
}

func (l *Limiter) pruneLocked(now time.Time) {
	if now.Sub(l.lastPrune) < 2*time.Minute {
		return
	}
	l.lastPrune = now

	for ip, b := range l.clients {
		if now.Sub(b.last) > l.ttl {
			delete(l.clients, ip)
		}
	}
}

func clientIP(r *http.Request) string {
	// We intentionally do NOT trust X-Forwarded-For by default (can be spoofed).
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
