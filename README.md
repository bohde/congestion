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
```
