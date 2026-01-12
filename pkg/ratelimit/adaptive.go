package ratelimit

import (
	"context"
	"sync"
	"sync/atomic"

	"golang.org/x/time/rate"
)

// AdaptiveLimiter implements AIMD (Additive Increase/Multiplicative Decrease)
// rate limiting that automatically adjusts based on server responses.
type AdaptiveLimiter struct {
	maxRPS         float64
	currentRPS     atomic.Value // float64
	limiter        *rate.Limiter
	consecutive429 int64
	successCount   int64
	mu             sync.Mutex
}

// New creates an AdaptiveLimiter with the specified maximum RPS ceiling.
func New(maxRPS float64) *AdaptiveLimiter {
	if maxRPS <= 0 {
		maxRPS = 100
	}

	a := &AdaptiveLimiter{
		maxRPS:  maxRPS,
		limiter: rate.NewLimiter(rate.Limit(maxRPS), int(maxRPS)),
	}
	a.currentRPS.Store(maxRPS)
	return a
}

// Wait blocks until the rate limiter allows an event.
func (a *AdaptiveLimiter) Wait(ctx context.Context) error {
	return a.limiter.Wait(ctx)
}

// RecordResponse adjusts the rate based on HTTP status codes.
// Call this after each probe request completes.
func (a *AdaptiveLimiter) RecordResponse(statusCode int) {
	a.mu.Lock()
	defer a.mu.Unlock()

	current := a.currentRPS.Load().(float64)

	if statusCode == 429 || statusCode == 503 {
		a.consecutive429++
		a.successCount = 0

		// Multiplicative decrease: halve RPS after 3 consecutive throttles
		if a.consecutive429 >= 3 {
			newRPS := current * 0.5
			if newRPS < 10 {
				newRPS = 10 // floor at 10 RPS
			}
			a.currentRPS.Store(newRPS)
			a.limiter.SetLimit(rate.Limit(newRPS))
			a.limiter.SetBurst(int(newRPS))
			a.consecutive429 = 0
		}
	} else {
		a.consecutive429 = 0
		a.successCount++

		// Additive increase: +10% every 100 successful requests
		if a.successCount >= 100 && current < a.maxRPS {
			newRPS := current * 1.1
			if newRPS > a.maxRPS {
				newRPS = a.maxRPS
			}
			a.currentRPS.Store(newRPS)
			a.limiter.SetLimit(rate.Limit(newRPS))
			a.limiter.SetBurst(int(newRPS))
			a.successCount = 0
		}
	}
}

// CurrentRPS returns the current rate limit.
func (a *AdaptiveLimiter) CurrentRPS() float64 {
	return a.currentRPS.Load().(float64)
}

// MaxRPS returns the configured maximum RPS ceiling.
func (a *AdaptiveLimiter) MaxRPS() float64 {
	return a.maxRPS
}
