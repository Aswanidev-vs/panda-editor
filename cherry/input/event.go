package input

// Event is anything the terminal delivers to the running app. Concrete types:
// KeyPress, Mouse, Paste, Focus, Resize, Interrupt, ErrorEvent.
type Event interface {
	isEvent()
}

// KeyPress is one key event. For printable input, Key is KeyNone and Rune
// carries the typed character (Mod may still flag Ctrl/Alt combos).
type KeyPress struct {
	Key  Key
	Rune rune
	Mod  Mod
}

// String renders the canonical spec form, e.g. "ctrl+shift+p", "a", "enter".
// Used for logging, help overlays and keymap matching.
func (k KeyPress) String() string {
	switch {
	case k.Key != KeyNone:
		name := k.Key.String()
		// Backtab's canonical spelling already includes shift.
		if k.Mod != 0 || k.Key == KeyBacktab {
			m := k.Mod
			if k.Key == KeyBacktab {
				m |= ModShift
			}
			return m.String() + name
		}
		return name
	case k.Rune != 0:
		r := k.Rune
		if k.Mod&ModCtrl != 0 && r >= 'a' && r <= 'z' {
			return k.Mod.String()[:len("ctrl+")] + string(r)
		}
		return k.Mod.String() + string(r)
	default:
		return ""
	}
}

// MouseAction describes what happened in a mouse event.
type MouseAction uint8

const (
	MousePress MouseAction = iota
	MouseRelease
	MouseMove // only delivered while dragging unless motion tracking is on
	MouseWheel
)

// WheelDir distinguishes scroll directions for MouseWheel actions.
type WheelDir uint8

const (
	WheelUp WheelDir = iota
	WheelDown
	WheelLeft
	WheelRight
)

// Mouse is a mouse event at character-cell resolution.
type Mouse struct {
	Action MouseAction
	Button int    // 0=left 1=middle 2=right; wheel events carry WheelDir in Button via Wheel field
	Wheel  WheelDir
	X, Y   int
	Mod    Mod
}

func (Mouse) isEvent() {}

// Paste carries bracketed-paste text from the terminal.
type Paste struct{ Text string }

func (Paste) isEvent() {}

// Focus reports terminal window focus changes (if supported).
type Focus struct{ Gained bool }

func (Focus) isEvent() {}

// Resize announces a new terminal size in cells.
type Resize struct{ Width, Height int }

func (Resize) isEvent() {}

// Interrupt is delivered when the terminal sends SIGINT / ctrl+c in raw mode.
type Interrupt struct{}

func (Interrupt) isEvent() {}

// ErrorEvent wraps an I/O failure from the driver; the app should decide
// whether to continue or shut down.
type ErrorEvent struct{ Err error }

func (ErrorEvent) isEvent() {}

func (KeyPress) isEvent() {}
