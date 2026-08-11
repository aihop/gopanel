package api

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const codeGitCommandTimeout = 15 * time.Second

type codeGitCappedBuffer struct {
	data      bytes.Buffer
	limit     int
	truncated bool
}

func (buffer *codeGitCappedBuffer) Write(content []byte) (int, error) {
	written := len(content)
	remaining := buffer.limit - buffer.data.Len()
	if remaining > 0 {
		if len(content) > remaining {
			content = content[:remaining]
			buffer.truncated = true
		}
		_, _ = buffer.data.Write(content)
	} else if len(content) > 0 {
		buffer.truncated = true
	}
	return written, nil
}

func runCodeGitReviewCommand(workDir string, allowDiffExit bool, outputLimit int, args ...string) (string, bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), codeGitCommandTimeout)
	defer cancel()
	commandArgs := append([]string{"-C", workDir}, args...)
	command := exec.CommandContext(ctx, "git", commandArgs...)
	command.Env = codeGitEnvironment()
	output := &codeGitCappedBuffer{limit: outputLimit}
	command.Stdout = output
	command.Stderr = output
	err := command.Run()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return "", output.truncated, errors.New("Git 操作超时")
	}
	if err != nil {
		var exitError *exec.ExitError
		if !(allowDiffExit && errors.As(err, &exitError) && exitError.ExitCode() == 1) {
			message := strings.TrimSpace(output.data.String())
			if message == "" {
				message = err.Error()
			}
			return "", output.truncated, fmt.Errorf("Git 操作失败：%s", message)
		}
	}
	return output.data.String(), output.truncated, nil
}

func parseCodeGitStatus(output string, workspacePrefix string) []codeGitFile {
	records := strings.Split(output, "\x00")
	files := make([]codeGitFile, 0, len(records))
	for index := 0; index < len(records); index++ {
		record := records[index]
		if len(record) < 4 || record[2] != ' ' {
			continue
		}
		indexStatus := string(record[0])
		worktreeStatus := string(record[1])
		filePath := record[3:]
		oldPath := ""
		if strings.Contains("RC", indexStatus) || strings.Contains("RC", worktreeStatus) {
			if index+1 < len(records) {
				index++
				oldPath = records[index]
			}
		}
		untracked := indexStatus == "?" && worktreeStatus == "?"
		workspacePath := path.Join(workspacePrefix, filepath.ToSlash(filePath))
		files = append(files, codeGitFile{
			Path: filePath, OldPath: oldPath, WorkspacePath: workspacePath,
			IndexStatus: indexStatus, WorktreeStatus: worktreeStatus,
			Staged: !untracked && indexStatus != " ", Changed: !untracked && worktreeStatus != " ", Untracked: untracked,
		})
	}
	return files
}

func parseCodeGitNumstat(output string) (int, int) {
	additions, deletions := 0, 0
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		fields := strings.SplitN(line, "\t", 3)
		if len(fields) < 3 {
			continue
		}
		added, addedErr := strconv.Atoi(fields[0])
		deleted, deletedErr := strconv.Atoi(fields[1])
		if addedErr == nil {
			additions += added
		}
		if deletedErr == nil {
			deletions += deleted
		}
	}
	return additions, deletions
}
