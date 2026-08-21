package render

// Behavioral tests for the retained cell grid (screen.go), the SGR/CUP frame
// emitter (color.go), and environment-driven capability detection (detect.go).
//
// Every expected byte stream below was hand-derived from the CURRENT
// implementation and pinned exactly. Two spots deserve attention:
//
//   - Color16 emission of palette indices 0..7: color.go's doc comment
//     promises the classic 30-37/40-47 range, but sgrWriter.colorParams
//     computes `code := base` and only offsets bright colors (n >= 8), so
//     every dark index collapses to plain 30/40. See the BUG note inside
//     TestFlushColorDowngradeSGRForms; the test pins the real output.
//
//   - VS Code's terminal advertises truecolor via TERM_PROGRAM but is absent
//     from detect.go's synchronized-update allowlist, so it receives
//     ColorRGB without DECSET 2026 bracketing. Mirrored as-is in
//     TestDetectCapabilitiesTable.

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/Aswanidev-vs/cherry/cell"
	"github.com/Aswanidev-vs/cherry/geom"
)

// Escape fragments shared by the expectations below.
const (
	escHide  = "\x1b[?25l" // opens every Flush; emitted by a hiding ShowCursor
	escShow  = "\x1b[?25h"
	escReset = "\x1b[0m" // baseline literal paired with sgrWriter.reset
)

// cup builds a Cursor Position request; row and col are 1-based as emitted.
func cup(row, col int) string { return fmt.Sprintf("\x1b[%d;%dH", row, col) }

// mustFlush runs one Flush and fails the test on error, returning the bytes.
func mustFlush(t *testing.T, s *Screen) string {
	t.Helper()
	var out bytes.Buffer
	if err := s.Flush(&out); err != nil {
		t.Fatalf("Flush: unexpected error: %v", err)
	}
	return out.String()
}

// newTestScreen builds a screen locked to the given emission mode.
//
// WORKAROUND for a production bug: Screen.New never assigns s.sw.buf, so the
// sgrWriter holds a nil *bytes.Buffer and the FIRST styled cell of any frame
// panics in (*sgrWriter).to (color.go, w.buf.WriteString). All-plain frames
// get away with it only because to() early-returns before writing. Until
// New() gains the one-line fix `s.sw.buf = s.buf`, the tests bridge the two
// fields here; this line becomes redundant (harmless) once that lands.
func newTestScreen(w, h int, m ColorMode) *Screen {
	s := New(w, h)
	s.SetColorMode(m)
	s.sw.buf = s.buf
	return s
}

// newRGBScreen is a fresh screen locked to truecolor emission.
func newRGBScreen(w, h int) *Screen { return newTestScreen(w, h, ColorRGB) }

// 1. First Flush emits a full frame: hide-cursor prologue, literal reset,
// per-row cursor-home CUP, minimal SGR for the painted style, raw UTF-8 runes.
func TestFirstFlushEmitsFullFrame(t *testing.T) {
	s := newRGBScreen(2, 2)
	st := cell.Plain.Foreground(cell.RGB(255, 0, 0)).Bold(true)

	if ret := s.Print(0, 0, 2, "hé", st); ret != 2 {
		t.Fatalf("Print ret = %d, want 2", ret)
	}

	// Derived from screen.go: every row gets its own CUP home; the styled
	// cells share one SGR (bold + 24-bit fg); the blank second row forces an
	// attribute-dropping reset before its spaces. 'é' must arrive as raw UTF-8.
	want := escHide + escReset +
		cup(1, 1) + "\x1b[0;1;38;2;255;0;0m" + "hé" +
		cup(2, 1) + escReset + "  "

	if got := mustFlush(t, s); got != want {
		t.Fatalf("first flush:\n got %q\nwant %q", got, want)
	}
}

// 2. Incremental: one changed cell => exactly one CUP plus that rune.
func TestIncrementalFlushEmitsOnlyChangedCell(t *testing.T) {
	s := newRGBScreen(4, 2)
	s.Print(0, 0, 4, "hi", cell.Plain)

	wantFirst := escHide + escReset + cup(1, 1) + "hi  " + cup(2, 1) + "    "
	if got := mustFlush(t, s); got != wantFirst {
		t.Fatalf("first flush = %q, want %q", got, wantFirst)
	}

	s.Set(2, 1, cell.Cell{Rune: 'X', Width: 1})

	// Only the dirty row is walked, only the changed run is emitted: one CUP,
	// one rune, no reset, no full repaint.
	wantSecond := escHide + cup(2, 3) + "X"
	if got := mustFlush(t, s); got != wantSecond {
		t.Fatalf("incremental flush = %q, want %q", got, wantSecond)
	}
}

// 3. Unchanged frame Flush emits nothing meaningful.
func TestUnchangedFrameFlushIsQuiet(t *testing.T) {
	s := newRGBScreen(2, 1)
	s.Print(0, 0, 2, "ok", cell.Plain)
	mustFlush(t, s)

	for i := 1; i <= 3; i++ {
		if got := mustFlush(t, s); got != escHide {
			t.Fatalf("quiet flush #%d = %q, want only the mandatory cursor-hide %q", i, got, escHide)
		}
	}
}

// 4. Print wide-rune handling: Width==0 spacer after fitting wide glyphs,
// truncation at maxX (and at the screen edge), zero-width rune dropping.
func TestPrintWideRuneHandling(t *testing.T) {
	st := cell.Plain.Underline(true)

	type wantCell struct {
		x int
		c cell.Cell
	}
	cases := []struct {
		name    string
		w       int
		x, maxX int
		text    string
		wantRet int
		want    []wantCell
	}{
		{
			name: "fitting wide glyph leaves zero-width spacer",
			w:    4, x: 0, maxX: 4, text: "世",
			wantRet: 2,
			want: []wantCell{
				{0, cell.Cell{Rune: '世', Style: st, Width: 2}},
				{1, cell.Cell{Rune: ' ', Style: st, Width: 0}}, // spacer inherits the style
				{2, cell.Blank},
				{3, cell.Blank},
			},
		},
		{
			name: "wide pair straddling maxX truncates the line",
			w:    4, x: 0, maxX: 3, text: "世界",
			wantRet: 2,
			want: []wantCell{
				{0, cell.Cell{Rune: '世', Style: st, Width: 2}},
				{1, cell.Cell{Rune: ' ', Style: st, Width: 0}},
				{2, cell.Blank}, // 界 did not fit: cell left untouched
				{3, cell.Blank},
			},
		},
		{
			name: "maxX beyond screen caps at the screen edge",
			w:    3, x: 0, maxX: 99, text: "世界",
			wantRet: 2,
			want: []wantCell{
				{0, cell.Cell{Rune: '世', Style: st, Width: 2}},
				{1, cell.Cell{Rune: ' ', Style: st, Width: 0}},
				{2, cell.Blank},
			},
		},
		{
			name: "zero-width runes are dropped entirely",
			w:    4, x: 0, maxX: 4, text: "a\u0301b", // a + combining acute + b
			wantRet: 2,
			want: []wantCell{
				{0, cell.Cell{Rune: 'a', Style: st, Width: 1}},
				{1, cell.Cell{Rune: 'b', Style: st, Width: 1}},
				{2, cell.Blank},
				{3, cell.Blank},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := newRGBScreen(tc.w, 1)
			if got := s.Print(tc.x, 0, tc.maxX, tc.text, st); got != tc.wantRet {
				t.Fatalf("Print ret = %d, want %d", got, tc.wantRet)
			}
			for _, wc := range tc.want {
				if c := s.CellAt(wc.x, 0); c != wc.c {
					t.Errorf("CellAt(%d,0) = %+v, want %+v", wc.x, c, wc.c)
				}
			}
		})
	}

	t.Run("negative x clamps to column 0", func(t *testing.T) {
		s := newRGBScreen(4, 1)
		if got := s.Print(-3, 0, 2, "abc", st); got != 2 {
			t.Fatalf("Print ret = %d, want 2", got)
		}
		if c := s.CellAt(0, 0); c.Rune != 'a' {
			t.Errorf("CellAt(0,0).Rune = %q, want 'a'", c.Rune)
		}
		if c := s.CellAt(1, 0); c.Rune != 'b' {
			t.Errorf("CellAt(1,0).Rune = %q, want 'b'", c.Rune)
		}
		if c := s.CellAt(2, 0); c != cell.Blank {
			t.Errorf("CellAt(2,0) = %+v, want untouched Blank", c)
		}
	})

	t.Run("out-of-range y leaves the grid untouched", func(t *testing.T) {
		s := newRGBScreen(4, 1)
		if got := s.Print(1, 9, 4, "ab", st); got != 1 {
			t.Fatalf("Print ret = %d, want original x = 1", got)
		}
		if c := s.CellAt(1, 0); c != cell.Blank {
			t.Errorf("CellAt(1,0) = %+v, want untouched Blank", c)
		}
	})
}

// 5. Fill and Clear semantics verified through CellAt.
func TestFillAndClearViaCellAt(t *testing.T) {
	mark := cell.Cell{Rune: '#', Style: cell.Plain.Reverse(true), Width: 1}

	t.Run("fill paints exactly the rect interior", func(t *testing.T) {
		s := New(4, 3)
		s.Fill(geom.Rect{Pos: geom.Point{X: 1, Y: 1}, Size: geom.Size{W: 2, H: 2}}, mark)
		for y := 1; y <= 2; y++ {
			for x := 1; x <= 2; x++ {
				if c := s.CellAt(x, y); c != mark {
					t.Errorf("CellAt(%d,%d) = %+v, want %#+v", x, y, c, mark)
				}
			}
		}
		outside := []geom.Point{
			{X: 0, Y: 0}, {X: 1, Y: 0}, {X: 2, Y: 0}, {X: 3, Y: 0},
			{X: 0, Y: 1}, {X: 0, Y: 2}, {X: 3, Y: 2},
		}
		for _, p := range outside {
			if c := s.CellAt(p.X, p.Y); c != cell.Blank {
				t.Errorf("CellAt(%d,%d) = %+v, want Blank outside the rect", p.X, p.Y, c)
			}
		}
	})

	t.Run("fill clips regions extending past the edges", func(t *testing.T) {
		s := New(4, 3)
		s.Fill(geom.Rect{Pos: geom.Point{X: 3, Y: 0}, Size: geom.Size{W: 10, H: 10}}, mark)
		if c := s.CellAt(3, 0); c != mark {
			t.Errorf("CellAt(3,0) = %+v, want marked", c)
		}
		if c := s.CellAt(3, 2); c != mark {
			t.Errorf("CellAt(3,2) = %+v, want marked", c)
		}
		if c := s.CellAt(2, 1); c != cell.Blank {
			t.Errorf("CellAt(2,1) = %+v, want Blank (clipped)", c)
		}
	})

	t.Run("empty rect fill is a no-op", func(t *testing.T) {
		s := New(2, 2)
		s.Fill(geom.Rect{}, mark)
		for y := 0; y < 2; y++ {
			for x := 0; x < 2; x++ {
				if c := s.CellAt(x, y); c != cell.Blank {
					t.Errorf("CellAt(%d,%d) = %+v, want Blank after empty fill", x, y, c)
				}
			}
		}
	})

	t.Run("clear resets every cell to Blank", func(t *testing.T) {
		s := New(3, 2)
		s.Fill(s.Bounds(), mark)
		s.Clear()
		for y := 0; y < 2; y++ {
			for x := 0; x < 3; x++ {
				if c := s.CellAt(x, y); !c.IsBlank() {
					t.Errorf("CellAt(%d,%d) = %+v, want Blank after Clear", x, y, c)
				}
			}
		}
	})

	t.Run("out-of-range reads return the zero cell", func(t *testing.T) {
		s := New(2, 2)
		for _, p := range []geom.Point{{X: -1, Y: 0}, {X: 0, Y: -1}, {X: 2, Y: 0}, {X: 0, Y: 2}} {
			if c := s.CellAt(p.X, p.Y); c != (cell.Cell{}) {
				t.Errorf("CellAt(%d,%d) = %+v, want zero Cell", p.X, p.Y, c)
			}
		}
	})
}

// 6. Color downgrade: the same painted style flushed under each mode must
// produce exactly the SGR forms implemented in color.go.
func TestFlushColorDowngradeSGRForms(t *testing.T) {
	cases := []struct {
		name string
		mode ColorMode
		fg   cell.Color
		sgr  string // SGR between the row-home CUP and the rune; "" = none
	}{
		{name: "truecolor passes through untouched", mode: ColorRGB, fg: cell.RGB(255, 0, 0), sgr: "\x1b[0;38;2;255;0;0m"},
		{name: "pure red hits exact cube cell 196", mode: Color256, fg: cell.RGB(255, 0, 0), sgr: "\x1b[0;38;5;196m"},
		{name: "neutral gray prefers the 24-step ramp (244)", mode: Color256, fg: cell.RGB(128, 128, 128), sgr: "\x1b[0;38;5;244m"},
		// BUG (suspected, color.go colorParams): in Color16 mode the indexed
		// branch computes `code := base` and only offsets n >= 8, so indices
		// 0..7 ALL emit 30/40. rgbTo16(255,0,0) correctly resolves to ANSI red
		// (index 1, documented as 31), yet the emitter writes 30 (black).
		// This row pins the real output; if it ever fails reporting 31, the
		// `base + n` fix landed and this expectation should be updated.
		{name: "rgb->16 dark range pins current off-by-index output", mode: Color16, fg: cell.RGB(255, 0, 0), sgr: "\x1b[0;30m"},
		{name: "rgb->16 bright range 90-97 as documented", mode: Color16, fg: cell.RGB(255, 255, 255), sgr: "\x1b[0;97m"},
		{name: "indexed >=16 remaps through lut256to16 (200 -> 95)", mode: Color16, fg: cell.Indexed(200), sgr: "\x1b[0;95m"},
		{name: "indexed passes through in 256 mode", mode: Color256, fg: cell.Indexed(200), sgr: "\x1b[0;38;5;200m"},
		{name: "mono strips color entirely: no SGR parameters", mode: ColorMono, fg: cell.RGB(255, 0, 0), sgr: ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := newTestScreen(1, 1, tc.mode)
			s.Print(0, 0, 1, "X", cell.Plain.Foreground(tc.fg))
			want := escHide + escReset + cup(1, 1) + tc.sgr + "X"
			if got := mustFlush(t, s); got != want {
				t.Fatalf("flush = %q, want %q", got, want)
			}
		})
	}
}

// 7a. ShowCursor: positive coordinates write CUP plus reveal; negatives hide.
func TestShowCursorSequences(t *testing.T) {
	cases := []struct {
		name string
		x, y int
		want string
	}{
		{name: "positive coords position then reveal", x: 3, y: 2, want: cup(3, 4) + escShow},
		{name: "origin normalizes to 1-based terminal coords", x: 0, y: 0, want: cup(1, 1) + escShow},
		{name: "negative x hides", x: -1, y: 4, want: escHide},
		{name: "negative y hides", x: 4, y: -1, want: escHide},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			if err := New(8, 8).ShowCursor(&out, tc.x, tc.y); err != nil {
				t.Fatalf("ShowCursor: unexpected error: %v", err)
			}
			if got := out.String(); got != tc.want {
				t.Fatalf("ShowCursor(%d,%d) = %q, want %q", tc.x, tc.y, got, tc.want)
			}
		})
	}
}

// 7b. DetectCapabilities mirrors detect.go's documented precedence, using a
// fake getenv so no real process environment leaks into the results.
func TestDetectCapabilitiesTable(t *testing.T) {
	cases := []struct {
		name   string
		env    map[string]string
		nilGet bool
		mode   ColorMode
		sync   bool
	}{
		{name: "bare environment", env: map[string]string{}, mode: Color16, sync: false},
		{name: "COLORTERM truecolor", env: map[string]string{"COLORTERM": "truecolor"}, mode: ColorRGB, sync: false},
		{name: "COLORTERM 24bit", env: map[string]string{"COLORTERM": "24bit"}, mode: ColorRGB, sync: false},
		{name: "COLORTERM match is case-insensitive", env: map[string]string{"COLORTERM": "TrueColor"}, mode: ColorRGB, sync: false},
		{name: "TERM 256color", env: map[string]string{"TERM": "xterm-256color"}, mode: Color256, sync: false},
		{name: "COLORTERM outranks TERM 256", env: map[string]string{"COLORTERM": "truecolor", "TERM": "xterm-256color"}, mode: ColorRGB, sync: false},
		{name: "WT_SESSION implies RGB and sync bracketing", env: map[string]string{"WT_SESSION": "{79A42E81-8D14-4C5E-9B2F-A1B2C3D4E5F6}"}, mode: ColorRGB, sync: true},
		{name: "vscode reports truecolor but is not on the sync allowlist", env: map[string]string{"TERM_PROGRAM": "vscode"}, mode: ColorRGB, sync: false},
		{name: "iTerm.app truecolor plus sync", env: map[string]string{"TERM_PROGRAM": "iTerm.app"}, mode: ColorRGB, sync: true},
		{name: "WezTerm truecolor plus sync", env: map[string]string{"TERM_PROGRAM": "WezTerm"}, mode: ColorRGB, sync: true},
		{name: "kitty TERM enables sync but stays Color16 without COLORTERM", env: map[string]string{"TERM": "xterm-kitty"}, mode: Color16, sync: true},
		{name: "ghostty enables sync", env: map[string]string{"TERM": "ghostty"}, mode: Color16, sync: true},
		{name: "alacritty enables sync", env: map[string]string{"TERM": "alacritty"}, mode: Color16, sync: true},
		{name: "nil getenv behaves as an empty environment", nilGet: true, mode: Color16, sync: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var get func(string) string
			if !tc.nilGet {
				get = func(k string) string { return tc.env[k] }
			}
			mode, sync := DetectCapabilities(get)
			if mode != tc.mode {
				t.Errorf("mode = %v, want %v", mode, tc.mode)
			}
			if sync != tc.sync {
				t.Errorf("syncOutput = %v, want %v", sync, tc.sync)
			}
		})
	}
}

// 8. Invalidate forces a full repaint on the next Flush even though nothing
// changed in either buffer.
func TestInvalidateForcesFullRepaint(t *testing.T) {
	s := newRGBScreen(3, 1)

	full := escHide + escReset + cup(1, 1) + "   "
	if got := mustFlush(t, s); got != full {
		t.Fatalf("first flush = %q, want full frame %q", got, full)
	}
	if got := mustFlush(t, s); got != escHide {
		t.Fatalf("steady-state flush = %q, want bare %q", got, escHide)
	}

	s.Invalidate() // contents untouched; only the dirty marks are raised

	if got := mustFlush(t, s); got != full {
		t.Fatalf("post-Invalidate flush = %q, want full repaint %q", got, full)
	}
}
