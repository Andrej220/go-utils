# grlimit — goroutine concurrency limiter (one‑shot)

`grlimit` is a tiny, dependency‑free **bounded‑concurrency executor**.  
It limits how many jobs run **at the same time**. `Submit` blocks when all slots are in use and resumes when a slot frees up or the caller’s context is canceled.

> This is **not** a classic worker “pool.” There’s no background worker fleet and no internal job queue. Each admitted job runs in its **own goroutine**; the “queue” is callers blocked in `Submit`.

---

## Install


```bash
go get github.com/Andrej220/go-utils/grlimit
```

---

## Quick start

```go
package main

import (
	"context"
	"fmt"
	"log"

	grlimit "github.com/Andrej220/go-utils/grlimit" 
)

// Optional adapter for quick inline jobs in examples/tests.
type JobFunc func(ctx context.Context) error
func (f JobFunc) Run(ctx context.Context) error { return f(ctx) }

func main() {
	g := grlimit.NewGate(8) // at most 8 jobs run concurrently

	// Consume errors until the gate shuts down.
	doneErrs := make(chan struct{})
	go func() {
		defer close(doneErrs)
		for err := range g.Errors() {
			log.Printf("job error: %v", err)
		}
	}()

	ctx := context.Background()

	// Submit some work. Submit blocks when all 8 slots are busy.
	for i := 0; i < 100; i++ {
		// Example job
		job := JobFunc(func(ctx context.Context) error {
			// do work; respect ctx
			return nil
		})
		if err := g.Submit(ctx, job); err != nil {
			// ctx canceled or gate shut down
			fmt.Println("submit:", err)
			break
		}
	}

	// Stop admissions and wait for all in‑flight jobs to finish.
	g.CloseAndWait()
	<-doneErrs
}
```

---

## API

```go
// Job represents a unit of work. Implementations should respect ctx for cancellation.
type Job interface {
    Run(ctx context.Context) error
}

// Gate limits the number of concurrently running jobs.
// One‑shot: after CloseAndWait, Submit returns ErrShutdown and Errors() is closed.
type Gate struct{ /* ... */ }

func NewGate(capacity int) *Gate
func (*Gate) Submit(ctx context.Context, j Job) error // blocks when full
func (*Gate) CloseAndWait()                           // shuts down & joins
func (*Gate) Errors() <-chan error                    // closes after join
func (*Gate) InUse() int                              // running jobs
func (*Gate) Available() int                          // free slots
func (*Gate) Capacity() int                           // max concurrency
```

**Errors:**  
- `ErrShutdown` — the gate has been closed and no longer accepts jobs.  
- `ErrNilJobSubmitted` — a nil job was submitted.

---

## Design

- Uses a **buffered channel as a semaphore**. Each admission sends a token (takes a slot); each job completion receives a token (releases the slot).
- `CloseAndWait()` flips the closed flag (future `Submit` → `ErrShutdown`) and then **fills the semaphore to capacity**, which blocks until all in‑flight jobs release their tokens. This is the join, implemented without a `sync.WaitGroup`.
- When the join completes, `Errors()` is **closed** so consumers can `range` and exit cleanly.

### Single‑use (one‑shot)

After `CloseAndWait()` returns, the gate **stays closed**. Create a new gate for another run.

### Cancellation & errors

- `Job.Run(ctx)` receives a context and should **return promptly** when `ctx` is canceled.
- Any non‑nil error returned from `Run` is sent to `Errors()`. The channel has a small buffer; if it fills, additional errors may be dropped (best‑effort). Size the buffer to your needs or log dropped sends.

### Concurrency semantics

- `Submit` blocks when at capacity; use the caller’s context for timeouts/deadlines.
- `InUse()` → number of currently running jobs.  
- `Capacity()` → maximum concurrency.  
- `Available()` → `Capacity() - InUse()`.

### Panic safety

Panics inside a job are **recovered** to avoid leaking a semaphore token. You should still log the panic (and stack) in your implementation.

---

## When to use

Use `grlimit` when you want **bounded concurrency** with simple admission control and a clean shutdown/join, but you **don’t need** a queued worker pool. If you need a fixed set of workers pulling from a buffered job queue, implement that separately (e.g., N workers reading from `jobs <-chan Job`).

---

## Testing & examples

Run the package tests:

```bash
go test -race -v ./...
```

A handy test adapter for jobs:

```go
type JobFunc func(ctx context.Context) error
func (f JobFunc) Run(ctx context.Context) error { return f(ctx) }
```

---

