// Command preview renders the real editor UI to PNG files so the design can be
// reviewed as pixels instead of guessed at from source.
//
// It is a separate module under an underscore-prefixed directory, so the root
// module's ./... pattern ignores it and it can depend on x/image without
// touching the editor's own dependency graph.
//
// The go tool ignores underscore-prefixed directories in package patterns, so
// this module is run from inside itself:
//
//	go -C _preview run .              → writes preview-out/*.png
//	go -C _preview run . -out shots
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/Aswanidev-vs/cherry/cell"

	"github.com/Aswanidev-vs/cherry"
	"github.com/Aswanidev-vs/cherry/geom"
	"github.com/Aswanidev-vs/cherry/input"
	"github.com/Aswanidev-vs/cherry/render"
	"github.com/Aswanidev-vs/cherry/widget"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"

	"github.com/Aswanidev-vs/panda-editor/editor/cli"
	"github.com/Aswanidev-vs/panda-editor/editor/shell"
	"github.com/Aswanidev-vs/panda-editor/editor/theme"
)

// cols and rows are settable so the narrow-terminal case can be rendered
// too; a layout that only works at 100 columns is not a layout.
var (
	cols = 100
	rows = 32
)

// defaultFg/defaultBg stand in for the terminal's own defaults, which a cell
// carrying cell.Plain leaves untouched. The highlighter emits cell.Plain for
// plain-text files, so without a stand-in those cells would render as the
// preview's own background and hide a real regression.
var (
	defaultFg = color.RGBA{0xE9, 0xE3, 0xD7, 0xFF}
	defaultBg = color.RGBA{0x19, 0x16, 0x14, 0xFF}
)

func main() {
	out := flag.String("out", "preview-out", "directory to write PNGs into")
	asText := flag.Bool("text", false, "print each frame as text instead of writing a PNG")
	marks := flag.Bool("marks", false, "render the splash-mark comparison sheet and exit")
	repo := flag.String("repo", "..", "path to the editor repository")
	flag.IntVar(&cols, "cols", 100, "terminal width to render")
	flag.IntVar(&rows, "rows", 32, "terminal height to render")
	flag.Parse()

	if *marks {
		if err := markMode(*out); err != nil {
			fmt.Fprintln(os.Stderr, "preview: marks:", err)
			os.Exit(1)
		}
		return
	}

	faces, err := loadFaces()
	if err != nil {
		fmt.Fprintln(os.Stderr, "preview: fonts:", err)
		os.Exit(1)
	}
	if err := os.MkdirAll(*out, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "preview:", err)
		os.Exit(1)
	}

	r := runner{repo: *repo, out: *out, faces: faces, text: *asText}
	for _, s := range scenarios {
		// The label goes out before the frame. Printing it afterwards puts
		// every heading above the wrong frame, which quietly invalidates a
		// whole review — it looks like a layout bug in the surface you did
		// not even mean to look at.
		if *asText {
			fmt.Printf("\n===== %s =====\n", s.name)
		}
		if err := r.run(s); err != nil {
			fmt.Fprintf(os.Stderr, "preview: %s: %v\n", s.name, err)
			os.Exit(1)
		}
		if !*asText {
			fmt.Println("wrote", filepath.Join(*out, s.name+".png"))
		}
	}
}

type scenario struct {
	name  string
	theme *theme.Palette
	files []string
	keys  []string // key specs, in order
}

// scenarios are the states worth reviewing. Each opens real source files from
// the repository, so the syntax highlighting in the shots is the real thing
// rather than a hand-written sample that can drift from the lexer config.
var scenarios = []scenario{
	{name: "01-editor", theme: &theme.Ink, files: []string{"editor/theme/theme.go"}, keys: []string{"down:18", "end"}},
	{name: "02-selection", theme: &theme.Ink, files: []string{"editor/highlight/contract.go"}, keys: []string{"ctrl+f", "type:Palette", "enter"}},
	{name: "03-tabs", theme: &theme.Ink, files: []string{"editor/document/contract.go", "DESIGN.md", "editor/theme/style.go"}, keys: []string{"down:12", "type: "}},
	{name: "04-welcome", theme: &theme.Ink},
	{name: "05-help", theme: &theme.Ink, files: []string{"editor/editorview/draw.go"}, keys: []string{"f1"}},
	{name: "06-dialog", theme: &theme.Ink, files: []string{"editor/views/views.go"}, keys: []string{"down:20", "type:x", "ctrl+q"}},
	{name: "07-search", theme: &theme.Ink, files: []string{"editor/shell/contract.go"}, keys: []string{"ctrl+f", "type:overlay"}},
	{name: "08-light", theme: &theme.Light, files: []string{"editor/theme/theme.go"}, keys: []string{"down:40", "end"}},
}

type runner struct {
	repo  string
	out   string
	faces *faces
	text  bool
}

func (r *runner) run(s scenario) error {
	theme.Current = *s.theme

	sh, err := shell.New(newStubApp(), shell.Options{
		Args:    &cli.Args{Files: r.fileSpecs(s.files)},
		Version: "2.0.0",
	})
	if err != nil {
		return err
	}
	for _, k := range s.keys {
		r.press(sh, k)
	}

	scr := render.New(cols, rows)
	scr.SetColorMode(render.ColorRGB)
	sh.Draw(&widget.DrawCtx{
		Rect:   geom.Rect{Size: geom.Size{W: cols, H: rows}},
		Screen: scr,
	})
	if r.text {
		printFrame(scr)
		return nil
	}
	return writePNG(filepath.Join(r.out, s.name+".png"), scr, r.faces)
}

// fileSpecs turns repo-relative paths into CLI file specs. A missing file is
// fine: the editor opens an empty buffer that remembers the path.
func (r *runner) fileSpecs(paths []string) []cli.FileSpec {
	out := make([]cli.FileSpec, 0, len(paths))
	for _, p := range paths {
		full := filepath.Join(r.repo, filepath.FromSlash(p))
		out = append(out, cli.FileSpec{Path: full})
	}
	return out
}

// press applies one scripted key. "type:TEXT" types a literal string, "down:N"
// repeats a navigation key N times; everything else is a single named key.
func (r *runner) press(sh *shell.Shell, spec string) {
	switch {
	case len(spec) > 5 && spec[:5] == "type:":
		for _, ru := range spec[5:] {
			sh.Handle(input.KeyPress{Rune: ru})
		}
	case len(spec) > 5 && spec[:5] == "down:":
		var n int
		fmt.Sscanf(spec[5:], "%d", &n)
		for i := 0; i < n; i++ {
			sh.Handle(input.KeyPress{Key: input.KeyDown})
		}
	case spec == "end":
		sh.Handle(input.KeyPress{Key: input.KeyEnd})
	case spec == "enter":
		sh.Handle(input.KeyPress{Key: input.KeyEnter})
	case spec == "f1":
		sh.Handle(input.KeyPress{Key: input.KeyF1})
	case spec == "ctrl+q":
		sh.Handle(input.KeyPress{Rune: 'q', Mod: input.ModCtrl})
	case spec == "ctrl+f":
		sh.Handle(input.KeyPress{Rune: 'f', Mod: input.ModCtrl})
	default:
		panic("preview: unhandled key spec " + spec)
	}
}

// ---------------------------------------------------------------------------
// PNG rendering
// ---------------------------------------------------------------------------

type faces struct {
	regular font.Face
	bold    font.Face
	italic  font.Face
	cellW   int
	cellH   int
	ascent  int
}

// fontCandidates are tried in order. Cascadia Code first: its box-drawing and
// block-element coverage is better than Consolas', and the whole design is
// drawn with those glyphs.
var fontCandidates = [][3]string{
	{"CascadiaCode.ttf", "CascadiaCode.ttf", "CascadiaCode.ttf"},
	{"consola.ttf", "consolab.ttf", "consolai.ttf"},
	{"lucon.ttf", "lucon.ttf", "lucon.ttf"},
}

func loadFaces() (*faces, error) {
	for _, c := range fontCandidates {
		reg, err := loadFace(filepath.Join(`C:\Windows\Fonts`, c[0]), 16)
		if err != nil {
			continue
		}
		bold, err := loadFace(filepath.Join(`C:\Windows\Fonts`, c[1]), 16)
		if err != nil {
			bold = reg
		}
		italic, err := loadFace(filepath.Join(`C:\Windows\Fonts`, c[2]), 16)
		if err != nil {
			italic = reg
		}
		// The cell grid is sized from the font's own advance and line
		// height, so glyphs land where the terminal would put them and a
		// column that is one pixel out shows up as a visible skew.
		m := reg.Metrics()
		adv, ok := reg.GlyphAdvance('M')
		if !ok {
			continue
		}
		return &faces{
			regular: reg,
			bold:    bold,
			italic:  italic,
			cellW:   adv.Ceil(),
			cellH:   m.Height.Ceil(),
			ascent:  m.Ascent.Ceil(),
		}, nil
	}
	return nil, fmt.Errorf("no usable monospace TTF found")
}

func loadFace(path string, size float64) (font.Face, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	f, err := opentype.Parse(b)
	if err != nil {
		return nil, err
	}
	return opentype.NewFace(f, &opentype.FaceOptions{Size: size, DPI: 72, Hinting: font.HintingFull})
}

func writePNG(path string, scr *render.Screen, f *faces) error {
	img := image.NewRGBA(image.Rect(0, 0, f.cellW*cols, f.cellH*rows))
	draw.Draw(img, img.Bounds(), &image.Uniform{defaultBg}, image.Point{}, draw.Src)

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			c := scr.CellAt(x, y)
			if c.Width == 0 {
				continue // trailing half of a wide glyph
			}
			px, py := x*f.cellW, y*f.cellH
			fg, bg := cellColors(c.Style)
			draw.Draw(img, image.Rect(px, py, px+f.cellW, py+f.cellH), &image.Uniform{bg}, image.Point{}, draw.Src)
			if c.Rune == 0 || c.Rune == ' ' {
				continue
			}
			face := f.regular
			if c.Style.Attrs&cell.AttrBold != 0 {
				face = f.bold
			} else if c.Style.Attrs&cell.AttrItalic != 0 {
				face = f.italic
			}
			d := &font.Drawer{Dst: img, Src: image.NewUniform(fg), Face: face}
			d.Dot = fixed.P(px, py+f.ascent)
			d.DrawString(string(c.Rune))
		}
	}

	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()
	return png.Encode(out, img)
}

// cellColors resolves a style to concrete RGBA, honouring reverse video the
// way a terminal does: by swapping the two, not by inverting them. A cell
// whose colour is the terminal default borrows the preview's stand-in, which
// is what the user's own terminal background would have supplied.
func cellColors(s cell.Style) (fg, bg color.RGBA) {
	fg, bg = defaultFg, defaultBg
	if s.Fg.IsRGB() {
		r, g, b := s.Fg.RGB()
		fg = color.RGBA{r, g, b, 0xFF}
	}
	if s.Bg.IsRGB() {
		r, g, b := s.Bg.RGB()
		bg = color.RGBA{r, g, b, 0xFF}
	}
	if s.Attrs&cell.AttrReverse != 0 {
		fg, bg = bg, fg
	}
	return fg, bg
}

// newStubApp hands the shell the *cherry.App it was designed around. The
// editor only reaches for it to reach the clipboard, which no scenario here
// touches, and a real App would need a controlling terminal that a headless
// render does not have.
func newStubApp() *cherry.App { return &cherry.App{} }

// printFrame writes the frame as plain text plus a column ruler. It cannot
// show colour, but it is exact about layout — alignment, truncation, which
// keys the chrome claims, and whether any row is a column short — which is
// most of what a layout regression looks like.
func printFrame(scr *render.Screen) {
	ruler := make([]rune, cols)
	for i := range ruler {
		if (i+1)%10 == 0 {
			ruler[i] = '|'
		} else {
			ruler[i] = '.'
		}
	}
	fmt.Printf("    %s\n", string(ruler))
	for y := 0; y < rows; y++ {
		var b []rune
		for x := 0; x < cols; x++ {
			c := scr.CellAt(x, y)
			r := c.Rune
			if r == 0 {
				r = ' '
			}
			b = append(b, r)
		}
		fmt.Printf("%3d|%s|\n", y, string(b))
	}
}

// markMode renders the splash-mark comparison sheet instead of the editor
// frames: choosing between candidate marks is a visual judgement, and seeing
// them side by side at the same size is the only reliable way to make it.
func markMode(out string) error {
	if err := checkMarkWidths(); err != nil {
		return err
	}
	faces, err := loadFaces()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		return err
	}
	path := filepath.Join(out, "marks.png")
	if err := writeMarkSheet(path, faces); err != nil {
		return err
	}
	fmt.Println("wrote", path)
	for _, c := range markCandidates {
		fmt.Printf("\n--- %s (%dx%d) ---\n%s\n", c.name, cell.StringWidth(c.rows[0]), len(c.rows), strings.Join(c.rows, "\n"))
	}
	return nil
}
