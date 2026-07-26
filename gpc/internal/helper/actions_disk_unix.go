//go:build !windows

package helper

import (
	"container/heap"
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"
)

// 伪文件系统一律不进：里面没有真实磁盘占用，读它们只会浪费时间甚至阻塞
var diskAlwaysSkip = []string{"/proc", "/sys", "/dev", "/run"}

const (
	diskScanDefaultMinSize int64 = 100 << 20
	diskScanDefaultTopN          = 200
	diskScanMaxTopN              = 1000
	diskScanTimeout              = 10 * time.Minute
)

type diskScanFile struct {
	Path    string    `json:"path"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"modTime"`
}

type diskScanResult struct {
	ScanID       string         `json:"scanId"`
	Roots        []string       `json:"roots"`
	MinSize      int64          `json:"minSize"`
	Files        []diskScanFile `json:"files"`
	ScannedFiles int64          `json:"scannedFiles"`
	ScannedBytes int64          `json:"scannedBytes"`
	Errors       int64          `json:"errors"`
	ExpiresIn    int            `json:"expiresIn"` // 秒
}

// actionDiskScan 只读扫描，天生允许走全盘——它不修改任何东西，风险等同于 FILE_LIST。
// 危险的是删除，删除由 checkScanGrantedPath 用这里产出的 scanId 约束。
func (s *Server) actionDiskScan(ctx context.Context, params map[string]interface{}) (string, error) {
	roots := getStringSlice(params, "roots")
	if len(roots) == 0 {
		roots = []string{"/"}
	}
	minSize := diskScanDefaultMinSize
	if v, ok := getInt(params, "minSize"); ok && v > 0 {
		minSize = int64(v)
	}
	topN := diskScanDefaultTopN
	if v, ok := getInt(params, "topN"); ok && v > 0 {
		topN = v
	}
	if topN > diskScanMaxTopN {
		topN = diskScanMaxTopN
	}
	crossDevice := getBool(params, "crossDevice")

	scanCtx, cancel := context.WithTimeout(ctx, diskScanTimeout)
	defer cancel()

	h := &diskFileHeap{limit: topN}
	heap.Init(h)
	var scannedFiles, scannedBytes, errCount int64

	for _, root := range roots {
		root = filepath.Clean(strings.TrimSpace(root))
		if root == "" || !filepath.IsAbs(root) {
			continue
		}
		rootDev, devOK := diskDeviceOf(root)
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if cerr := scanCtx.Err(); cerr != nil {
				return cerr
			}
			if err != nil {
				errCount++
				if d != nil && d.IsDir() {
					return fs.SkipDir
				}
				return nil
			}
			if d.IsDir() {
				for _, skip := range diskAlwaysSkip {
					if path == skip || strings.HasPrefix(path, skip+string(os.PathSeparator)) {
						return fs.SkipDir
					}
				}
				if !crossDevice && devOK && path != root {
					if dev, ok := diskDeviceOf(path); ok && dev != rootDev {
						return fs.SkipDir
					}
				}
				return nil
			}
			if !d.Type().IsRegular() {
				return nil
			}
			info, ierr := d.Info()
			if ierr != nil {
				errCount++
				return nil
			}
			scannedFiles++
			scannedBytes += info.Size()
			if info.Size() >= minSize {
				h.push(diskScanFile{Path: path, Size: info.Size(), ModTime: info.ModTime()})
			}
			return nil
		})
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return "", err
			}
			errCount++
		}
	}

	files := h.sorted()
	// 只把达到授权门槛的路径写进 store：低于门槛的即使在结果里也永远删不掉，
	// 记进去只会白占内存
	granted := make([]string, 0, len(files))
	for _, f := range files {
		if f.Size >= diskGrantMinSize && !s.isDiskProtectedPath(f.Path) {
			granted = append(granted, f.Path)
		}
	}
	scanID, err := diskScanStore.save(granted)
	if err != nil {
		return "", err
	}

	out := diskScanResult{
		ScanID:       scanID,
		Roots:        roots,
		MinSize:      minSize,
		Files:        files,
		ScannedFiles: scannedFiles,
		ScannedBytes: scannedBytes,
		Errors:       errCount,
		ExpiresIn:    int(diskScanTTL.Seconds()),
	}
	b, err := json.Marshal(out)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// actionFileTruncate 把文件截断为 0，用于清理正在被写入的日志。
// 对日志来说 truncate 比 rm 正确得多：rm 掉正在被进程持有的文件，
// inode 不会释放，磁盘空间根本不会回来，还得重启进程。
func (s *Server) actionFileTruncate(ctx context.Context, params map[string]interface{}) (string, error) {
	_ = ctx
	p := getString(params, "path")
	if p == "" {
		return "", errors.New("invalid params: path is empty")
	}
	scanID := getString(params, "scanId")

	var abs string
	var err error
	if scanID != "" {
		abs, err = s.checkScanGrantedPath(scanID, p)
	} else {
		abs, err = s.cleanAndCheckPath(p, true)
		if err == nil && s.isDiskProtectedPath(abs) {
			err = errPathProtected
		}
	}
	if err != nil {
		return "", err
	}
	if !diskIsRegularFile(abs) {
		return "", errNotRegularFile
	}
	if err := os.Truncate(abs, 0); err != nil {
		return "", err
	}
	return "ok", nil
}

func diskIsRegularFile(path string) bool {
	fi, err := os.Lstat(path)
	if err != nil {
		return false
	}
	return fi.Mode().IsRegular()
}

func diskDeviceOf(path string) (uint64, bool) {
	fi, err := os.Lstat(path)
	if err != nil {
		return 0, false
	}
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, false
	}
	return uint64(st.Dev), true
}

type diskFileHeap struct {
	items []diskScanFile
	limit int
}

func (h diskFileHeap) Len() int            { return len(h.items) }
func (h diskFileHeap) Less(i, j int) bool  { return h.items[i].Size < h.items[j].Size }
func (h diskFileHeap) Swap(i, j int)       { h.items[i], h.items[j] = h.items[j], h.items[i] }
func (h *diskFileHeap) Push(x interface{}) { h.items = append(h.items, x.(diskScanFile)) }
func (h *diskFileHeap) Pop() interface{} {
	old := h.items
	n := len(old)
	it := old[n-1]
	h.items = old[:n-1]
	return it
}

func (h *diskFileHeap) push(item diskScanFile) {
	if h.Len() < h.limit {
		heap.Push(h, item)
		return
	}
	if h.Len() > 0 && item.Size > h.items[0].Size {
		h.items[0] = item
		heap.Fix(h, 0)
	}
}

func (h *diskFileHeap) sorted() []diskScanFile {
	out := make([]diskScanFile, len(h.items))
	copy(out, h.items)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Size != out[j].Size {
			return out[i].Size > out[j].Size
		}
		return out[i].Path < out[j].Path
	})
	return out
}
