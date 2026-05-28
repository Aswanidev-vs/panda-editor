package highlight

import (
	"regexp"
	"strings"

	"github.com/Aswanidev-vs/panda-editor/internal/theme"
	"github.com/alecthomas/chroma"
	"github.com/alecthomas/chroma/lexers"
	"github.com/charmbracelet/lipgloss"
)

type TokenType int

const (
	TokenNormal TokenType = iota
	TokenComment
	TokenKeyword
	TokenString
	TokenNumber
	TokenFunction
	TokenType_
	TokenOperator
	TokenBuiltin
	TokenPreprocessor
)

type Span struct {
	Text  string
	Token TokenType
}

var (
	goKeywords = []string{
		"break", "case", "chan", "const", "continue", "default", "defer",
		"else", "fallthrough", "for", "func", "go", "goto", "if",
		"import", "interface", "map", "package", "range", "return",
		"select", "struct", "switch", "type", "var",
	}
	goBuiltins = []string{
		"append", "cap", "close", "complex", "copy", "delete", "imag",
		"len", "make", "new", "panic", "print", "println", "real",
		"recover", "true", "false", "nil", "iota",
	}
	goTypes = []string{
		"bool", "byte", "complex64", "complex128", "error", "float32",
		"float64", "int", "int8", "int16", "int32", "int64", "rune",
		"string", "uint", "uint8", "uint16", "uint32", "uint64",
		"uintptr", "any", "comparable",
	}

	pyKeywords = []string{
		"and", "as", "assert", "async", "await", "break", "class",
		"continue", "def", "del", "elif", "else", "except", "finally",
		"for", "from", "global", "if", "import", "in", "is", "lambda",
		"nonlocal", "not", "or", "pass", "raise", "return", "try",
		"while", "with", "yield",
	}
	pyBuiltins = []string{
		"True", "False", "None", "self", "cls", "print", "len", "range",
		"int", "str", "float", "list", "dict", "set", "tuple", "bool",
		"type", "object", "super", "isinstance", "issubclass", "hasattr",
		"getattr", "setattr", "delattr", "property", "staticmethod",
		"classmethod", "enumerate", "zip", "map", "filter", "sorted",
		"reversed", "min", "max", "sum", "abs", "any", "all", "open",
		"input", "format", "repr", "id", "hash", "callable", "iter",
		"next", "slice", "vars", "dir", "help", "exec", "eval",
	}

	jsKeywords = []string{
		"abstract", "arguments", "async", "await", "boolean", "break",
		"byte", "case", "catch", "char", "class", "const", "continue",
		"debugger", "default", "delete", "do", "double", "else", "enum",
		"export", "extends", "false", "final", "finally", "float", "for",
		"function", "goto", "if", "implements", "import", "in",
		"instanceof", "int", "interface", "let", "long", "native", "new",
		"null", "of", "package", "private", "protected", "public",
		"return", "short", "static", "super", "switch", "synchronized",
		"this", "throw", "throws", "transient", "true", "try", "typeof",
		"undefined", "var", "void", "volatile", "while", "with", "yield",
	}
	jsBuiltins = []string{
		"console", "window", "document", "Array", "Object", "String",
		"Number", "Boolean", "Math", "JSON", "Promise", "Map", "Set",
		"WeakMap", "WeakSet", "Symbol", "Proxy", "Reflect", "Error",
		"TypeError", "RangeError", "SyntaxError", "parseInt", "parseFloat",
		"isNaN", "isFinite", "setTimeout", "setInterval", "clearTimeout",
		"clearInterval", "fetch", "require", "module", "exports", "process",
		"Buffer",
	}

	rsKeywords = []string{
		"as", "async", "await", "break", "const", "continue", "crate",
		"dyn", "else", "enum", "extern", "fn", "for", "if", "impl", "in",
		"let", "loop", "match", "mod", "move", "mut", "pub", "ref",
		"return", "self", "Self", "static", "struct", "super", "trait",
		"type", "unsafe", "use", "where", "while", "yield",
	}
	rsBuiltins = []string{
		"bool", "char", "f32", "f64", "i8", "i16", "i32", "i64", "i128",
		"isize", "str", "u8", "u16", "u32", "u64", "u128", "usize",
		"String", "Vec", "Box", "Option", "Result", "Some", "None", "Ok",
		"Err", "true", "false", "println", "print", "format", "panic",
		"assert", "assert_eq", "assert_ne", "todo", "unimplemented",
		"unreachable", "dbg", "vec", "include", "include_str",
		"include_bytes", "env", "cfg",
	}

	cKeywords = []string{
		"auto", "break", "case", "char", "const", "continue", "default",
		"do", "double", "else", "enum", "extern", "float", "for", "goto",
		"if", "int", "long", "register", "return", "short", "signedfa",
		"sizeof", "static", "struct", "switch", "typedef", "union",
		"unsigned", "void", "volatile", "while",
	}

	javaKeywords = []string{
		"abstract", "assert", "boolean", "break", "byte", "case", "catch",
		"char", "class", "const", "continue", "default", "do", "double",
		"else", "enum", "extends", "final", "finally", "float", "for",
		"goto", "if", "implements", "import", "instanceof", "int",
		"interface", "long", "native", "new", "package", "private",
		"protected", "public", "return", "short", "static", "strictfp",
		"super", "switch", "synchronized", "this", "throw", "throws",
		"transient", "try", "void", "volatile", "while", "var", "yield",
		"record", "sealed", "permits", "with",
	}

	rubyKeywords = []string{
		"BEGIN", "END", "alias", "and", "begin", "break", "case", "class",
		"def", "defined?", "do", "else", "elsif", "end", "ensure", "false",
		"for", "if", "in", "module", "next", "nil", "not", "or", "redo",
		"rescue", "retry", "return", "self", "super", "then", "true",
		"undef", "unless", "until", "when", "while", "yield", "__FILE__",
		"__LINE__", "__dir__",
	}

	shellKeywords = []string{
		"if", "then", "else", "elif", "fi", "case", "esac", "for",
		"select", "while", "until", "do", "done", "in", "function",
		"time", "coproc", "break", "continue", "return", "exit",
		"export", "readonly", "declare", "local", "typeset", "unset",
		"shift", "source", "trap", "wait", "exec", "eval",
	}
)

var (
	lineCommentPatterns = map[string]string{
		"go":         "//",
		"python":     "#",
		"javascript": "//",
		"typescript": "//",
		"jsx":        "//",
		"rust":       "//",
		"c":          "//",
		"cpp":        "//",
		"java":       "//",
		"ruby":       "#",
		"php":        "//",
		"shell":      "#",
		"lua":        "--",
		"sql":        "--",
		"yaml":       "#",
		"toml":       "#",
		"ini":        ";",
	}

	blockCommentStart = map[string]string{
		"go":         "/*",
		"javascript": "/*",
		"typescript": "/*",
		"jsx":        "/*",
		"rust":       "/*",
		"c":          "/*",
		"cpp":        "/*",
		"java":       "/*",
		"php":        "/*",
		"css":        "/*",
		"scss":       "/*",
		"lua":        "--[[",
	}

	blockCommentEnd = map[string]string{
		"go":         "*/",
		"javascript": "*/",
		"typescript": "*/",
		"jsx":        "*/",
		"rust":       "*/",
		"c":          "*/",
		"cpp":        "*/",
		"java":       "*/",
		"php":        "*/",
		"css":        "*/",
		"scss":       "*/",
		"lua":        "]]",
	}

	numberRe = regexp.MustCompile(`\b(0x[0-9a-fA-F]+|0b[01]+|0o[0-7]+|\d+\.?\d*([eE][+-]?\d+)?i?|0[Xx][0-9a-fA-F]+)\b`)
)

func tokeniseWithChroma(lexer chroma.Lexer, line string) []Span {
	iter, err := lexer.Tokenise(nil, line)
	if err != nil {
		return []Span{{Text: line, Token: TokenNormal}}
	}
	var spans []Span
	for token := iter(); token != chroma.EOF; token = iter() {
		text := token.String()
		if text == "" {
			continue
		}
		spans = append(spans, Span{Text: text, Token: tokenTypeToToken(token.Type)})
	}
	if len(spans) == 0 {
		return []Span{{Text: line, Token: TokenNormal}}
	}
	return spans
}

func HighlightLine(line, language string, inBlockComment bool) ([]Span, bool) {
	if language == "text" || language == "markdown" || language == "" {
		return []Span{{Text: line, Token: TokenNormal}}, false
	}

	lexer := lexerForLanguage(language)
	if lexer == nil {
		return []Span{{Text: line, Token: TokenNormal}}, false
	}

	startDelim := blockCommentStart[language]
	endDelim := blockCommentEnd[language]

	// If continuing a block comment from previous line
	if inBlockComment && endDelim != "" {
		idx := strings.Index(line, endDelim)
		if idx >= 0 {
			// Closing delimiter found — split into comment + code
			commentPart := line[:idx+len(endDelim)]
			codePart := line[idx+len(endDelim):]
			spans := []Span{{Text: commentPart, Token: TokenComment}}
			spans = append(spans, tokeniseWithChroma(lexer, codePart)...)
			return spans, false
		}
		// Entire line is still inside block comment
		return []Span{{Text: line, Token: TokenComment}}, true
	}

	spans := tokeniseWithChroma(lexer, line)

	// Detect if line ends with an unclosed block comment
	stillInComment := false
	if startDelim != "" && endDelim != "" && len(spans) > 0 {
		last := spans[len(spans)-1]
		if last.Token == TokenComment && strings.Contains(last.Text, startDelim) && !strings.Contains(last.Text, endDelim) {
			stillInComment = true
		}
	}

	return spans, stillInComment
}

func lexerForLanguage(language string) chroma.Lexer {
	// Chroma expects lexer names like "go", "python", "javascript", etc.
	// Most of our buffer language values already match.
	return lexers.Get(language)
}

func tokenTypeToToken(tt chroma.TokenType) TokenType {
	switch tt.Category() {
	case chroma.Comment:
		return TokenComment
	case chroma.Keyword:
		return TokenKeyword
	case chroma.Literal:
		// further refine:
		if tt == chroma.String {
			return TokenString
		}
		if tt == chroma.Number || tt == chroma.NumberInteger || tt == chroma.NumberFloat {
			return TokenNumber
		}
		// fallback:
		return TokenString
	case chroma.Name:
		// Many languages tag builtins/types as Name; heuristics:
		if tt == chroma.NameBuiltin {
			return TokenBuiltin
		}
		if tt == chroma.NameFunction {
			return TokenFunction
		}
		if tt == chroma.NameClass {
			return TokenType_
		}
		return TokenNormal
	case chroma.Operator:
		return TokenOperator
	case chroma.Punctuation:
		return TokenNormal
	default:
		return TokenNormal
	}
}

func TokenToStyle(token TokenType) lipgloss.Style {
	t := theme.CurrentTheme
	switch token {
	case TokenComment:
		return lipgloss.NewStyle().Foreground(t.Comment).Italic(true)
	case TokenKeyword:
		return lipgloss.NewStyle().Foreground(t.Keyword).Bold(true)
	case TokenString:
		return lipgloss.NewStyle().Foreground(t.String)
	case TokenNumber:
		return lipgloss.NewStyle().Foreground(t.Number)
	case TokenFunction:
		return lipgloss.NewStyle().Foreground(t.Function)
	case TokenType_:
		return lipgloss.NewStyle().Foreground(t.Type)
	case TokenOperator:
		return lipgloss.NewStyle().Foreground(t.Operator)
	case TokenBuiltin:
		return lipgloss.NewStyle().Foreground(t.Builtin)
	case TokenPreprocessor:
		return lipgloss.NewStyle().Foreground(t.AccentAlt)
	default:
		return lipgloss.NewStyle().Foreground(t.Fg)
	}
}

func RenderLine(line, language string, inBlockComment bool) (string, bool) {
	spans, stillInComment := HighlightLine(line, language, inBlockComment)
	var sb strings.Builder
	for _, span := range spans {
		style := TokenToStyle(span.Token)
		sb.WriteString(style.Render(span.Text))
	}
	return sb.String(), stillInComment
}

func getKeywords(lang string) []string {
	switch lang {
	case "go":
		return goKeywords
	case "python":
		return pyKeywords
	case "javascript", "typescript", "jsx":
		return jsKeywords
	case "rust":
		return rsKeywords
	case "c", "cpp":
		return cKeywords
	case "java":
		return javaKeywords
	case "ruby":
		return rubyKeywords
	case "shell":
		return shellKeywords
	default:
		return nil
	}
}

func getBuiltins(lang string) []string {
	switch lang {
	case "go":
		return goBuiltins
	case "python":
		return pyBuiltins
	case "javascript", "typescript", "jsx":
		return jsBuiltins
	case "rust":
		return rsBuiltins
	default:
		return nil
	}
}

func getTypes(lang string) []string {
	switch lang {
	case "go":
		return goTypes
	default:
		return nil
	}
}

func contains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

func isAlpha(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

func isAlphaNum(r rune) bool {
	return isAlpha(r) || (r >= '0' && r <= '9') || r == '_'
}

func isOperator(r rune) bool {
	switch r {
	case '+', '-', '*', '/', '%', '=', '!', '<', '>', '&', '|', '^', '~', '?', ':', '.':
		return true
	}
	return false
}

func utf8RuneIndex(s string, byteIdx int) int {
	return len([]rune(s[:byteIdx]))
}
