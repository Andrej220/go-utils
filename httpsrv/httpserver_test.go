package srvx

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"syscall"
	"testing"
	"time"

	lg "github.com/azargarov/go-utils/zlog"
)

type dto struct{ Name string }

func mustJSON(t testing.TB, v any) io.Reader {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal JSON: %v", err)
	}
	return bytes.NewReader(b)
}

func shortCfg() ServerConfig {
	return ServerConfig{
		Addr:            "127.0.0.1",
		Port:            "0",
		ShutdownTimeout: 200 * time.Millisecond,
		Logger:          lg.Discard, // keep tests quiet
	}

}
func strconvI(p int) string { return strconv.Itoa(p) }

func TestRunServer_GracefulShutdownOnSignal(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/ok", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	done := make(chan error, 1)
	go func() { done <- RunServer(mux, shortCfg()) }()

	// Give the server a moment to start
	time.Sleep(100 * time.Millisecond)

	// Send SIGINT to this process; RunServer should catch it and exit cleanly
	_ = syscall.Kill(os.Getpid(), syscall.SIGINT)

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("RunServer returned error on graceful shutdown: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("RunServer did not exit after SIGINT")
	}
}

func TestRunServer_PortInUseReturnsError(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()
	port := ln.Addr().(*net.TCPAddr).Port

	// Using same port should cause immediate ListenAndServe error (serveErr path)
	cfg := ServerConfig{
		Addr:            "127.0.0.1",
		Port:            func() string { return strconvI(port) }(),
		ShutdownTimeout: 200 * time.Millisecond,
		Logger:          lg.Discard,
	}
	err = RunServer(http.NewServeMux(), cfg)
	if err == nil {
		t.Fatal("expected error due to address already in use, got nil")
	}
}

func TestValidationHandler_UnsupportedContentType(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { t.Fatal("next should not be called") })
	h := NewValidationHandler[dto](next)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "text/plain")
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("want %d got %d", http.StatusUnsupportedMediaType, rr.Code)
	}
	if !bytes.Contains(rr.Body.Bytes(), []byte(`"unsupported_media_type"`)) {
		t.Fatalf("want JSON error with unsupported_media_type, got: %s", rr.Body.String())
	}
}

func TestValidationHandler_EmptyBody(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { t.Fatal("next should not be called") })
	h := NewValidationHandler[dto](next)

	req := httptest.NewRequest(http.MethodPost, "/", nil) // Body == nil
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("want 400 got %d", rr.Code)
	}

	resp := APIError{}

	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp.Code != ErrCodeEmptyBody || rr.Code != http.StatusBadRequest {
		t.Fatalf("want error=%s got %v", ErrCodeEmptyBody, resp.Code)
	}
}

func TestValidationHandler_UnknownFields(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { t.Fatal("next should not be called") })
	h := NewValidationHandler[dto](next)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{"Name":"a","extra":1}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("want 400 got %d", rr.Code)
	}
	if !bytes.Contains(rr.Body.Bytes(), []byte(`"invalid_json"`)) {
		t.Fatalf("unexpected body: %s", rr.Body.String())
	}
}

func TestValidationHandler_SuccessAndContext(t *testing.T) {
	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		got, ok := GetRequest[dto](r.Context())
		if !ok || got == nil || got.Name != "alice" {
			t.Fatalf("GetRequest failed: ok=%v got=%v", ok, got)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	h := NewValidationHandler[dto](next)

	req := httptest.NewRequest(http.MethodPost, "/", mustJSON(t, dto{Name: "alice"}))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("want 204 got %d", rr.Code)
	}
	if !nextCalled {
		t.Fatal("next handler not called")
	}
}

func TestWithRequest_PerTypeKeyIsolation(t *testing.T) {
	type A struct{ X int }
	type B struct{ Y int }

	ctx := context.Background()
	ctx = WithRequest(ctx, &A{X: 1})
	ctx = WithRequest(ctx, &B{Y: 2})

	if a, ok := GetRequest[A](ctx); !ok || a == nil || a.X != 1 {
		t.Fatalf("GetRequest[A] failed: %v %v", ok, a)
	}
	if b, ok := GetRequest[B](ctx); !ok || b == nil || b.Y != 2 {
		t.Fatalf("GetRequest[B] failed: %v %v", ok, b)
	}
}

func TestNormalize_UsesEnvPort(t *testing.T) {
	const key = "SRVX_TEST_PORT"
	t.Setenv(key, "9099")
	got := normalize(ServerConfig{EnvPortKey: key})
	if got.Port != "9099" {
		t.Fatalf("env port not applied, got %q", got.Port)
	}
}

// nil handler should not crash
func TestRunServer_DefaultHandlerWhenNil(t *testing.T) {
	done := make(chan error, 1)
	go func() { done <- RunServer(nil, shortCfg()) }()

	time.Sleep(100 * time.Millisecond)
	_ = syscall.Kill(os.Getpid(), syscall.SIGINT)

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("RunServer returned error: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for RunServer to exit")
	}
}
