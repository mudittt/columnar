package formatter

// rubyControlFlow lists Ruby's statement-leading keywords. Ruby uses
// `do`/`end` blocks and inline `if`/`unless` modifiers; only *leading*
// control-flow keywords trigger a break here.
var rubyControlFlow = map[string]bool{
	"if":     true,
	"elsif":  true,
	"else":   true,
	"unless": true,
	"case":   true,
	"when":   true,
	"while":  true,
	"until":  true,
	"for":    true,
	"begin":  true,
	"rescue": true,
	"ensure": true,
	"end":    true,
	"do":     true,
	"def":    true,
	"class":  true,
	"module": true,
	"return": true,
	"yield":  true,
	"raise":  true,
	"next":   true,
	"break":  true,
	"redo":   true,
	"retry":  true,
	"require": true,
	"require_relative": true,
}

// RubyBreak is the GroupBreakFunc for Ruby.
func RubyBreak(prev, curr []string) bool {
	if UniversalBreak(prev, curr) {
		return true
	}
	return startsWithRubyControlFlow(prev) != startsWithRubyControlFlow(curr)
}

func startsWithRubyControlFlow(cells []string) bool {
	if len(cells) == 0 {
		return false
	}
	return rubyControlFlow[firstWord(cells[0])]
}
