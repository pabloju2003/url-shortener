package ratelimit

import (
	"sync"
	"time"
)

type TokenBucket struct {
	tokens chan struct{}
	ticker *time.Ticker
	done   chan struct{}
}

func NewTokenBucket(rate int, capacity int) *TokenBucket {
	tb := &TokenBucket{
		tokens: make(chan struct{}, capacity),
		ticker: time.NewTicker(time.Second / time.Duration(rate)),
		done:   make(chan struct{}),
	}

	for range capacity {
		tb.tokens <- struct{}{}
	}

	go func() {
		for {
			select {
			case <-tb.done:
				return
			case <-tb.ticker.C:
				select {
				case tb.tokens <- struct{}{}:
				default:
				}
			}
		}
	}()

	return tb
}

func (tb *TokenBucket) Allow() bool {
	select {
	case <-tb.tokens:
		return true
	default:
		return false
	}
}

func (tb *TokenBucket) Stop() {
	close(tb.done)
	tb.ticker.Stop()
}

type IPRateLimiter struct {
	limiters map[string]*TokenBucket
	mu       sync.RWMutex
	rate     int
	capacity int
}

func NewIPRateLimiter(rate int, capacity int) *IPRateLimiter {
	return &IPRateLimiter{
		limiters: make(map[string]*TokenBucket),
		rate:     rate,
		capacity: capacity,
	}
}

func (l *IPRateLimiter) GetLimiter(ip string) *TokenBucket {
	l.mu.RLock()
	tb, ok := l.limiters[ip]
	l.mu.RUnlock()
	if ok {
		return tb
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	// Re-check after acquiring write lock to avoid double creation.
	if tb, ok = l.limiters[ip]; ok {
		return tb
	}
	tb = NewTokenBucket(l.rate, l.capacity)
	l.limiters[ip] = tb
	return tb
}
