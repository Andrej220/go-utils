# autostr 🧩 — Tag-based struct-to-string converter for Go

[![Go Reference](https://pkg.go.dev/badge/github.com/azargarov/autostr.svg)](https://pkg.go.dev/github.com/azargarov/autostr)
[![Go Tests](https://github.com/azargarov/autostr/actions/workflows/ci.yml/badge.svg)](https://github.com/azargarov/autostr/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/azargarov/autostr)](https://goreportcard.com/report/github.com/azargarov/autostr)

`autostr` is a small, reflection-based Go library that automatically converts structs into human-readable strings using struct tags — similar to how `encoding/json` converts structs into JSON.

It’s designed for logging, debugging, and CLI output when you want control over **which fields** are shown and **how** they’re displayed — without writing manual `String()` methods.

---

## ✨ Features

- Tag-driven field inclusion (`string:"include"`)
- Custom field labels via tag (`display:"Alias"`)
- Nested struct and pointer traversal
- Optional `AutoString()` override per type
- Safe cycle detection for linked data
- Configurable separators and zero-value display
- Default configuration with lazy fallback (like `http.Client`)

---

## Example
```Bash
type Person struct {
    Name string `string:"include" display:"FullName"`
    Age  int    `string:"include"`
    ID   int    // excluded
}
p := Person{Name: "Alice", Age: 30}
fmt.Println(autostr.String(p)) // Output: FullName: Alice, Age: 30
```
---
## 📦 Installation

```bash
go get github.com/azargarov/go-utils/autostr
