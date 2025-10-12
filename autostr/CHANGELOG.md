# Changelog

All notable changes to the `autostr` module will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.5] - 2025-10-12

### Added
- **PrettyPrint mode** (`Config.PrettyPrint`) for aligned, multi-line value formatting with proper indentation.
- **Helper functions:**
  - `measureKeyColumnWidth` — computes key column width for aligned output.
  - `formatValueAligned` — aligns multi-line values under their corresponding key columns.
- **Extended test suite:**
  - Added PrettyPrint, alignment, and newline normalization tests.
  - Verified zero-value field exclusion effects on column width.
  - Confirmed support for custom separators including `\n`.

## [0.1.4] - 2025-10-09
### Fixed
- Docs and CI badge finalized; code formatted with `gofmt -s`.
### Maintenance
- Retracted versions [0.1.0–0.1.3] due to earlier force-moved tag and docs cleanup. Use 0.1.4+.

## [0.1.1] - 2025-10-09

### Added
- Support for `format` tag to customize field value formatting (e.g., `format:"%03d"` for zero-padded integers).

### Changed
- Updated `DefaultShowZeroValue` to `true` to include zero-value fields by default.

## [0.1.0] - 2025-09-08

### Added
- Initial release of `autostr`, a tag-based struct-to-string conversion library.
- Support for converting structs to human-readable strings using `string:"include"` tags.
- Custom field naming via `display` tags (e.g., `display:"FullName"`).
- Configurable separators (`Separator`, `FieldValueSeparator`) and zero-value display (`ShowZeroValue`).
- Support for the `AutoStringer` interface to override default string conversion.
- Safe handling of cyclic references with `<cycle>` output.
- Nested struct and pointer traversal.
- Comprehensive test suite covering basic functionality, nested structs, cycles, and configuration.
- CI/CD pipeline with GitHub Actions for testing, vetting, and code coverage.