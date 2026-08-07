package api

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const maxCodeCommitFileBytes = maxCodeSnapshotFileBytes

var codeSensitiveCommitNames = map[string]struct{}{
	".env": {}, ".netrc": {}, ".npmrc": {}, "credentials.json": {},
	"id_dsa": {}, "id_ecdsa": {}, "id_ed25519": {}, "id_rsa": {},
}

var codeSensitiveCommitExtensions = map[string]struct{}{
	".key": {}, ".p12": {}, ".pem": {}, ".pfx": {},
}

var codePrivateKeyMarkers = [][]byte{
	[]byte("-----BEGIN PRIVATE KEY-----"),
	[]byte("-----BEGIN RSA PRIVATE KEY-----"),
	[]byte("-----BEGIN EC PRIVATE KEY-----"),
	[]byte("-----BEGIN OPENSSH PRIVATE KEY-----"),
}

func validateCodeGitSaveFiles(workDir string) error {
	paths, err := codeGitChangedFilePaths(workDir)
	if err != nil {
		return err
	}
	for _, relativePath := range paths {
		if err := validateCodeGitSaveFile(workDir, relativePath); err != nil {
			return err
		}
	}
	return nil
}

func validateCodeGitStagedChanges(workDir string) error {
	if _, err := runCodeGit(workDir, "diff", "--cached", "--check"); err != nil {
		return fmt.Errorf("暂存区存在未解决的冲突标记或空白错误，请修复后再提交：%w", err)
	}
	return nil
}

func codeGitChangedFilePaths(workDir string) ([]string, error) {
	changed, err := runCodeGitBytes(workDir, nil, "diff", "--name-only", "--diff-filter=ACMR", "-z", "HEAD")
	if err != nil {
		return nil, err
	}
	untracked, err := runCodeGitBytes(workDir, nil, "ls-files", "--others", "--exclude-standard", "-z")
	if err != nil {
		return nil, err
	}
	seen := make(map[string]struct{})
	for _, output := range [][]byte{changed, untracked} {
		for _, rawPath := range bytes.Split(output, []byte{0}) {
			relativePath := strings.TrimSpace(string(rawPath))
			if relativePath != "" {
				seen[filepath.Clean(relativePath)] = struct{}{}
			}
		}
	}
	paths := make([]string, 0, len(seen))
	for path := range seen {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths, nil
}

func validateCodeGitSaveFile(workDir, relativePath string) error {
	baseName := strings.ToLower(filepath.Base(relativePath))
	if codeSensitiveEnvironmentFile(baseName) {
		return fmt.Errorf("拒绝提交敏感配置文件：%s", relativePath)
	}
	if _, blocked := codeSensitiveCommitNames[baseName]; blocked {
		return fmt.Errorf("拒绝提交敏感凭据文件：%s", relativePath)
	}
	if _, blocked := codeSensitiveCommitExtensions[strings.ToLower(filepath.Ext(baseName))]; blocked {
		return fmt.Errorf("拒绝提交私钥或证书凭据文件：%s", relativePath)
	}
	path := filepath.Join(workDir, relativePath)
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return nil
	}
	if info.Size() > maxCodeCommitFileBytes {
		return fmt.Errorf("拒绝提交超大文件 %s：大小超过 %d MiB", relativePath, maxCodeCommitFileBytes>>20)
	}
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	prefix, err := io.ReadAll(io.LimitReader(file, 1024*1024))
	if err != nil {
		return err
	}
	for _, marker := range codePrivateKeyMarkers {
		if bytes.Contains(prefix, marker) {
			return fmt.Errorf("拒绝提交包含私钥内容的文件：%s", relativePath)
		}
	}
	return nil
}

func codeSensitiveEnvironmentFile(baseName string) bool {
	if baseName == ".env.example" || baseName == ".env.sample" || baseName == ".env.template" {
		return false
	}
	return baseName == ".env" || strings.HasPrefix(baseName, ".env.")
}
