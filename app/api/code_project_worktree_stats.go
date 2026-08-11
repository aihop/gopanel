package api

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type codeProjectWorktreeStats struct {
	Additions int
	Deletions int
	Files     int
}

func loadCodeProjectWorktreeStats(root string) codeProjectWorktreeStats {
	status, err := runCodeGit(root, "status", "--porcelain=v1", "-z", "--untracked-files=all")
	if err != nil || status == "" {
		return codeProjectWorktreeStats{}
	}
	files := codeProjectUnsavedFiles(status)
	working, _ := runCodeGit(root, "diff", "--numstat", "--no-renames")
	staged, _ := runCodeGit(root, "diff", "--cached", "--numstat", "--no-renames")
	workingAdditions, workingDeletions := parseCodeGitNumstat(working)
	stagedAdditions, stagedDeletions := parseCodeGitNumstat(staged)
	stats := codeProjectWorktreeStats{
		Additions: workingAdditions + stagedAdditions,
		Deletions: workingDeletions + stagedDeletions,
		Files:     len(files),
	}
	untracked, _ := runCodeGit(root, "ls-files", "--others", "--exclude-standard", "-z")
	for _, filePath := range strings.Split(untracked, "\x00") {
		filePath = filepath.Clean(strings.TrimSpace(filePath))
		if filePath != "." {
			stats.Additions += countCodeProjectUntrackedLines(filepath.Join(root, filePath))
		}
	}
	return stats
}

func codeProjectUnsavedFiles(status string) map[string]struct{} {
	files := make(map[string]struct{})
	records := strings.Split(status, "\x00")
	for index := 0; index < len(records); index++ {
		record := records[index]
		if len(record) < 4 {
			continue
		}
		filePath := filepath.Clean(record[3:])
		if filePath != "." {
			files[filePath] = struct{}{}
		}
		if record[0] == 'R' || record[0] == 'C' || record[1] == 'R' || record[1] == 'C' {
			index++
		}
	}
	return files
}

func countCodeProjectUntrackedLines(filePath string) int {
	info, err := os.Lstat(filePath)
	if err != nil || info.IsDir() {
		return 0
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return 1
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
