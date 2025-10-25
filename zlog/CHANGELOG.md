
---

## 🕑 **CHANGELOG.md (updated for zLog rename)**

```markdown
# Changelog

All notable changes to this project will be documented in this file.

---

## [v0.2.1] - 2025-10-25
### Changed
- Renamed internal `zapLogger` → **`zLog`** for naming consistency.
- Updated all references, tests, and README documentation.
- Improved `With(...)` in `defaultLogger` to preserve output routing.
- Minor README fixes (correct `go get` path, expanded interface docs).

---

## [v0.2.0] - 2025-10-25
### Added
- `ZLogger.RedirectStdLog(level)` to redirect global `log` package output.
- `ZLogger.RedirectOutput(writer, level)` to dynamically redirect logs per writer.
- New `defaultLogger` with per-level `*log.Logger` instances for thread-safe stdlib fallback.
- `TestZapRedirectOutput` and `TestDefaultRedirectOutputLevels` for verifying redirect behavior.
- `noopLogger` (a.k.a. `zlog.Discard`) with full interface compliance.
- Enhanced `flatten()` helper for deterministic `key=value` formatting.

### Changed
- Switched all `defaultLogger` methods to **pointer receivers**.
- `With(...)` now preserves existing output mappings (`RedirectOutput` safety).
- `Info`, `Warn`, `Error`, and `Debug` use dedicated per-level loggers (no `SetOutput` churn).
- `RedirectStdLogger` restores previous `log` prefix and flags properly.
- `FromContext` now returns `*defaultLogger` (pointer) instead of value.

### Fixed
- Panic when using `defaultLogger` before initializing mutex.
- Level filtering in `zLog.RedirectOutput` now correctly uses a `LevelEnabler`.

---

## [v0.1.0] - 2025-10-01
### Added
- Initial release with:
  - `ZLogger` interface
  - Zap-based logger (`zLog`)
  - Stdlib fallback logger (`defaultLogger`)
  - Context helpers (`Attach`, `FromContext`)
  - Field helpers (`String`, `Int`, `Bool`, etc.)
  - `Discard` noop logger for tests
