package autostr

// tag-based struct-to-string conversion
//
// Example:
//   type Person struct {
//       Name string `string:"include"`
//       Age  int    `string:"include"`
//   }
//   p := Person{Name: "Alice", Age: 30}
//   fmt.Println(autostr.String(p)) // Name: Alice, Age: 30

import (
	"fmt"
	"reflect"
	"strings"
)

type AutoStringer interface {
	AutoString() string
}

const (
    DefaultIncludeTag          = "string"
    DefaultIncludeValue        = "include"
    DefaultFieldNameTag        = "display"
    DefaultSeparator           = ", "
    DefaultFieldValueSeparator = ": "
)

type Config struct {
	IncludeTag           string // struct tag to include fields (default: "string")
	IncludeValue         string // tag value that includes field (default: "include")
	FieldNameTag		 string // struct tag to rename field
	FieldValueSeparator *string // separator between Field and Value (default: ": ")
	Separator           *string // field separator (default: ", ")
	ShowZeroValue        bool   // whether to show zero values (default: false)
}

func Ptr[T any](v T) *T { return &v }

func DefaultConfig() Config {
	return Config{
        IncludeTag:          DefaultIncludeTag,
        IncludeValue:        DefaultIncludeValue,
        FieldNameTag:        DefaultFieldNameTag,
        Separator:           Ptr(DefaultSeparator),
        FieldValueSeparator: Ptr(DefaultFieldValueSeparator),
        ShowZeroValue:       false,
	}
}

func ensureDefaults(cfg *Config) {
    if cfg.IncludeTag == "" {
        cfg.IncludeTag = DefaultIncludeTag
    }
    if cfg.IncludeValue == "" {
        cfg.IncludeValue = DefaultIncludeValue
    }
    if cfg.FieldNameTag == "" {
        cfg.FieldNameTag = DefaultFieldNameTag
    }
    if cfg.Separator == nil {
        cfg.Separator = Ptr(DefaultSeparator)
    }
    if cfg.FieldValueSeparator == nil {
        cfg.FieldValueSeparator = Ptr(DefaultFieldValueSeparator)
    }
}

// String renders any value to string using struct tags and Config.
// If the value (or *value) implements AutoString, that is used instead.
func String(obj any, config ...Config) string {
	cfg := DefaultConfig()
	if len(config) > 0 {
		cfg = config[0]
	}

	ensureDefaults(&cfg)

	// Prefer user-defined AutoString on value.
	if s, ok := any(obj).(AutoStringer); ok {
		return s.AutoString()
	}
	if vt := reflect.TypeOf(obj); vt != nil && vt.Kind() != reflect.Pointer {
		pv := reflect.New(vt)
		pv.Elem().Set(reflect.ValueOf(obj))
		if s, ok := pv.Interface().(AutoStringer); ok {
			return s.AutoString()
		}
	}

	visited := make(map[uintptr]bool) // cycle detection on pointers
	return stringifyValue(reflect.ValueOf(obj), cfg, visited)
}

func stringifyValue(v reflect.Value, cfg Config, visited map[uintptr]bool) string {
	if !v.IsValid() {
		return "<nil>"
	}

	if v.Kind() == reflect.Interface {
		if v.IsNil() {
			return "<nil>"
		}
		return stringifyValue(v.Elem(), cfg, visited)
	}

	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return "<nil>"
		}
		ptr := v.Pointer()
		if visited[ptr] {
			return "<cycle>"
		}
		visited[ptr] = true
		return stringifyValue(v.Elem(), cfg, visited)
	}

	if v.Kind() != reflect.Struct {
		return fmt.Sprintf("%v", v.Interface())
	}

	t := v.Type()
	var sb strings.Builder
	sb.Grow(64)
	
	sep := *cfg.Separator
	kv := *cfg.FieldValueSeparator

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		ft := t.Field(i)

		if !field.CanInterface() {
			continue
		}

		tag := ft.Tag.Get(cfg.IncludeTag)
		if tag != cfg.IncludeValue {
			continue
		}

		if !cfg.ShowZeroValue && isZeroValue(field) {
			continue
		}

		if sb.Len() > 0 {
			sb.WriteString(sep)
		}
		displayName := ft.Tag.Get(cfg.FieldNameTag)
		if displayName ==""{
			displayName = ft.Name
		}
		sb.WriteString(displayName)
		sb.WriteString(kv)
		sb.WriteString(formatValueWithVisited(field, cfg, visited))
	}
	return sb.String()
}

func formatValueWithVisited(field reflect.Value, cfg Config, visited map[uintptr]bool) string {
	switch field.Kind() {
	case reflect.Interface, reflect.Pointer:
		return stringifyValue(field, cfg, visited)
	case reflect.Struct:
		if hasAutoStringTags(field, cfg) {
			return stringifyValue(field, cfg, visited)
		}
	}
	return fmt.Sprintf("%v", field.Interface())
}

func isZeroValue(field reflect.Value) bool {
	switch field.Kind() {
	case reflect.String:
		return field.String() == ""
	case reflect.Ptr, reflect.Slice, reflect.Map, reflect.Interface, reflect.Chan, reflect.Func:
		return field.IsNil()
	default:
		return field.IsZero()
	}
}

func hasAutoStringTags(v reflect.Value, cfg Config) bool {
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		ft := t.Field(i)
		if ft.Tag.Get(cfg.IncludeTag) == cfg.IncludeValue {
			return true
		}
	}
	return false
}
