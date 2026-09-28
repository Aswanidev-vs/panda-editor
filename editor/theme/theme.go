// Package theme holds the editor's design system: the surface ramp, the ink
// ramp, the single signal colour, the syntax families, the glyph vocabulary
// and the precomputed styles that recurring roles use.
//
// Two themes ship. Ink (dark) is the default; Vellum (light) is its paper
// counterpart. Both are expressed in the same semantic field names, so no
// widget may hard-code a hex value — it asks the palette for a role
// ("surface", "rule", "signal") and gets that theme's exact shade.
//
// Cherry renders these as 24-bit RGB on capable terminals and downgrades them
// automatically (256 / 16 / mono) on constrained ones, so the editor can
// always request the exact shade and still be readable everywhere.
//
// The framework (cherry) stays domain-agnostic; the colour identity of the
// editor lives here so every widget draws from one source of truth. The
// visual rules these colours implement are specified in DESIGN.md.
package theme

import "github.com/Aswanidev-vs/cherry/cell"

// Palette is one editor theme: every surface, ink and syntax family as a cell
// colour. The zero value is the terminal default, which no field uses.
//
// Fields fall into five groups.
//
// Surfaces — a value ramp from deepest to lightest, so a surface reads as
// nearer or farther by how dark it is rather than by hue.
//
//	Sunken  chrome wells: gutter, tab strip, hint strip
//	Base    the editor body and status strip
//	Raised  one step up: active tab, modal frame interior
//	Overlay the top step: popup body, control chips
//	Rule    hairlines, separators, tick rules
//
// Ink — text on those surfaces, from primary to faint.
//
// Accent — exactly one signal colour. It is reserved for UI state (mode
// badge, active tab, focus, cursor, readouts) and is never used for syntax,
// so chrome stays legible while code scrolls underneath it.
//
// Syntax — one colour per chroma token family.
type Palette struct {
	// Name identifies the theme in a status readout or a theme switcher.
	Name string

	// Surfaces.
	Sunken  cell.Color
	Base    cell.Color
	Raised  cell.Color
	Overlay cell.Color
	Rule    cell.Color

	// Ink.
	Fg      cell.Color
	FgDim   cell.Color
	FgFaint cell.Color

	// Signal.
	Accent    cell.Color
	AccentAlt cell.Color
	AccentDim cell.Color

	// Text-area state.
	CursorLine cell.Color
	Selection  cell.Color
	Cursor     cell.Color

	// Diagnostics.
	Error   cell.Color
	Warning cell.Color
	Success cell.Color

	// Syntax families. Punct is separate from Fg so punctuation can recede
	// below ordinary text instead of competing with it.
	Comment  cell.Color
	Keyword  cell.Color
	String   cell.Color
	Number   cell.Color
	Function cell.Color
	Type     cell.Color
	Operator cell.Color
	Builtin  cell.Color
	Punct    cell.Color

	// Legacy aliases. These predate the surface ramp and are kept so older
	// consumers keep compiling; newPalette derives them from the semantic
	// fields above, so a theme can never disagree with itself. Prefer the
	// semantic name in new code.
	Bg            cell.Color
	GutterBg      cell.Color
	StatusBar     cell.Color
	TabBg         cell.Color
	TabActiveBg   cell.Color
	TabFg         cell.Color
	TabActiveFg   cell.Color
	LineNum       cell.Color
	LineNumActive cell.Color
	Border        cell.Color
}

// newPalette derives the legacy alias fields from the semantic ones. A
// theme is always described once, in semantic terms; the aliases exist only
// so that the ramp cannot be applied inconsistently across two spellings of
// the same role.
func newPalette(p Palette) Palette {
	p.Bg = p.Base
	p.GutterBg = p.Sunken
	p.StatusBar = p.Base
	p.TabBg = p.Sunken
	p.TabActiveBg = p.Raised
	p.TabFg = p.FgFaint
	p.TabActiveFg = p.Fg
	p.LineNum = p.FgFaint
	p.LineNumActive = p.Accent
	p.Border = p.Rule
	return p
}

// hex parses "#rrggbb" / "#rgb" into a cell colour.
//
// A malformed literal is a programming error, not a runtime condition: a
// silently-defaulted shade produces an invisible UI bug that no test can see.
// Panicking here turns a typo into an immediate, obvious failure.
func hex(s string) cell.Color {
	c, ok := cell.Hex(s)
	if !ok {
		panic("theme: malformed colour literal " + s)
	}
	return c
}

// Ink is the default theme: a warm near-black drafting substrate, bone-white
// ink, one ochre signal and a cool-leaning syntax ramp. Cool syntax against
// a warm surface keeps code from competing with the ochre chrome.
var Ink = newPalette(Palette{
	Name: "Ink",

	// Surfaces: carbon black lifting to graphite.
	Sunken:  hex("#100E0C"),
	Base:    hex("#191614"),
	Raised:  hex("#221D19"),
	Overlay: hex("#2B241E"),
	Rule:    hex("#332B23"),

	// Ink: bone white, warm all the way down.
	Fg:      hex("#E9E3D7"),
	FgDim:   hex("#A79C8B"),
	FgFaint: hex("#6F6558"),

	// Signal: drafting ochre, its cooler sibling, and its dimmed form.
	Accent:    hex("#E8A33D"),
	AccentAlt: hex("#C97B4A"),
	AccentDim: hex("#8A6224"),

	// Text-area state. The cursor-line wash is a 4% lift off Base and the
	// selection is ochre-tinted so selected code reads as inked over.
	CursorLine: hex("#1F1A15"),
	Selection:  hex("#3A2F1E"),
	Cursor:     hex("#E8A33D"),

	// Diagnostics.
	Error:   hex("#E05B4A"),
	Warning: hex("#E8A33D"),
	Success: hex("#8FBF6A"),

	// Syntax: terracotta keywords, sage strings, dusty-blue numbers, a pale
	// straw for functions (the most frequent token stays close to plain
	// text), muted teal types, and punctuation dimmer than everything.
	Comment:  hex("#6F6558"),
	Keyword:  hex("#E2836A"),
	String:   hex("#93B884"),
	Number:   hex("#7FA8C4"),
	Function: hex("#D9C6A2"),
	Type:     hex("#6FB3A8"),
	Operator: hex("#B8AFA2"),
	Builtin:  hex("#9A8FC4"),
	Punct:    hex("#8A8175"),
})

// Vellum is the light counterpart: warm paper, near-black ink, burnt ochre
// signal. Same ramp structure and the same syntax relationships as Ink, so
// both themes read as one design in two media.
var Vellum = newPalette(Palette{
	Name: "Vellum",

	// Surfaces: paper stock.
	Sunken:  hex("#EDE7DB"),
	Base:    hex("#F7F2E8"),
	Raised:  hex("#FDFAF3"),
	Overlay: hex("#FFFFFF"),
	Rule:    hex("#DCD1BC"),

	// Ink.
	Fg:      hex("#221E19"),
	FgDim:   hex("#5A5248"),
	FgFaint: hex("#948A7B"),

	// Signal.
	Accent:    hex("#A85B12"),
	AccentAlt: hex("#8A4A6B"),
	AccentDim: hex("#C79A6A"),

	// Text-area state.
	CursorLine: hex("#EFE8DA"),
	Selection:  hex("#DCC9A0"),
	Cursor:     hex("#A85B12"),

	// Diagnostics.
	Error:   hex("#B03A2B"),
	Warning: hex("#A85B12"),
	Success: hex("#4A7A2E"),

	// Syntax.
	Comment:  hex("#948A7B"),
	Keyword:  hex("#9C3B1E"),
	String:   hex("#4A6B2A"),
	Number:   hex("#2F5C7A"),
	Function: hex("#6B4E14"),
	Type:     hex("#1F6360"),
	Operator: hex("#7A7167"),
	Builtin:  hex("#5B4B8A"),
	Punct:    hex("#8A8175"),
})

// Dark and Light are the original names for the two themes, kept so existing
// code keeps compiling. New code should use Ink and Vellum.
var (
	Dark  = Ink
	Light = Vellum
)

// Current is the palette every widget reads. It starts at Ink. Because the
// theme is swappable, widgets must resolve their styles inside Draw (or via
// the helpers in style.go) rather than capturing them into package-level
// vars at init time, which would freeze whatever Current happened to be.
var Current = Ink
