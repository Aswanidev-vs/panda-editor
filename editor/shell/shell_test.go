package shell

import (
	"testing"

	"github.com/Aswanidev-vs/cherry"
	"github.com/Aswanidev-vs/cherry/geom"
	"github.com/Aswanidev-vs/cherry/input"
	"github.com/Aswanidev-vs/cherry/render"
	"github.com/Aswanidev-vs/cherry/widget"

	"github.com/Aswanidev-vs/cherry/cell"

	"github.com/Aswanidev-vs/panda-editor/editor/cli"
	"github.com/Aswanidev-vs/panda-editor/editor/theme"
)

// draw renders one frame into an off-screen buffer; it must never panic.
func draw(t *testing.T, s *Shell) {
	t.Helper()
	scr := render.New(80, 24)
	s.Draw(&widget.DrawCtx{
		Rect:   geom.Rect{Size: geom.Size{W: 80, H: 24}},
		Screen: scr,
	})
}

func TestShellDrawAndFlows(t *testing.T) {
	app := (*cherry.App)(nil) // headless: clipboard stays nil-safe
	sh, err := New(app, Options{Args: &cli.Args{}, Version: "test"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	draw(t, sh)

	// Insert a character through the editor (no overlay): delegates to view.
	if !sh.Handle(input.KeyPress{Key: input.KeyNone, Rune: 'a'}) {
		t.Fatal("expected 'a' to be handled by the editor view")
	}
	draw(t, sh)

	// New tab, then switch tabs.
	sh.Handle(input.KeyPress{Key: input.KeyNone, Rune: 'n', Mod: input.ModCtrl})
	sh.Handle(input.KeyPress{Key: input.KeyPageDown, Mod: input.ModCtrl})
	sh.Handle(input.KeyPress{Key: input.KeyPageUp, Mod: input.ModCtrl})
	draw(t, sh)

	// Goto-line overlay: open, type "3", confirm.
	sh.Handle(input.KeyPress{Key: input.KeyNone, Rune: 'g', Mod: input.ModCtrl})
	if sh.ov != ovInput {
		t.Fatal("ctrl+g should open the goto input overlay")
	}
	sh.Handle(input.KeyPress{Key: input.KeyNone, Rune: '3'})
	sh.Handle(input.KeyPress{Key: input.KeyEnter})
	if sh.ov != ovNone {
		t.Fatal("goto overlay should close after Enter")
	}
	draw(t, sh)

	// Help popup opens and dismisses on any key.
	sh.Handle(input.KeyPress{Key: input.KeyF1})
	if sh.ov != ovHelp {
		t.Fatal("F1 should open help")
	}
	draw(t, sh)
	sh.Handle(input.KeyPress{Key: input.KeyEscape})
	if sh.ov != ovNone {
		t.Fatal("help should dismiss on keypress")
	}

	// Search overlay opens and closes.
	sh.Handle(input.KeyPress{Key: input.KeyNone, Rune: 'f', Mod: input.ModCtrl})
	if sh.ov != ovInput {
		t.Fatal("ctrl+f should open search")
	}
	sh.Handle(input.KeyPress{Key: input.KeyEscape})
	if sh.ov != ovNone {
		t.Fatal("search should cancel on Esc")
	}
	draw(t, sh)
}

func TestShellWelcomeDismiss(t *testing.T) {
	app := (*cherry.App)(nil)
	sh, err := New(app, Options{Args: &cli.Args{}, Version: "test"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if sh.ov != ovWelcome {
		t.Fatal("no files should show the welcome splash")
	}
	sh.Handle(input.KeyPress{Key: input.KeyNone, Rune: 'i'})
	if sh.ov != ovNone {
		t.Fatal("welcome should dismiss on any key")
	}
}

// The chrome used to build its styles into package-level vars at init, which
// froze whichever palette happened to be current and made a theme switch a
// silent no-op. Styles must now be resolved per frame, so switching themes and
// drawing again has to produce genuinely different cells — and switching back
// has to restore them exactly.
func TestThemeIsResolvedPerFrameNotFrozenAtInit(t *testing.T) {
	before := theme.Current
	t.Cleanup(func() { theme.Current = before })

	frame := func(p theme.Palette) cell.Style {
		theme.Current = p
		s, err := New(&cherry.App{}, Options{
			Args:    &cli.Args{Files: []cli.FileSpec{{Path: ""}}},
			Version: "test",
		})
		if err != nil {
			t.Fatalf("New: %v", err)
		}
		scr := render.New(60, 6)
		s.Draw(&widget.DrawCtx{Rect: geom.Rect{Size: geom.Size{W: 60, H: 6}}, Screen: scr})
		return scr.CellAt(30, 2).Style // the editor body
	}

	ink := frame(theme.Ink)
	vellum := frame(theme.Vellum)
	if ink.Bg == vellum.Bg {
		t.Fatalf("switching theme did not change the editor surface: both are %#v", ink.Bg)
	}
	if vellum.Bg != theme.Light.Base {
		t.Errorf("Vellum surface = %#v, want the light palette's Base %#v", vellum.Bg, theme.Light.Base)
	}
	if back := frame(theme.Ink); back != ink {
		t.Errorf("switching back gave %#v, want the original %#v", back, ink)
	}
}
