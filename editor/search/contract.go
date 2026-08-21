// Package search implements find and replace over text buffers with
// case-sensitive, whole-word and regex options. Every position is a byte
// offset into the buffer; the view converts to lines via Buffer helpers.
// Matches wrap around the buffer edges so "no match" means truly none.
package search

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/Aswanidev-vs/panda-editor/editor/textbuf"
)

// matchCap bounds MatchCount and ReplaceAll so pathological input cannot run
// forever.
const matchCap = 100000

// Query is one find/replace request.
type Query struct {
	Text          string
	Regex         bool
	CaseSensitive bool
	Word          bool // match whole words only (ignored with Regex)
}

// Valid reports whether the query can match anything (non-empty text, and a
// regex that compiles).
func (q Query) Valid() bool {
	if q.Text == "" {
		return false
	}
	if q.Regex {
		_, err := q.regexpFor(false)
		return err == nil
	}
	return true
}

// regexpFor compiles the query's regex once per flavour. The plain flavour
// carries (?i) for case-insensitive queries; the boundary flavour
// additionally wraps the source in (?m)^(?:...)$ so it only accepts a whole
// call window at a time — that is what lets replace-time verification tell
// a true match apart from a coincidental substring.
func (q Query) regexpFor(boundary bool) (*regexp.Regexp, error) {
	src := q.Text
	if !q.CaseSensitive {
		src = "(?i)" + src
	}
	if boundary {
		src = "(?m)^(?:" + src + ")$"
	}
	return regexp.Compile(src)
}

// cursor reports the first candidate whose start is at or after cur.
type cursor interface {
	next(text string, cur int) (start, end int, ok bool)
}

// literalCursor finds the literal needle in the document. Case-sensitive
// lookups use strings.Index on the original buffer text. Case-insensitive
// lookups index an aligned lowered copy when lowering preserves the byte
// lengths of needle and text, then guard the matched content with
// equal-rune (fold) comparison over the original; when lowering would shift
// byte offsets, the scan falls back to rune-wise fold comparison over the
// original so ranges stay exact.
type literalCursor struct {
	needle string // original needle text
	fold   bool   // case-insensitive matching

	lowNeedle string // strings.ToLower(needle)
	lowOK     bool   // lowering kept needle's byte length

	src  string // document text the cached lowered copy belongs to
	low  string // strings.ToLower(src), byte-aligned with src when lowOKall
	lowT bool   // low is usable for byte indexing
}

func (l *literalCursor) next(text string, cur int) (int, int, bool) {
	n := len(l.needle)
	if n == 0 || cur < 0 || cur > len(text) {
		return 0, 0, false
	}
	if !l.fold {
		i := strings.Index(text[cur:], l.needle)
		if i < 0 {
			return 0, 0, false
		}
		s := cur + i
		return s, s + n, true
	}
	if l.lowOK {
		if l.src != text {
			l.src = text
			l.low = strings.ToLower(text)
			l.lowT = len(l.low) == len(text)
		}
		if l.lowT {
			for cur+n <= len(l.low) {
				i := strings.Index(l.low[cur:], l.lowNeedle)
				if i < 0 {
					return 0, 0, false
				}
				s := cur + i
				e := s + n
				if equalFoldWindow(text, s, e, l.needle) {
					return s, e, true
				}
				cur = nextRuneOff(text, s) // guard failed: resume one rune on
			}
			return 0, 0, false
		}
	}
	for cur+n <= len(text) {
		if m := foldMatch(text, l.needle, cur); m >= 0 {
			return cur, cur + m, true
		}
		cur = nextRuneOff(text, cur)
	}
	return 0, 0, false
}

// equalFoldWindow guards lowered-copy matches: it requires the window to end
// exactly at e, so a byte-length-changing fold can never leak a range.
func equalFoldWindow(text string, s, e int, needle string) bool {
	if e > len(text) || foldMatch(text, needle, s) != e-s {
		return false
	}
	return true
}

// foldMatch reports whether text[i:] begins with needle under rune-wise case
// folding, returning the bytes consumed and -1 otherwise.
func foldMatch(text, needle string, i int) int {
	rest := text[i:]
	eaten := 0
	for len(needle) > 0 {
		if len(rest) == 0 {
			return -1
		}
		nr, ns := utf8.DecodeRuneInString(needle)
		r, rs := utf8.DecodeRuneInString(rest)
		if nr != r && !foldEquals(nr, r) {
			return -1
		}
		needle = needle[ns:]
		rest = rest[rs:]
		eaten += rs
	}
	return eaten
}

// foldEquals mirrors unicode.SimpleFold equivalence for the common case
// where a and b differ only by case.
func foldEquals(a, b rune) bool {
	if a == b {
		return true
	}
	for r := unicode.SimpleFold(a); r != a; r = unicode.SimpleFold(r) {
		if r == b {
			return true
		}
	}
	return false
}

// wordCursor applies the whole-word boundary rule: a candidate only wins
// when the runes immediately before its start and at its end are not
// letters, digits, or underscores. Rejected candidates advance one rune.
type wordCursor struct {
	needle string
	fold   bool
}

func (w *wordCursor) next(text string, cur int) (int, int, bool) {
	n := len(w.needle)
	if n == 0 || cur < 0 || cur > len(text) {
		return 0, 0, false
	}
	for cur+n <= len(text) {
		end := 0
		if !w.fold {
			if strings.HasPrefix(text[cur:], w.needle) {
				end = cur + n
			}
		} else if m := foldMatch(text, w.needle, cur); m >= 0 {
			end = cur + m
		}
		if end > 0 && wordOK(text, cur, end) {
			return cur, end, true
		}
		cur = nextRuneOff(text, cur)
	}
	return 0, 0, false
}

// regexCursor reports regexp matches on the call site text[cur:]. Zero-width
// candidates are skipped so the scan always advances (one rune past the
// rejected start).
type regexCursor struct {
	re *regexp.Regexp
}

func (r *regexCursor) next(text string, cur int) (int, int, bool) {
	if cur < 0 || cur > len(text) {
		return 0, 0, false
	}
	for cur <= len(text) {
		loc := r.re.FindStringSubmatchIndex(text[cur:])
		if loc == nil {
			return 0, 0, false
		}
		s, e := cur+loc[0], cur+loc[1]
		if s == e {
			if s >= len(text) {
				return 0, 0, false // zero-width at document end: never a match
			}
			cur = nextRuneOff(text, s)
			continue
		}
		return s, e, true
	}
	return 0, 0, false
}

// cursorFor builds the scanner for q. err is only non-nil for a regex that
// does not compile. The Word flag is ignored for regex queries (contract:
// whole-word filtering applies to literal text only).
func (q Query) cursorFor() (cursor, error) {
	if q.Regex {
		re, err := q.regexpFor(false)
		if err != nil {
			return nil, err
		}
		return &regexCursor{re: re}, nil
	}
	if q.Word {
		return &wordCursor{needle: q.Text, fold: !q.CaseSensitive}, nil
	}
	if q.CaseSensitive {
		return &literalCursor{needle: q.Text}, nil
	}
	low := strings.ToLower(q.Text)
	return &literalCursor{needle: q.Text, fold: true, lowNeedle: low,
		lowOK: len(low) == len(q.Text)}, nil
}

// FindNext returns the first match whose start is at or after fromByte,
// wrapping from zero; ok is false when the buffer holds no match.
func (q Query) FindNext(buf *textbuf.Buffer, fromByte int) (start, end int, ok bool) {
	if !q.Valid() {
		return 0, 0, false
	}
	c, err := q.cursorFor()
	if err != nil {
		return 0, 0, false
	}
	text := buf.Text()
	n := len(text)
	cur0 := fromByte
	wrapped := fromByte < 0 || fromByte > n
	if wrapped {
		cur0 = 0
	}
	if s, e, found := scanNext(c, text, cur0); found {
		return s, e, true
	}
	if !wrapped && cur0 > 0 {
		if s, e, found := scanNext(c, text, 0); found {
			return s, e, true
		}
	}
	return 0, 0, false
}

// scanNext advances the cursor from cur0. Empty-range candidates are not
// matches: traversal steps one rune forward instead so zero-width patterns
// never wedge the scan at the document end.
func scanNext(c cursor, text string, cur0 int) (int, int, bool) {
	if cur0 < 0 {
		cur0 = 0
	}
	if cur0 > len(text) {
		cur0 = len(text)
	}
	cur := cur0
	for steps := 0; steps <= len(text)+1; steps++ {
		s, e, ok := c.next(text, cur)
		if !ok {
			return 0, 0, false
		}
		if s == e {
			if cur >= len(text) {
				return 0, 0, false
			}
			cur = nextRuneOff(text, s)
			continue
		}
		return s, e, true
	}
	return 0, 0, false
}

// FindPrev returns the closest match ending at or before fromByte, wrapping
// from the buffer end.
func (q Query) FindPrev(buf *textbuf.Buffer, fromByte int) (start, end int, ok bool) {
	if !q.Valid() {
		return 0, 0, false
	}
	c, err := q.cursorFor()
	if err != nil {
		return 0, 0, false
	}
	text := buf.Text()
	n := len(text)
	from := fromByte
	if from <= 0 || from > n {
		from = n // whole document, scanned backwards
	}
	if s, e, found := scanPrev(c, text, from); found {
		return s, e, true
	}
	if from != n {
		if s, e, found := scanPrev(c, text, n); found {
			return s, e, true
		}
	}
	return 0, 0, false
}

// scanPrev walks every candidate in document order and keeps the last one
// whose span ends at or before end0. Because a later candidate can be
// shorter than an earlier one, the sweep cannot stop at the first overlong
// match.
func scanPrev(c cursor, text string, end0 int) (int, int, bool) {
	if end0 > len(text) {
		end0 = len(text)
	}
	cur := 0
	bestS, bestE := 0, 0
	have := false
	for steps := 0; steps <= len(text)+1 && cur <= len(text); steps++ {
		s, e, ok := c.next(text, cur)
		if !ok {
			break
		}
		if s == e {
			if cur >= len(text) {
				break
			}
			cur = nextRuneOff(text, s)
			continue
		}
		if e <= end0 {
			bestS, bestE, have = s, e, true
		}
		cur = nextRuneOff(text, s)
	}
	return bestS, bestE, have
}

// MatchCount counts the matches in the whole buffer, capped at matchCap.
// Each match advances one rune past its start, which keeps the walk forward
// even for zero-width regex candidates.
func (q Query) MatchCount(buf *textbuf.Buffer) int {
	if !q.Valid() {
		return 0
	}
	c, err := q.cursorFor()
	if err != nil {
		return 0
	}
	text := buf.Text()
	count := 0
	cur := 0
	for steps := 0; steps <= len(text)+1 && cur <= len(text) && count < matchCap; steps++ {
		s, _, ok := c.next(text, cur)
		if !ok {
			break
		}
		count++
		if cur >= len(text) {
			break
		}
		cur = nextRuneOff(text, s)
	}
	return count
}

// ReplaceMatch replaces the range [start,end) with with (regex expansions
// $1.. are supported for regex queries) as a single undo step and returns
// the byte offset just past the inserted text.
func (q Query) ReplaceMatch(buf *textbuf.Buffer, start, end int, with string) int {
	n := buf.Len()
	if start < 0 {
		start = 0
	}
	if end < start {
		end = start
	}
	if start > n {
		start = n
	}
	if end > n {
		end = n
	}
	repl := with
	if q.Regex && q.Valid() {
		repl = string(expandRange(q, buf.Text(), start, end, with))
	}
	buf.BeginGroup()
	buf.DeleteRange(start, end)
	buf.Insert(start, repl)
	buf.EndGroup()
	return start + len(repl)
}

// ReplaceAll replaces every match in one grouped undo step and returns the
// number of replacements performed. Matches are located on the untouched
// document up front; the edit loop then walks them in order carrying a
// running byte delta (inserted minus deleted so far) that shifts each queued
// range into place before the delete/insert pair.
func (q Query) ReplaceAll(buf *textbuf.Buffer, with string) int {
	if !q.Valid() {
		return 0
	}
	c, err := q.cursorFor()
	if err != nil {
		return 0
	}
	text := buf.Text()
	type span struct{ s, e int }
	var ms []span
	cur := 0
	for steps := 0; steps <= len(text)+1 && cur <= len(text) && len(ms) < matchCap; steps++ {
		s, e, ok := c.next(text, cur)
		if !ok {
			break
		}
		if s == e {
			if cur >= len(text) {
				break
			}
			cur = nextRuneOff(text, s)
			continue
		}
		ms = append(ms, span{s, e})
		cur = e
	}
	if len(ms) == 0 {
		return 0
	}
	buf.BeginGroup()
	defer buf.EndGroup()
	delta := 0
	for _, m := range ms {
		s, e := m.s+delta, m.e+delta
		repl := with
		if q.Regex {
			repl = string(expandRange(q, text, m.s, m.e, with))
		}
		buf.DeleteRange(s, e)
		buf.Insert(s, repl)
		delta += len(repl) - (e - s)
	}
	return len(ms)
}

// expandRange computes the replacement for window [s,e) of text. The window
// is first verified against the anchored (boundary) pattern; if it does not
// cover the whole window, with is returned verbatim. Verified windows are
// run through the plain pattern with $1.. expansion.
func expandRange(q Query, text string, s, e int, with string) []byte {
	if s < 0 || e > len(text) || s > e {
		return []byte(with)
	}
	window := text[s:e]
	ct, err := q.regexpFor(true)
	if err != nil {
		return []byte(with)
	}
	loc := ct.FindStringIndex(window)
	if loc == nil || loc[0] != 0 || loc[1] != len(window) {
		return []byte(with)
	}
	plain, err := q.regexpFor(false)
	if err != nil {
		return []byte(with)
	}
	sub := plain.FindStringSubmatchIndex(window)
	if sub == nil {
		return []byte(with)
	}
	return plain.Expand(nil, []byte(with), []byte(window), sub)
}

// wordOK enforces the whole-word boundary around [s,e) in text.
func wordOK(text string, s, e int) bool {
	if s > 0 {
		r, _ := utf8.DecodeLastRuneInString(text[:s])
		if isWordRune(r) {
			return false
		}
	}
	if e < len(text) {
		r, _ := utf8.DecodeRuneInString(text[e:])
		if isWordRune(r) {
			return false
		}
	}
	return true
}

func isWordRune(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

// nextRuneOff returns the offset one whole rune after i (len(text) carries).
func nextRuneOff(text string, i int) int {
	if i >= len(text) {
		return len(text)
	}
	_, n := utf8.DecodeRuneInString(text[i:])
	return i + n
}
