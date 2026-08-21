//go:build darwin || ios || dragonfly || freebsd || netbsd || openbsd

package term

import "golang.org/x/sys/unix"

// BSD-family termios ioctl requests. These cannot live in term_unix.go:
// the constants only exist per-GOOS and IoctlGetTermios's request parameter
// is int here but uint on Linux, so both sides need their own declaration.
const (
	ioctlReadTermios  = unix.TIOCGETA
	ioctlWriteTermios = unix.TIOCSETA
)
