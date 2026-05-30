package cmd

import (
	"bufio"
	"fmt"
	"os/exec"
	"strings"
)

// ExecCmdWithStream 执行一个 shell 命令，并通过 outputFn 回调逐行输出 stdout 内容。
// workDir 如果非空则设置命令的工作目录。
// 返回 combined stderr 作为 error 的一部分（如果命令失败）。
func ExecCmdWithStream(cmdStr, workDir string, outputFn func(line string)) error {
	cmd := exec.Command("bash", "-c", cmdStr)
	if workDir != "" {
		cmd.Dir = workDir
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("error creating stdout pipe: %v", err)
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("error creating stderr pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("error starting command: %v", err)
	}

	// 异步读取 stderr
	stderrCh := make(chan string, 100)
	go func() {
		scanner := bufio.NewScanner(stderrPipe)
		var sb strings.Builder
		for scanner.Scan() {
			line := scanner.Text()
			if sb.Len() > 0 {
				sb.WriteString("\n")
			}
			sb.WriteString(line)
			if outputFn != nil {
				outputFn(line)
			}
		}
		stderrCh <- sb.String()
		close(stderrCh)
	}()

	// 同步逐行读取 stdout 并回调
	stdoutScanner := bufio.NewScanner(stdoutPipe)
	for stdoutScanner.Scan() {
		line := stdoutScanner.Text()
		if outputFn != nil {
			outputFn(line)
		}
	}

	// 等待 stderr 读完
	stderrOutput := <-stderrCh

	if err := cmd.Wait(); err != nil {
		stderrOutput = strings.TrimSpace(stderrOutput)
		if stderrOutput != "" {
			return fmt.Errorf("error: %v, stderr: %s", err, stderrOutput)
		}
		return fmt.Errorf("error: %v", err)
	}

	return nil
}
