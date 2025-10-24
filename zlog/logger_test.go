package zlog

import (
	"bufio"
	"bytes"
	"context"
	"go.uber.org/zap/zapcore"
	"io"
	"log"
	"os"
	"strings"
	"testing"
	"time"
)

func TestStdLoggerAt_DefaultBackend_RoutesToError(t *testing.T) {
	var buf bytes.Buffer
	oldOut, oldFlags := log.Writer(), log.Flags()
	log.SetOutput(&buf)
	log.SetFlags(0)
	defer func() { log.SetOutput(oldOut); log.SetFlags(oldFlags) }()

	d := defaultLogger{logger: log.New(log.Default().Writer(), "", log.LstdFlags)}
	std := StdLoggerAt(d, zapcore.ErrorLevel)

	std.Println("boom")

	out := buf.String()
	if !strings.Contains(out, "ERROR: boom") {
		t.Fatalf("stdlog adapter did not route to defaultLogger.Error; got: %q", out)
	}
}

func TestStdLoggerAt_ZapBackend_NoPanic(t *testing.T) {
	lg := New(&Config{
		ServiceName: "test-service",
		Debug:       false,
		Format:      ZLoggerJsonFormat,
	})

	std := StdLoggerAt(lg, zapcore.ErrorLevel)
	if std == nil {
		t.Fatal("StdLoggerAt returned nil *log.Logger for zap backend")
	}

	std.Println("zap stdlog bridge smoke test")
}

func TestStdLoggerAt_DefaultBackend_RoutesToWarn(t *testing.T) {
	var buf bytes.Buffer
	oldOut, oldFlags := log.Writer(), log.Flags()
	log.SetOutput(&buf)
	log.SetFlags(0)
	defer func() { log.SetOutput(oldOut); log.SetFlags(oldFlags) }()

	d := defaultLogger{logger: log.New(log.Default().Writer(), "", log.LstdFlags)}
	std := StdLoggerAt(d, zapcore.WarnLevel)

	std.Println("heads up")

	out := buf.String()
	if !strings.Contains(out, "WARN: heads up") {
		t.Fatalf("expected WARN routing; got: %q", out)
	}
}

func TestNew_ProductionConfig(t *testing.T) {
	cfg := &Config{
		ServiceName: "test-service",
		Debug:       false,
		Format:      ZLoggerJsonFormat,
	}

	logger := New(cfg)
	if _, ok := logger.(*zapLogger); !ok {
		t.Error("Expected zapLogger in production mode")
	}
}

func TestNew_DebugConfig(t *testing.T) {
	cfg := &Config{
		ServiceName: "test-service",
		Debug:       true,
		Format:      ZLoggerConsoleFormat,
	}

	logger := New(cfg)
	if _, ok := logger.(*zapLogger); !ok {
		t.Error("Expected zapLogger in debug mode")
	}
}

// Test invalid config that should trigger fallback
func TestNew_FallbackToDefault(t *testing.T) {
	old := log.Writer()
	defer log.SetOutput(old)

	var buf bytes.Buffer
	log.SetOutput(&buf)

	cfg := &Config{
		ServiceName: "test-service",
		Debug:       false,
		Format:      "invalid format", // <-- must be an invalid encoding format
	}

	logger := New(cfg)
	if _, ok := logger.(defaultLogger); !ok {
		t.Error("Expecte default logger")
	}
}

func TestContextIntegration(t *testing.T) {
	logger := New(&Config{ServiceName: "test"})
	ctx := Attach(context.Background(), logger)

	retrieved := FromContext(ctx)
	if retrieved != logger {
		t.Error("Logger not properly retrieved from context")
	}
}

func TestFromContext_NoLogger(t *testing.T) {
	// Test empty context returns default logger
	logger := FromContext(context.Background())
	if _, ok := logger.(defaultLogger); !ok {
		t.Error("Expected defaultLogger from empty context")
	}
}

func TestWithFields(t *testing.T) {
	logger := New(&Config{ServiceName: "test"})
	loggerWithFields := logger.With(String("key", "value"))

	// Verify it returns the same type
	if _, ok := loggerWithFields.(*zapLogger); !ok {
		t.Error("With() should return same logger type")
	}
}

// Test that noopLogger methods don't panic
func TestDiscardLogger(t *testing.T) {
	Discard.Info("test")
	Discard.Error("test")
	Discard.With(String("key", "value"))
}

func TestDebugFromEnv(t *testing.T) {
	t.Setenv("APP_DEBUG", "true")
	if !DebugFromEnv() {
		t.Error("DebugFromEnv should detect true")
	}

	t.Setenv("APP_DEBUG", "false")
	if DebugFromEnv() {
		t.Error("DebugFromEnv should detect false")
	}
}

func TestFlattenFunction(t *testing.T) {
	result := flatten(String("user", "alice"), Int("age", 30))

	if !strings.Contains(result, "user=alice") || !strings.Contains(result, "age=30") {
		t.Errorf("Flatten failed: %s", result)
	}
}

// test force stderr functionality
func hijackFile(f **os.File) (restore func(), r *os.File, err error) {
	pr, pw, err := os.Pipe()
	if err != nil {
		return nil, nil, err
	}
	old := *f
	*f = pw
	restore = func() {
		_ = pw.Close()
		*f = old
		_ = pr.Close()
	}
	return restore, pr, nil
}

func readAllWithDeadline(t *testing.T, r io.Reader) string {
	t.Helper()
	var buf bytes.Buffer
	br := bufio.NewReader(r)

	done := make(chan struct{})
	go func() {
		io.Copy(&buf, br)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
	}
	return buf.String()
}

func TestNew_ForceStderr_RoutesAllToStderr(t *testing.T) {
	restoreStdout, rOut, err := hijackFile(&os.Stdout)
	if err != nil {
		t.Fatalf("stdout hijack failed: %v", err)
	}
	defer restoreStdout()

	restoreStderr, rErr, err := hijackFile(&os.Stderr)
	if err != nil {
		restoreStdout()
		t.Fatalf("stderr hijack failed: %v", err)
	}
	defer restoreStderr()

	lg := New(&Config{
		ServiceName: "test-svc",
		Debug:       false,
		Format:      ZLoggerJsonFormat,
		ForceStderr: true,
	})
	lg.Info("hello stderr", String("k", "v"))
	_ = lg.Sync()

	_ = os.Stdout.Close()
	_ = os.Stderr.Close()

	// ---- Assert: nothing on stdout; something on stderr ----
	stdout := readAllWithDeadline(t, rOut)
	stderr := readAllWithDeadline(t, rErr)

	if strings.Contains(stdout, "hello stderr") {
		t.Fatalf("expected no output on stdout, but found: %q", stdout)
	}
	if !strings.Contains(stderr, "hello stderr") {
		t.Fatalf("expected log on stderr, got: %q", stderr)
	}
}
