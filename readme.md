# columnar

> elastic tabstop alignment for every language.

## why this exists

i spent a while writing go, and somewhere along the way i got used to code that looked like this:

```go
var (
    name        = "Alice"
    age         = 30
    isActive    = true
    emailAddr   = "alice@example.com"
)
```

then i'd switch to a python or typescript file and everything would go back to looking like unaligned prose:

```python
name = "Alice"
age = 30
is_active = True
email_address = "alice@example.com"
acceleration_due_to_gravity = 9.8
```

it's the same idea as gofmt's elastic tabstops, but language-agnostic. point it at almost any file and it lines up the columns:

```python
name                        = "Alice"
age                         = 30
is_active                   = True
email_address               = "alice@example.com"
acceleration_due_to_gravity = 9.8
```

that's it. that's the whole pitch.

## what it does

`columnar` reads a file, groups consecutive lines that share the same indent, splits each line into tab-separated cells, and runs them through go's [`text/tabwriter`](https://pkg.go.dev/text/tabwriter) to pad columns to a uniform width. blank lines and indent changes break groups so nested blocks don't stretch outer columns.

it works across:

- **c family** — java, javascript, typescript, c, c++, c#, kotlin, swift, dart, php
- **scripting** — python, ruby, lua, shell (`bash`/`zsh`/`sh`)
- **systems** — go, rust
- **config** — makefile, toml, yaml, ini, `.properties`, `.env`

…and falls back to plain text for anything else.

the tokenizer is deliberately dumb — it only knows about comments, string literals, brackets, and whitespace. it never parses syntax, never joins or splits lines, never touches string contents, never rewrites comment text. the only thing it changes is horizontal whitespace.

## examples

**imports line up by their `from` clause:**

```js
// before
import { useState, useEffect, useCallback } from "react";
import { Button, Input, Card } from "./components";
import { fetchUsers, createUser, deleteUser } from "./api";
import { formatDate } from "./utils";

// after
import { useState, useEffect, useCallback }   from "react";
import { Button, Input, Card }                from "./components";
import { fetchUsers, createUser, deleteUser } from "./api";
import { formatDate }                         from "./utils";
```

**struct fields, map entries, switch cases, enum members, trailing comments** — all fall into columns the same way. poke around [`testdata/`](testdata/) for the full gallery.

## installation

depending on how you want to use it:

- **homebrew (cli binary)** — see [`homebrew-columnar`](https://github.com/mudittt/homebrew-columnar). one-liner tap + install.
- **vs code** — see [`vscode-columnar`](https://github.com/mudittt/vscode-columnar). format-on-save, keybindings, the usual.
- **from source** — `go install github.com/mudittt/columnar@latest`

## usage

```bash
columnar format path/to/file.py           # print formatted output to stdout
columnar format -w path/to/file.py        # write in place
columnar format -d path/to/file.py        # check mode — exit 1 if it would change
columnar format -l python path/to/file    # force a language
columnar format -c .columnar.json file.py # use a specific config
```

if a `.columnar.json` exists in the working directory it's picked up automatically.

## configuration

a `.columnar.json` lets you toggle alignment features and tune spacing. all fields are optional — defaults turn everything on.

```json
{
  "minColumnGap": 1,
  "maxColumnWidth": 80,
  "indentSize": 4,
  "alignAssignments": true,
  "alignOperators": true,
  "alignComments": true,
  "alignMethodChains": true,
  "alignTernary": true,
  "alignEnums": true,
  "alignSwitchCases": true,
  "alignMapEntries": true,
  "alignStructFields": true,
  "alignImports": true,
  "alignFunctionParams": true,
  "alignArrayColumns": true,
  "formatMultilineStrings": false,
  "languages": {
    "python": { "commentToken": "#" }
  }
}
```

## how it works, briefly

the pipeline is deliberately small:

1. [`cmd/format.go`](cmd/format.go) reads the file and detects the language from its extension (or `-l`).
2. [`formatter.Format()`](internal/formatter/formatter.go) walks lines, tracking indent and multi-line-string state.
3. [`formatter.Tokenize()`](internal/formatter/tokenizer.go) splits each line into cells using only generic rules (brackets are atomic, strings are atomic, line comments become the trailing cell).
4. consecutive same-indent lines are grouped into a block, then flushed through `text/tabwriter`, which handles the elastic padding.

language-specific group-break heuristics live in [`group_cfamily.go`](internal/formatter/group_cfamily.go), [`group_python.go`](internal/formatter/group_python.go), [`group_ruby.go`](internal/formatter/group_ruby.go), [`group_lua.go`](internal/formatter/group_lua.go), and [`group_shell.go`](internal/formatter/group_shell.go). they decide when two adjacent same-indent lines shouldn't share a column block (e.g. an `if` header shouldn't align with the assignment above it).

**one invariant** runs through all of it: the formatter only changes whitespace. never the text, never the token order, never string or comment content. if `columnar` ever alters a character that isn't a space or tab, that's a bug.

## development

```bash
go build ./...                    # build
go test ./...                     # run all tests
go run main.go format testdata/case01_assignments/before.py
```

tests are fixture-based: drop a `before.<ext>` and `expected.<ext>` into a new `testdata/case*/` directory and the runner picks it up. every case is also run twice to verify idempotency — formatting an already-formatted file must be a no-op.

## license

[mit](LICENSE).
