package congestion_test

import (
	"context"
	"time"

	"github.com/joshbohde/congestion"
)

func doRequest() error {
	return nil
}

func Example() {
	const (
		HighPriority = 100
	)

	// A limiter can be used to manage concurrent access to a rate limited resource
	limiter := congestion.New(congestion.Config{
		Capacity: 10,
		MaxLimit: 10,
	})

	// backoff manages a single retryable request with priority
	backoff := congestion.Backoff{
		Step:     100 * time.Millisecond,
		Limiter:  &limiter,
		Priority: HighPriority,
	}
	defer backoff.Close()

	// Try the request until either it succeeds, or the context is canceled
	for backoff.Try(context.Background()) {

		// Make some sort of request to the rate limited resource
		err := doRequest()

		// If this error signals we are overloading the server, we'll retry
		if err != nil {
			continue
		}

		// Otherwise return
		return
	}

}
