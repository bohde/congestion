package congestion

import (
	"context"
	"errors"
	"sync"
)

// Dropped is the error that will be returned if this token is dropped
var Dropped = errors.New("dropped")

type Config struct {
	Capacity int
	MaxLimit int
}

type Limiter struct {
	mu       sync.Mutex
	waiters  priorityQueue
	stage    stage
	capacity int

	acksLeft    int
	outstanding int
	limit       int
	maxLimit    int
}

func New(cfg Config) Limiter {
	return Limiter{
		stage:    slowStart,
		limit:    1,
		acksLeft: 1,
		capacity: cfg.Capacity,
		maxLimit: cfg.MaxLimit,
		waiters:  newQueue(),
	}
}

// Acquire a Lock with FIFO ordering, respecting the context. Returns an error it fails to acquire.
func (l *Limiter) Acquire(ctx context.Context, priority int) error {
	l.mu.Lock()

	// Fast path if we are unblocked.
	if l.outstanding < l.limit && l.waiters.Len() == 0 {
		l.outstanding++
		l.mu.Unlock()
		return nil
	}

	// If our queue is full, drop
	if l.waiters.Len() == l.capacity {
		l.mu.Unlock()
		return Dropped
	}

	r := rendezvouz{
		priority: priority,
		errChan:  make(chan error),
	}

	l.waiters.Push(&r)
	l.mu.Unlock()

	select {

	case err := <-r.errChan:
		return err

	case <-ctx.Done():
		err := ctx.Err()

		l.mu.Lock()

		select {
		case err = <-r.errChan:
		default:
			l.waiters.Remove(&r)
		}

		l.mu.Unlock()

		return err
	}
}

func (l *Limiter) ack() {
	// If we are waiting on acks, decrement and move on
	if l.stage == recovering {
		l.acksLeft = l.limit
		// Implement a waiting period of our limit before scaling again
		l.stage = waiting
		return
	}

	if l.acksLeft > 1 {
		l.acksLeft--
		return
	}

	switch l.stage {

	case waiting:
		if l.outstanding == l.limit {
			l.stage = increasing
		}

	// If we're in slow start, double our limit
	case slowStart:
		l.limit = l.limit * 2

	// If we're increasing increment
	case increasing:
		l.limit++
	}

	if l.limit > l.maxLimit {
		l.limit = l.maxLimit
	}

	// reset acks left for next stage transition
	l.acksLeft = l.limit
}

// Release a previously acquired lock.
func (l *Limiter) Release() {
	l.mu.Lock()

	l.ack()

	l.outstanding--

	if l.outstanding < 0 {
		l.mu.Unlock()
		panic("lock: bad release")
	}

	keepGoing := true
	for keepGoing && l.outstanding < l.limit {
		keepGoing = l.deque()
	}

	l.mu.Unlock()
}

func (l *Limiter) decrease() {
	l.limit = (l.limit * 3) / 4
	if l.limit < 1 {
		l.limit = 1
	}
	l.acksLeft = l.limit

}

// Signal that we need to backoff, and decrease our limit.
func (l *Limiter) Backoff() {
	l.mu.Lock()

	switch l.stage {

	// Decrease limit if we were not recovering
	case slowStart:
		l.decrease()
	case waiting:
		l.decrease()
	case increasing:
		l.decrease()

	// If we are recovering for more than the ack period, we decrease the limit again
	case recovering:
		if l.acksLeft > 1 {
			l.acksLeft--
		} else {
			l.decrease()
		}
	}

	l.stage = recovering

	l.mu.Unlock()
}

// Pull instances off the queue until we no longer drop
func (l *Limiter) deque() bool {
	rendezvouz := l.waiters.Pop()

	// Nothing to dequeue, so return
	if rendezvouz == nil {
		return false
	}

	l.outstanding++
	rendezvouz.Signal()

	return true
}
