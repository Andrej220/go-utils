package grlimit

import (
	"context"
	"errors"
	"testing"
	"time"
)

// Adaptor to build jobs inline in tests.
type JobFunc func(ctx context.Context) error
func (f JobFunc) Run(ctx context.Context) error { return f(ctx) }

func startErrConsumer(g *Gate) (done chan struct{}, got chan error) {
	done = make(chan struct{})
	got = make(chan error, 16)
	go func() {
		defer close(done)
		for err := range g.Errors() {
			got <- err
		}
	}()
	return done, got
}

func TestBlocksAtCapacityAndReleases(t *testing.T) {
	g := NewGate(1)
	errsDone, _ := startErrConsumer(g)

	hold := make(chan struct{})
	first := JobFunc(func(ctx context.Context) error {
		<-hold
		return nil
	})
	if err := g.Submit(context.Background(), first); err != nil {
		t.Fatalf("submit first: %v", err)
	}

	// Second should block; we use a short timeout context to prove it blocks.
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Millisecond)
	defer cancel()
	second := JobFunc(func(ctx context.Context) error { return nil })

	if err := g.Submit(ctx, second); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline exceeded while full, got %v", err)
	}

	// Now free the first; second should get in immediately.
	close(hold)
	if err := g.Submit(context.Background(), second); err != nil {
		t.Fatalf("submit after release: %v", err)
	}

	g.CloseAndWait()
	<-errsDone
}

func TestShutdownPreventsAdmission(t *testing.T) {
	g := NewGate(2)
	errsDone, _ := startErrConsumer(g)

	g.CloseAndWait()

	if err := g.Submit(context.Background(), JobFunc(func(ctx context.Context) error { return nil })); !errors.Is(err, ErrShutdown) {
		t.Fatalf("expected ErrShutdown after CloseAndWait, got %v", err)
	}
	<-errsDone
}

func TestErrorsForwarded(t *testing.T) {
	g := NewGate(1)
	errCh := g.Errors()

	want := errors.New("boom")
	if err := g.Submit(context.Background(), JobFunc(func(ctx context.Context) error { return want })); err != nil {
		t.Fatalf("submit: %v", err)
	}
	
	g.CloseAndWait()
	select {
	case err := <-errCh:
		if !errors.Is(err, want) {
			t.Fatalf("want %v, got %v", want, err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("timed out waiting for error")
	}
}

func TestPanicDoesNotLeakSlot(t *testing.T) {
	g := NewGate(1)
	errsDone, _ := startErrConsumer(g)

	// First job panics; token must still be released by defer.
	_ = g.Submit(context.Background(), JobFunc(func(ctx context.Context) error {
		panic("kaboom")
	}))

	// Second should still be able to enter soon.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := g.Submit(ctx, JobFunc(func(ctx context.Context) error { return nil })); err != nil {
		t.Fatalf("submit after panic job: %v", err)
	}

	g.CloseAndWait()
	<-errsDone
}

func TestInUseAndAvailable(t *testing.T) {
	g := NewGate(3)
	errsDone, _ := startErrConsumer(g)

	hold := make(chan struct{})
	job := JobFunc(func(ctx context.Context) error { <-hold; return nil })
	if err := g.Submit(context.Background(), job); err != nil {
		t.Fatalf("submit: %v", err)
	}

	if got := g.InUse(); got != 1 {
		t.Fatalf("InUse = %d, want 1", got)
	}
	if got := g.Available(); got != 2 {
		t.Fatalf("Available = %d, want 2", got)
	}

	close(hold)
	g.CloseAndWait()
	<-errsDone
}

func TestSubmitRespectsContextBeforeAdmission(t *testing.T) {
	g := NewGate(1)
	errsDone, _ := startErrConsumer(g)

	hold := make(chan struct{})
	_ = g.Submit(context.Background(), JobFunc(func(ctx context.Context) error {
		<-hold
		return nil
	}))

	// Gate is full; this submit should time out and must not increase InUse.
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Millisecond)
	defer cancel()
	_ = g.Submit(ctx, JobFunc(func(ctx context.Context) error { return nil })) // expect ctx error

	if got := g.InUse(); got != 1 {
		t.Fatalf("InUse = %d, want 1", got)
	}

	close(hold)
	g.CloseAndWait()
	<-errsDone
}
