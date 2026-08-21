//go:build !windows

package term

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/Aswanidev-vs/cherry/geom"
	"github.com/Aswanidev-vs/cherry/input"
	"golang.org/x/sys/unix"
)

// platformTTY holds the saved line-discipline state. Unix termios belongs to
// the tty device rather than individual fds, so stdin's settings govern
// stdout too and capturing them once covers both streams.
type platformTTY struct {
	saved unix.Termios
}

func newPlatform(in, out *os.File) (platformTTY, error) {
	saved, err := unix.IoctlGetTermios(int(in.Fd()), ioctlReadTermios)
	if err != nil {
		return platformTTY{}, fmt.Errorf("%w: stdin: %v", ErrNotATerminal, err)
	}
	if _, err := unix.IoctlGetTermios(int(out.Fd()), ioctlReadTermios); err != nil {
		return platformTTY{}, fmt.Errorf("%w: stdout: %v", ErrNotATerminal, err)
	}
	return platformTTY{saved: *saved}, nil
}

// enterRaw applies cfmakeraw semantics to the saved line discipline;
// caller (Start) holds t.mu.
func (t *TTY) enterRaw() error {
	raw := t.plat.saved
	// ISIG off is what makes ^C arrive in Read as a 0x03 byte for the input
	// parser instead of raising SIGINT.
	raw.Iflag &^= unix.IGNBRK | unix.BRKINT | unix.PARMRK | unix.ISTRIP |
		unix.INLCR | unix.IGNCR | unix.ICRNL | unix.IXON
	raw.Oflag &^= unix.OPOST
	raw.Lflag &^= unix.ECHO | unix.ECHONL | unix.ICANON | unix.ISIG | unix.IEXTEN
	raw.Cflag &^= unix.CSIZE | unix.PARENB
	raw.Cflag |= unix.CS8
	raw.Cc[unix.VMIN] = 1
	raw.Cc[unix.VTIME] = 0
	return unix.IoctlSetTermios(int(os.Stdin.Fd()), ioctlWriteTermios, &raw)
}

func (t *TTY) restoreRaw() {
	// Best effort by contract (Restore has no error return); worst case the
	// shell's own reset or process exit fixes the line discipline.
	_ = unix.IoctlSetTermios(int(os.Stdin.Fd()), ioctlWriteTermios, &t.plat.saved)
}

func (t *TTY) size() (geom.Size, error) {
	ws, err := unix.IoctlGetWinsize(int(os.Stdout.Fd()), unix.TIOCGWINSZ)
	if err != nil {
		return geom.Size{}, err
	}
	return geom.Size{W: int(ws.Col), H: int(ws.Row)}, nil
}

func (t *TTY) watchResizes(stop <-chan struct{}) {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGWINCH)
	defer signal.Stop(sig)
	for {
		select {
		case <-stop:
			return
		case <-sig:
			// SIGWINCH coalesces bursts; querying Size now always yields the
			// freshest dimensions even if several resizes were collapsed.
			if sz, err := t.size(); err == nil {
				t.sendEvent(input.Resize{Width: sz.W, Height: sz.H}, stop)
			}
		}
	}
}
