// Package highlight turns one source line into styled spans, threading
// block-comment state across consecutive lines. It is backed by a chroma
// regex lexer matched by file extension and maps chroma token kinds onto
// cell styles from a built-in dark palette (indexed colours only). Nothing
// here knows about the viewport; the view caches chains of (line hash, state
// in) itself.
package highlight

import (
	"path/filepath"
	"strings"
	"sync"

	"github.com/alecthomas/chroma"
	"github.com/alecthomas/chroma/lexers"

	"github.com/Aswanidev-vs/cherry/cell"
)

// State threads block-context between consecutive Style calls.
type State uint8

const (
	StateNormal  State = iota // default
	StateComment              // a block comment is still open at end of line
)

// Span is one styled run whose Text is a substring of the original line
// (tabs stay untouched; the view decides how wide they are).
type Span struct {
	Text  string
	Style cell.Style
}

// Palette: one cell.Style per chroma token family, drawn from the indexed
// 256-colour palette only (no RGB) so it degrades gracefully on 16-colour
// terminals.
var (
	styleKeyword = cell.Plain.Foreground(cell.Indexed(75)).Bold(true)    // Keywords
	styleString  = cell.Plain.Foreground(cell.Indexed(114))              // Strings, Chars
	styleComment = cell.Plain.Foreground(cell.Indexed(245)).Italic(true) // Comments, docstrings
	styleNumber  = cell.Plain.Foreground(cell.Indexed(173))              // Numbers
	stylePunct   = cell.Plain.Foreground(cell.Indexed(251))              // Operators, Punctuation
	styleName    = cell.Plain.Foreground(cell.Indexed(252))              // Name tokens
	stylePreproc = cell.Plain.Foreground(cell.Indexed(180))              // Preprocessor fields
	styleOther   = cell.Plain.Foreground(cell.Indexed(252)).Italic(true) // Everything else
)

// extensions maps supported file extensions (lower case) to chroma lexer
// aliases; Go, Python, JavaScript/TypeScript, JSON, YAML/Markdown, C/C++,
// Rust, Shell, SQL, HTML and CSS are covered. Anything else falls back to
// plain text.
var extensions = map[string]string{
	".go":   "go",
	".py":   "python",
	".pyi":  "python",
	".pyw":  "python",
	".js":   "js",
	".mjs":  "js",
	".jsx":  "jsx",
	".ts":   "ts",
	".tsx":  "tsx",
	".json": "json",
	".yaml": "yaml",
	".yml":  "yaml",
	".md":   "md",
	".c":    "c",
	".h":    "c",
	".cpp":  "cpp",
	".cc":   "cpp",
	".cxx":  "cpp",
	".hpp":  "cpp",
	".hh":   "cpp",
	".hxx":  "cpp",
	".rs":   "rust",
	".sh":   "bash",
	".bash": "bash",
	".zsh":  "bash",
	".sql":  "sql",
	".html": "html",
	".htm":  "html",
	".css":  "css",
}

// chromaCache interns chroma lexers so every NewLexer call is cheap: chroma
// compiles a lexer's rules lazily on first use, then the engine is shared.
var chromaCache struct {
	mu    sync.Mutex
	langs map[string]chroma.Lexer
}

func cachedLexer(alias string) chroma.Lexer {
	chromaCache.mu.Lock()
	defer chromaCache.mu.Unlock()
	if chromaCache.langs == nil {
		chromaCache.langs = make(map[string]chroma.Lexer, len(extensions))
	}
	if l, ok := chromaCache.langs[alias]; ok {
		return l
	}
	l := lexers.Get(alias)
	chromaCache.langs[alias] = l // nil results are cached too
	return l
}

// Lexer is a per-file syntax engine selected at construction time.
type Lexer struct {
	name       string // language id, e.g. "go" or "plaintext"
	clang      chroma.Lexer
	terminator string // "*/" for /*-style languages, "" otherwise
}

// NewLexer picks the language from path's extension. Unknown extensions
// yield a plain lexer whose Name is "plaintext" and whose Style returns the
// whole line as one default-styled span.
func NewLexer(path string) *Lexer {
	alias := extensions[strings.ToLower(filepath.Ext(path))]
	if alias == "" {
		return &Lexer{name: "plaintext"}
	}
	clang := cachedLexer(alias)
	if clang == nil {
		return &Lexer{name: "plaintext"}
	}
	return &Lexer{
		name:       strings.ToLower(clang.Config().Name),
		clang:      clang,
		terminator: terminatorFor(alias),
	}
}

// terminatorFor returns the block-comment terminator of /*-style languages;
// only those can stay open across lines, because chroma re-tokenises every
// line from its root state.
func terminatorFor(alias string) string {
	switch alias {
	case "c", "cpp", "rust", "js", "ts", "jsx", "css", "sql", "go":
		return "*/"
	}
	return ""
}

// Name reports the language id, e.g. "go", "python", "plaintext".
func (l *Lexer) Name() string { return l.name }

// tokeniseOptions tokenises each LINE independently in the lexer's root
// state; EnsureLF is off so \r characters in line content are never
// rewritten.
var tokeniseOptions = &chroma.TokeniseOptions{State: "root", Nested: false, EnsureLF: false}

// appendSpan appends one styled run, merging it with the previous span when
// the styles are identical so the view receives contiguous runs. The
// concatenation of all span texts always equals the original line.
func appendSpan(spans []Span, text string, style cell.Style) []Span {
	if text == "" {
		return spans
	}
	if n := len(spans); n > 0 && spans[n-1].Style == style {
		spans[n-1].Text += text
		return spans
	}
	return append(spans, Span{Text: text, Style: style})
}

// Style renders one line. in must carry the State of the previous line
// (StateNormal for the first). The returned spans concatenate to exactly the
// input text; an empty line yields a nil slice and out == in.
func (l *Lexer) Style(line string, in State) (spans []Span, out State) {
	if line == "" {
		return nil, in // an empty line renders nothing; state carries over
	}
	if l.clang == nil {
		// Plaintext: the whole line as one default-styled span.
		return []Span{{Text: line, Style: cell.Plain}}, in
	}
	out = in

	// chroma can panic on degenerate input (delegate lexers, rule bugs); an
	// editor must survive any file content, so degrade to one plain span.
	defer func() {
		if recover() != nil {
			spans = []Span{{Text: line, Style: cell.Plain}}
			out = in
		}
	}()

	if in == StateComment && l.terminator != "" {
		// Continuation of a still-open block comment: a pseudo open marker
		// stands at the head of the line. Everything up to and including the
		// first terminator is comment styled; the remainder (if any)
		// re-enters the normal chroma pass. Without a terminator the whole
		// line remains a comment and the state carries over.
		if i := strings.Index(line, l.terminator); i >= 0 {
			end := i + len(l.terminator)
			spans = appendSpan(nil, line[:end], styleComment)
			line = line[end:]
			if line == "" {
				return spans, StateNormal
			}
			out = StateNormal
		} else {
			return []Span{{Text: line, Style: styleComment}}, StateComment
		}
	}

	// Normal pass: chroma tokenises the line in the root state.
	toks, err := chroma.Tokenise(l.clang, tokeniseOptions, line)
	if err != nil {
		return spans, in
	}

	for _, tok := range toks {
		v := tok.Value
		comment := isCommentType(tok.Type)
		if strings.HasSuffix(v, "\n") {
			// An unterminated block comment comes back as a comment token
			// whose value spans the whole open comment INCLUDING the newline
			// terminator chroma appends (EnsureNL). The file's data layer
			// never keeps the newline in line content, so trim it off the
			// span text and open the comment state. /*-style only: // and #
			// line comments carry the newline too but never continue.
			v = strings.TrimSuffix(v, "\n")
			if comment && l.terminator != "" &&
				strings.Contains(v, "/*") && !strings.Contains(v, l.terminator) {
				out = StateComment
			}
		}
		spans = appendSpan(spans, v, styleFor(tok.Type, comment))
	}
	return spans, out
}

// styleFor maps a chroma token type onto the palette by token family band
// (chroma lays token types out in 100-wide bands per family).
func styleFor(t chroma.TokenType, comment bool) cell.Style {
	switch {
	case t == chroma.LiteralStringDoc: // docstrings read as comments
		return styleComment
	case t >= chroma.Keyword && t <= chroma.KeywordType:
		return styleKeyword
	case t >= chroma.LiteralString && t <= chroma.LiteralStringSymbol:
		return styleString
	case t >= chroma.LiteralNumber && t <= chroma.LiteralNumberOct:
		return styleNumber
	case t >= chroma.Comment && t <= chroma.CommentSpecial:
		return styleComment
	case t == chroma.CommentPreproc || t == chroma.CommentPreprocFile:
		return stylePreproc
	case t == chroma.Operator || t == chroma.OperatorWord || t == chroma.Punctuation:
		return stylePunct
	case t >= chroma.Name && t <= chroma.NameVariableMagic:
		return styleName
	default: // Text/Whitespace, Generics, plain Literals, Errors, ...
		return styleOther
	}
}

// isCommentType reports whether a chroma token type belongs to one of the
// comment families (comments proper plus preprocessor fields).
func isCommentType(t chroma.TokenType) bool {
	return (t >= chroma.Comment && t <= chroma.CommentSpecial) ||
		t == chroma.CommentPreproc || t == chroma.CommentPreprocFile
}
