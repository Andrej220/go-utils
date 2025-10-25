package zlog

import (
	"fmt"
	"go.uber.org/zap/zapcore"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func flatten(fields ...zapcore.Field) string {
	enc := zapcore.NewMapObjectEncoder()
	for _, f := range fields {
		f.AddTo(enc)
	}
	if len(enc.Fields) == 0 {
		return ""
	}
	keys := make([]string, 0, len(enc.Fields))
	for k := range enc.Fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	pairs := make([]string, 0, len(keys))
	for _, k := range keys {
		pairs = append(pairs, fmt.Sprintf("%s=%v", k, enc.Fields[k]))
	}
	return strings.Join(pairs, ", ")
}

// detectAppName attempts to detect the process' executable name to prefill the
// "app" field in the stdlib fallback logger.
func detectAppName() string {
	exe, err := os.Executable()
	if err == nil {
		return filepath.Base(exe)
	}
	return filepath.Base(os.Args[0])
}

//func _flatten(fields ...zapcore.Field) string {
//	enc := zapcore.NewConsoleEncoder(zapcore.EncoderConfig{ConsoleSeparator: " "})
//	buf, _ := enc.EncodeEntry(zapcore.Entry{}, fields)
//	defer buf.Free()
//	return strings.TrimSpace(buf.String())
//}
