package formatter

// pythonControlFlow lists Python statement-leading keywords that mark a
// line as structural rather than a plain assignment/expression. Python has
// no `{`/`}`, so all grouping falls on this rule alone.
var pythonControlFlow = map[string]bool{
	"if":       true,
	"elif":     true,
	"else":     true,
	"for":      true,
	"while":    true,
	"try":      true,
	"except":   true,
	"finally":  true,
	"with":     true,
	"return":   true,
	"yield":    true,
	"raise":    true,
	"pass":     true,
	"break":    true,
	"continue": true,
	"def":      true,
	"class":    true,
	"async":    true,
	"await":    true,
	"match":    true,
	"case":     true,
	"assert":   true,
	"import":   true,
	"from":     true,
	"global":   true,
	"nonlocal": true,
	"lambda":   true,
	"del":      true,
}

// PythonBreak is the GroupBreakFunc for Python. Same shape as
// CFamilyBreak: break when exactly one of the two adjacent lines starts
// with a control-flow keyword.
func PythonBreak(prev, curr []string) bool {
	if UniversalBreak(prev, curr) {
		return true
	}
	return startsWithPythonControlFlow(prev) != startsWithPythonControlFlow(curr)
}

func startsWithPythonControlFlow(cells []string) bool {
	if len(cells) == 0 {
		return false
	}
	return pythonControlFlow[firstWord(cells[0])]
}
