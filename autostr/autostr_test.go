package autostr_test

import (
	"github.com/azargarov/go-utils/autostr"
	"strings"
	"testing"
)

type Person struct {
	Name string `string:"include" display:"FullName"`
	Age  int    `string:"include"`
}

type Car struct {
	Make  string  `string:"include"`
	Price float32 `string:"include" format:"%.2f"`
}

type Outer struct {
	Person Person `string:"include"`
	Note   string `string:"include"`
}

type WithPtr struct {
	Next *WithPtr `string:"include"`
	V    int      `string:"include"`
}

type WithAuto struct {
	X int `string:"include"`
}

func (w *WithAuto) AutoString() string { return "WithAuto<X>" }

func Test_Defaults_Basic(t *testing.T) {
	p := Person{Name: "Alice", Age: 30}
	got := autostr.String(p)
	want := "FullName: Alice, Age: 30"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func Test_Nested_WithTags(t *testing.T) {
	o := Outer{
		Person: Person{Name: "Bob", Age: 40},
		Note:   "n1",
	}
	got := autostr.String(o)
	want := "Person: FullName: Bob, Age: 40, Note: n1"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func Test_EmptySeparators_AreRespected(t *testing.T) {
	cfg := autostr.DefaultConfig()
	cfg.Separator = autostr.Ptr("")
	cfg.FieldValueSeparator = autostr.Ptr("")

	p := Person{Name: "A", Age: 1}
	got := autostr.String(p, cfg)
	want := "FullNameAAge1"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func Test_ShowZeroValue(t *testing.T) {
	p := Person{Name: "", Age: 0}
	cfg := autostr.DefaultConfig()
	cfg.ShowZeroValue = true
	got := autostr.String(p, cfg)
	want := "FullName: , Age: 0"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func Test_NilInterface_IsHandled(t *testing.T) {
	var x any = nil
	got := autostr.String(x)
	if got != "<nil>" {
		t.Fatalf("got %q, want <nil>", got)
	}
}

func Test_PointerReceiver_AutoString_Detected(t *testing.T) {
	got1 := autostr.String(WithAuto{X: 1})
	got2 := autostr.String(&WithAuto{X: 1})
	if got1 != "WithAuto<X>" || got2 != "WithAuto<X>" {
		t.Fatalf("AutoString not used: got1=%q got2=%q", got1, got2)
	}
}

func Test_Cycle_Detected(t *testing.T) {
	a := &WithPtr{V: 1}
	b := &WithPtr{V: 2}
	a.Next = b
	b.Next = a

	got := autostr.String(a)
	if got == "" || got == "<nil>" || got == "V: 1" {
		t.Fatalf("unexpected cycle handling: %q", got)
	}
	if !containsAll(got, []string{"V: 1", "V: 2", "<cycle>"}) {
		t.Fatalf("expected to contain V:1, V:2 and <cycle>, got %q", got)
	}
}

func Test_FieldDisplayNameTag(t *testing.T) {
	p := Person{Name: "Zoe", Age: 7}
	got := autostr.String(p)
	if got[:8] != "FullName" {
		t.Fatalf("display tag not applied: %q", got)
	}
}

func Test_FormatTag(t *testing.T) {
	p := Car{Make: "Opel", Price: 2.20}
	got := autostr.String(p, autostr.DefaultConfig())
	// "Make: Opel, Price: 2.20" -> the value starts at index 19
	if len(got) < 23 || got[19:] != "2.20" {
		t.Fatalf("format tag not applied correctly, got %q", got)
	}
}

func Test_ZeroValueConfig_FillsDefaults(t *testing.T) {
	cfg := autostr.Config{} // zero-value config; ensureDefaults should fill most defaults
	got := autostr.String(Person{"A", 2}, cfg)
	if got == "" || got == "<nil>" {
		t.Fatalf("defaults not applied: %q", got)
	}
}

// --- PrettyPrint & alignment tests ---

// Uses PrettyPrint=true to ensure multi-line values align under the value column.
// Also verifies that Windows newlines are normalized and trailing newlines trimmed.
func Test_PrettyPrint_MultilineAlignment(t *testing.T) {
	type Doc struct {
		Title string `string:"include" display:"T"`
		Body  string `string:"include" display:"Body"`
	}
	v := Doc{
		Title: "One",
		Body:  "line1\r\nline2\n", // includes Windows-style \r\n and trailing \n
	}

	cfg := autostr.DefaultConfig()
	cfg.PrettyPrint = true

	got := autostr.String(v, cfg)
	// indent = max(len("T"), len("Body")) = 4
	// For "T" (len=1): pad=3 => "T" + "   : One"
	// Separator between fields is ", "
	// For "Body" (len=4): pad=0, first line ": line1"; subsequent lines prefixed with 4 spaces + ": "
	// Windows newlines normalized and trailing newline trimmed
	want := "T   : One, Body: line1\n    : line2"
	if got != want {
		t.Fatalf("PrettyPrint alignment mismatch.\nGot:\n%q\nWant:\n%q", got, want)
	}
}

// Ensures that when ShowZeroValue=false, zero-value fields are omitted from both output
// and width calculation (so long key names that are zero don't affect alignment).
func Test_PrettyPrint_WidthIgnoresZeroWhenShowZeroFalse(t *testing.T) {
	type S struct {
		LongZero string `string:"include" display:"VeryLongKey"`
		Short    string `string:"include" display:"K"`
	}
	v := S{
		LongZero: "",    // zero-value, should be omitted
		Short:    "val", // only field shown
	}

	cfg := autostr.DefaultConfig()
	cfg.PrettyPrint = true
	cfg.ShowZeroValue = false

	got := autostr.String(v, cfg)
	// Because LongZero is omitted, indent = len("K") = 1; pad for "K" is 0
	// So no extra left spaces before ": "
	want := "K: val"
	if got != want {
		t.Fatalf("width calc with zero omitted mismatch.\nGot:  %q\nWant: %q", got, want)
	}
}

// Ensures PrettyPrint plays nicely with custom separators.
func Test_PrettyPrint_CustomSeparators_Newline(t *testing.T) {
	type P struct {
		A string `string:"include" display:"A"`
		B string `string:"include" display:"BBB"`
	}
	v := P{A: "x", B: "y\nz"}

	cfg := autostr.DefaultConfig()
	cfg.PrettyPrint = true
	cfg.Separator = autostr.Ptr("\n")
	cfg.FieldValueSeparator = autostr.Ptr(" : ")

	// indent = 3; colon column aligns across all lines:
	// "A"   -> pad=2 + " : "  => "A   : x"
	// "BBB" -> pad=0 + " : "  => "BBB : y"
	// cont  -> indent(3) + " : " => "    : z" (4 spaces before ':')
	want := "A   : x\nBBB : y\n    : z"
	got := autostr.String(v, cfg)
	if got != want {
		t.Fatalf("PrettyPrint with custom separators mismatch.\nGot:\n%q\nWant:\n%q", got, want)
	}
}

// Ensures PrettyPrint plays nicely with custom separators.
func Test_PrettyPrint_PipeSeparators(t *testing.T) {
	type P struct {
		A string `string:"include" display:"A"`
		B string `string:"include" display:"BBB"`
	}
	v := P{A: "x", B: "y\nz"}

	cfg := autostr.DefaultConfig()
	cfg.PrettyPrint = true
	cfg.Separator = autostr.Ptr(" | ")
	cfg.FieldValueSeparator = autostr.Ptr(" : ")

	want := "A   : x | BBB : y\n    : z"
	got := autostr.String(v, cfg)
	if got != want {
		t.Fatalf("PrettyPrint with custom separators mismatch.\nGot:\n%q\nWant:\n%q", got, want)
	}
}

// helpers
func containsAll(s string, parts []string) bool {
	for _, p := range parts {
		if !strings.Contains(s, p) {
			return false
		}
	}
	return true
}

type Bench struct {
	A string `string:"include"`
	B int    `string:"include"`
	C bool   `string:"include"`
	D int64  `string:"include"`
}

func BenchmarkString_Defaults(b *testing.B) {
	v := Bench{A: "hello", B: 42, C: true, D: 12345}
	for i := 0; i < b.N; i++ {
		_ = autostr.String(v)
	}
}
