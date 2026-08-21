// Tests for the highlight package: written against the public API only (a
// file/line view of the outline document).
package highlight

import (
	"strings"
	"testing"

	chromacell "github.com/Aswanidev-vs/cherry/cell"
)

// join concatenates every span's text for an exact-equality check.
func join(spans []Span) string {
	var b strings.Builder
	for _, s := range spans {
		b.WriteString(s.Text)
	}
	return b.String()
}

// assertConcat is the document promise: spans in order, concatenation
// equals the input line.
func assertConcat(t *testing.T, spans []Span, line string) {
	t.Helper()
	if got := join(spans); got != line {
		t.Fatalf("span concatenation = %q, want %q", got, line)
	}
}

// fgIsIndexed checks a style carries an indexed (256-colour) foreground n.
func fgIsIndexed(t *testing.T, style chromacell.Style, n uint8) {
	t.Helper()
	if !style.Fg.IsIndexed() || style.Fg.Index() != n {
		t.Fatalf("style.Foreground = %#v, want Indexed(%d)", style.Fg, n)
	}
}

// TestPlaintextPassthrough: unknownextension picks the plaintext lexer, whose
// Style returns one default span over the whole line.
func TestPlaintextPassthrough(t *testing.T) {
	l := NewLexer("foo/readme.txt")
	if l.Name() != "plaintext" {
		t.Fatalf("Name = %q, want \"plaintext\"", l.Name())
	}

	spans, out := l.Style("hello world", StateNormal)
	if len(spans) != 1 {
		t.Fatalf("len(spans) = %d, want 1", len(spans))
	}
	if spans[0].Text != "hello world" {
		t.Fatalf("span text = %q, want %q", spans[0].Text, "hello world")
	}
	if !spans[0].Style.Fg.IsDefault() {
		t.Fatalf("span style fg = %#v, want default colour", spans[0].Style.Fg)
	}
	if out != StateNormal {
		t.Fatalf("out = %v, want StateNormal", out)
	}

	// empty line still renders nothing and keeps state.
	spans, out = l.Style("", StateNormal)
	if spans != nil || out != StateNormal {
		t.Fatalf("empty plaintext line: spans = %#v, out = %v, want nil / StateNormal", spans, out)
	}
}

// TestGoKeywordNumberCommentString: the four token families of the outline get
// their four palette colours.
func TestGoKeywordNumberCommentString(t *testing.T) {
	l := NewLexer("main.go")
	if l.Name() != "go" {
		t.Fatalf("Name = %q, want \"go\"", l.Name())
	}

	// keyword + braces
	spans, out := l.Style("if true {", StateNormal)
	assertConcat(t, spans, "if true {")
	if out != StateNormal {
		t.Fatalf("out = %v, want StateNormal", out)
	}
	var kwCount int
	for _, s := range spans {
		if s.Style.Attrs&chromacell.AttrBold != 0 && s.Style.Fg.IsIndexed() && s.Style.Fg.Index() == 75 {
			if s.Text != "if" && s.Text != "true" {
				t.Fatalf("keyword span = %q, want \"if\" or \"true\"", s.Text)
			}
			kwCount++
		}
	}
	if kwCount != 2 {
		t.Fatalf("keyword spans = %d, want 2", kwCount)
	}

	// number
	spans, _ = l.Style("n := 42", StateNormal)
	assertConcat(t, spans, "n := 42")
	var numCount int
	for _, s := range spans {
		if s.Style.Fg.IsIndexed() && s.Style.Fg.Index() == 173 {
			if s.Text != "42" {
				t.Fatalf("number span = %q, want \"42\"", s.Text)
			}
			numCount++
		}
	}
	if numCount != 1 {
		t.Fatalf("number spans = %d, want 1", numCount)
	}

	// comment (single line; newline appended by chroma and trimmed back off)
	spans, _ = l.Style("// note here", StateNormal)
	assertConcat(t, spans, "// note here")
	var comCount int
	for _, s := range spans {
		if s.Style.Attrs&chromacell.AttrItalic != 0 && s.Style.Fg.IsIndexed() && s.Style.Fg.Index() == 245 {
			comCount++
		}
	}
	if comCount == 0 {
		t.Fatalf("no comment span found in %#v", spans)
	}
	if spans[0].Text != "// note here" {
		t.Fatalf("comment span = %q, want \"// note here\"", spans[0].Text)
	}

	// string
	spans, _ = l.Style(`s := "hello"`, StateNormal)
	assertConcat(t, spans, `s := "hello"`)
	var strCount int
	for _, s := range spans {
		if s.Style.Fg.IsIndexed() && s.Style.Fg.Index() == 114 {
			if s.Text != `"hello"` {
				t.Fatalf("string span = %q, want %q", s.Text, `"hello"`)
			}
			strCount++
		}
	}
	if strCount != 1 {
		t.Fatalf("string spans = %d, want 1", strCount)
	}
}

// TestBlockCommentContinuation: a /* comment that opens on one line, carries
// through an empty middle line, and closes on a third.
func TestBlockCommentContinuation(t *testing.T) {
	l := NewLexer("comment.c")

	open, out1 := l.Style("/* open comment", StateNormal)
	assertConcat(t, open, "/* open comment")
	if out1 != StateComment {
		t.Fatalf("after line 1: out = %v, want StateComment", out1)
	}
	for _, s := range open {
		if !(s.Style.Fg.IsIndexed() && s.Style.Fg.Index() == 245) {
			t.Fatalf("span %q inside unterminated /* comment should be comment styled", s.Text)
		}
	}

	// line 2 stays inside the comment; the state must carry across.
	mid, out2 := l.Style("still inside", out1)
	assertConcat(t, mid, "still inside")
	if out2 != StateComment {
		t.Fatalf("after line 2: out = %v, want StateComment", out2)
	}
	if len(mid) != 1 || !(mid[0].Style.Fg.IsIndexed() && mid[0].Style.Fg.Index() == 245) {
		t.Fatalf("comment continuation span = %#v, want one 245-coloured span", mid)
	}

	// line 3 closes the comment and carries more code.
	close, out3 := l.Style("end */ int x", out2)
	assertConcat(t, close, "end */ int x")
	if out3 != StateNormal {
		t.Fatalf("after line 3: out = %v, want StateNormal", out3)
	}
	if close[0].Text != "end */" {
		t.Fatalf("first span = %q, want \"end */\"", close[0].Text)
	}
	if !(close[0].Style.Fg.IsIndexed() && close[0].Style.Fg.Index() == 245) {
		t.Fatalf("closing span should carry the comment colour, got %#v", close[0].Style.Fg)
	}
}

// TestEmptyLineReturnsNil: an empty line produces a nil slice and out == in.
func TestEmptyLineReturnsNil(t *testing.T) {
	l := NewLexer("main.go")

	spans, out := l.Style("", StateNormal)
	if spans != nil {
		t.Fatalf("Spans = %#v, want nil", spans)
	}
	if out != StateNormal {
		t.Fatalf("out = %v, want StateNormal", out)
	}

	spans, out = l.Style("", StateComment)
	if spans != nil {
		t.Fatalf("Spans = %#v, want nil", spans)
	}
	if out != StateComment {
		t.Fatalf("out = %v, want StateComment", out)
	}
}

// TestTabsPassThroughUntouched: indentation is kept verbatim in the spans.
func TestTabsPassThroughUntouched(t *testing.T) {
	l := NewLexer("indented.py")
	line := "\tfoo()\t"
	spans, _ := l.Style(line, StateNormal)
	assertConcat(t, spans, line)

	whole := join(spans)
	if !strings.HasPrefix(whole, "\t") || !strings.HasSuffix(whole, "\t") {
		t.Fatalf("joined spans = %q, want leading and trailing tab preserved", whole)
	}
}

// TestUnknownExtensionYieldsPlaintextDefaultSpan: unknown extension picks
// the plaintext lexer whose Name is "plaintext".
func TestUnknownExtensionYieldsPlaintextDefaultSpan(t *testing.T) {
	l := NewLexer("mystery.zzz")
	if l.Name() != "plaintext" {
		t.Fatalf("Name = %q, want \"plaintext\"", l.Name())
	}
	spans, out := l.Style("some content", StateNormal)
	if len(spans) != 1 {
		t.Fatalf("len(spans) = %d, want 1", len(spans))
	}
	if spans[0].Text != "some content" {
		t.Fatalf("span text = %q, want %q", spans[0].Text, "some content")
	}
	if !spans[0].Style.Fg.IsDefault() {
		t.Fatalf("plaintext span should use the default colour, got %#v", spans[0].Style.Fg)
	}
	if out != StateNormal {
		t.Fatalf("out = %v, want StateNormal", out)
	}
}

// TestSupportedLanguagesSmoke: every supported family picks its lexer and
// keeps the span-concatenation promise on a representative first line.
func TestSupportedLanguagesSmoke(t *testing.T) {
	cases := []struct {
		path string
		name string
		line string
	}{
		{"a/main.go", "go", `fmt.Println("hi")`},
		{"b.py", "python", "def foo(): pass"},
		{"c.js", "javascript", "const x = 1;"},
		{"d.ts", "typescript", "let n: number = 2;"},
		{"e.json", "json", `{"k": 5}`},
		{"f.yaml", "yaml", "key: value"},
		{"g.md", "markdown", "# Heading"},
		{"h.c", "c", "int f(int a) { return 1; }"},
		{"i.rs", "rust", "fn main() {}"},
		{"j.sh", "bash", "echo hi"},
		{"k.sql", "sql", "SELECT 1;"},
		{"l.html", "html", `<div class="a">hi</div>`},
		{"m.css", "css", `.a { color: #fff; }`},
		{"n.pyi", "python", "x: int"},
		{"o.mjs", "javascript", "export {}"},
	}
	for _, c := range cases {
		l := NewLexer(c.path)
		if l.Name() != c.name {
			t.Errorf("%s: Name = %q, want %q", c.path, l.Name(), c.name)
		}
		spans, _ := l.Style(c.line, StateNormal)
		assertConcat(t, spans, c.line)
	}
}

// TestCommentContinuationCloseOnly: comment line closed by its terminator, the
// pseudo open marker stays attached so the span goes right up to the terminator.
func TestCommentContinuationCloseOnly(t *testing.T) {
	l := NewLexer("main.cpp")
	spans, out := l.Style("*/", StateComment)
	assertConcat(t, spans, "*/")
	if out != StateNormal {
		t.Fatalf("out = %v, want StateNormal", out)
	}
	if len(spans) != 1 {
		t.Fatalf("len(spans) = %d, want 1 span covering the full close marker", len(spans))
	}
	if !(spans[0].Style.Fg.IsIndexed() && spans[0].Style.Fg.Index() == 245) {
		t.Fatalf("close-only span should stay comment styled, got %#v", spans[0].Style.Fg)
	}
}

// TestNestedCommentOpenWithTerminatorOnLine: a /* that closes itself on the
// same line stays inside the normal pass.
func TestClosedCommentSameLine(t *testing.T) {
	l := NewLexer("main.c")
	line := "int x; /* quick */ int y;"
	spans, out := l.Style(line, StateNormal)
	assertConcat(t, spans, line)
	if out != StateNormal {
		t.Fatalf("out = %v, want StateNormal for a same-line closed comment", out)
	}
}

// TestNoFileReadsOrPanic: construction itself must never panic or fail for a
// wide set of paths.
func TestNoFileReadsOrPanic(t *testing.T) {
	paths := []string{
		"", ".", "..", "noext", "/foo/bar/baz.zzz", "x.go", "dir/x.h",
		"X.JSON", // case insensitivity
	}
	for _, p := range paths {
		l := NewLexer(p)
		_ = l.Name()
		spans, out := l.Style("x", StateNormal)
		assertConcat(t, spans, "x")
		if out != StateNormal {
			t.Errorf("%q: out = %v, want StateNormal", p, out)
		}
	}
}

// TestLeadingWhitespaceGo: tab-indented code inside a block comment scenario
// reuses indentation without splitting spans.
func TestLeadingWhitespaceGo(t *testing.T) {
	l := NewLexer("ws.go")
	spans, out := l.Style("\tx := y", StateNormal)
	assertConcat(t, spans, "\tx := y")
	if out != StateNormal {
		t.Fatalf("out = %v, want StateNormal", out)
	}
}

// TestSearchQueryHashtags: Python '#' comments leave the state normal, since
// they cannot continue across lines.
func TestPythonHashCommentStateNormal(t *testing.T) {
	l := NewLexer("script.py")
	spans, out := l.Style("# note", StateNormal)
	assertConcat(t, spans, "# note")
	if out != StateNormal {
		t.Fatalf("out = %v, want StateNormal for a Python line comment", out)
	}
}
