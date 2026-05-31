package ratelimit

import (
	"testing"
	"time"
)

func TestTokenBucket_Allow(t *testing.T) {
	const capacity = 5
	tb := NewTokenBucket(1, capacity)
	defer tb.Stop()

	for i := range capacity {
		if !tb.Allow() {
			t.Fatalf("expected Allow()=true on request %d/%d", i+1, capacity)
		}
	}

	if tb.Allow() {
		t.Fatal("expected Allow()=false after exhausting all tokens")
	}
}

func TestTokenBucket_Refill(t *testing.T) {
	// 10 tokens/s, capacity 1 — drain it, wait for refill.
	tb := NewTokenBucket(10, 1)
	defer tb.Stop()

	if !tb.Allow() {
		t.Fatal("expected initial token to be available")
	}
	if tb.Allow() {
		t.Fatal("expected bucket to be empty after draining")
	}

	// At 10 tok/s one token arrives every 100 ms; wait 3× to be safe.
	time.Sleep(300 * time.Millisecond)

	if !tb.Allow() {
		t.Fatal("expected token to be available after refill period")
	}
}

func TestIPRateLimiter_DifferentIPs(t *testing.T) {
	limiter := NewIPRateLimiter(10, 2)

	lb1 := limiter.GetLimiter("1.2.3.4")
	lb2 := limiter.GetLimiter("5.6.7.8")
	defer lb1.Stop()
	defer lb2.Stop()

	if lb1 == lb2 {
		t.Fatal("different IPs must have independent token buckets")
	}

	// Drain ip1 completely.
	lb1.Allow()
	lb1.Allow()

	if lb1.Allow() {
		t.Fatal("ip1 bucket should be exhausted")
	}
	// ip2 must still have tokens.
	if !lb2.Allow() {
		t.Fatal("ip2 bucket must be independent and still have tokens")
	}

	// Same IP returns the same bucket.
	if limiter.GetLimiter("1.2.3.4") != lb1 {
		t.Fatal("GetLimiter must return the same bucket for the same IP")
	}
}
