package cherry

import (
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/Aswanidev-vs/cherry/geom"
	"github.com/Aswanidev-vs/cherry/input"
	"github.com/Aswanidev-vs/cherry/layout"
	"github.com/Aswanidev-vs/cherry/render"
	"github.com/Aswanidev-vs/cherry/term"
	"github.com/Aswanidev-vs/cherry/widget"
)

// App owns the terminal, the event pump and the root widget. It is the only
// piece of cherry an application must construct.
//
// The loop: term bytes -> input.Pump -> routed events -> widget tree ->
// render diff flush. Rendering happens after every delivered event; the
// diffing flusher makes no-op frames cheap, so callers never invalidate by hand.
type App struct {
	tty    *term.TTY
	screen *render.Screen
	root   widget.Widget

	events chan input.Event
	done   chan struct{}
	once   sync.Once

	bindings []binding
	clipboard string // app-level clipboard mirror; also pushed to the OS via OSC 52
	lastKey   string // shown by debugging hosts; not part of the widget tree
}

type binding struct {
	spec string
	fn   func(a *App)
}

// New opens the terminal in raw mode and prepares the render surface.
func New() (*App, error) {
	tty, err := term.Open()
	if err != nil {
		return nil, fmt.Errorf("cherry: no controlling terminal: %w", err)
	}
	if err := tty.Start(); err != nil {
		tty.Restore()
		return nil, fmt.Errorf("cherry: raw mode failed: %w", err)
	}
	w, h := initialSize(tty)
	mode, syncOut := render.DetectCapabilities(os.Getenv)
	screen := render.New(w, h)
	screen.SetColorMode(mode)
	screen.SetSyncOutput(syncOut)
	return &App{
		tty:    tty,
		screen: screen,
		events: make(chan input.Event, 64),
		done:   make(chan struct{}),
	}, nil
}

// SetRoot installs the widget tree drawn every frame.
func (a *App) SetRoot(w widget.Widget) { a.root = w }

// Bind routes key presses whose canonical form equals spec (e.g. "ctrl+q",
// "esc", "?") to fn. Later bindings win; bindings fire before the widget
// tree sees the event.
func (a *App) Bind(spec string, fn func(a *App)) {
	a.bindings = append(a.bindings, binding{strings.ToLower(spec), fn})
}

// LastKey reports the canonical form of the most recent key press.
func (a *App) LastKey() string { return a.lastKey }

// Quit stops Run. Safe from any callback.
func (a *App) Quit() { a.once.Do(func() { close(a.done) }) }

// Run pumps events until Quit or terminal loss, redrawing after each one.
// Terminal state (raw mode, alt screen, mouse reporting) is always restored.
func (a *App) Run() error {
	a.enter()
	defer a.leave()

	input.Pump(a.tty, a.tty.Events(), a.events, func(err error) {
		select {
		case a.events <- input.ErrorEvent{Err: err}:
		default:
		}
	})

	a.draw()
	for {
		select {
		case <-a.done:
			return nil
		case ev := <-a.events:
			switch e := ev.(type) {
			case input.Resize:
				a.screen.Resize(e.Width, e.Height)
			case input.ErrorEvent:
				a.Quit()
			case input.KeyPress:
				a.lastKey = e.String()
				if a.dispatchKey(e) {
					break
				}
				if a.root != nil {
					a.root.Handle(e)
				}
			default:
				if a.root != nil {
					a.root.Handle(ev)
				}
			}
			a.draw()
		}
	}
}

func (a *App) dispatchKey(k input.KeyPress) bool {
	spec := strings.ToLower(k.String())
	for i := len(a.bindings) - 1; i >= 0; i-- {
		if a.bindings[i].spec == spec && a.bindings[i].fn != nil {
			a.bindings[i].fn(a)
			return true
		}
	}
	return false
}

// draw lays the root over the whole screen, flushes the diff and places the
// hardware cursor where the root asks (or hides it when it does not care).
func (a *App) draw() {
	if a.root == nil {
		return
	}
	sz := a.screen.Size()
	rects := layout.Solve(geom.Size{W: sz.W, H: sz.H}, layout.Vertical,
		[]layout.Spec{{Fill: true}})
	a.root.Draw(&widget.DrawCtx{Rect: rects[0], Screen: a.screen})
	_ = a.screen.Flush(a.tty)
	if c, ok := a.root.(widget.Cursorer); ok {
		if p, vis := c.CursorPos(); vis {
			_ = a.screen.SetCursor(a.tty, p.X, p.Y)
		} else {
			_ = a.screen.SetCursor(a.tty, -1, -1)
		}
		return
	}
	_ = a.screen.SetCursor(a.tty, -1, -1)
}

// SetClipboard stores text in the in-process clipboard and mirrors it into
// the real system clipboard via OSC 52, which most terminals honour.
func (a *App) SetClipboard(text string) {
	a.clipboard = text
	if text == "" {
		return
	}
	a.tty.Write(osc52Set(text))
}

// ClipboardText returns the last text stored via SetClipboard. Braces are
// stripped so callers can feed the result straight into an editor.
func (a *App) ClipboardText() string { return a.clipboard }

// osc52Set encodes text as an OSC 52 clipboard-set sequence: ESC ] 52 ; c ;
// <base64> BEL, which terminals honour to fill the real system clipboard.
func osc52Set(text string) []byte {
	b64 := base64.StdEncoding
	payload := make([]byte, b64.EncodedLen(len(text)))
	b64.Encode(payload, []byte(text))
	return append(append([]byte("\x1b]52;c;"), payload...), '\a')
}

const (
	seqAltEnter = "\x1b[?1049h\x1b[2J\x1b[?25l\x1b[?1000h\x1b[?1006h"
	seqAltLeave = "\x1b[?1006l\x1b[?1000l\x1b[0m\x1b[?25h\x1b[?1049l"
)

func (a *App) enter() {
	_, _ = a.tty.Write([]byte(seqAltEnter))
}

func (a *App) leave() {
	a.tty.Restore()
	_, _ = a.tty.Write([]byte(seqAltLeave))
}

func initialSize(tty *term.TTY) (w, h int) {
	if sz, err := tty.Size(); err == nil && sz.W > 0 && sz.H > 0 {
		return sz.W, sz.H
	}
	return 80, 24
}

var _ io.Writer = (*term.TTY)(nil)
