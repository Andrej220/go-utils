package grlimit

import (
	"context"
	"errors"
	"sync/atomic"
)

var (
	ErrShutdown        = errors.New("gate is shutting down")
	ErrNilJobSubmitted = errors.New("nil job submitted")
)

const (
	defaultErrBuffer = 10
)

// Job represents a unit of work. Implementations should respect ctx for cancellation.
type Job interface {
	Run(ctx context.Context) error
}

// Gate limits the number of concurrently running jobs.
// One-shot: after CloseAndWait, Submit will return ErrShutdown and Errors() is closed.
type Gate struct {
	closed atomic.Bool
	sem    chan struct{} // max concurrent jobs
	errs   chan error
}

// NewGate creates a new go routine limiter with the given capacity.
func NewGate(cap int) *Gate {
	if cap <= 0 {
		cap = 1
	}

	return &Gate{
		sem:  make(chan struct{}, cap),
		errs: make(chan error, defaultErrBuffer),
	}
}

// Submit blocks until a slot is available or ctx is canceled.
// Returns ErrShutdown after the gate has been closed.
func (g *Gate) Submit(ctx context.Context, jb Job) error {

	if jb == nil {
		return ErrNilJobSubmitted
	}

	if g.closed.Load() {
		return ErrShutdown
	}

	select {
	case g.sem <- struct{}{}: //take a slot
		// prevent starting after shutdown flipped
		if g.closed.Load() {
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

func (g *Gate) InUse() int           { return len(g.sem) }
func (g *Gate) Capacity() int        { return cap(g.sem) }
func (g *Gate) Errors() <-chan error { return g.errs }
func (g *Gate) Available() int       { return g.Capacity() - g.InUse() }

func (g *Gate) worker(ctx context.Context, jb Job) {
	defer func() { <-g.sem }() // Release ticket
	defer func() {
		if r := recover(); r != nil {
			//TODO: log panic
		}
	}()

	select {
	case <-ctx.Done():
		return
	default:
	}
	err := jb.Run(ctx)
	if err != nil {
		select {
		case g.errs <- err:
		default:
			//TODO: chanell appears to be full, log it
		}
	}

}
