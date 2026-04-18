package formatter

// GroupBreakFunc decides whether to split the alignment group between two
// adjacent same-indent non-blank lines. Returning true ends the current
// group before curr; curr starts a new group.
//
// Group-break rules are language-specific by design. Each language owns its
// own rule in a dedicated group_<lang>.go file so edge cases can be tuned
// per-language without affecting other languages.
//
// prev and curr are the tokenized cells of the two lines (without indent).
// Both are guaranteed non-empty when this function is called.
type GroupBreakFunc func(prev, curr []string) bool

// UniversalBreak is a language-agnostic rule that every language-specific
// break function should compose with. It fires when either line consists
// solely of a block delimiter (`{` or `}`) — a structural line with no
// alignment content in any C-family syntax.
func UniversalBreak(prev, curr []string) bool {
	return isBraceOnly(prev) || isBraceOnly(curr)
}

func isBraceOnly(cells []string) bool {
	if len(cells) != 1 {
		return false
	}
	c := cells[0]
	return c == "{" || c == "}" || c == "};"
}

// firstWord returns the leading identifier-ish run of a cell, stopping at
// the first space, opening bracket, or punctuation that ends a keyword.
// Used by per-language rules to match the first token against a keyword
// set regardless of whether the keyword is followed by `(`, `{`, `:` etc.
func firstWord(cell string) string {
	for i := 0; i < len(cell); i++ {
		c := cell[i]
		if c == ' ' || c == '\t' || c == '(' || c == '{' || c == '[' || c == ':' || c == ';' {
			return cell[:i]
		}
	}
	return cell
}
