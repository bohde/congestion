package congestion

import (
	"context"
	"math/rand"
	"time"
)

// Backoff implements exponential backoff retries on top of a Limiter
type Backoff struct {
	Step     time.Duration
	Limiter  *Limiter
	Priority int
	Error    error

	runs        int
	shouldClose bool
}

// Close will close resources associated with the Backoff
func (r *Backoff) Close() {
	if r.shouldClose {
		r.Limiter.Release()
		r.shouldClose = false
	}
}

// acquire will acquire the underlying limiter, managing state
func (r *Backoff) acquire(ctx context.Context) bool {
	err := r.Limiter.Acquire(ctx, r.Priority)
	if err != nil {
		r.Error = err
		return false
	}
	r.shouldClose = true
	return true
}

// Try will block this attempt until it's no longer limited, or the context is cancelled
func (r *Backoff) Try(ctx context.Context) bool {
	// If this is our first run, we always try to acquire
	if r.runs == 0 {
		r.runs++
		return r.acquire(ctx)
	}

	// Otherwise, we are retrying, and have to signal a backoff
	r.Limiter.Backoff()
	r.Close()

	// Generate the next time this retry can run, and check if that is after the deadline
	nextWakeup := time.Now().Add(time.Duration((rand.Float64() + 0.5) * float64(r.Step)))
	if deadline, ok := ctx.Deadline(); ok {
		if nextWakeup.After(deadline) {
			r.Error = context.DeadlineExceeded
			return false
		}
	}

	// Increase our priority that way we get scheduled ahead of other similar priority traffic
	r.Priority++
	// Update our step
	r.Step = (r.Step * 3) / 2

	// Re-enqueue at our new priority
	if !r.acquire(ctx) {
		return false
	}

	// If we've already been delayed enough, go ahead
	timeLeft := time.Now().Sub(nextWakeup)
	if timeLeft < 0 {
		return true
	}

	// Otherwise, block until either the context is cancelled
	t := time.NewTimer(timeLeft)
	select {
	case <-t.C:
		t.Stop()
		return true
	case <-ctx.Done():
		r.Error = ctx.Err()
		return false
	}

}
