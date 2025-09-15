package zlog

import (
	"context"
	"strings"
	"testing"
	"log"
	"bytes"
)

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