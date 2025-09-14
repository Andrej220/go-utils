// Package grlimit (go routine limiter) implements a simple, one-shot bounded-concurrency gate.
//
// Overview
//
// Gate limits the number of jobs that may run concurrently. Call Submit to
// admit a Job; it will block when the Gate is at capacity and return when a
// slot is available or the caller’s context is canceled. Each admitted job
// runs in its own goroutine. This is a concurrency grlimit, not a fixed worker
// pool—there is no background worker fleet and no internal job queue. The
// “queue” is simply callers blocked in Submit.
//
// Design
//
// Internally, Gate uses a buffered channel as a semaphore. Each admission
// sends a token (takes a slot); each job completion receives a token (releases
// the slot). CloseAndWait marks the gate as closed (future Submit calls fail
// with ErrShutdown) and then “fills” the semaphore to capacity, which blocks
// until all in-flight jobs release their tokens—this is how the join is
// implemented without a sync.WaitGroup. When the join completes, the error
// stream is closed.
//
// Single-use
//
// Gate is intentionally one-shot. After CloseAndWait returns, the gate stays
// closed and Submit always returns ErrShutdown. Create a new Gate for another
// run.
//
// Cancellation & errors
//
// Job.Run receives a context and should return promptly when ctx is canceled.
// Any non-nil error returned from Run is sent to Errors(). Errors() is a
// best-effort channel with a small buffer; if it fills, additional errors may
// be dropped (callers should log or size the buffer appropriately). Errors()
// closes after CloseAndWait returns; consumers can range over it and exit
// cleanly.
//
// Concurrency semantics
//
//   • Submit blocks when at capacity; use caller context for timeouts/deadlines.
//   • InUse() is the number of currently running jobs.
//   • Capacity() is the maximum concurrency.
//   • Available() = Capacity() - InUse().
//
// Panic safety
//
// Panics inside a job are recovered to avoid leaking a semaphore token. You
// should still log the panic (see TODO in worker).
//
// When to use
//
// Use Gate when you want bounded concurrency with simple admission control and
// a clean shutdown/join, but you do not need a queued worker pool. If you need
// a fixed set of workers pulling from a buffered job queue, implement that
// pattern separately (N workers reading from jobs <-chan Job).
//
// Example
//
//  g := grlimit.NewGate(8) // at most 8 jobs at once
//
//  // Consume errors until the gate shuts down.
//  doneErrs := make(chan struct{})
//  go func() {
//      defer close(doneErrs)
//      for err := range g.Errors() {
//          // handle/log err
//          _ = err
//      }
//  }()
//
//  ctx := context.Background()
//  for _, j := range jobs {
//      // Submit blocks when all 8 slots are busy, or returns ctx.Err() on cancel.
//      if err := g.Submit(ctx, j); err != nil {
//          // ctx canceled or gate shut down
//          break
//      }
//  }
//
//  // Stop admissions, wait for all running jobs to finish, then close Errors().
//  g.CloseAndWait()
//  <-doneErrs
//
// API summary
//
//  type Job interface {
//      Run(ctx context.Context) error
//  }
//
//  type Gate struct { /* one-shot bounded-concurrency grlimit */ }
//    func NewGate(capacity int) *Gate
//    func (*Gate) Submit(ctx context.Context, j Job) error      // blocks when full
//    func (*Gate) CloseAndWait()                                 // shuts down & joins
//    func (*Gate) Errors() <-chan error                          // closes after join
//    func (*Gate) InUse() int
//    func (*Gate) Available() int
//    func (*Gate) Capacity() int


package grlimit

import (
	"context"
	"errors"
	"sync/atomic"
)

var (
	ErrShutdown  = errors.New("gate is shutting down")
	ErrNilJobSubmitted = errors.New("nil job submitted")
)

const(
	defaultErrBuffer = 10
)

// Job represents a unit of work. Implementations should respect ctx for cancellation.
type Job interface {
	Run(ctx context.Context) error
}

// Gate limits the number of concurrently running jobs.
// One-shot: after CloseAndWait, Submit will return ErrShutdown and Errors() is closed.
type Gate struct{
	closed			atomic.Bool
	sem 			chan struct{}	// max concurrent jobs
	errs			chan error
}

// NewGate creates a new go routine limiter with the given capacity.
func NewGate(cap int) *Gate{
	if cap <= 0{ cap = 1 }

	return &Gate{
		sem: make(chan struct{}, cap),
		errs: make(chan error, defaultErrBuffer),
	}
}

// Submit blocks until a slot is available or ctx is canceled.
// Returns ErrShutdown after the gate has been closed.
func (g *Gate) Submit(ctx context.Context, jb Job) error{

	if jb == nil{
		return ErrNilJobSubmitted
	}

	if g.closed.Load() {
		return ErrShutdown
	}

	select{
	case g.sem <- struct{}{}: 		//take a slot
	    // prevent starting after shutdown flipped
		if g.closed.Load(){
			<-g.sem
			return ErrShutdown
		}
		go g.worker(ctx, jb)	    
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// CloseAndWait stops admissions and waits for all in-flight jobs to finish.
// Afterwards, Errors() is closed and Submit will return ErrShutdown.
func (g *Gate) CloseAndWait() {
	if g.closed.Swap(true) {
		return // already closed
	}
	// Acquire all capacity tokens, this blocks until no job holds a token
	for i := 0; i < g.Capacity(); i++ {
		g.sem <- struct{}{}
	}
	      // draining queue -> idle state, if reuse is implemented
	      //for i := 0; i < g.Capacity(); i++ {
	      //	<-g.sem
	      //}
	close(g.errs)
}

func (g *Gate) InUse() int{ return len(g.sem) }
func (g *Gate) Capacity() int{ return cap(g.sem) }
func (g *Gate) Errors() <-chan error { return g.errs }
func (g *Gate) Available() int  { return g.Capacity() - g.InUse() }

func (g *Gate)worker(ctx context.Context, jb Job){
	defer func(){<-g.sem}()	// Release ticket
	defer func() { 
		if r := recover(); r != nil{
			//TODO: log panic
		}
	}()

	select {
	case <- ctx.Done():
		return
	default:
	}
	err := jb.Run(ctx)
	if err != nil{
		select {
		case g.errs <- err:
		default:
			//TODO: chanell appears to be full, log it
		}
	}

}
