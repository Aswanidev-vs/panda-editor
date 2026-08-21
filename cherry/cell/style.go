package cell

// Color is a terminal color: Default, one of 256 indexed colors, or 24-bit RGB.
// The zero value is Default.
type Color struct {
	// bits 0-23: RGB value; bits 24-31: mode (see below)
	v uint32
}

const (
	modeDefault uint32 = iota << 29
	modeIndexed
	modeRGB
)

// mode isolates the top three bits where the color-kind tag lives; the
// mode* constants are pre-shifted into those bits, so they compare directly.
func (c Color) mode() uint32 { return c.v & (7 << 29) }

// DefaultColor leaves the terminal's default color untouched.
var DefaultColor = Color{}

// Indexed returns palette color n (0-255).
func Indexed(n uint8) Color { return Color{v: uint32(n) | modeIndexed} }

// RGB returns a truecolor value.
func RGB(r, g, b uint8) Color {
	return Color{v: uint32(r)<<16 | uint32(g)<<8 | uint32(b) | modeRGB}
}

func (c Color) IsDefault() bool { return c.mode() == modeDefault }
func (c Color) IsIndexed() bool { return c.mode() == modeIndexed }
func (c Color) IsRGB() bool     { return c.mode() == modeRGB }
func (c Color) Index() uint8    { return uint8(c.v & 0xFF) }
func (c Color) RGB() (r, g, b uint8) {
	return uint8(c.v >> 16), uint8(c.v >> 8), uint8(c.v)
}

// Hex parses "#rrggbb" / "#rgb" strings; ok is false on malformed input.
func Hex(s string) (Color, bool) {
	if len(s) > 0 && s[0] == '#' {
		s = s[1:]
	}
	switch len(s) {
	case 3:
		var v [3]uint8
		for i := 0; i < 3; i++ {
			n, ok := hexNibble(s[i])
			if !ok {
				return DefaultColor, false
			}
			v[i] = n * 17
		}
		return RGB(v[0], v[1], v[2]), true
	case 6:
		var v [3]uint8
		for i := 0; i < 3; i++ {
			hi, ok1 := hexNibble(s[i*2])
			lo, ok2 := hexNibble(s[i*2+1])
			if !ok1 || !ok2 {
				return DefaultColor, false
			}
			v[i] = hi<<4 | lo
		}
		return RGB(v[0], v[1], v[2]), true
	}
	return DefaultColor, false
}

func hexNibble(b byte) (uint8, bool) {
	switch {
	case b >= '0' && b <= '9':
		return b - '0', true
	case b >= 'a' && b <= 'f':
		return b - 'a' + 10, true
	case b >= 'A' && b <= 'F':
		return b - 'A' + 10, true
	}
	return 0, false
}

// Attr is a set of text attributes.
type Attr uint16

const (
	AttrBold Attr = 1 << iota
	AttrFaint
	AttrItalic
	AttrUnderline
	AttrBlink
	AttrReverse
	AttrInvisible
	AttrStrikethrough
)

// Style is an immutable cell style. The zero value renders with terminal defaults.
type Style struct {
	Fg    Color
	Bg    Color
	Attrs Attr
}

// Plain is the default style.
var Plain = Style{}

func (s Style) Foreground(c Color) Style { s.Fg = c; return s }
func (s Style) Background(c Color) Style { s.Bg = c; return s }
func (s Style) Bold(on bool) Style       { return s.attr(AttrBold, on) }
func (s Style) Italic(on bool) Style     { return s.attr(AttrItalic, on) }
func (s Style) Underline(on bool) Style  { return s.attr(AttrUnderline, on) }
func (s Style) Reverse(on bool) Style    { return s.attr(AttrReverse, on) }
func (s Style) Faint(on bool) Style      { return s.attr(AttrFaint, on) }
func (s Style) Strike(on bool) Style     { return s.attr(AttrStrikethrough, on) }

func (s Style) attr(a Attr, on bool) Style {
	if on {
		s.Attrs |= a
	} else {
		s.Attrs &^= a
	}
	return s
}
