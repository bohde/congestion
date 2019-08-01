# Congestion

[![Build Status](https://travis-ci.org/joshbohde/congestion.svg?branch=master)](https://travis-ci.org/joshbohde/congestion)
[![GoDoc](https://godoc.org/github.com/joshbohde/congestion?status.svg)](https://godoc.org/github.com/joshbohde/congestion)


A congestion control limiter with priorities. Useful if you are
working with a service that implements dynamic rate limits with a way
of signaling when you're over the rate limit, e.g. [HTTP 429](https://httpstatuses.com/429).

It works by limiting the number of outstanding concurrent requests. It gradually increments concurrency, until it hits a limit, at which point, it decreases concurrency by 25%.


## Installation

```
$ go get github.com/joshbohde/congestion
```

## Usage

```
import (
	"context"
	"log"

	"github.com/joshbohde/congestion"
)

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
```
