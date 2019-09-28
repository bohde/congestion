package congestion

import (
	"context"
	"testing"
)

func TestRelease(t *testing.T) {
	cases := []struct {
		Stage         stage
		AcksLeft      int
		Outstanding   int
		Limit         int
		Expected      int
		ExpectedStage stage
	}{
		{recovering, 1, 10, 10, 10, waiting},
		{waiting, 2, 10, 10, 10, waiting},
		{waiting, 1, 10, 10, 10, increasing},
		{waiting, 1, 9, 10, 10, waiting},
		{slowStart, 2, 10, 10, 10, slowStart},
		{slowStart, 1, 10, 10, 20, slowStart},
		{increasing, 2, 10, 10, 10, increasing},
		{increasing, 1, 10, 10, 11, increasing},
	}

	for _, tc := range cases {
		l := Limiter{
			stage:       tc.Stage,
			acksLeft:    tc.AcksLeft,
			outstanding: tc.Outstanding,
			limit:       tc.Limit,
			maxLimit:    1000,
		}

		l.Release()

		actual := l.limit
		actualStage := l.stage

		if actual != tc.Expected || actualStage != tc.ExpectedStage {
			t.Errorf("Ack %s acksLeft=%d limit=%d is %d %s, expected %d %s", tc.Stage, tc.AcksLeft, tc.Limit, actual, actualStage, tc.Expected, tc.ExpectedStage)
		}
	}
}

func TestRateBackoff(t *testing.T) {
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

func BenchmarkLimiter(b *testing.B) {
	b.Run("Unblocked", func(b *testing.B) {
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
	})

	b.Run("Blocked", func(b *testing.B) {
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

	})

}
