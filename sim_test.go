package congestion

import (
	"context"
	"errors"
	"flag"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

var sim = flag.Bool("sim", false, "run simulation test")

func msToWait(perSec int64) time.Duration {
	ms := rand.ExpFloat64() / (float64(perSec) / 1000)
	return time.Duration(ms * float64(time.Millisecond))
}

type Capped struct {
	mu  sync.Mutex
	cur int64
	cap int64
}

func (s *Capped) Lock() error {
	s.mu.Lock()
	if s.cur >= s.cap {
		s.mu.Unlock()
		return errors.New("too many threads")
	}

	s.cur++
	s.mu.Unlock()

	return nil
}

func (s *Capped) Unlock() {
	s.mu.Lock()
	s.cur--
	s.mu.Unlock()
}

// Simulate 2 concurrent process with 1000 reqs/second each, fighting
// for a process that can process 10 concurrent at 100 reqs/second
// This should converge to each limiter getting 5, with a success rate of 50% on each
func TestConcurrentSimulation(t *testing.T) {
	if !(*sim) {
		t.Log("Skipping sim since -sim not passed")
		t.Skip()
	}

	const (
		perSecond   = 1000
		testSeconds = 1
		iterations  = perSecond * testSeconds
	)

	wg := sync.WaitGroup{}

	c := Capped{cap: 10}

	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			inner := sync.WaitGroup{}

			limiter := New(Config{
				Capacity: 100,
				MaxLimit: 20,
			})

			success := int64(0)

			for i := 0; i < iterations; i++ {
				time.Sleep(msToWait(perSecond))
				inner.Add(1)

				go func() {
					defer inner.Done()

					err := limiter.Acquire(context.Background(), 0)

					if err != nil {
						return
					}

					defer limiter.Release()

					err = c.Lock()
					if err != nil {
						limiter.Backoff()
						return
					}
					defer c.Unlock()

					time.Sleep(msToWait(100))
					atomic.AddInt64(&success, 1)

				}()
			}

			// Wait for the inner loop to finish
			inner.Wait()

			t.Logf("limit=%d, success=%f", limiter.limit, (float64(success) / iterations))

		}()
	}

	wg.Wait()

}

// Simulate 2 concurrent process with 1000 reqs/second each, using the
// same limiter at different priorities.  The upstream process that
// can process 10 concurrent at 100 reqs/second. This should converge
// on a limiter of 10, with the higher priority having most of the successful requests
func TestBackoffConcurrentSimulation(t *testing.T) {
	if !(*sim) {
		t.Log("Skipping sim since -sim not passed")
		t.Skip()
	}

	const (
		perSecond   = 1000
		testSeconds = 1
		iterations  = testSeconds * perSecond
	)

	wg := sync.WaitGroup{}

	c := Capped{cap: 10}
	limiter := New(Config{
		Capacity: 100,
		MaxLimit: 20,
	})

	for i := 0; i < 2; i++ {
		priority := i * 10
		wg.Add(1)
		go func() {
			defer wg.Done()
			inner := sync.WaitGroup{}

			success := int64(0)

			for i := 0; i < iterations; i++ {
				time.Sleep(msToWait(perSecond))
				inner.Add(1)

				go func() {

					defer inner.Done()
					b := Backoff{
						Limiter:  &limiter,
						Step:     10 * time.Millisecond,
						Priority: priority,
					}
					defer b.Close()

					for b.Try(context.Background()) {

						err := c.Lock()
						if err != nil {
							continue
						}

						time.Sleep(msToWait(100))

						c.Unlock()

						atomic.AddInt64(&success, 1)
						break
					}

				}()
			}

			// Wait for the inner loop to finish
			inner.Wait()

			t.Logf("priority=%d, limit=%d, success=%f", priority, limiter.limit, (float64(success) / (iterations)))

		}()
	}

	wg.Wait()

}
