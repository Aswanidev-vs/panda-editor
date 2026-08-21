//go:build aix || linux || solaris || zos

package term

import "golang.org/x/sys/unix"

// POSIX termios ioctl requests for Linux, AIX, Solaris and z/OS. Split out of
// term_unix.go because BSD-family GOOS values lack TCGETS/TCSETS entirely;
// see term_unix_bsd.go for the other half.
const (
	ioctlReadTermios  = unix.TCGETS
	ioctlWriteTermios = unix.TCSETS
)
