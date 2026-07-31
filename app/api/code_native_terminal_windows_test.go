//go:build windows

package api

import (
	"os/exec"
	"strings"
	"testing"

	"golang.org/x/sys/windows"
)

func TestWindowsTerminalCommandWrapsNPMShim(t *testing.T) {
	t.Setenv("COMSPEC", `C:\Windows\System32\cmd.exe`)
	command := exec.Command(`C:\Users\dev\AppData\Roaming\npm\codex.cmd`, "--version")
	executable, commandLine := windowsTerminalCommand(command)
	if executable != `C:\Windows\System32\cmd.exe` {
		t.Fatalf("unexpected command shell: %s", executable)
	}
	if !strings.Contains(commandLine, "/d /s /c") || !strings.Contains(commandLine, "codex.cmd") || !strings.Contains(commandLine, "--version") {
		t.Fatalf("unexpected command line: %s", commandLine)
	}
}

func TestWindowsEnvironmentBlockIsSortedAndTerminated(t *testing.T) {
	block, err := windowsEnvironmentBlock([]string{"Path=C:\\bin", "APP_ENV=dev"})
	if err != nil {
		t.Fatal(err)
	}
	if len(block) < 2 || block[len(block)-1] != 0 || block[len(block)-2] != 0 {
		t.Fatalf("environment block is not double terminated: %#v", block)
	}
	decoded := windows.UTF16ToString(block)
	if decoded != "APP_ENV=dev" {
		t.Fatalf("environment block is not sorted: %q", decoded)
	}
}

func TestConPTYSizeUsesTerminalDefaults(t *testing.T) {
	size := conptySize(0, 0)
	if size.X != 80 || size.Y != 24 {
		t.Fatalf("unexpected default terminal size: %#v", size)
	}
}
