package render

import (
	"bytes"
	"sync"

	"github.com/Aswanidev-vs/cherry/cell"
)

// Color downgrading: RGB -> 256 -> 16.

// cubeLevels are the xterm 6x6x6 cube channel values.
var cubeLevels = [6]int{0, 95, 135, 175, 215, 255}

// ansi16RGB is the classic VGA palette used as the target space for 16-color
// terminals.
var ansi16RGB = [16][3]int{
	{0x00, 0x00, 0x00}, {0xAA, 0x00, 0x00}, {0x00, 0xAA, 0x00}, {0xAA, 0x55, 0x00},
	{0x00, 0x00, 0xAA}, {0xAA, 0x00, 0xAA}, {0x00, 0xAA, 0xAA}, {0xAA, 0xAA, 0xAA},
	{0x55, 0x55, 0x55}, {0xFF, 0x55, 0x55}, {0x55, 0xFF, 0x55}, {0xFF, 0xFF, 0x55},
	{0x55, 0x55, 0xFF}, {0xFF, 0x55, 0xFF}, {0x55, 0xFF, 0xFF}, {0xFF, 0xFF, 0xFF},
}

func dist2(r, g, b, or, og, ob int) int {
	dr, dg, db := r-or, g-og, b-ob
	return dr*dr + dg*dg + db*db
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// rgbTo256 maps a 24-bit color to the nearest xterm-256 entry: whichever of
// the rounded 6x6x6 cube cell and the 24-step grayscale ramp is closer wins.
func rgbTo256(r, g, b uint8) uint8 {
	ri, gi, bi := int(r), int(g), int(b)

	qr := nearestCubeLevel(ri)
	qg := nearestCubeLevel(gi)
	qb := nearestCubeLevel(bi)
	ci := 16 + 36*qr + 6*qg + qb
	cr, cg, cb := cubeLevels[qr], cubeLevels[qg], cubeLevels[qb]
	dCube := dist2(ri, gi, bi, cr, cg, cb)

	lum := (ri + gi + bi) / 3
	gi256 := 232 + (lum+5-8)/10
	if gi256 < 232 {
		gi256 = 232
	} else if gi256 > 255 {
		gi256 = 255
	}
	gr := 8 + 10*(gi256-232)
	dGray := dist2(ri, gi, bi, gr, gr, gr)

	if dGray < dCube {
		return uint8(gi256)
	}
	return uint8(ci)
}

func nearestCubeLevel(v int) int {
	best, bd := 0, abs(v-cubeLevels[0])
	for i := 1; i < 6; i++ {
		if d := abs(v - cubeLevels[i]); d < bd {
			best, bd = i, d
		}
	}
	return best
}

// rgbTo16 finds the nearest of the 16 ANSI colors by squared RGB distance.
// Ties resolve to the lower index.
func rgbTo16(r, g, b uint8) uint8 {
	ri, gi, bi := int(r), int(g), int(b)
	best, bd := 0, 1<<30
	for i, c := range ansi16RGB {
		if d := dist2(ri, gi, bi, c[0], c[1], c[2]); d < bd {
			best, bd = i, d
		}
	}
	return uint8(best)
}

// palette256 reconstructs the xterm-256 RGB value for index n.
func palette256(n uint8) (int, int, int) {
	switch {
	case n < 16:
		c := &ansi16RGB[n]
		return c[0], c[1], c[2]
	case n < 232:
		i := int(n) - 16
		return cubeLevels[i/36], cubeLevels[(i/6)%6], cubeLevels[i%6]
	default:
		v := 8 + 10*(int(n)-232)
		return v, v, v
	}
}

var (
	lut16Once sync.Once
	lut16     [256]uint8
)

// lut256to16 lazily precomputes the nearest ANSI-16 index for every 256-palette
// entry, so indexed colors downgrade without repeated color math.
func lut256to16() *[256]uint8 {
	lut16Once.Do(func() {
		for i := range lut16 {
			r, g, b := palette256(uint8(i))
			lut16[i] = rgbTo16(uint8(r), uint8(g), uint8(b))
		}
	})
	return &lut16
}

// downgradeRGB maps a 24-bit color into the given mode's space: the nearest
// 256-palette index, the nearest ANSI-16 index, the unchanged RGB value, or
// DefaultColor for mono.
func downgradeRGB(r, g, b uint8, m ColorMode) cell.Color {
	switch m {
	case Color256:
		return cell.Indexed(rgbTo256(r, g, b))
	case Color16:
		return cell.Indexed(rgbTo16(r, g, b))
	case ColorRGB:
		return cell.RGB(r, g, b)
	default: // ColorMono
		return cell.DefaultColor
	}
}

// resolveColor normalizes any terminal color to what the current mode can
// emit: RGB steps down when mode<ColorRGB, indexed values beyond 16 map through
// the LUT in 16-color mode, and mono strips all color.
func resolveColor(c cell.Color, m ColorMode) cell.Color {
	switch {
	case c.IsDefault():
		return c
	case m == ColorMono:
		return cell.DefaultColor
	case m == Color16 && c.IsRGB():
		r, g, b := c.RGB()
		return downgradeRGB(r, g, b, m)
	case m == Color256 && c.IsRGB():
		r, g, b := c.RGB()
		return downgradeRGB(r, g, b, m)
	case m == Color16 && c.IsIndexed() && c.Index() >= 16:
		return cell.Indexed(lut256to16()[c.Index()])
	default:
		return c
	}
}

// sgrWriter accumulates the minimal SGR transitions between successive cell
// styles into the shared frame buffer. Its tracked state mirrors what the
// terminal currently holds, so it persists across frames and runs; steady-state
// emission allocates nothing.
type sgrWriter struct {
	buf  *bytes.Buffer
	cur  cell.Style
	have bool // cur reflects a style the terminal actually received
	mode ColorMode
}

type attrCode struct {
	bit  cell.Attr
	code byte // SGR parameter, all standard attributes are 1..9
}

var attrCodes = [...]attrCode{
	{cell.AttrBold, 1},
	{cell.AttrFaint, 2},
	{cell.AttrItalic, 3},
	{cell.AttrUnderline, 4},
	{cell.AttrBlink, 5},
	{cell.AttrReverse, 7},
	{cell.AttrInvisible, 8},
	{cell.AttrStrikethrough, 9},
}

// reset forgets the tracked style. Callers that wipe the screen pair this with
// a literal "\e[0m" so the terminal matches again.
func (w *sgrWriter) reset() {
	w.cur = cell.Style{}
	w.have = false
}

// to emits the escape sequence moving the terminal from the tracked style to
// next, then tracks it. Nothing is written when next already applies.
func (w *sgrWriter) to(next cell.Style) {
	if w.have && w.cur == next {
		return
	}

	nfg, nbg := resolveColor(next.Fg, w.mode), resolveColor(next.Bg, w.mode)
	pfg, pbg := resolveColor(w.cur.Fg, w.mode), resolveColor(w.cur.Bg, w.mode)

	// Attributes cannot be selectively removed, so any attribute present in
	// cur but missing from next forces a reset plus full restatement.
	dropAttrs := w.have && ^next.Attrs&w.cur.Attrs != 0
	addAttrs := next.Attrs
	if !dropAttrs && w.have {
		addAttrs = next.Attrs &^ w.cur.Attrs
	}
	if dropAttrs || !w.have {
		pfg, pbg = cell.DefaultColor, cell.DefaultColor
	}
	fgCh, bgCh := nfg != pfg, nbg != pbg

	if !dropAttrs && addAttrs == 0 && !fgCh && !bgCh {
		w.cur, w.have = next, true
		return
	}

	w.buf.WriteString("\x1b[")
	first := true
	if dropAttrs || !w.have {
		w.buf.WriteByte('0')
		first = false
	}
	for _, ac := range attrCodes {
		if addAttrs&ac.bit != 0 {
			if !first {
				w.buf.WriteByte(';')
			}
			w.buf.WriteByte('0' + ac.code)
			first = false
		}
	}
	if fgCh && !nfg.IsDefault() {
		first = w.colorParams(nfg, false, first)
	}
	if bgCh && !nbg.IsDefault() {
		w.colorParams(nbg, true, first)
	}
	w.buf.WriteByte('m')
	w.cur, w.have = next, true
}

// writeUint appends v in decimal without strconv's allocation.
func writeUint(buf *bytes.Buffer, v uint32) {
	if v == 0 {
		buf.WriteByte('0')
		return
	}
	var tmp [10]byte
	i := len(tmp)
	for v > 0 {
		i--
		tmp[i] = byte('0' + v%10)
		v /= 10
	}
	buf.Write(tmp[i:])
}

// colorParams appends foreground/background parameters for a resolved color:
// 38;5;N / 48;5;N indexed, 38;2;r;g;b / 48;2;r;g;b RGB, or plain 30-37/90-97
// (40-47/100-107) ANSI codes in 16-color mode.
func (w *sgrWriter) colorParams(c cell.Color, bg bool, first bool) bool {
	base := uint8(30)
	if bg {
		base = 40
	}
	if !first {
		w.buf.WriteByte(';')
	}
	switch {
	case c.IsIndexed():
		n := c.Index()
		if w.mode == Color16 {
			code := base
			if n >= 8 {
				code = base + 60 + n - 8
			}
			writeUint(w.buf, uint32(code))
		} else {
			if bg {
				w.buf.WriteString("48;5;")
			} else {
				w.buf.WriteString("38;5;")
			}
			writeUint(w.buf, uint32(n))
		}
	case c.IsRGB():
		r, g, b := c.RGB()
		if bg {
			w.buf.WriteString("48;2;")
		} else {
			w.buf.WriteString("38;2;")
		}
		writeUint(w.buf, uint32(r))
		w.buf.WriteByte(';')
		writeUint(w.buf, uint32(g))
		w.buf.WriteByte(';')
		writeUint(w.buf, uint32(b))
	default: // DefaultColor reaching here only in mono mode: skip silently
		return first
	}
	return false
}
