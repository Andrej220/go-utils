# Roadmap

## Core behavior
- [ ] **Interface-cycle detection**: detect cycles reachable only via `interface{}` (separate `visitedIface` keyed by underlying pointer identity when available; fall back to (type, address) pairs for pointer-backed values).
- [ ] **Max-depth & truncation**: `Config.MaxDepth` and `Config.Truncate` (runes), with ellipsis `…` to avoid runaway output.
- [ ] **Per-field options**: support tag modifiers akin to struct tags, e.g. `string:"include,omitempty"` (omit zero per-field even if `ShowZeroValue=true`) and `string:"-"` to force-skip.
- [ ] **Field order control**: optional `order:"<int>"` tag to override struct declaration order.
- [ ] **Nested/inline control**: `inline:"true"` to flatten embedded structs (print their fields at the parent level).
- [ ] **Collections**: rules for slices/maps (limit length via `MaxItems`, stable map ordering by key fmt).
- [ ] **Stringer hooks**: prefer `fmt.Stringer` / `encoding.TextMarshaler` / `json.Marshaler` before reflection when present (opt-in via config).
- [ ] **Time & duration helpers**: default friendly formats with per-field override (`timeFormat:"2006-01-02 15:04"`).

## Formatting & UX
- [ ] **Unicode-aware alignment**: measure key width in runes (and optionally visual cells) to align with non-ASCII keys; guard without external deps, optionally add a `runeWidth` strategy hook.
- [ ] **Tab-stop alignment**: `indentChar="\t"` support with configurable tab width.
- [ ] **ANSI styling (opt-in)**: simple color themes for keys/colons/values for CLI logs; no ANSI by default.
- [ ] **PrettyPrint policy**: knobs for line wrapping (`WrapAt`), continuation prefix string (not only `indentChar+sep`).

## Performance
- [ ] **Reflection cache**: cache field metadata per `reflect.Type` (included fields, display names, formatters, zero-check funcs) using `sync.Map`.
- [ ] **Formatter fast-paths**: avoid `fmt.Sprintf` when `format == "%v"` and kind is basic (int, float, bool, string).
- [ ] **Builder reuse**: optional `sync.Pool` for `strings.Builder`/buffers in hot paths.
- [ ] **Zero-check specialization**: precompute zero-check closures per field kind to reduce `reflect.Value` calls.
- [ ] **Bench targets**: add micro-benchmarks for small/medium structs; set budget (e.g., ≤ 2 allocs, ≤ 1µs for small struct on amd64).

## Reliability & Safety
- [ ] **Fuzzing**: `go test -fuzz` corpus for random nested types (pointers, maps, slices, interfaces) to harden cycle/nil paths.
- [ ] **Panic safety**: recover from unexpected reflection panics and print `<error: ...>` when `Config.Safe=true`.
- [ ] **Race scan**: CI job with `-race` and parallel fuzz to catch shared cache hazards.

## API & DX
- [ ] **Stable API surface**: mark `Config` and `String` as v1-stable; keep internals (`ensureDefaults`, etc.) unexported.
- [ ] **Writer API**: `Write(obj any, w io.Writer, cfg ...Config) (int, error)` to avoid intermediate strings.
- [ ] **Examples**: GoDoc examples for PrettyPrint, nested structs, custom formats, zero-value policies.
- [ ] **README gallery**: before/after snippets for PrettyPrint, custom separators (`"\n"`, `" | "`), and multiline alignment.
- [ ] **Compatibility note**: document how map order is stabilized and any guarantees (or lack thereof).

## Housekeeping
- [ ] **Issue labels**: perf, formatting, api, bug, docs.
- [ ] **CI matrix**: Go 1.22–1.23; linux/windows/darwin; `-race`, `-covermode=atomic`.
- [ ] **Release checklist**: bump, changelog, tag, `pkg.go.dev` badges, example screenshots for PrettyPrint.

### Notes on interface-cycle detection
- Track pointer cycles as you do now (`map[uintptr]bool`).
- For `interface{}` values:
  - If the dynamic value is a pointer, use its address as usual.
  - If it’s a non-pointer but references heap data (e.g., slice/map), you can key by the pointer to its underlying header (read via reflection) to detect cycles through containers.
  - Keep a separate `visitedContainers` keyed by (type, pointer) for slices/maps.
- Keep it conservative: if identity can’t be established safely, don’t mark as visited to avoid false positives; rely on depth limits as a safety net.

