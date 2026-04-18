package files

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
)

func IsSymlink(mode os.FileMode) bool {
	return mode&os.ModeSymlink != 0
}

func IsBlockDevice(mode os.FileMode) bool {
	return mode&os.ModeDevice != 0 && mode&os.ModeCharDevice == 0
}

func GetMimeType(path string) string {
	file, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer file.Close()

	buffer := make([]byte, 512)
	_, err = file.Read(buffer)
	if err != nil {
		return ""
	}
	mimeType := http.DetectContentType(buffer)
	return mimeType
}

func GetSymlink(path string) string {
	linkPath, err := os.Readlink(path)
	if err != nil {
		return ""
	}
	return linkPath
}

func GetUsername(uid int) string {
	usr, err := user.LookupId(strconv.Itoa(uid))
	if err != nil {
		return ""
	}
	return usr.Username
}

func GetGroup(gid int) string {
	usr, err := user.LookupGroupId(strconv.Itoa(gid))
	if err != nil {
		return ""
	}
	return usr.Name
}

const dotCharacter = 46

func IsHidden(path string) bool {
	return path[0] == dotCharacter
}

func countLines(path string) (int, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()
	reader := bufio.NewReader(file)
	count := 0
	for {
		_, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				if count > 0 {
					count++
				}
				return count, nil
			}
			return count, err
		}
		count++
	}
}

func ReadFileByLine(filename string, page, pageSize int, latest bool) (lines []string, isEndOfFile bool, total int, err error) {
	if !NewFileOp().Stat(filename) {
		return
	}
	file, err := os.Open(filename)
	if err != nil {
		return
	}
	defer file.Close()

	totalLines, err := countLines(filename)
	if err != nil {
		return
	}
	total = (totalLines + pageSize - 1) / pageSize
	reader := bufio.NewReaderSize(file, 8192)

	if latest {
		page = total
	}
	currentLine := 0
	startLine := (page - 1) * pageSize
	endLine := startLine + pageSize

	for {
		line, _, err := reader.ReadLine()
		if err == io.EOF {
			break
		}
		if currentLine >= startLine && currentLine < endLine {
			lines = append(lines, string(line))
		}
		currentLine++
		if currentLine >= endLine {
			break
		}
	}

	isEndOfFile = currentLine < endLine
	return
}

func GetParentMode(path string) (os.FileMode, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return 0, err
	}

	for {
		fileInfo, err := os.Stat(absPath)
		if err == nil {
			return fileInfo.Mode() & os.ModePerm, nil
		}
		if !os.IsNotExist(err) {
			return 0, err
		}

		parentDir := filepath.Dir(absPath)
		if parentDir == absPath {
			return 0, fmt.Errorf("no existing directory found in the path: %s", path)
		}
		absPath = parentDir
	}
}

func IsInvalidChar(name string) bool {
	return strings.Contains(name, "&")
}

func IsEmptyDir(dir string) bool {
	f, err := os.Open(dir)
	if err != nil {
		return false
	}
	defer f.Close()
	_, err = f.Readdirnames(1)
	return err == io.EOF
}

// CopyDirContents 复制源目录内容到目标目录（不包括源目录本身）
func CopyDirContents(src, dst string) error {
	// 确保目标目录存在
	if err := os.MkdirAll(dst, 0755); err != nil {
		return fmt.Errorf("创建目标目录失败: %w", err)
	}

	// 遍历源目录
	entries, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("读取源目录失败: %w", err)
	}

	filesUtil := NewFileOp()

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		if err = filesUtil.Copy(srcPath, dst); err != nil {
			return fmt.Errorf("copy dir %s failed: %w", srcPath, err)
		}
	}

	return nil
}
