package cell

// Cell is one character position on screen. Width is the display width of
// Rune (1 or 2); wide runes occupy their trailing cell as Width==0 spacer.
type Cell struct {
	Rune  rune
	Style Style
	Width uint8
}

// Blank is an empty cell in the terminal's default style.
var Blank = Cell{Rune: ' ', Width: 1}

// IsBlank reports whether c renders as default-styled space.
func (c Cell) IsBlank() bool {
	return c.Rune == ' ' && c.Style == Plain
}
