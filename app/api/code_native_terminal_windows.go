//go:build windows

package api

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
)

type windowsNativeTerminal struct {
	input     *os.File
	output    *os.File
	console   windows.Handle
	closeOnce sync.Once
}

func nativeTerminalPlatformSupported() bool {
	major, _, build := windows.RtlGetNtVersionNumbers()
	return major > 10 || major == 10 && build >= 17763
}

func startNativeTerminal(command *exec.Cmd, cols, rows uint16) (nativeTerminal, error) {
	if command == nil || strings.TrimSpace(command.Path) == "" {
		return nil, errors.New("terminal command is empty")
	}
	inputRead, inputWrite, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	outputRead, outputWrite, err := os.Pipe()
	if err != nil {
		_ = inputRead.Close()
		_ = inputWrite.Close()
		return nil, err
	}
	closePipes := func() {
		_ = inputRead.Close()
		_ = inputWrite.Close()
		_ = outputRead.Close()
		_ = outputWrite.Close()
	}
	var console windows.Handle
	if err := windows.CreatePseudoConsole(conptySize(cols, rows), windows.Handle(inputRead.Fd()), windows.Handle(outputWrite.Fd()), 0, &console); err != nil {
		closePipes()
		return nil, err
	}
	attributeList, err := windows.NewProcThreadAttributeList(1)
	if err != nil {
		windows.ClosePseudoConsole(console)
		closePipes()
		return nil, err
	}
	defer attributeList.Delete()
	if err := attributeList.Update(windows.PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE, unsafe.Pointer(&console), unsafe.Sizeof(console)); err != nil {
		windows.ClosePseudoConsole(console)
		closePipes()
		return nil, err
	}
	executable, commandLine := windowsTerminalCommand(command)
	executableUTF16, err := windows.UTF16PtrFromString(executable)
	if err != nil {
		windows.ClosePseudoConsole(console)
		closePipes()
		return nil, err
	}
	commandLineUTF16, err := windows.UTF16PtrFromString(commandLine)
	if err != nil {
		windows.ClosePseudoConsole(console)
		closePipes()
		return nil, err
	}
	currentDirUTF16, err := optionalWindowsString(command.Dir)
	if err != nil {
		windows.ClosePseudoConsole(console)
		closePipes()
		return nil, err
	}
	environment, err := windowsEnvironmentBlock(command.Env)
	if err != nil {
		windows.ClosePseudoConsole(console)
		closePipes()
		return nil, err
	}
	startupInfo := windows.StartupInfoEx{
		StartupInfo:             windows.StartupInfo{Cb: uint32(unsafe.Sizeof(windows.StartupInfoEx{}))},
		ProcThreadAttributeList: attributeList.List(),
	}
	processInfo := windows.ProcessInformation{}
	err = windows.CreateProcess(
		executableUTF16,
		commandLineUTF16,
		nil,
		nil,
		false,
		windows.EXTENDED_STARTUPINFO_PRESENT|windows.CREATE_UNICODE_ENVIRONMENT,
		&environment[0],
		currentDirUTF16,
		&startupInfo.StartupInfo,
		&processInfo,
	)
	if err != nil {
		windows.ClosePseudoConsole(console)
		closePipes()
		return nil, err
	}
	command.Process, err = os.FindProcess(int(processInfo.ProcessId))
	_ = windows.CloseHandle(processInfo.Thread)
	_ = windows.CloseHandle(processInfo.Process)
	if err != nil {
		windows.ClosePseudoConsole(console)
		closePipes()
		return nil, err
	}
	_ = inputRead.Close()
	_ = outputWrite.Close()
	return &windowsNativeTerminal{input: inputWrite, output: outputRead, console: console}, nil
}

func windowsTerminalCommand(command *exec.Cmd) (string, string) {
	arguments := append([]string(nil), command.Args...)
	if len(arguments) == 0 {
		arguments = []string{command.Path}
	} else {
		arguments[0] = command.Path
	}
	extension := strings.ToLower(filepath.Ext(command.Path))
	if extension != ".cmd" && extension != ".bat" {
		return command.Path, windows.ComposeCommandLine(arguments)
	}
	commandShell := strings.TrimSpace(os.Getenv("COMSPEC"))
	if commandShell == "" {
		commandShell = filepath.Join(os.Getenv("SystemRoot"), "System32", "cmd.exe")
	}
	payload := windows.ComposeCommandLine(arguments)
	prefix := windows.ComposeCommandLine([]string{commandShell, "/d", "/s", "/c"})
	return commandShell, prefix + " \"" + payload + "\""
}

func optionalWindowsString(value string) (*uint16, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	return windows.UTF16PtrFromString(value)
}

func windowsEnvironmentBlock(environment []string) ([]uint16, error) {
	if len(environment) == 0 {
		environment = os.Environ()
	}
	environment = append([]string(nil), environment...)
	sort.Slice(environment, func(left, right int) bool {
		return strings.ToLower(environment[left]) < strings.ToLower(environment[right])
	})
	block := make([]uint16, 0, len(environment)*2)
	for _, item := range environment {
		if strings.IndexByte(item, 0) >= 0 {
			return nil, errors.New("environment contains NUL")
		}
		block = append(block, utf16.Encode([]rune(item))...)
		block = append(block, 0)
	}
	return append(block, 0), nil
}

func conptySize(cols, rows uint16) windows.Coord {
	if cols == 0 {
		cols = 80
	}
	if rows == 0 {
		rows = 24
	}
	return windows.Coord{X: int16(cols), Y: int16(rows)}
}

func (terminal *windowsNativeTerminal) Read(buffer []byte) (int, error) {
	return terminal.output.Read(buffer)
}

func (terminal *windowsNativeTerminal) Write(buffer []byte) (int, error) {
	return terminal.input.Write(buffer)
}

func (terminal *windowsNativeTerminal) Resize(cols, rows uint16) error {
	return windows.ResizePseudoConsole(terminal.console, conptySize(cols, rows))
}

func (terminal *windowsNativeTerminal) Close() error {
	var closeErr error
	terminal.closeOnce.Do(func() {
		if err := terminal.input.Close(); err != nil {
			closeErr = err
		}
		windows.ClosePseudoConsole(terminal.console)
		if err := terminal.output.Close(); closeErr == nil && err != nil {
			closeErr = err
		}
	})
	return closeErr
}
