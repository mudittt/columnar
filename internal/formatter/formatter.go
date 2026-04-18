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

	indentUnit := detectIndentUnit(lines)
	if indentUnit <= 0 {
		indentUnit = 1
	}

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
			tw.Write([]byte(r.indent + strings.Join(r.cells, "\t") + "\n"))
		}
		tw.Flush()
		out.WriteString(trimTrailingSpaces(buf.String()))
		group = group[:0]
	}

	prevIndent := ""
	mlStringDelim := ""
	inBlockComment := false
	for _, line := range lines {
		indent, rest := SplitIndent(line)

		// When AlignComments is disabled, emit block comment bodies verbatim.
		if !cfg.AlignComments && langCfg.BlockCommentOpen != "" {
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

func detectIndentUnit(lines []string) int {
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
			if strings.Contains(indent, "\t") {
				return 1
			}
			return len(indent)
		}
	}
	return 1
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
