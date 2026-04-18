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
	for _, line := range lines {
		indent, rest := SplitIndent(line)
		if strings.TrimSpace(rest) == "" {
			flushGroup()
			out.WriteString("\n")
			continue
		}
		if len(group) > 0 && indent != prevIndent {
			flushGroup()
		}
		cells := Tokenize(rest, langCfg)
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
		indent, _ := SplitIndent(line)
		if len(indent) > 0 {
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
	depth := len(indent) / indentUnit
	if depth == 0 && len(indent) > 0 {
		depth = 1
	}
	return strings.Repeat(" ", depth*targetSize)
}
