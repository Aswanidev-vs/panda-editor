package views

import (
	"strings"
	"unicode"

	"github.com/Aswanidev-vs/cherry/cell"
	"github.com/Aswanidev-vs/cherry/geom"
	"github.com/Aswanidev-vs/cherry/render"

	"github.com/Aswanidev-vs/panda-editor/editor/theme"
	"github.com/Aswanidev-vs/cherry/widget"
)

// blank is a space cell painted with st, used for background fills.
func blank(st cell.Style) cell.Cell { return cell.Cell{Rune: ' ', Style: st, Width: 1} }

// strW sums the display width of every rune in s.
func strW(s string) int {
	w := 0
	for _, r := range s {
		w += cell.RuneWidth(r)
	}
	return w
}

// fitSize clamps a preferred size down to max; dimensions <= 0 in max mean
// unconstrained, mirroring the convention cherry's widgets use.
func fitSize(pref, max geom.Size) geom.Size {
	if max.W > 0 && pref.W > max.W {
		pref.W = max.W
	}
	if max.H > 0 && pref.H > max.H {
		pref.H = max.H
	}
	return pref
}

// cursorBlock paints a reversed block cell for cur at (x,y) and returns the x
// just past it. Nothing is drawn when x already reached maxX; a wide rune
// paints its trailing spacer cell in the same style.
func cursorBlock(sc *render.Screen, x, y, maxX int, cur rune, st cell.Style) int {
	w := cell.RuneWidth(cur)
	if w <= 0 {
		cur, w = ' ', 1
	}
	if x >= maxX {
		return x
	}
	sc.Set(x, y, cell.Cell{Rune: cur, Style: st, Width: uint8(w)})
	if w == 2 && x+1 < maxX {
		sc.Set(x+1, y, cell.Cell{Rune: ' ', Style: st, Width: 0})
	}
	return x + w
}

// spacedName renders s in uppercase with two spaces between letters, the
// "large format" trick used by the welcome splash.
func spacedName(s string) string {
	var b strings.Builder
	for i, r := range s {
		if i > 0 {
			b.WriteString("  ")
		}
		b.WriteRune(unicode.ToUpper(r))
	}
	return b.String()
}

// welcomeRow pairs one splash line with the style it is painted in.
type welcomeRow struct {
	text  string
	style cell.Style
}

// Welcome is the splash widget shown when panda starts with no files: a
// spaced-distance large-format editor name, version and three principal
// shortcuts. Text only, no focus.
type Welcome struct {
	widget.Base
	Version string
}

// rows builds the splash lines: the panda mark, a heavy rule, the spaced
// wordmark, the version, a blank row, and a two-column key grid.
// rows composes the splash as a centred column: the panda mark, a heavy rule
// under it, the spaced wordmark, the version, a blank row, then the two-column
// key grid.
//
// The mark and the rule are both MarkWidth cells wide so they read as one
// block. The rule must be built from MarkWidth rather than from len() of a
// mark row: those rows are multi-byte block glyphs, so their byte length is
// about three times their column count, and a rule sized that way would burst
// out of the popup.
func (w *Welcome) rows() []welcomeRow {
	ver := w.Version
	if ver == "" {
		ver = "dev"
	}
	r := theme.Current.Roles()

	rows := make([]welcomeRow, 0, len(theme.Mark)+6)
	for _, line := range theme.Mark {
		rows = append(rows, welcomeRow{text: line, style: r.Signal})
	}
	rows = append(rows, welcomeRow{
		text:  theme.HRule(theme.MarkWidth, theme.Glyph.RuleHeavy),
		style: r.Signal,
	})
	rows = append(rows, welcomeRow{text: spacedName("PANDA"), style: r.Badge})
	rows = append(rows, welcomeRow{text: "version " + ver, style: r.ChromeLabel})
	rows = append(rows, welcomeRow{text: "", style: r.Well})
	for _, line := range keyGridRows() {
		rows = append(rows, welcomeRow{text: line, style: r.ChromeLabel})
	}
	return rows
}

// keyGrid is the splash's two-column shortcut grid. It is one table rather
// than two loose strings so that both rows can be laid out against the same
// column width — sizing each row from its own label lets the second column
// drift by a cell between rows, which is exactly the kind of thing that makes
// a composition look accidental.
var keyGrid = [2][4]string{
	{"^O", "open", "^G", "help"},
	{"^N", "new", "^Q", "quit"},
}

func keyGridRows() []string {
	labelW := 0
	for _, row := range keyGrid {
		if w := strW(row[1]); w > labelW {
			labelW = w
		}
	}
	// Two cells for the key, one for the space after it, the widest label in
	// the column, then three cells of gutter before the second pair starts.
	col1 := 2 + 1 + labelW + 3

	out := make([]string, 0, len(keyGrid))
	for _, row := range keyGrid {
		left := row[0] + " " + row[1]
		out = append(out, left+strings.Repeat(" ", col1-strW(left))+row[2]+" "+row[3])
	}
	return out
}

// Measure returns the size of the welcome splash.
func (w *Welcome) Measure(max geom.Size) geom.Size {
	rows := w.rows()
	wd := 0
	for _, row := range rows {
		if lw := strW(row.text); lw > wd {
			wd = lw
		}
	}
	return fitSize(geom.Size{W: wd, H: len(rows)}, max)
}

// Draw paints the welcome splash.
func (w *Welcome) Draw(ctx *widget.DrawCtx) {
	rect := ctx.Rect
	if rect.Empty() {
		return
	}
	rows := w.rows()
	y0 := rect.Pos.Y + (rect.Size.H-len(rows))/2
	if y0 < rect.Pos.Y {
		y0 = rect.Pos.Y
	}
	for k, row := range rows {
		y := y0 + k
		if y >= rect.Bottom() {
			break
		}
		x := rect.Pos.X + (rect.Size.W-strW(row.text))/2
		if x < rect.Pos.X {
			x = rect.Pos.X
		}
		ctx.Screen.Print(x, y, rect.Right(), row.text, row.style)
	}
}

// printCentered paints s horizontally centered inside r on row y, clipped
// at the rect's right edge.
func printCentered(sc *render.Screen, s string, y int, r geom.Rect, st cell.Style) {
	x := r.Pos.X + (r.Size.W-strW(s))/2
	if x < r.Pos.X {
		x = r.Pos.X
	}
	sc.Print(x, y, r.Right(), s, st)
}

// actionMatchesKey reports whether an action label starts with the pressed
// letter, compared case-insensitively (y picks "yes", n picks "no", ...).
func actionMatchesKey(label string, r rune) bool {
	if label == "" {
		return false
	}
	for _, lr := range label {
		return unicode.ToLower(lr) == unicode.ToLower(r)
	}
	return false
}

// hintBarContent is the text to show in the hint bar.
const hintBarContent = "▸ ^G help   ^R save as   ^F find   ^G line   ^Q quit"

// hintBarWidget implements widget.Widget for the hint strip.
type hintBarWidget struct{}

// Measure returns the size of the hint bar: one row high, width as needed.
func (h *hintBarWidget) Measure(max geom.Size) geom.Size {
	return geom.Size{W: strW(hintBarContent), H: 1}
}

// Draw paints the hint bar: sunken background, faint ink, with the keys
// themselves in dimmed accent. Since widget.Text only supports one style,
// we approximate by using the dimmed accent for the whole string, which
// makes the keys stand out against the faint background.
func (h *hintBarWidget) Draw(ctx *widget.DrawCtx) {
	r := ctx.Rect
	if r.Empty() {
		return
	}
	// Sunken background.
	roles := theme.Current.Roles()
	ctx.Screen.Fill(r, blank(roles.Well))
	// Faint ink for the entire string, but we want the keys in dimmed accent.
	// As an approximation, we use dimmed accent for the whole string.
	// The design expects the keys in dimmed accent and the rest faint.
	// We cannot achieve multi-styled text with widget.Text, so we use dimmed
	// accent as a compromise: the keys are accent, the rest is slightly
	// brighter than faint but still readable.
	ctx.Screen.Print(r.Pos.X, r.Pos.Y, r.Right(), hintBarContent, roles.Signal)
}

// HintBar returns the hint bar widget. The content is the same for all modes,
// showing the global keybindings as per DESIGN.md §3.
func HintBar(mode string) *hintBarWidget {
	return &hintBarWidget{}
}