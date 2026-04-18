//go:build windows
// +build windows

package terminal

import "errors"

// ResizeTerminal stub on windows
func (lcmd *LocalCommand) ResizeTerminal(width int, height int) error {
	return errors.New("ResizeTerminal: not supported on windows")
}
