// Package formatter implements the core elastic-tabstop alignment engine.
//
// The approach mirrors gofmt's: tokenize each line into tab-separated cells
// and run them through text/tabwriter, which groups consecutive lines with
// matching column structure and pads each column to its max width.
//
// Group boundaries:
//   - Blank lines (no non-whitespace content) break groups.
//   - Indent changes break groups — otherwise lines at deeper indent would
//     stretch the outer column widths.
//
// Within a group, tabwriter handles the elastic padding automatically.
package formatter

import (
	"bytes"
	"strings"
	"text/tabwriter"

	"github.com/mudittt/columnar/internal/config"
)

// updateMLState scans line given the current multiline-string delimiter and
// returns the active delimiter after the line ends. Returns "" when no
// multiline string is open at end of line.
func updateMLState(line, currentDelim string, delims []string) string {
	if len(delims) == 0 {
		return currentDelim
	}
	s := line
	inML := currentDelim
	for len(s) > 0 {
		if inML != "" {
			idx := strings.Index(s, inML)
			if idx < 0 {
				return inML // delimiter not closed on this line
			}
			s = s[idx+len(inML):]
			inML = ""
			continue
		}
		earliest := -1
		var found string
		for _, d := range delims {
			if idx := strings.Index(s, d); idx >= 0 && (earliest < 0 || idx < earliest) {
				earliest = idx
				found = d
			}
		}
		if earliest < 0 {
			break
		}
		s = s[earliest+len(found):]
		inML = found
	}
	return inML
}

type row struct {
	indent string
	cells  []string
}

// Format takes source code and returns it with elastic tabstop alignment.
// If cfg is nil, sensible defaults are used.
func Format(src, language string, cfg *config.Config) (string, error) {
	if cfg == nil {
		cfg = config.Default()
	}
	langCfg := GetLangConfig(language)
	if override, ok := cfg.Languages[langCfg.Name]; ok {
		if override.CommentToken != "" {
			langCfg.LineComment = override.CommentToken
		}
	}

	hasTrailingNewline := strings.HasSuffix(src, "\n")
	lines := strings.Split(src, "\n")
	if hasTrailingNewline && len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	tabWidth := cfg.IndentSize
	if tabWidth <= 0 {
		tabWidth = 4
	}
	indentUnit := detectIndentUnit(lines, tabWidth)
	if indentUnit <= 0 {
		indentUnit = 1
	}
	useTabs := detectTabIndent(lines)

	var out strings.Builder
	var group []row
	padding := cfg.MinColumnGap
	if padding <= 0 {
		padding = 1
	}

	flushGroup := func() {
		if len(group) == 0 {
			return
		}
		var buf bytes.Buffer
		tw := tabwriter.NewWriter(&buf, 0, 1, padding, ' ', 0)
		for _, r := range group {
			// Feed spaces-only indent to tabwriter — a literal \t there would
			// be treated as a cell delimiter. We'll convert the leading
			// spaces back to tabs post-flush when the source uses tabs.
			tw.Write([]byte(r.indent + strings.Join(r.cells, "\t") + "\n"))
		}
		tw.Flush()
		flushed := trimTrailingSpaces(buf.String())
		if useTabs {
			flushed = leadingSpacesToTabs(flushed, cfg.IndentSize)
		}
		out.WriteString(flushed)
		group = group[:0]
	}

	prevIndent := ""
	mlStringDelim := ""
	inBlockComment := false
	for _, line := range lines {
		rawIndent, rest := SplitIndent(line)
		// Expand leading-indent tabs to spaces so (a) mixed tab/space files
		// normalize consistently and (b) literal tabs never reach tabwriter
		// as the indent prefix (it would treat them as cell delimiters).
		indent := expandIndentTabs(rawIndent, tabWidth)

		// Block comment bodies are always emitted verbatim — modifying comment
		// text would violate the whitespace-only invariant. (AlignComments
		// governs trailing `// comment` alignment, not block-comment bodies.)
		if langCfg.BlockCommentOpen != "" {
			if inBlockComment {
				flushGroup()
				out.WriteString(line + "\n")
				prevIndent = indent
				if strings.Contains(rest, langCfg.BlockCommentClose) {
					inBlockComment = false
				}
				continue
			}
			if idx := strings.Index(rest, langCfg.BlockCommentOpen); idx >= 0 {
				after := rest[idx+len(langCfg.BlockCommentOpen):]
				if !strings.Contains(after, langCfg.BlockCommentClose) {
					flushGroup()
					inBlockComment = true
					out.WriteString(line + "\n")
					prevIndent = indent
					continue
				}
			}
		}

		if !cfg.FormatMultilineStrings {
			// Lines inside an unclosed multiline string must be emitted verbatim.
			if mlStringDelim != "" {
				flushGroup()
				mlStringDelim = updateMLState(rest, mlStringDelim, langCfg.MultilineStringDelims)
				out.WriteString(line + "\n")
				prevIndent = indent
				continue
			}

			if strings.TrimSpace(rest) != "" {
				// Check if this line opens a multiline string that doesn't close here.
				if newDelim := updateMLState(rest, "", langCfg.MultilineStringDelims); newDelim != "" {
					flushGroup()
					mlStringDelim = newDelim
					out.WriteString(line + "\n")
					prevIndent = indent
					continue
				}
			}
		}

		cells := Tokenize(rest, langCfg)
		if len(group) > 0 {
			if indent != prevIndent {
				flushGroup()
			} else if langCfg.GroupBreak != nil && langCfg.GroupBreak(group[len(group)-1].cells, cells) {
				flushGroup()
			}
		}
		normalizedIndent := normalizeIndent(indent, indentUnit, cfg.IndentSize)
		group = append(group, row{indent: normalizedIndent, cells: cells})
		prevIndent = indent
	}
	flushGroup()

	result := out.String()
	if hasTrailingNewline {
		if !strings.HasSuffix(result, "\n") {
			result += "\n"
		}
	} else {
		result = strings.TrimSuffix(result, "\n")
	}
	return result, nil
}

func trimTrailingSpaces(s string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}
	return strings.Join(lines, "\n")
}

func detectIndentUnit(lines []string, tabWidth int) int {
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		indent, rest := SplitIndent(line)
		if len(indent) > 0 {
			// Skip block comment continuation lines (e.g. " * text") — their
			// single-space indent would corrupt indent-unit detection.
			if strings.HasPrefix(rest, "*") {
				continue
			}
			return len(expandIndentTabs(indent, tabWidth))
		}
	}
	return 1
}

// detectTabIndent reports whether the source predominantly uses tab
// indentation. When true, we emit tabs in the output indent so we respect
// the file's existing style instead of forcing spaces.
func detectTabIndent(lines []string) bool {
	tabCount, spaceCount := 0, 0
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		switch line[0] {
		case '\t':
			tabCount++
		case ' ':
			spaceCount++
		}
	}
	return tabCount > spaceCount
}

// leadingSpacesToTabs replaces leading space runs with tabs at unit-sized
// stops. Applied after tabwriter has flushed, so inter-cell padding spaces
// are preserved while the indent prefix is converted back to tabs.
func leadingSpacesToTabs(s string, unit int) string {
	if unit <= 0 {
		return s
	}
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lead := 0
		for lead < len(line) && line[lead] == ' ' {
			lead++
		}
		tabs := lead / unit
		if tabs > 0 {
			lines[i] = strings.Repeat("\t", tabs) + line[tabs*unit:]
		}
	}
	return strings.Join(lines, "\n")
}

// expandIndentTabs replaces tabs in the leading indent with spaces, using
// tabWidth as the stop. Done so mixed tab/space files normalize uniformly
// and literal tabs never reach tabwriter as the indent prefix.
func expandIndentTabs(indent string, tabWidth int) string {
	if !strings.ContainsRune(indent, '\t') {
		return indent
	}
	if tabWidth <= 0 {
		tabWidth = 4
	}
	var b strings.Builder
	col := 0
	for _, c := range indent {
		if c == '\t' {
			n := tabWidth - (col % tabWidth)
			b.WriteString(strings.Repeat(" ", n))
			col += n
			continue
		}
		b.WriteRune(c)
		col++
	}
	return b.String()
}

func normalizeIndent(indent string, indentUnit, targetSize int) string {
	if len(indent) == 0 {
		return ""
	}
	// Sub-unit indents (e.g. the " *" lines in block comments) are not real
	// indent levels — preserve them verbatim instead of quantizing them.
	if len(indent) < indentUnit {
		return indent
	}
	depth := len(indent) / indentUnit
	return strings.Repeat(" ", depth*targetSize)
}
