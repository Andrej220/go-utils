package zlog_test

import (
	"context"
	"log"
	"os"

	"github.com/azargarov/go-utils/zlog"
	"go.uber.org/zap/zapcore"
)

func ExampleNewDefault() {
	lg := zlog.NewDefault("my-service")
	defer lg.Sync()

	lg.Info("started",
		zlog.String("env", "dev"),
		zlog.Int("port", 8080),
	)
	// Output:
	// (output format depends on environment)
}

func ExampleAttach() {
	lg := zlog.NewDefault("svc")
	defer lg.Sync()

	ctx := zlog.Attach(context.Background(), lg)
	zlog.FromContext(ctx).Info("request", zlog.String("path", "/health"))
	// Output:
	// (output format depends on environment)
}

func ExampleZLogger_RedirectStdLog() {
	lg := zlog.NewDefault("svc")
	defer lg.Sync()

	restore := lg.RedirectStdLog(zapcore.WarnLevel)
	defer restore()

	log.Println("stdlib message promoted to WARN")
	// Output:
	// (output format depends on environment)
}

func ExampleZLogger_RedirectOutput() {
	lg := zlog.NewDefault("svc")
	defer lg.Sync()

	f, _ := os.CreateTemp("", "app.log")
	defer os.Remove(f.Name())

	restore := lg.RedirectOutput(f, zapcore.InfoLevel)
	defer restore()

	lg.Info("written to file")
	// Output:
	// (no stdout; message is written to file)
}
