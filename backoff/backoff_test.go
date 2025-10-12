package backoff

import (
	"testing"
	"time"
)

func TestNew(t *testing.T) {

	b := New(InitialBackoff, MaxBackoff, time.Now().UnixNano())
	if b == nil {
		t.Error("Expected non-nil Backoff instance")
		return
	}

	if b.current != InitialBackoff {
		t.Errorf("Expected current equal %d, got %d", InitialBackoff, b.current)
	}

	if b.max != MaxBackoff {
		t.Errorf("Expected max equal %d, got %d", MaxBackoff, b.max)
	}
}

func TestNext(t *testing.T) {
	b := New(InitialBackoff, MaxBackoff, time.Now().UnixNano())

	nxt := b.Next()

	if nxt < 0 || nxt > MaxBackoff {
		t.Error("Unxpected next value")
	}
}

func TestReset(t *testing.T) {
	b := New(InitialBackoff, MaxBackoff, time.Now().UnixNano())

	_ = b.Next()
	b.Reset(InitialBackoff)

	if b.current != InitialBackoff {
		t.Errorf("After Reset cxpected current equal to initial value %d, got %d", InitialBackoff, b.current)
	}
}
