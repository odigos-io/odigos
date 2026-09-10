package server

import (
	"net"
	"net/http"
	"sync"
	"time"
)

// rateLimiter is a simple per-key token bucket used to bound OTLP abuse.
type rateLimiter struct {
	mu        sync.Mutex
	buckets   map[string]*bucket
	rate      float64 // tokens per second
	burst     float64
	lastSweep time.Time
}

type bucket struct {
	tokens   float64
	last     time.Time
	lastSeen time.Time
}

func newRateLimiter(perMinute int, burst int) *rateLimiter {
	if perMinute <= 0 {
		perMinute = 120
	}
	if burst <= 0 {
		burst = perMinute / 4
		if burst < 5 {
			burst = 5
		}
	}
	return &rateLimiter{
		buckets:   make(map[string]*bucket),
		rate:      float64(perMinute) / 60.0,
		burst:     float64(burst),
		lastSweep: time.Now(),
	}
}

func (l *rateLimiter) allow(key string) bool {
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()

	if now.Sub(l.lastSweep) > 5*time.Minute {
		l.sweepLocked(now)
		l.lastSweep = now
	}

	b := l.buckets[key]
	if b == nil {
		b = &bucket{tokens: l.burst, last: now, lastSeen: now}
		l.buckets[key] = b
	}

	elapsed := now.Sub(b.last).Seconds()
	b.tokens += elapsed * l.rate
	if b.tokens > l.burst {
		b.tokens = l.burst
	}
	b.last = now
	b.lastSeen = now

	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

func (l *rateLimiter) sweepLocked(now time.Time) {
	for k, b := range l.buckets {
		if now.Sub(b.lastSeen) > 10*time.Minute {
			delete(l.buckets, k)
		}
	}
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
