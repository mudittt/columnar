package formatter

// shellControlFlow lists shell-script statement-leading keywords.
var shellControlFlow = map[string]bool{
	"if":       true,
	"then":     true,
	"elif":     true,
	"else":     true,
	"fi":       true,
	"for":      true,
	"while":    true,
	"until":    true,
	"do":       true,
	"done":     true,
	"case":     true,
	"esac":     true,
	"function": true,
	"return":   true,
	"break":    true,
	"continue": true,
	"exit":     true,
}

// ShellBreak is the GroupBreakFunc for bash/zsh/sh. Config-assignment
// blocks (FOO=bar) should stay grouped; control-flow lines should break.
func ShellBreak(prev, curr []string) bool {
	if UniversalBreak(prev, curr) {
		return true
	}
	return startsWithShellControlFlow(prev) != startsWithShellControlFlow(curr)
}

func startsWithShellControlFlow(cells []string) bool {
	if len(cells) == 0 {
		return false
	}
	return shellControlFlow[firstWord(cells[0])]
}
