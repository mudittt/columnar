package formatter

// cFamilyControlFlow lists the leading keywords that mark a line as a
// structural / control-flow statement rather than a declaration or
// expression. A line starting with any of these should not share an
// alignment group with an adjacent declaration or assignment — otherwise
// tabwriter would pad unrelated columns (e.g. `try {` aligning with a
// `String x = ...;` above it).
//
// The set is intentionally a superset of what any single C-family language
// uses. A keyword that doesn't exist in a given language simply never
// appears as a first token, so there is no harm in including it.
var cFamilyControlFlow = map[string]bool{
	"if":       true,
	"else":     true,
	"for":      true,
	"foreach":  true,
	"while":    true,
	"do":       true,
	"switch":   true,
	"case":     true,
	"default":  true,
	"try":      true,
	"catch":    true,
	"finally":  true,
	"return":   true,
	"throw":    true,
	"throws":   true,
	"break":    true,
	"continue": true,
	"goto":     true,
	"yield":    true,
	"defer":    true,
	"go":       true,
	"await":    true,
	"async":    true,
	"using":    true,
	"match":    true,
	"loop":     true,
	"when":     true,
}

// CFamilyBreak is the GroupBreakFunc for curly-brace languages:
// C, C++, C#, Java, Kotlin, Swift, JavaScript, TypeScript, Go, Rust,
// Dart, PHP, Scala, Groovy.
//
// It composes the universal brace-only rule with a control-flow rule:
// two adjacent same-indent lines should not share an alignment group if
// exactly one of them begins with a control-flow keyword. This covers
// failure cases 1 and 3 from todov2.md (a `try {` or `if (...)` line
// adjacent to a variable declaration) without touching field-declaration
// blocks or `String s = ""` / `this.s = ""` style assignments, neither
// of which starts with a control-flow keyword.
func CFamilyBreak(prev, curr []string) bool {
	if UniversalBreak(prev, curr) {
		return true
	}
	return startsWithControlFlow(prev) != startsWithControlFlow(curr)
}

func startsWithControlFlow(cells []string) bool {
	if len(cells) == 0 {
		return false
	}
	return cFamilyControlFlow[firstWord(cells[0])]
}
