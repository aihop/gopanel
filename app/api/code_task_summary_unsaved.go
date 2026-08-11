package api

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func applyCodeTaskUnsavedStats(summary *codeTaskSummary, stats codeTaskDiffStats) {
	if stats.Files == 0 {
		return
	}
	summary.HasUnsavedChanges = true
	summary.UnsavedAdditions += stats.Additions
	summary.UnsavedDeletions += stats.Deletions
	summary.UnsavedFiles += stats.Files
}

func loadCodeTaskUnsavedStats(root string) codeTaskDiffStats {
	root = filepath.Clean(strings.TrimSpace(root))
	if root == "." || !isCodeGitWorktree(root) {
		return codeTaskDiffStats{}
	}
	status, err := runCodeGitBytes(root, nil, "status", "--porcelain=v1", "-z", "--untracked-files=all")
	if err != nil || len(status) == 0 {
		return codeTaskDiffStats{}
	}
	files, untracked := codeTaskUnsavedFiles(parseCodeGitStatus(string(status), ""))
	if len(files) == 0 {
		return codeTaskDiffStats{}
	}
	args := append([]string{"diff", "HEAD", "--numstat", "--no-renames"}, codeTaskDiffPathspec()...)
	tracked, _ := runCodeGit(root, args...)
	additions, deletions := parseCodeGitNumstat(tracked)
	stats := codeTaskDiffStats{Additions: additions, Deletions: deletions, Files: len(files)}
	for _, filePath := range untracked {
		stats.Additions += countCodeTaskUntrackedLines(filepath.Join(root, filePath))
	}
	return stats
}

func codeTaskUnsavedFiles(status []codeGitFile) (map[string]struct{}, []string) {
	files := make(map[string]struct{})
	untracked := make([]string, 0)
	for _, file := range status {
		filePath := filepath.Clean(file.Path)
		if filePath == "." || isCodeTaskDiffExcludedFile(filePath) {
			continue
		}
		files[filePath] = struct{}{}
		if file.Untracked {
			untracked = append(untracked, filePath)
		}
	}
	return files, untracked
}

func isCodeTaskDiffExcludedFile(filePath string) bool {
	name := filepath.Base(filePath)
	for _, excluded := range codeTaskDiffExcludedFiles {
		if name == excluded {
			return true
		}
	}
	return false
}

func countCodeTaskUntrackedLines(filePath string) int {
	info, err := os.Lstat(filePath)
	if err != nil || info.IsDir() {
		return 0
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return 1
	}
	if !info.Mode().IsRegular() {
		return 0
	}
	file, err := os.Open(filePath)
	if err != nil {
		return 0
	}
	defer file.Close()
	buffer := make([]byte, 32*1024)
	lines, total := 0, 0
	lastNewline := false
	for {
		read, readErr := file.Read(buffer)
		if read > 0 {
			chunk := buffer[:read]
			if bytes.IndexByte(chunk, 0) >= 0 {
				return 0
			}
			lines += bytes.Count(chunk, []byte{'\n'})
			total += read
			lastNewline = chunk[read-1] == '\n'
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return 0
		}
	}
	if total > 0 && !lastNewline {
		lines++
	}
	return lines
}
