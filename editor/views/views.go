package views

import (
	"strings"
	"unicode"

	"github.com/Aswanidev-vs/cherry/cell"
	"github.com/Aswanidev-vs/cherry/geom"
	"github.com/Aswanidev-vs/cherry/render"
)

// Chrome palette shared by every widget in the package: a calm dark surface
// (236), light text (252), a single blue accent (75) and a dimmed variant
// (240) for secondary information.
var (
	colFG     = cell.Indexed(252)
	colBG     = cell.Indexed(236)
	colAccent = cell.Indexed(75)
	colDim    = cell.Indexed(240)
)

var (
	styleText    = cell.Style{}.Foreground(colFG).Background(colBG)
	styleDim     = cell.Style{}.Foreground(colDim).Background(colBG)
	styleAccent  = cell.Style{}.Foreground(colAccent).Background(colBG).Bold(true)
	styleInverse = cell.Style{}.Foreground(colBG).Background(colAccent).Bold(true)
	styleBorder  = cell.Style{}.Foreground(colAccent).Background(colBG)
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

// hintStrips are the nano-style key hints HintBar shows per mode.
var hintStrips = map[string]string{
	"insert":  "^G Help    ^O Write Out    ^W Where Is    ^K Cut    ^U Paste    ^X Exit",
	"search":  "^G Help    Enter Confirm    ^C Cancel",
	"dialog":  "Enter Choose    Esc Cancel    y/n Quick Pick",
	"welcome": "^O Open    ^Q Quit    ^G Help",
}

const hintFallback = "^G Help    Esc Close"

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

// rows builds the splash lines: the spaced editor name, the version and the
// three principal shortcuts, each kept shorter than the name line.
func (w *Welcome) rows() []welcomeRow {
	ver := w.Version
	if ver == "" {
		ver = "dev"
	}
	return []welcomeRow{
		{spacedName("PANDA"), styleAccent},
		{"version " + ver, styleDim},
		{"", styleText},
		{"^O open", styleDim},
		{"^Q quit", styleDim},
		{"^G help", styleDim},
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
