package theme

import (
	"reflect"
	"testing"

	"github.com/Aswanidev-vs/cherry/cell"
	"github.com/Aswanidev-vs/cherry/geom"
	"github.com/Aswanidev-vs/cherry/render"
)

func themes() map[string]Palette {
	return map[string]Palette{"Ink": Ink, "Vellum": Vellum}
}

// Every field must carry a real colour. The zero value is the terminal
// default, which silently renders as whatever the user's terminal happens to
// use — a whole theme of unset fields looks almost right and is impossible to
// spot in a screenshot.
func TestEveryPaletteFieldIsSet(t *testing.T) {
	for name, p := range themes() {
		v := p
		v.Bg, v.GutterBg, v.StatusBar, v.TabBg, v.TabActiveBg = nil5(v.Bg), nil5(v.GutterBg), nil5(v.StatusBar), nil5(v.TabBg), nil5(v.TabActiveBg)
		v.TabFg, v.TabActiveFg, v.LineNum, v.LineNumActive, v.Border = nil5(v.TabFg), nil5(v.TabActiveFg), nil5(v.LineNum), nil5(v.LineNumActive), nil5(v.Border)
		count := 0
		visit(v, func(field string, c cell.Color) {
			count++
			if c.IsDefault() {
				t.Errorf("%s.%s is unset (terminal default)", name, field)
			}
		})
		if count != 26 {
			t.Errorf("%s: visited only %d fields, expected all 26 semantic fields", name, count)
		}
	}
}

// The legacy alias fields must agree with the semantic ones, or two spellings
// of the same role would render as different colours.
func TestLegacyAliasesMatchSemanticFields(t *testing.T) {
	for name, p := range themes() {
		cases := []struct {
			field       string
			legacy, sem cell.Color
		}{
			{"Bg", p.Bg, p.Base},
			{"GutterBg", p.GutterBg, p.Sunken},
			{"StatusBar", p.StatusBar, p.Base},
			{"TabBg", p.TabBg, p.Sunken},
			{"TabActiveBg", p.TabActiveBg, p.Raised},
			{"TabFg", p.TabFg, p.FgFaint},
			{"TabActiveFg", p.TabActiveFg, p.Fg},
			{"LineNum", p.LineNum, p.FgFaint},
			{"LineNumActive", p.LineNumActive, p.Accent},
			{"Border", p.Border, p.Rule},
		}
		for _, c := range cases {
			if c.legacy != c.sem {
				t.Errorf("%s.%s = %#v, want the same value as its semantic field", name, c.field, c.legacy)
			}
		}
	}
}

// The accent is reserved for UI state. A syntax family that shares it would
// make chrome indistinguishable from code, which is the one thing the palette
// is built to prevent.
func TestSyntaxDoesNotReuseTheAccent(t *testing.T) {
	for name, p := range themes() {
		syntax := map[string]cell.Color{
			"Keyword": p.Keyword, "String": p.String, "Number": p.Number,
			"Function": p.Function, "Type": p.Type, "Operator": p.Operator,
			"Builtin": p.Builtin, "Comment": p.Comment, "Punct": p.Punct,
		}
		for field, c := range syntax {
			if c == p.Accent {
				t.Errorf("%s: syntax %s reuses the accent", name, field)
			}
			if c == p.Fg {
				t.Errorf("%s: syntax %s is indistinguishable from body text", name, field)
			}
			if c.IsDefault() {
				t.Errorf("%s: syntax %s is unset", name, field)
			}
		}
	}
}

func TestMixEndpointsAndInterpolation(t *testing.T) {
	a := cell.RGB(0, 0, 0)
	b := cell.RGB(200, 100, 40)
	if got := Mix(a, b, 0); got != a {
		t.Errorf("Mix(a,b,0) = %#v, want a", got)
	}
	if got := Mix(a, b, 1); got != b {
		t.Errorf("Mix(a,b,1) = %#v, want b", got)
	}
	got := Mix(a, b, 0.5)
	if r, gg, bb := got.RGB(); r != 100 || gg != 50 || bb != 20 {
		t.Errorf("Mix(a,b,0.5) = %d,%d,%d, want 100,50,20", r, gg, bb)
	}
	// Out-of-range t clamps rather than extrapolating into an invalid colour.
	if got := Mix(a, b, 2); got != b {
		t.Errorf("Mix(a,b,2) = %#v, want b", got)
	}
	if got := Mix(a, b, -1); got != a {
		t.Errorf("Mix(a,b,-1) = %#v, want a", got)
	}
}

// A terminal default is not a colour, so blending one must not produce a
// colour where there was none.
func TestMixTreatsDefaultAsUnset(t *testing.T) {
	c := cell.RGB(10, 20, 30)
	if got := Mix(cell.DefaultColor, c, 0.5); got != c {
		t.Errorf("Mix(default,c,0.5) = %#v, want c", got)
	}
	if got := Mix(c, cell.DefaultColor, 0.5); got != c {
		t.Errorf("Mix(c,default,0.5) = %#v, want c", got)
	}
}

// A glyph that is not exactly one cell wide would shift every column after it,
// so the vocabulary is asserted rather than trusted. The list is walked by
// reflection so a glyph added to the vocabulary later cannot skip the check.
func TestEveryGlyphIsOneCellWide(t *testing.T) {
	v := reflect.ValueOf(Glyph)
	checked := 0
	for i := 0; i < v.NumField(); i++ {
		g := rune(v.Field(i).Int())
		if g == 0 {
			t.Errorf("glyph %s is unset", v.Type().Field(i).Name)
			continue
		}
		if w := cell.RuneWidth(g); w != 1 {
			t.Errorf("glyph %s (%q, U+%04X) is %d cells wide, want 1 — it would break column alignment",
				v.Type().Field(i).Name, g, g, w)
		}
		checked++
	}
	if checked == 0 {
		t.Fatal("no glyphs were checked")
	}
}

// The splash mark is drawn as five centred rows inside a popup whose width
// comes from Measure, so a row that is not the declared width would shift the
// wordmark off centre.
func TestSplashMarkRowsAreUniformWidth(t *testing.T) {
	for i, row := range Mark {
		if w := cell.StringWidth(row); w != MarkWidth {
			t.Errorf("mark row %d is %d cells wide, want %d (row %q)", i, w, MarkWidth, row)
		}
	}
	if MarkWidth == 0 {
		t.Error("MarkWidth is zero")
	}
}

func TestScrimDarkensWithoutMovingGlyphs(t *testing.T) {
	before := Current
	defer func() { Current = before }()
	Current = Ink

	sc := render.New(4, 2)
	lit := cell.Style{}.Foreground(Ink.Fg).Background(Ink.Raised)
	sc.Fill(geom.Rect{Size: geom.Size{W: 4, H: 2}}, cell.Cell{Rune: ' ', Style: lit, Width: 1})
	sc.Print(0, 0, 4, "abc", lit)

	Scrim(geom.Rect{Size: geom.Size{W: 4, H: 2}}, sc)

	c := sc.CellAt(0, 0)
	if c.Rune != 'a' {
		t.Errorf("scrim moved a glyph: got %q, want 'a'", c.Rune)
	}
	if c.Style.Bg == lit.Bg {
		t.Error("scrim left the background untouched")
	}
	wantR, wantG, wantB := Ink.Raised.RGB()
	// The background must have moved toward Sunken, not merely changed.
	gotR, gotG, gotB := c.Style.Bg.RGB()
	if !(gotR < wantR && gotG < wantG && gotB < wantB) {
		t.Errorf("scrim background %d,%d,%d did not move toward sunken from %d,%d,%d",
			gotR, gotG, gotB, wantR, wantG, wantB)
	}
}

// Scrimming must stay inside the given rect: a modal's scrim that darkened
// the whole screen would also dim the chrome the user still needs.
func TestScrimRespectsItsRect(t *testing.T) {
	before := Current
	defer func() { Current = before }()
	Current = Ink

	sc := render.New(4, 2)
	lit := cell.Style{}.Foreground(Ink.Fg).Background(Ink.Raised)
	sc.Fill(geom.Rect{Size: geom.Size{W: 4, H: 2}}, cell.Cell{Rune: ' ', Style: lit, Width: 1})

	Scrim(geom.Rect{Size: geom.Size{W: 4, H: 1}}, sc)

	if sc.CellAt(0, 0).Style.Bg == lit.Bg {
		t.Error("row inside the scrim rect was not darkened")
	}
	if sc.CellAt(0, 1).Style.Bg != lit.Bg {
		t.Error("row outside the scrim rect was darkened")
	}
}

// Roles must never hand back a default-colour style: a chrome surface with no
// background paints as a hole in the terminal's own background.
func TestRolesAreFullyColoured(t *testing.T) {
	before := Current
	defer func() { Current = before }()

	for _, name := range []string{"Ink", "Vellum"} {
		p := themes()[name]
		Current = p
		r := Current.Roles()
		for field, s := range map[string]cell.Style{
			"Well": r.Well, "Surface": r.Surface, "Raised": r.Raised,
			"Overlay": r.Overlay, "Hairline": r.Hairline, "Text": r.Text,
			"TextDim": r.TextDim, "TextFaint": r.TextFaint, "Badge": r.Badge,
			"Mark": r.Mark, "Rule": r.Rule, "RuleActive": r.RuleActive,
			"Rail": r.Rail, "GutterNum": r.GutterNum, "GutterCur": r.GutterCur,
			"Body": r.Body, "Frame": r.Frame, "ActionFocus": r.ActionFocus,
			"ReadoutValue": r.ReadoutValue,
		} {
			if s.Fg.IsDefault() || s.Bg.IsDefault() {
				t.Errorf("%s Roles().%s has an unset colour: %+v", name, field, s)
			}
		}
	}
}

func TestStyleAppliesAttributes(t *testing.T) {
	s := Style(Ink.Accent, Ink.Raised, cell.AttrBold, cell.AttrBlink)
	if s.Attrs&cell.AttrBold == 0 {
		t.Error("AttrBold was not applied")
	}
	if s.Attrs&cell.AttrBlink == 0 {
		t.Error("AttrBlink was not applied")
	}
	if s.Fg != Ink.Accent || s.Bg != Ink.Raised {
		t.Errorf("Style colours = %#v/%#v, want the palette's", s.Fg, s.Bg)
	}
}

func TestHRuleClamps(t *testing.T) {
	if got := HRule(3, Glyph.RuleLight); len([]rune(got)) != 3 {
		t.Errorf("HRule(3) = %q, want 3 cells", got)
	}
	if got := HRule(-2, Glyph.RuleLight); got != "" {
		t.Errorf("HRule(-2) = %q, want empty", got)
	}
	if got := VRule(0, Glyph.TickHeavy); got != "" {
		t.Errorf("VRule(0) = %q, want empty", got)
	}
}

// ---------------------------------------------------------------------------

// nil5 returns a zero colour, used to blank the legacy fields so visit can
// walk the semantic ones without double-counting.
func nil5(c cell.Color) cell.Color { return cell.Color{} }

// visit calls fn for every exported colour field of p.
func visit(p Palette, fn func(field string, c cell.Color)) {
	fn("Sunken", p.Sunken)
	fn("Base", p.Base)
	fn("Raised", p.Raised)
	fn("Overlay", p.Overlay)
	fn("Rule", p.Rule)
	fn("Fg", p.Fg)
	fn("FgDim", p.FgDim)
	fn("FgFaint", p.FgFaint)
	fn("Accent", p.Accent)
	fn("AccentAlt", p.AccentAlt)
	fn("AccentDim", p.AccentDim)
	fn("CursorLine", p.CursorLine)
	fn("Selection", p.Selection)
	fn("Cursor", p.Cursor)
	fn("Error", p.Error)
	fn("Warning", p.Warning)
	fn("Success", p.Success)
	fn("Comment", p.Comment)
	fn("Keyword", p.Keyword)
	fn("String", p.String)
	fn("Number", p.Number)
	fn("Function", p.Function)
	fn("Type", p.Type)
	fn("Operator", p.Operator)
	fn("Builtin", p.Builtin)
	fn("Punct", p.Punct)
}
