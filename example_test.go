package congestion_test

import (
	"context"
	"log"

	"github.com/joshbohde/congestion"
)

func foo() error {
	return nil
}

func Example() {
	const HighPriority = 100

	limiter := congestion.New(congestion.Config{
		Capacity: 10,
		MaxLimit: 10,
	})

	err := limiter.Acquire(context.Background(), HighPriority)

	// If we get an error, the queue was full
	if err != nil {
		log.Print("Got an error:", err)
		return
	}
	defer limiter.Release()

	// Make some sort of request
	err = foo()

	// If this error signals we are overloading the server, we need to backoff
	if err != nil {
		limiter.Backoff()
	}

}
