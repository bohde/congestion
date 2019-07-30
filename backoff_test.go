package congestion

import (
	"context"
	"testing"
	"time"
)

func TestBackoff(t *testing.T) {
	c := New(Config{10, 10})
	b := Backoff{
		Limiter: &c,
		Step:    10 * time.Millisecond,
	}

	ok := b.Try(context.Background())
	if !ok {
		t.Error("Try failed", ok, b.Error)
	}

	b.Close()
}

func TestBackoffTryFailsOnCancelledContext(t *testing.T) {
	c := New(Config{10, 10})
	c.limit = 0

	b := Backoff{
		Limiter: &c,
		Step:    10 * time.Millisecond,
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	ok := b.Try(ctx)
	if ok {
		t.Error("Expected an error", b.Error)
		b.Close()
	}

}
