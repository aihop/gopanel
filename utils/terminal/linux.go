//go:build !windows
// +build !windows

package terminal

import (
	"golang.org/x/sys/unix"
)

// ResizeTerminal for Unix-like systems
func (lcmd *LocalCommand) ResizeTerminal(width int, height int) error {
	ws := &unix.Winsize{
		Row:    uint16(height),
		Col:    uint16(width),
		Xpixel: 0,
		Ypixel: 0,
	}
	return unix.IoctlSetWinsize(int(lcmd.pty.Fd()), unix.TIOCSWINSZ, ws)
}
