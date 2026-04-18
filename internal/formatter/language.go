package formatter

import (
	"path/filepath"
	"strings"
)

// LangConfig captures the minimal syntax facts we need to tokenize a line.
type LangConfig struct {
	Name              string
	LineComment       string
	BlockCommentOpen  string
	BlockCommentClose string
	StringQuotes      []rune
	// MultilineStringDelims lists delimiters that open and close multi-line
	// string literals (e.g. `"""` for Python, "`" for Go/JS). Lines inside
	// an unclosed multi-line string are emitted verbatim — the formatter must
	// not touch string content.
	MultilineStringDelims []string
	// AssignRHSAtomic: when true, after a standalone `=` the rest of the
	// line (up to any trailing comment) is kept as a single cell rather
	// than being split on whitespace. Used for config-style files where
	// `KEY = one two three` has a single logical value.
	AssignRHSAtomic bool
	// GroupBreak decides whether to split the alignment group between two
	// adjacent same-indent lines. Defined per-language in a dedicated
	// group_<lang>.go file; nil means "never break on content" (only blank
	// lines and indent changes split groups).
	GroupBreak GroupBreakFunc
}

var languages = map[string]LangConfig{
	"java":        {Name: "java", LineComment: "//", BlockCommentOpen: "/*", BlockCommentClose: "*/", StringQuotes: []rune{'"', '\''}, GroupBreak: CFamilyBreak},
	"javascript":  {Name: "javascript", LineComment: "//", BlockCommentOpen: "/*", BlockCommentClose: "*/", StringQuotes: []rune{'"', '\'', '`'}, MultilineStringDelims: []string{"`"}, GroupBreak: CFamilyBreak},
	"typescript":  {Name: "typescript", LineComment: "//", BlockCommentOpen: "/*", BlockCommentClose: "*/", StringQuotes: []rune{'"', '\'', '`'}, MultilineStringDelims: []string{"`"}, GroupBreak: CFamilyBreak},
	"python":      {Name: "python", LineComment: "#", StringQuotes: []rune{'"', '\''}, MultilineStringDelims: []string{`"""`, `'''`}, GroupBreak: PythonBreak},
	"rust":        {Name: "rust", LineComment: "//", BlockCommentOpen: "/*", BlockCommentClose: "*/", StringQuotes: []rune{'"'}, GroupBreak: CFamilyBreak},
	"go":          {Name: "go", LineComment: "//", BlockCommentOpen: "/*", BlockCommentClose: "*/", StringQuotes: []rune{'"', '`'}, MultilineStringDelims: []string{"`"}, GroupBreak: CFamilyBreak},
	"c":           {Name: "c", LineComment: "//", BlockCommentOpen: "/*", BlockCommentClose: "*/", StringQuotes: []rune{'"', '\''}, GroupBreak: CFamilyBreak},
	"cpp":         {Name: "cpp", LineComment: "//", BlockCommentOpen: "/*", BlockCommentClose: "*/", StringQuotes: []rune{'"', '\''}, GroupBreak: CFamilyBreak},
	"csharp":      {Name: "csharp", LineComment: "//", BlockCommentOpen: "/*", BlockCommentClose: "*/", StringQuotes: []rune{'"', '\''}, MultilineStringDelims: []string{`"""`}, GroupBreak: CFamilyBreak},
	"kotlin":      {Name: "kotlin", LineComment: "//", BlockCommentOpen: "/*", BlockCommentClose: "*/", StringQuotes: []rune{'"', '\''}, MultilineStringDelims: []string{`"""`}, GroupBreak: CFamilyBreak},
	"swift":       {Name: "swift", LineComment: "//", BlockCommentOpen: "/*", BlockCommentClose: "*/", StringQuotes: []rune{'"'}, MultilineStringDelims: []string{`"""`}, GroupBreak: CFamilyBreak},
	"ruby":        {Name: "ruby", LineComment: "#", StringQuotes: []rune{'"', '\''}, GroupBreak: RubyBreak},
	"php":         {Name: "php", LineComment: "//", BlockCommentOpen: "/*", BlockCommentClose: "*/", StringQuotes: []rune{'"', '\''}, GroupBreak: CFamilyBreak},
	"dart":        {Name: "dart", LineComment: "//", BlockCommentOpen: "/*", BlockCommentClose: "*/", StringQuotes: []rune{'"', '\''}, MultilineStringDelims: []string{`"""`, `'''`}, GroupBreak: CFamilyBreak},
	"lua":         {Name: "lua", LineComment: "--", StringQuotes: []rune{'"', '\''}, GroupBreak: LuaBreak},
	"makefile":    {Name: "makefile", LineComment: "#", StringQuotes: []rune{'"', '\''}, AssignRHSAtomic: true},
	"shellscript": {Name: "shellscript", LineComment: "#", StringQuotes: []rune{'"', '\''}, AssignRHSAtomic: true, GroupBreak: ShellBreak},
	"properties":  {Name: "properties", LineComment: "#", AssignRHSAtomic: true},
	"ini":         {Name: "ini", LineComment: ";", AssignRHSAtomic: true},
	"toml":        {Name: "toml", LineComment: "#", StringQuotes: []rune{'"', '\''}, AssignRHSAtomic: true},
	"yaml":        {Name: "yaml", LineComment: "#", StringQuotes: []rune{'"', '\''}},
	"plaintext":   {Name: "plaintext"},
}

var extensionMap = map[string]string{
	".java":       "java",
	".py":         "python",
	".js":         "javascript",
	".mjs":        "javascript",
	".cjs":        "javascript",
	".jsx":        "javascript",
	".ts":         "typescript",
	".tsx":        "typescript",
	".rs":         "rust",
	".go":         "go",
	".c":          "c",
	".h":          "c",
	".cpp":        "cpp",
	".cc":         "cpp",
	".cxx":        "cpp",
	".hpp":        "cpp",
	".hh":         "cpp",
	".cs":         "csharp",
	".kt":         "kotlin",
	".kts":        "kotlin",
	".swift":      "swift",
	".rb":         "ruby",
	".php":        "php",
	".dart":       "dart",
	".lua":        "lua",
	".sh":         "shellscript",
	".bash":       "shellscript",
	".zsh":        "shellscript",
	".properties": "properties",
	".ini":        "ini",
	".env":        "properties",
	".toml":       "toml",
	".yaml":       "yaml",
	".yml":        "yaml",
	".txt":        "plaintext",
	".mk":         "makefile",
}

// DetectLanguage infers a language name from a file path.
func DetectLanguage(path string) string {
	base := filepath.Base(path)
	lower := strings.ToLower(base)
	if lower == "makefile" || lower == "gnumakefile" || strings.HasSuffix(lower, ".mk") {
		return "makefile"
	}
	if strings.HasPrefix(lower, ".env") {
		return "properties"
	}
	ext := strings.ToLower(filepath.Ext(base))
	if lang, ok := extensionMap[ext]; ok {
		return lang
	}
	return "plaintext"
}

// GetLangConfig returns the LangConfig for a given language name.
// Unknown names fall back to plaintext.
func GetLangConfig(name string) LangConfig {
	if cfg, ok := languages[strings.ToLower(name)]; ok {
		return cfg
	}
	return languages["plaintext"]
}
