package shell

import (
	"testing"

	"github.com/Aswanidev-vs/cherry"
	"github.com/Aswanidev-vs/cherry/geom"
	"github.com/Aswanidev-vs/cherry/input"
	"github.com/Aswanidev-vs/cherry/render"
	"github.com/Aswanidev-vs/cherry/widget"

	"github.com/Aswanidev-vs/panda-editor/editor/cli"
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
