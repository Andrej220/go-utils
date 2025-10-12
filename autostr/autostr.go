// Tag-based struct-to-string conversion.
//
// The autostr package provides a reflection-based utility to convert Go structs into human-readable strings using struct tags.
// It is designed for logging, debugging, and CLI output, offering flexible configuration for field inclusion, naming, formatting, and separators.
// If a type implements the AutoStringer interface, its AutoString method is used instead of reflection-based conversion.
//
// Example:
//
//	type Person struct {
//	    Name string `string:"include" display:"FullName" format:"%s"`
//	    Age  int    `string:"include" format:"%d years"`
//	}
//	p := Person{Name: "Alice", Age: 30}
//	fmt.Println(autostr.String(p)) // Output: FullName: Alice, Age: 30 years
package autostr

import (
	"fmt"
	"reflect"
	"strings"
)

// AutoStringer defines an interface for types that provide their own string representation.
// Types implementing AutoStringer will use their AutoString method instead of reflection-based conversion.
type AutoStringer interface {
	AutoString() string
}

// Constants defining default values for configuration.
const (
	// DefaultIncludeTag is the default struct tag key for including fields in the string output.
	DefaultIncludeTag = "string"
	// DefaultIncludeValue is the default tag value that indicates a field should be included.
	DefaultIncludeValue = "include"
	// DefaultFieldNameTag is the default struct tag key for renaming fields in the output.
	DefaultFieldNameTag = "display"
	// DefaultSeparator is the default separator between fields in the output.
	DefaultSeparator = ", "
	// DefaultFieldValueSeparator is the default separator between field names and their values.
	DefaultFieldValueSeparator = ": "
	// DefaultShowZeroValue determines whether zero values are included by default.
	DefaultShowZeroValue = true
	// DefaultFormat is the default format string for field values when no format tag is specified.
	DefaultFormat = "%v"
	// DefaultFormatTag is the default struct tag key for specifying field value formats.
	DefaultFormatTag = "format"
)

// Config defines options for customizing the string conversion process.
type Config struct {
	IncludeTag          string  // IncludeTag specifies the struct tag key for including fields (default: "string").
	IncludeValue        string  // IncludeValue specifies the tag value that includes a field (default: "include").
	FieldNameTag        string  // FieldNameTag specifies the struct tag key for renaming fields (default: "display").
	FieldValueSeparator *string // FieldValueSeparator is the separator between field names and values (default: ": ").
	Separator           *string // Separator is the separator between fields (default: ", ").
	ShowZeroValue       bool    // ShowZeroValue determines whether zero-value fields are included (default: true).
	FormatTag           string  // FormatTag specifies the struct tag key for formatting field values (default: "format").
	PrettyPrint         bool    // print multiline values in a pretty way
}

// Ptr creates a pointer to a value of any type.
// It is a helper function for setting pointer-based fields in Config, such as Separator or FieldValueSeparator.
//
// Example:
//
//	cfg := Config{Separator: Ptr(":")} // Sets Separator to ":"
func Ptr[T any](v T) *T { return &v }

// DefaultConfig returns a Config with default values for struct-to-string conversion.
// The defaults are:
//   - IncludeTag: "string"
//   - IncludeValue: "include"
//   - FieldNameTag: "display"
//   - Separator: ", "
//   - FieldValueSeparator: ": "
//   - ShowZeroValue: true
//   - FormatTag: "format"
//
// Example:
//
//	cfg := DefaultConfig()
//	fmt.Println(String(Person{Name: "Alice", Age: 30}, cfg)) // Output: Name: Alice, Age: 30
func DefaultConfig() Config {
	return Config{
		IncludeTag:          DefaultIncludeTag,
		IncludeValue:        DefaultIncludeValue,
		FieldNameTag:        DefaultFieldNameTag,
		Separator:           Ptr(DefaultSeparator),
		FieldValueSeparator: Ptr(DefaultFieldValueSeparator),
		ShowZeroValue:       DefaultShowZeroValue,
		FormatTag:           DefaultFormatTag,
	}
}

// ensureDefaults sets default values for unset Config fields.
// It is an internal helper function and not intended for public use.
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
	if cfg.FormatTag == "" {
		cfg.FormatTag = DefaultFormatTag
	}
	if cfg.FieldValueSeparator == nil {
		cfg.FieldValueSeparator = Ptr(DefaultFieldValueSeparator)
	}
}

// String converts a value to a human-readable string using struct tags and an optional Config.
// If the value (or its pointer) implements AutoStringer, its AutoString method is used.
// If no Config is provided, DefaultConfig is used.
// The function handles nested structs, pointers, interfaces, and cyclic references safely.
//
// Example:
//
//	type Person struct {
//	    Name string `string:"include" display:"FullName" format:"%s"`
//	    Age  int    `string:"include" format:"%d years"`
//	}
//	p := Person{Name: "Alice", Age: 30}
//	fmt.Println(String(p)) // Output: FullName: Alice, Age: 30 years
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

// stringifyValue converts a reflect.Value to a string based on the provided Config and visited pointers.
// It is an internal helper function and not intended for public use.
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

	var indent int
	if cfg.PrettyPrint {
		indent = measureKeyColumnWidth(v, cfg)
	}

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
		if displayName == "" {
			displayName = ft.Name
		}
		sb.WriteString(displayName)
		val := formatValueWithVisited(field, ft.Tag.Get(cfg.FormatTag), cfg, visited)
		if cfg.PrettyPrint {
			pad := indent - len(displayName)
			val = formatValueAligned(val, " ", kv, indent, pad)
		} else {
			sb.WriteString(kv)
		}
		sb.WriteString(val)
	}
	return sb.String()
}

// formatValueAligned formats a value string so that its first line follows a
// key/column and its subsequent lines align under the value column.
//   - The first line is rendered as: strings.Repeat(indentChar, pad) + separator + firstLine
//   - Each following line is rendered as: strings.Repeat(indentChar, indent) + separator + line
//
// The function normalizes Windows newlines ("\r\n" → "\n") and trims only
// trailing '\n' from val. If val is empty, it returns an empty string.
// Parameters:
//   - val: the raw value text (may be multi-line).
//   - indentChar: the single-character string used for indentation (e.g., " ").
//   - separator: the string placed between key and value (e.g., ": " or "\t\t").
//   - indent: the total indent (in indentChar units) used for subsequent lines
//     (typically the maximum key width).
//   - pad: extra left padding (in indentChar units) applied before separator on
//     the first line (typically indent - len(key); negative values are treated as 0).
//
// Note: This function only formats the value portion; callers should write the key
// text before calling this function.
func formatValueAligned(val, indentChar, separator string, indent, pad int) string {
	if val == "" {
		return ""
	}
	if pad < 0 {
		pad = 0
	}
	val = strings.ReplaceAll(val, "\r\n", "\n")
	val = strings.Trim(val, "\n")
	lines := strings.Split(val, "\n")
	first := strings.Repeat(indentChar, pad) + separator + lines[0]
	if len(lines) == 1 {
		return first
	}

	prefix := strings.Repeat(indentChar, indent) + separator
	out := make([]string, 0, len(lines))
	out = append(out, first)
	for _, ln := range lines[1:] {
		out = append(out, prefix+ln)
	}
	return strings.Join(out, "\n")
}

// measureKeyColumnWidth returns the maximum printable key width for v,
// considering autostr rules. It inspects only exported struct fields that
// (1) are tagged with cfg.IncludeTag=cfg.IncludeValue and
// (2) are not zero when cfg.ShowZeroValue is false.
// If a field has cfg.FieldNameTag, that value is used as the key; otherwise
// the struct field name is used. Non-struct, nil, or unsupported values return 0.
func measureKeyColumnWidth(v reflect.Value, cfg Config) int {

	for v.IsValid() && (v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface) {
		if v.IsNil() {
			return 0
		}
		v = v.Elem()
	}
	if !v.IsValid() || v.Kind() != reflect.Struct {
		return 0
	}

	max := 0
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		ft := t.Field(i)
		fv := v.Field(i)

		if !fv.CanInterface() {
			continue
		}
		if ft.Tag.Get(cfg.IncludeTag) != cfg.IncludeValue {
			continue
		}
		if !cfg.ShowZeroValue && isZeroValue(fv) {
			continue
		}

		name := ft.Tag.Get(cfg.FieldNameTag)
		if name == "" {
			name = ft.Name
		}
		if n := len(name); n > max {
			max = n
		}
	}
	return max
}

// formatValueWithVisited formats a reflect.Value using the specified format string, Config, and visited pointers.
// It is an internal helper function and not intended for public use.
func formatValueWithVisited(field reflect.Value, format string, cfg Config, visited map[uintptr]bool) string {
	switch field.Kind() {
	case reflect.Interface, reflect.Pointer:
		return stringifyValue(field, cfg, visited)
	case reflect.Struct:
		if hasAutoStringTags(field, cfg) {
			return stringifyValue(field, cfg, visited)
		}
	}
	if format == "" {
		format = DefaultFormat
	}
	return fmt.Sprintf(format, field.Interface())
}

// isZeroValue checks if a reflect.Value represents a zero value for its type.
// It is an internal helper function and not intended for public use.
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

// hasAutoStringTags checks if a struct value has any fields with the include tag specified in Config.
// It is an internal helper function and not intended for public use.
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
