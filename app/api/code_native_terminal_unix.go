//go:build !windows

package api

import (
	"os"
	"os/exec"

	"github.com/creack/pty"
)

type unixNativeTerminal struct {
	*os.File
}

func nativeTerminalPlatformSupported() bool {
	return true
}

func startNativeTerminal(command *exec.Cmd, cols, rows uint16) (nativeTerminal, error) {
	terminal, err := pty.StartWithSize(command, &pty.Winsize{Cols: cols, Rows: rows})
	if err != nil {
		return nil, err
	}
	return &unixNativeTerminal{File: terminal}, nil
}

func (terminal *unixNativeTerminal) Resize(cols, rows uint16) error {
	return pty.Setsize(terminal.File, &pty.Winsize{Cols: cols, Rows: rows})
}
