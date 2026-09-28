package theme

import (
	"github.com/Aswanidev-vs/cherry/cell"
	"github.com/Aswanidev-vs/cherry/geom"
	"github.com/Aswanidev-vs/cherry/render"
)

// Mix blends a toward b by t in 24-bit RGB: t=0 returns a, t=1 returns b,
// and values between interpolate per channel. It is how the surface ramp is
// stepped and how the modal scrim darkens whatever is already on screen.
//
// A terminal default is not a colour to blend, so it is treated as "unset":
// mixing a default toward anything returns the other operand unchanged.
// Indexed colours are likewise returned as-is — the screen buffer only ever
// holds the palette's RGB values, and the 256/16/mono downgrade happens later,
// at Flush, so an indexed operand here means the caller handed in a colour the
// ramp never produced.
func Mix(a, b cell.Color, t float64) cell.Color {
	if a.IsDefault() {
		return b
	}
	if b.IsDefault() {
		return a
	}
	if t <= 0 {
		return a
	}
	if t >= 1 {
		return b
	}
	if !a.IsRGB() || !b.IsRGB() {
		return a
	}
	ar, ag, ab := a.RGB()
	br, bg, bb := b.RGB()
	return cell.RGB(
		lerp8(ar, br, t),
		lerp8(ag, bg, t),
		lerp8(ab, bb, t),
	)
}

func lerp8(a, b uint8, t float64) uint8 {
	v := float64(a) + (float64(b)-float64(a))*t
	if v <= 0 {
		return 0
	}
	if v >= 255 {
		return 255
	}
	return uint8(v + 0.5)
}

// Style builds a cell style from a palette, so callers never assemble
// foreground/background pairs by hand:
//
//	theme.Style(theme.Current.Fg, theme.Current.Base)
//	theme.Style(theme.Current.Accent, theme.Current.Raised, cell.AttrBold)
func Style(fg, bg cell.Color, attrs ...cell.Attr) cell.Style {
	s := cell.Plain.Foreground(fg).Background(bg)
	for _, a := range attrs {
		s.Attrs |= a
	}
	return s
}

// Roles holds the recurring cell styles: a widget that needs "a readout value
// on the status strip" asks for ReadoutValue instead of re-deriving the same
// foreground/background/weight triple — which is how two surfaces end up
// disagreeing about what a value looks like.
//
// Roles belong to a Palette, so a widget resolves them from the theme it is
// actually drawing with (theme.Current.Roles()) rather than from a value
// captured at init, which would outlive a theme switch.
type Roles struct {
	// Chrome surfaces.
	Well        cell.Style // sunken chrome background
	Surface     cell.Style // base background
	Raised      cell.Style // one step up
	Overlay     cell.Style // modal interior
	Hairline    cell.Style // rule between segments
	ChromeLabel cell.Style // faint text on a chrome surface

	// Ink.
	Text      cell.Style // primary text on Base
	TextDim   cell.Style // secondary text
	TextFaint cell.Style // tertiary text, disabled, comments

	// Signal.
	Badge    cell.Style // mode badge: the loudest element in the chrome
	BadgeAlt cell.Style // the badge, for a second mode
	Signal   cell.Style // accent text on a chrome surface
	Mark     cell.Style // the blinking unsaved dot

	// Readouts (status strip).
	ReadoutLabel cell.Style // small-caps label
	ReadoutValue cell.Style // the value itself
	ReadoutNote  cell.Style // fixed facts like encoding and line ending

	// Editor body.
	Rule       cell.Style // the gutter's measurement rule
	RuleActive cell.Style // the same rule, on the cursor line
	Rail       cell.Style // the gutter's cursor marker
	GutterNum  cell.Style // inactive line numbers
	GutterCur  cell.Style // the cursor line's number
	EOF        cell.Style // the marker past the last line
	Body       cell.Style // plain text in the editor

	// Modals.
	Frame       cell.Style // the popup's border
	FrameTitle  cell.Style // the popup's title
	DialogBody  cell.Style // dialog message
	ActionIdle  cell.Style // an unselected dialog action
	ActionFocus cell.Style // the selected dialog action
}

// Roles resolves the recurring cell styles for this palette. Call it inside
// Draw — never cache the result in a package-level var, or it will outlive a
// theme switch.
func (p Palette) Roles() Roles {
	return Roles{
		Well:        Style(p.FgFaint, p.Sunken),
		Surface:     Style(p.Fg, p.Base),
		Raised:      Style(p.Fg, p.Raised),
		Overlay:     Style(p.Fg, p.Overlay),
		Hairline:    Style(p.Rule, p.Base),
		ChromeLabel: Style(p.FgFaint, p.Sunken),

		Text:      Style(p.Fg, p.Base),
		TextDim:   Style(p.FgDim, p.Base),
		TextFaint: Style(p.FgFaint, p.Base),

		Badge:    Style(p.Accent, p.Raised, cell.AttrBold),
		BadgeAlt: Style(p.AccentAlt, p.Raised, cell.AttrBold),
		Signal:   Style(p.Accent, p.Sunken),
		Mark:     Style(p.Accent, p.Base, cell.AttrBlink),

		ReadoutLabel: Style(p.FgFaint, p.Base),
		ReadoutValue: Style(p.Fg, p.Base, cell.AttrBold),
		ReadoutNote:  Style(p.FgFaint, p.Base),

		Rule:       Style(p.Rule, p.Base),
		RuleActive: Style(p.AccentDim, p.CursorLine),
		Rail:       Style(p.Accent, p.CursorLine),
		GutterNum:  Style(p.FgFaint, p.Sunken),
		GutterCur:  Style(p.Accent, p.CursorLine, cell.AttrBold),
		EOF:        Style(p.FgFaint, p.Base),
		Body:       Style(p.Fg, p.Base),

		Frame:       Style(p.AccentDim, p.Overlay),
		FrameTitle:  Style(p.Accent, p.Overlay, cell.AttrBold),
		DialogBody:  Style(p.FgDim, p.Overlay),
		ActionIdle:  Style(p.FgFaint, p.Overlay),
		ActionFocus: Style(p.Accent, p.Raised, cell.AttrBold),
	}
}

// ScrimStrength is how far the scrim pulls the editor back behind a modal.
// It is strong enough that the popup reads as the only lit surface, and light
// enough that the code behind it is still recognisable as context.
const ScrimStrength = 0.65

// Scrim darkens every cell already painted in r toward Sunken, which is how a
// modal interrupts instead of merely covering. Because it reads what is on
// screen and re-blends it, it works for arbitrary content — text, rules,
// chrome — without the caller having to redraw anything underneath.
//
// It runs before the modal is drawn, over the viewport the modal covers.
func Scrim(r geom.Rect, sc *render.Screen) {
	if sc == nil || r.Empty() {
		return
	}
	p := Current
	base := p.Sunken
	area := r.Intersect(sc.Bounds())
	for y := area.Pos.Y; y < area.Bottom(); y++ {
		for x := area.Pos.X; x < area.Right(); x++ {
			c := sc.CellAt(x, y)
			// Never scrim the wide-glyph spacer cell: it is not a glyph of
			// its own, and restyling it would break the pairing with its
			// lead cell.
			if c.Width == 0 {
				continue
			}
			c.Style.Fg = Mix(c.Style.Fg, base, ScrimStrength)
			c.Style.Bg = Mix(c.Style.Bg, base, ScrimStrength)
			sc.Set(x, y, c)
		}
	}
}
