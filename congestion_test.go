package congestion

import (
	"context"
	"testing"
)

func TestAck(t *testing.T) {
	cases := []struct {
		Stage    stage
		Limit    int
		MaxLimit int
		Expected int
	}{
		{recovering, 1, 100, 1},
		{waiting, 1, 100, 1},
		{slowStart, 1, 100, 2},
		{slowStart, 50, 100, 100},
		{slowStart, 52, 100, 100},
		{increasing, 52, 100, 53},
		{increasing, 99, 100, 100},
		{increasing, 100, 100, 100},
	}

	for _, tc := range cases {
		l := Limiter{
			stage:    tc.Stage,
			limit:    tc.Limit,
			maxLimit: tc.MaxLimit,
		}

		l.ack()

		actual := l.limit

		if actual != tc.Expected {
			t.Errorf("Ack %s limit=%d maxLimit=%d is %d, expected %d", tc.Stage, tc.Limit, tc.MaxLimit, actual, tc.Expected)
		}
	}

}

func TestBackoff(t *testing.T) {
	cases := []struct {
		Stage    stage
		AcksLeft int
		Limit    int
		Expected int
	}{
		{recovering, 2, 100, 100},
		{recovering, 1, 100, 75},
		{waiting, 1, 100, 75},
		{slowStart, 1, 100, 75},
		{increasing, 1, 100, 75},
		{increasing, 1, 10, 7},
		{increasing, 1, 2, 1},
		{increasing, 1, 1, 1},
	}

	for _, tc := range cases {
		l := Limiter{
			stage:    tc.Stage,
			acksLeft: tc.AcksLeft,
			limit:    tc.Limit,
			maxLimit: 1000,
		}

		l.Backoff()

		actual := l.limit

		if actual != tc.Expected {
			t.Errorf("Ack %s acksLeft=%d limit=%d is %d, expected %d", tc.Stage, tc.AcksLeft, tc.Limit, actual, tc.Expected)
		}
	}

}

func TestLimiter(t *testing.T) {
	c := New(Config{10, 10})

	err := c.Acquire(context.Background(), 100)
	if err != nil {
		t.Error("Got an error:", err)
	}

	c.Release()
}

func TestAcquireFailsForCanceledContext(t *testing.T) {
	c := New(Config{10, 10})
	c.limit = 0

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := c.Acquire(ctx, 100)
	if err == nil {
		t.Error("Expected an error:", err)
		c.Release()
	}

}

func TestLimiterCanHaveMultiple(t *testing.T) {
	const concurrent = 4

	c := New(Config{10, 10})
	c.limit = concurrent

	ctx := context.Background()

	for i := 0; i < concurrent; i++ {
		err := c.Acquire(ctx, 100)
		if err != nil {
			t.Error("Got an error:", err)
			return
		}
	}

	for i := 0; i < concurrent; i++ {
		c.Release()
	}
}

func BenchmarkLimiterUnblocked(b *testing.B) {
	c := New(Config{10, 10})

	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		err := c.Acquire(ctx, 100)

		if err != nil {
			b.Log("Got an error:", err)
			return
		}
		c.Release()
	}
	b.StopTimer()
}

func BenchmarkLimiterBlocked(b *testing.B) {
	const concurrent = 4

	c := New(Config{10, 10})
	c.limit = concurrent

	ctx := context.Background()

	// Acquire maximum outstanding to avoid fast path
	for i := 0; i < concurrent; i++ {
		err := c.Acquire(ctx, 100)
		if err != nil {
			b.Error("Got an error:", err)
			return
		}
	}

	b.ResetTimer()

	// Race the release and the acquire in order to benchmark slow path
	for i := 0; i < b.N; i++ {
		go func() {
			c.Release()

		}()
		err := c.Acquire(ctx, 100)

		if err != nil {
			b.Log("Got an error:", err)
			return
		}
	}
}
