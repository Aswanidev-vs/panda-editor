//go:build windows

package term

import (
	"fmt"
	"os"
	"time"

	"github.com/Aswanidev-vs/cherry/geom"
	"github.com/Aswanidev-vs/cherry/input"
	"golang.org/x/sys/windows"
)

// resizePollInterval is how often the watcher samples the console size;
// Windows has no SIGWINCH equivalent reachable from a plain ReadFile loop.
const resizePollInterval = 400 * time.Millisecond

type platformTTY struct {
	in, out           windows.Handle
	savedIn, savedOut uint32
}

func newPlatform(in, out *os.File) (platformTTY, error) {
	p := platformTTY{
		in:  windows.Handle(in.Fd()),
		out: windows.Handle(out.Fd()),
	}
	if err := windows.GetConsoleMode(p.in, &p.savedIn); err != nil {
		return platformTTY{}, fmt.Errorf("%w: stdin: %v", ErrNotATerminal, err)
	}
	if err := windows.GetConsoleMode(p.out, &p.savedOut); err != nil {
		return platformTTY{}, fmt.Errorf("%w: stdout: %v", ErrNotATerminal, err)
	}
	return p, nil
}

// enterRaw switches stdin to VT-aware unbuffered input and stdout to VT
// output. Consoles that predate the VT flags reject the new bits; those fall
// back to cooked-off input and clear t.vtSupported so callers can degrade
// instead of rendering garbage escapes. Caller (Start) holds t.mu.
func (t *TTY) enterRaw() error {
	p := &t.plat

	rawIn := p.savedIn&^(windows.ENABLE_LINE_INPUT|windows.ENABLE_ECHO_INPUT|
		windows.ENABLE_PROCESSED_INPUT|windows.ENABLE_MOUSE_INPUT|windows.ENABLE_WINDOW_INPUT|
		windows.ENABLE_QUICK_EDIT_MODE|windows.ENABLE_INSERT_MODE) |
		windows.ENABLE_EXTENDED_FLAGS | windows.ENABLE_VIRTUAL_TERMINAL_INPUT
	vtIn := setModeSticks(p.in, rawIn, windows.ENABLE_VIRTUAL_TERMINAL_INPUT)
	if !vtIn {
		// Legacy fallback: still disable line editing/echo/processing so keys
		// arrive unbuffered (and ^C won't kill the process), but escape
		// sequences will reach Read untranslated.
		legacy := p.savedIn &^ (windows.ENABLE_LINE_INPUT |
			windows.ENABLE_ECHO_INPUT | windows.ENABLE_PROCESSED_INPUT)
		if err := windows.SetConsoleMode(p.in, legacy); err != nil {
			return fmt.Errorf("term: SetConsoleMode(stdin): %w", err)
		}
	}

	// MOUSE/WINDOW records are discarded by ReadFile anyway; disabling their
	// capture lets VT mouse sequences flow as bytes once the renderer opts in,
	// and clearing QUICK_EDIT stops click-selections from freezing output.
	rawOut := p.savedOut | windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING |
		windows.DISABLE_NEWLINE_AUTO_RETURN
	vtOut := setModeSticks(p.out, rawOut, windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING)
	if !vtOut {
		// Pre-1607 Windows 10 builds know VT processing but not DNAR.
		vtOut = setModeSticks(p.out, p.savedOut|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING,
			windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING)
	}

	// Caller (Start) holds t.mu, so vtSupported needs no extra locking here.
	t.vtSupported = vtIn && vtOut
	return nil
}

// setModeSticks applies mode and verifies the console actually kept bit:
// some console hosts accept unknown flags without honoring them.
func setModeSticks(h windows.Handle, mode, bit uint32) bool {
	if err := windows.SetConsoleMode(h, mode); err != nil {
		return false
	}
	var got uint32
	if err := windows.GetConsoleMode(h, &got); err != nil {
		return false
	}
	return got&bit != 0
}

func (t *TTY) restoreRaw() {
	p := &t.plat
	_ = windows.SetConsoleMode(p.in, p.savedIn)
	_ = windows.SetConsoleMode(p.out, p.savedOut)
}

func (t *TTY) size() (geom.Size, error) {
	var csbi windows.ConsoleScreenBufferInfo
	if err := windows.GetConsoleScreenBufferInfo(t.plat.out, &csbi); err != nil {
		return geom.Size{}, err
	}
	return geom.Size{
		W: int(csbi.Window.Right - csbi.Window.Left + 1),
		H: int(csbi.Window.Bottom - csbi.Window.Top + 1),
	}, nil
}

func (t *TTY) watchResizes(stop <-chan struct{}) {
	last, err := t.size()
	if err != nil {
		last = geom.Size{}
	}
	ticker := time.NewTicker(resizePollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			cur, err := t.size()
			if err != nil || cur == last {
				continue
			}
			last = cur
			t.sendEvent(input.Resize{Width: cur.W, Height: cur.H}, stop)
		}
	}
}
