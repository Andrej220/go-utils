// logger_fromctx_stderr_test.go
package zlog

import (
	"context"
	"os"
	"strings"
	"testing"
)

//func hijackFile(f **os.File) (restore func(), r *os.File, err error) {
//	pr, pw, err := os.Pipe()
//	if err != nil { return nil, nil, err }
//	old := *f
//	*f = pw
//	return func() { _ = pw.Close(); *f = old; _ = pr.Close() }, pr, nil
//}
//
//func readAllWithDeadline(t *testing.T, r io.Reader) string {
//	t.Helper()
//	var buf bytes.Buffer
//	done := make(chan struct{})
//	go func() { _, _ = io.Copy(&buf, bufio.NewReader(r)); close(done) }()
//	select {
//	case <-done:
//	case <-time.After(200 * time.Millisecond):
//	}
//	return buf.String()
//}

func TestFromContext_DefaultLoggerWritesToStderr(t *testing.T) {
	// Capture stdout (to assert nothing goes there)
	restoreOut, rOut, err := hijackFile(&os.Stdout)
	if err != nil {
		t.Fatalf("stdout hijack failed: %v", err)
	}
	defer restoreOut()

	// Capture stderr
	restoreErr, rErr, err := hijackFile(&os.Stderr)
	if err != nil {
		t.Fatalf("stderr hijack failed: %v", err)
	}
	defer restoreErr()

	// No logger in context -> defaultLogger (uses stdlib log)
	ctx := context.Background()
	lg := FromContext(ctx)

	lg.Info("fallback path")

	// Close writers to flush/unblock readers
	_ = os.Stdout.Close()
	_ = os.Stderr.Close()

	stdout := readAllWithDeadline(t, rOut)
	stderr := readAllWithDeadline(t, rErr)

	if strings.Contains(stdout, "fallback path") {
		t.Fatalf("expected no output on stdout, got: %q", stdout)
	}
	if !strings.Contains(stderr, "INFO: fallback path") {
		t.Fatalf("expected default logger to write to stderr; got: %q", stderr)
	}
}
