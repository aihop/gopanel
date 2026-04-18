package terminal

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/aihop/gopanel/global"
	"github.com/creack/pty"
)

const (
	DefaultCloseSignal  = syscall.SIGINT
	DefaultCloseTimeout = 10 * time.Second
)

type LocalCommand struct {
	closeSignal  syscall.Signal
	closeTimeout time.Duration

	cmd *exec.Cmd
	pty *os.File
}

func NewCommand(initCmd []string) (*LocalCommand, error) {
	cmd := exec.Command("docker", initCmd...)
	if term := os.Getenv("TERM"); term != "" {
		cmd.Env = append(os.Environ(), "TERM="+term)
	} else {
		cmd.Env = append(os.Environ(), "TERM=xterm")
	}

	pty, err := pty.Start(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to start command: %w", err)
	}

	lcmd := &LocalCommand{
		closeSignal:  DefaultCloseSignal,
		closeTimeout: DefaultCloseTimeout,
		cmd:          cmd,
		pty:          pty,
	}

	return lcmd, nil
}

func (lcmd *LocalCommand) Read(p []byte) (n int, err error) {
	return lcmd.pty.Read(p)
}

func (lcmd *LocalCommand) Write(p []byte) (n int, err error) {
	return lcmd.pty.Write(p)
}

func (lcmd *LocalCommand) Close() error {
	if lcmd.cmd != nil && lcmd.cmd.Process != nil {
		_ = lcmd.cmd.Process.Kill()
	}
	_ = lcmd.pty.Close()
	return nil
}

func (lcmd *LocalCommand) Wait(quitChan chan bool) {
	if err := lcmd.cmd.Wait(); err != nil {
		global.LOG.Errorf("ssh session wait failed, err: %v", err)
		setQuit(quitChan)
	}
}
