# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this project is

`columnar` is a CLI tool that applies gofmt-style elastic tabstop alignment to source code in any language. It tokenizes lines, groups consecutive lines with the same indent level, and uses Go's `text/tabwriter` to pad columns so tokens align vertically.

## Commands

```bash
go build ./...                    # build
go test ./...                     # run all tests
go test ./internal/formatter/...  # run formatter tests only
go run main.go format <file>      # format a file to stdout
go run main.go format -w <file>   # format in place
go run main.go format -d <file>   # check mode (exit 1 if would change)
go run main.go format -l python <file>  # override language detection
go run main.go format -c .columnar.json <file>  # explicit config
```

## Architecture

**Pipeline:** `cmd/format.go` → `formatter.Format()` → `formatter.Tokenize()` → `text/tabwriter`

Three key packages:

- **`internal/formatter`** — core engine. `Format()` splits source into lines, detects indent unit, groups consecutive same-indent non-blank lines into a `[]row`, then flushes each group through `tabwriter`. Group boundaries: blank lines and indent changes.
- **`internal/formatter/tokenizer.go`** — `Tokenize()` splits a post-indent line into tab-separated cells. Respects bracket depth (content inside `()[]{}` is atomic), string literals, line comments (appended as final cell), block comments (absorbed into current cell), and standalone `=` operators. Config-style languages (`makefile`, `shellscript`, `properties`, `ini`, `toml`) use `AssignRHSAtomic=true` so the RHS of `=` is a single cell.
- **`internal/formatter/language.go`** — `LangConfig` maps language names to comment tokens and string quote chars. `DetectLanguage()` maps file extensions to language names.
- **`internal/config/config.go`** — loads `.columnar.json`; all fields default to enabled. Config is looked up from CWD automatically if not specified.

## Task Management

- Plan First: Write plan to 'tasks/todo.md' with checkable items
- Verify Plan: Check in before starting implementation
- Track Progress: Mark items complete as you go
- Explain Changes: High-level summary at each step
- Document Results: Add review to 'tasks/todo.md'
- Capture Lessons: Update 'tasks/lessons.md' after corrections

## Core Principles

- Simplicity First: Make every change as simple as possible. Impact minimal code.
- No Laziness: Find root causes. No temporary fixes. Senior developer standards.
- Minimal Impact: Changes should only touch what's necessary. Avoid introducing bugs.

## Testing

Fixture-based: `testdata/case<N>_<name>/before.<ext>` → `expected.<ext>`. Tests also verify idempotency (format twice = same result). To add a new case, add `before.*` and `expected.*` files under a new `testdata/case*/` directory — the test runner picks them up automatically via glob.

## Critical invariants

The formatter must only change whitespace — never join/split lines, never touch string literal content, never modify comment text (only their column position).
