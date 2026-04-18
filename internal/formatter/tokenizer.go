package formatter

import (
	"strings"
	"unicode"
)

// SplitIndent separates leading whitespace from the rest of a line.
// Leading whitespace is returned verbatim so tabs vs spaces are preserved.
func SplitIndent(line string) (indent, rest string) {
	i := 0
	for i < len(line) && (line[i] == ' ' || line[i] == '\t') {
		i++
	}
	return line[:i], line[i:]
}

// Tokenize splits the post-indent portion of a line into alignment cells.
//
// Rules:
//   - Whitespace at bracket-depth 0 separates cells.
//   - Content inside matched () [] {} stays in a single cell (a balanced
//     parenthesised expression is atomic).
//   - String literals (using cfg.StringQuotes) are atomic.
//   - A line comment (cfg.LineComment) consumes the rest of the line into a
//     single final cell.
//   - Block comments are absorbed into the current cell.
//
// The tokenizer is language-agnostic by design: it knows only about comments,
// strings, brackets and whitespace. It does not parse syntax.
func Tokenize(rest string, cfg LangConfig) []string {
	runes := []rune(rest)
	var cells []string
	var cur strings.Builder
	depth := 0

	flush := func() {
		if cur.Len() > 0 {
			cells = append(cells, cur.String())
			cur.Reset()
		}
	}

	i := 0
	for i < len(runes) {
		c := runes[i]

		// Line comment: consume rest of line as one cell.
		if cfg.LineComment != "" && depth == 0 && hasPrefixAt(runes, i, cfg.LineComment) {
			flush()
			cells = append(cells, string(runes[i:]))
			return cells
		}

		// Block comment: absorb into current cell (keep with preceding code).
		if cfg.BlockCommentOpen != "" && hasPrefixAt(runes, i, cfg.BlockCommentOpen) {
			end := indexAt(runes, i+len([]rune(cfg.BlockCommentOpen)), cfg.BlockCommentClose)
			if end < 0 {
				flush()
				cells = append(cells, string(runes[i:]))
				return cells
			}
			endIdx := end + len([]rune(cfg.BlockCommentClose))
			cur.WriteString(string(runes[i:endIdx]))
			i = endIdx
			continue
		}

		// Whitespace at top level → cell boundary.
		if depth == 0 && unicode.IsSpace(c) {
			flush()
			for i < len(runes) && unicode.IsSpace(runes[i]) {
				i++
			}
			continue
		}

		// Strings.
		if isStringQuote(c, cfg.StringQuotes) {
			end := findStringEnd(runes, i)
			cur.WriteString(string(runes[i:end]))
			i = end
			continue
		}

		// Standalone `=`: split into its own cell so key=value aligns even
		// without surrounding whitespace. We skip compound operators like
		// ==, !=, <=, >=, =>, +=, -=, *=, /=, %=, :=, !==, ===, &=, |=, ^=.
		if depth == 0 && c == '=' && isStandaloneEquals(runes, i) {
			flush()
			cells = append(cells, "=")
			i++
			for i < len(runes) && unicode.IsSpace(runes[i]) {
				i++
			}
			if cfg.AssignRHSAtomic {
				rhs, comment := splitTrailingComment(string(runes[i:]), cfg)
				rhs = strings.TrimRight(rhs, " \t")
				if rhs != "" {
					cells = append(cells, rhs)
				}
				if comment != "" {
					cells = append(cells, comment)
				}
				return coalesceCommas(cells)
			}
			continue
		}

		switch c {
		case '(', '[', '{':
			depth++
			cur.WriteRune(c)
			i++
			continue
		case ')', ']', '}':
			if depth > 0 {
				depth--
			}
			cur.WriteRune(c)
			i++
			continue
		}

		cur.WriteRune(c)
		i++
	}

	flush()
	return coalesceCommas(cells)
}

// coalesceCommas merges cells across a trailing-comma boundary so that a
// comma-separated list written on one line reads as a single logical cell.
//
// Rationale: "import Flask, request, jsonify" tokenises as three cells, but
// for alignment it should behave like one — otherwise tabwriter starts
// padding between list members and we lose the comma-delimited look.
func coalesceCommas(cells []string) []string {
	if len(cells) == 0 {
		return cells
	}
	out := make([]string, 0, len(cells))
	for _, c := range cells {
		if len(out) > 0 && endsWithComma(out[len(out)-1]) {
			out[len(out)-1] = out[len(out)-1] + " " + c
			continue
		}
		out = append(out, c)
	}
	return out
}

// splitTrailingComment returns (codePart, commentPart) for a string that may
// contain a trailing line comment. Respects strings so a `#` inside a quoted
// literal is not treated as a comment.
func splitTrailingComment(s string, cfg LangConfig) (string, string) {
	if cfg.LineComment == "" {
		return s, ""
	}
	runes := []rune(s)
	for i := 0; i < len(runes); i++ {
		c := runes[i]
		if isStringQuote(c, cfg.StringQuotes) {
			i = findStringEnd(runes, i) - 1
			continue
		}
		if hasPrefixAt(runes, i, cfg.LineComment) {
			return string(runes[:i]), string(runes[i:])
		}
	}
	return s, ""
}

// isStandaloneEquals reports whether the `=` at runes[i] is a bare assignment
// operator rather than part of a compound operator such as ==, !=, <=, >=,
// =>, +=, etc.
func isStandaloneEquals(runes []rune, i int) bool {
	// Adjacent-to-right compound: ==, =>.
	if i+1 < len(runes) {
		next := runes[i+1]
		if next == '=' || next == '>' {
			return false
		}
	}
	// Adjacent-to-left compound: preceded by an operator char.
	if i > 0 {
		prev := runes[i-1]
		switch prev {
		case '=', '<', '>', '!', '+', '-', '*', '/', '%', ':', '~', '&', '|', '^':
			return false
		}
	}
	return true
}

func endsWithComma(s string) bool {
	if s == "" {
		return false
	}
	// Use rune-safe last char.
	last := s[len(s)-1]
	return last == ','
}

func hasPrefixAt(runes []rune, i int, s string) bool {
	sr := []rune(s)
	if i+len(sr) > len(runes) {
		return false
	}
	for k := 0; k < len(sr); k++ {
		if runes[i+k] != sr[k] {
			return false
		}
	}
	return true
}

func indexAt(runes []rune, from int, s string) int {
	sr := []rune(s)
	if len(sr) == 0 {
		return from
	}
	for i := from; i+len(sr) <= len(runes); i++ {
		match := true
		for k := 0; k < len(sr); k++ {
			if runes[i+k] != sr[k] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

func isStringQuote(c rune, quotes []rune) bool {
	for _, q := range quotes {
		if q == c {
			return true
		}
	}
	return false
}

// findStringEnd returns the index just after the closing quote of a string
// that starts at start. Handles backslash escapes. If the string is unclosed
// it returns len(runes).
func findStringEnd(runes []rune, start int) int {
	quote := runes[start]
	i := start + 1
	for i < len(runes) {
		c := runes[i]
		if c == '\\' && i+1 < len(runes) {
			i += 2
			continue
		}
		if c == quote {
			return i + 1
		}
		i++
	}
	return len(runes)
}
