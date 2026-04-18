package formatter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestFixtures walks testdata/ and for every before.<ext> runs Format and
// compares against expected.<ext>. It also verifies that running Format a
// second time on the output is a no-op (idempotency).
func TestFixtures(t *testing.T) {
	beforeFiles, err := filepath.Glob("../../testdata/case*/before.*")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	if len(beforeFiles) == 0 {
		t.Fatal("no fixtures found under testdata/")
	}

	for _, before := range beforeFiles {
		before := before
		dir := filepath.Dir(before)
		ext := filepath.Ext(before)
		expected := filepath.Join(dir, "expected"+ext)
		name := filepath.Base(dir) + "/" + filepath.Base(before)

		t.Run(name, func(t *testing.T) {
			beforeSrc, err := os.ReadFile(before)
			if err != nil {
				t.Fatalf("read before: %v", err)
			}
			wantSrc, err := os.ReadFile(expected)
			if err != nil {
				t.Fatalf("read expected: %v", err)
			}
			lang := DetectLanguage(expected)
			got, err := Format(string(beforeSrc), lang, nil)
			if err != nil {
				t.Fatalf("Format: %v", err)
			}
			if got != string(wantSrc) {
				t.Errorf("mismatch for %s (lang=%s)\n=== got ===\n%s\n=== want ===\n%s",
					name, lang, visualize(got), visualize(string(wantSrc)))
				return
			}
			// Idempotency: formatting the formatted output is a no-op.
			got2, err := Format(got, lang, nil)
			if err != nil {
				t.Fatalf("Format (2nd pass): %v", err)
			}
			if got2 != got {
				t.Errorf("not idempotent for %s\n=== pass1 ===\n%s\n=== pass2 ===\n%s",
					name, visualize(got), visualize(got2))
			}
		})
	}
}

// visualize makes trailing whitespace and tabs visible in error messages.
func visualize(s string) string {
	lines := strings.Split(s, "\n")
	var b strings.Builder
	for i, line := range lines {
		trailing := len(line) - len(strings.TrimRight(line, " \t"))
		b.WriteString(strings.TrimRight(line, " \t"))
		if trailing > 0 {
			b.WriteString(strings.Repeat("·", trailing))
		}
		if i < len(lines)-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

// TestStringLiteralsUntouched verifies that content inside quoted strings is
// never modified by the formatter, even if it contains tokens that would
// normally split cells.
func TestStringLiteralsUntouched(t *testing.T) {
	cases := []struct {
		lang string
		src  string
	}{
		{"java", `String a = "one = two = three";
String ab = "x";`},
		{"python", `a = "x = y = z"
abcdef = "y"`},
		{"javascript", "const x = `a = b = c`;\nconst yyy = 1;"},
	}
	for _, tc := range cases {
		got, err := Format(tc.src, tc.lang, nil)
		if err != nil {
			t.Fatalf("%s: %v", tc.lang, err)
		}
		// The literal content must survive verbatim.
		for _, wanted := range []string{`"one = two = three"`, `"x = y = z"`, "`a = b = c`"} {
			if strings.Contains(tc.src, wanted) && !strings.Contains(got, wanted) {
				t.Errorf("string literal %q was modified in %s lang output:\n%s", wanted, tc.lang, got)
			}
		}
	}
}

// TestBlankLinesBreakGroups verifies that a blank line splits an alignment
// group so the two halves can have different column widths.
func TestBlankLinesBreakGroups(t *testing.T) {
	src := `a = 1
bb = 2

c = 3
ddddd = 4
`
	got, err := Format(src, "python", nil)
	if err != nil {
		t.Fatalf("Format: %v", err)
	}
	// If groups were joined, `a` would be padded to width 5 (matching ddddd).
	// With the blank-line split, `a` should be padded only to width 2.
	want := `a  = 1
bb = 2

c     = 3
ddddd = 4
`
	if got != want {
		t.Errorf("blank-line grouping wrong\n=== got ===\n%s\n=== want ===\n%s",
			visualize(got), visualize(want))
	}
}

// TestIndentChangesBreakGroups verifies that a deeper-indented block doesn't
// stretch the enclosing column widths.
func TestIndentChangesBreakGroups(t *testing.T) {
	src := `a = 1
bb = 2
    nestedlong = 3
    x = 4
`
	got, err := Format(src, "python", nil)
	if err != nil {
		t.Fatalf("Format: %v", err)
	}
	want := `a  = 1
bb = 2
    nestedlong = 3
    x          = 4
`
	if got != want {
		t.Errorf("indent-change grouping wrong\n=== got ===\n%s\n=== want ===\n%s",
			visualize(got), visualize(want))
	}
}

// TestNoTrailingNewlinePreserved verifies that a file without a trailing
// newline doesn't gain one after formatting.
func TestNoTrailingNewlinePreserved(t *testing.T) {
	src := "x = 1\ny = 2"
	got, err := Format(src, "python", nil)
	if err != nil {
		t.Fatalf("Format: %v", err)
	}
	if strings.HasSuffix(got, "\n") {
		t.Errorf("gained a trailing newline: %q", got)
	}
}

// TestDetectLanguage spot-checks a few file extensions.
func TestDetectLanguage(t *testing.T) {
	cases := map[string]string{
		"main.java":    "java",
		"app.py":       "python",
		"index.ts":     "typescript",
		"config.rs":    "rust",
		"Makefile":     "makefile",
		"makefile":     "makefile",
		".env":         "properties",
		".env.local":   "properties",
		"styles.rb":    "ruby",
		"notes.txt":    "plaintext",
		"unknown.xyz":  "plaintext",
		"something.go": "go",
	}
	for path, want := range cases {
		got := DetectLanguage(path)
		if got != want {
			t.Errorf("DetectLanguage(%q) = %q, want %q", path, got, want)
		}
	}
}
