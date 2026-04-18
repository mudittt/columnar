package formatter

// luaControlFlow lists Lua's statement-leading keywords.
var luaControlFlow = map[string]bool{
	"if":       true,
	"then":     true,
	"elseif":   true,
	"else":     true,
	"end":      true,
	"for":      true,
	"while":    true,
	"repeat":   true,
	"until":    true,
	"do":       true,
	"function": true,
	"local":    false, // `local` is a declaration prefix — do NOT break
	"return":   true,
	"break":    true,
	"goto":     true,
}

// LuaBreak is the GroupBreakFunc for Lua.
func LuaBreak(prev, curr []string) bool {
	if UniversalBreak(prev, curr) {
		return true
	}
	return startsWithLuaControlFlow(prev) != startsWithLuaControlFlow(curr)
}

func startsWithLuaControlFlow(cells []string) bool {
	if len(cells) == 0 {
		return false
	}
	return luaControlFlow[firstWord(cells[0])]
}
