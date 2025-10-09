package autostr_test

import (
	"testing"
	"strings"
	"github.com/azargarov/go-utils/autostr"
)

type Person struct {
	Name string `string:"include" display:"FullName"`
	Age  int    `string:"include"`
}

type Car struct{
	Make 	string `string:"include"`
	Price	float32 `string:"include" format:"%.2f"`
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
	// Person has tagged fields; Outer prints Person using tags too.
	// Order is struct field order.
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
	// Zero values included because ShowZeroValue=true
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
	// value receiver path should detect pointer AutoString via reflect.New trick
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
	b.Next = a // cycle

	got := autostr.String(a)
	// Next is included, but cycles collapse to <cycle>
	// Exact formatting depends on field order
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
	if got[19:] != "2.20" {
		t.Fatalf("display tag not applied: %q", got)
	}
}
func Test_ZeroValueConfig_FillsDefaults(t *testing.T) {
	// Users can pass zero-value Config and expect defaults to apply
	cfg := autostr.Config{}
	got := autostr.String(Person{"A", 2}, cfg)
	if got == "" || got == "<nil>" {
		t.Fatalf("defaults not applied: %q", got)
	}
}

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
