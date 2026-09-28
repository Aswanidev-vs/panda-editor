package theme

import "strings"

// glyphs is the editor's drawing vocabulary, grouped so callers write
// theme.Glyph.TickLight rather than a flat list of package constants. One
// namespace keeps the drawing language a single decision instead of a
// scattering of lookalike characters, and it stays greppable.
//
// Only four Unicode blocks are used — Box Drawing, Block Elements, Geometric
// Shapes and Arrows. All of them are single-width in cherry's width table, so
// no glyph here can shift a column and break alignment. Private-use codepoints
// (Nerd Font icons) are deliberately excluded: they render as tofu on most
// terminals, and a broken row is worse than a plain one.
type glyphs struct {
	// Rules and rails. Hairlines carry structure; heavy rules mark the one
	// edge that matters.
	RuleLight rune // U+2500 continuous hairline
	RuleHeavy rune // U+2501 heavy rule: the splash mark's underline
	TickLight rune // U+2506 dashed rail: the gutter's measurement rule
	TickHeavy rune // U+2502 solid rail: the same rule, on the cursor line
	Divider   rune // U+2502 solid hairline: between status-strip segments

	// Blocks.
	BlockFull  rune // U+2588 full block
	BlockUpper rune // U+2580 upper half: the splash mark's ears
	BlockLower rune // U+2584 lower half
	BlockLeft  rune // U+258C left half: the gutter marker and tab tick
	BlockRight rune // U+2590 right half
	BlockLight rune // U+2591 light shade: the splash mark's eye patches

	// Splash-mark corners, used only inside Mark.
	CornerTL rune // U+259B upper one eighth, left
	CornerTR rune // U+259C upper one eighth, right

	// Marks and pointers.
	DotFilled  rune // U+25CF filled circle: unsaved-changes indicator
	DotOpen    rune // U+25CB hollow circle: clean-state counterpart
	MiddleDot  rune // U+00B7: hairline separator, and the past-EOF marker
	Diamond    rune // U+25C6
	PointerR   rune // U+25B8 points at the current selection
	PointerL   rune // U+25C2
	ChevronL   rune // U+2039 single left angle quote: the tab-overflow marker
	CheckMark  rune // U+2713
	CrossMark  rune // U+2717
	Ellipsis   rune // U+2026
	ArrowUp    rune // U+2191
	ArrowRight rune // U+2192
}

// Glyph is the shared glyph vocabulary.
var Glyph = glyphs{
	RuleLight:  '─',
	RuleHeavy:  '━',
	TickLight:  '┆',
	TickHeavy:  '│',
	Divider:    '│',
	BlockFull:  '█',
	BlockUpper: '▀',
	BlockLower: '▄',
	BlockLeft:  '▌',
	BlockRight: '▐',
	BlockLight: '░',
	CornerTL:   '▛',
	CornerTR:   '▜',
	DotFilled:  '●',
	DotOpen:    '○',
	MiddleDot:  '·',
	Diamond:    '◆',
	PointerR:   '▸',
	PointerL:   '◂',
	ChevronL:   '‹',
	CheckMark:  '✓',
	CrossMark:  '✗',
	Ellipsis:   '…',
	ArrowUp:    '↑',
	ArrowRight: '→',
}

// The splash mark: a panda face assembled from block elements, nineteen
// columns wide and six rows high — see DESIGN.md §5.
//
// Two decisions carry the whole design. The ears sit on their own rows with a
// blank row between them and the head, because ears fused into the head's top
// edge read as a bump on a skull rather than as ears. And the face has both
// eye patches and a solid muzzle: a hollow outline with nothing in the middle
// reads as a mask, and a thin nose alone reads as a smudge. Every glyph is
// single-width, so the mark can never shift a column.
const (
	MarkRow0 = "    ▄▀▄     ▄▀▄    "
	MarkRow1 = "   █▛▀▜█   █▛▀▜█   "
	MarkRow2 = "                   "
	MarkRow3 = "  ▛▀▀▀▀▀▀▀▀▀▀▀▀▀▜  "
	MarkRow4 = "  █ ░░░ ███ ░░░ █  "
	MarkRow5 = "  ▙▄▄▄▄▄▄▄▄▄▄▄▄▄▟  "
)

// MarkWidth is the display width of every row of the splash mark. It is the
// column count, not the byte count: the rows are multi-byte block glyphs, so
// len() would report roughly three times the real width and every layout that
// centres on it would be wrong.
const MarkWidth = 19

// Mark is the splash mark as six rows, ready to draw.
var Mark = [6]string{MarkRow0, MarkRow1, MarkRow2, MarkRow3, MarkRow4, MarkRow5}

// HRule returns a horizontal rule of n cells. n is clamped at zero.
func HRule(n int, glyph rune) string {
	if n <= 0 {
		return ""
	}
	return strings.Repeat(string(glyph), n)
}

// VRule returns a vertical rule of n cells. It is a convenience for callers
// painting column by column; drawing a rail is normally a single Set per row.
func VRule(n int, glyph rune) string {
	if n <= 0 {
		return ""
	}
	return strings.Repeat(string(glyph), n)
}
