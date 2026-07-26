// Package diskscan 提供磁盘大文件/大目录扫描。
//
// 设计要点（改动前请先读完）：
//   - 用 WalkDir 而不是 Walk：WalkDir 基于 DirEntry，不会对每个文件做 Stat，
//     大目录下快数倍；只有需要 size 时才 d.Info()。
//   - 默认不跨文件系统：比对根目录的设备号，否则扫 / 会掉进 NFS、容器 overlay，
//     结果全是噪音而且慢到不可用。
//   - 不跟随符号链接：跟随会绕过调用方的路径边界，也可能陷入环。
//   - Top-N 用最小堆增量维护，内存 O(N) 而不是 O(全盘文件数)。
package diskscan

import (
	"container/heap"
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync/atomic"
	"time"
)

// 无论调用方传什么，这些目录一律不进入遍历：伪文件系统里没有真实占用，
// 读它们只会浪费时间甚至阻塞（/proc/<pid>/fd 之类）。
var alwaysSkipDirs = []string{"/proc", "/sys", "/dev", "/run"}

// 容器存储目录默认单独归类：里面的层文件不能直接 rm，必须走 docker/podman prune。
var containerStoreHints = []string{
	"/var/lib/docker",
	"/var/lib/containers",
	"/var/lib/podman",
	".local/share/containers",
}

const (
	// DefaultMinSize 默认只关心 ≥100MB 的文件，再小的清了也没意义
	DefaultMinSize int64 = 100 << 20
	// DefaultTopN 结果条数上限
	DefaultTopN = 200
	// DefaultTopDirs 目录聚合条数上限
	DefaultTopDirs = 50
)

// Options 扫描参数
type Options struct {
	Roots       []string // 扫描根，默认 ["/"]
	MinSize     int64    // 只收录 ≥ 该体积的文件
	TopN        int      // 大文件保留条数
	TopDirs     int      // 大目录保留条数
	CrossDevice bool     // 是否跨文件系统，默认 false
	SkipDirs    []string // 调用方追加的跳过目录
}

func (o *Options) normalize() {
	if len(o.Roots) == 0 {
		o.Roots = []string{"/"}
	}
	if o.MinSize <= 0 {
		o.MinSize = DefaultMinSize
	}
	if o.TopN <= 0 {
		o.TopN = DefaultTopN
	}
	if o.TopDirs <= 0 {
		o.TopDirs = DefaultTopDirs
	}
}

// FileItem 命中的大文件
type FileItem struct {
	Path        string    `json:"path"`
	Size        int64     `json:"size"`
	ModTime     time.Time `json:"modTime"`
	Category    string    `json:"category"`
	IsContainer bool      `json:"isContainer"` // 容器存储目录下的文件，不能直接删
	// Removable 这个文件在当前运行条件下能不能被清理。
	// 不能清理的原因五花八门（只读卷、属主不是自己、系统关键路径…），
	// 必须在列表里提前标出来——否则用户一个个点过去才发现全都失败，
	// 这正是 macOS 上的实际观感：扫 / 出来的大文件几乎全在只读封印卷上，
	// 或者属主是 root 而面板跑在普通用户下。
	Removable bool   `json:"removable"`
	Reason    string `json:"reason,omitempty"` // Removable 为 false 时说明原因
}

// DirItem 目录占用聚合（只统计该目录直属文件，不含子目录，避免父目录重复吃掉子目录的量）
type DirItem struct {
	Path  string `json:"path"`
	Size  int64  `json:"size"`
	Count int64  `json:"count"`
}

// Progress 扫描进度，供 SSE 推送
type Progress struct {
	ScannedFiles int64  `json:"scannedFiles"`
	ScannedBytes int64  `json:"scannedBytes"`
	CurrentDir   string `json:"currentDir"`
	Errors       int64  `json:"errors"`
}

// Result 扫描结果
type Result struct {
	Roots        []string   `json:"roots"`
	MinSize      int64      `json:"minSize"`
	Files        []FileItem `json:"files"`
	Dirs         []DirItem  `json:"dirs"`
	ScannedFiles int64      `json:"scannedFiles"`
	ScannedBytes int64      `json:"scannedBytes"`
	Errors       int64      `json:"errors"`
	StartedAt    time.Time  `json:"startedAt"`
	FinishedAt   time.Time  `json:"finishedAt"`
}

// Scan 遍历 opts.Roots，返回大文件与大目录。
// onProgress 可为 nil；不为 nil 时按 progressInterval 节流回调，不要在回调里做重活。
func Scan(ctx context.Context, opts Options, onProgress func(Progress)) (*Result, error) {
	opts.normalize()
	res := &Result{Roots: opts.Roots, MinSize: opts.MinSize, StartedAt: time.Now()}

	top := &fileHeap{limit: opts.TopN}
	heap.Init(top)
	dirAgg := make(map[string]*DirItem, 1024)

	var scannedFiles, scannedBytes, errCount int64
	lastReport := time.Now()
	report := func(dir string, force bool) {
		if onProgress == nil {
			return
		}
		if !force && time.Since(lastReport) < 300*time.Millisecond {
			return
		}
		lastReport = time.Now()
		onProgress(Progress{
			ScannedFiles: atomic.LoadInt64(&scannedFiles),
			ScannedBytes: atomic.LoadInt64(&scannedBytes),
			CurrentDir:   dir,
			Errors:       atomic.LoadInt64(&errCount),
		})
	}

	// alwaysSkip 对根目录也生效（扫 /proc 本身就没意义）。
	// rootGuarded 是平台跳过 + 调用方指定的跳过：当扫描根本身就落在某条跳过规则
	// 里面时，整条规则必须对这个根失效——只豁免根路径本身是不够的，
	// 那样它的每个子目录还是会被跳掉，结果就是扫出个位数文件。
	// macOS 上数据卷挂在 /System/Volumes/Data，指定它为根时正会踩到这条。
	alwaysSkip := alwaysSkipDirs
	rootGuardedAll := append(append([]string{}, platformSkipDirs...), opts.SkipDirs...)

	for _, root := range opts.Roots {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}
		root = filepath.Clean(root)
		// 剔除那些「本身是扫描根的祖先或就是扫描根」的跳过规则
		rootGuardedSkip := make([]string, 0, len(rootGuardedAll))
		for _, sk := range rootGuardedAll {
			sk = filepath.Clean(strings.TrimSpace(sk))
			if sk == "" || sk == "." {
				continue
			}
			if root == sk || strings.HasPrefix(root, sk+string(os.PathSeparator)) {
				continue // 用户就是要扫这里面，跳过规则对本次根失效
			}
			rootGuardedSkip = append(rootGuardedSkip, sk)
		}
		rootDev, devOK := deviceOf(root)
		if !devOK && !opts.CrossDevice {
			// 拿不到设备号（非 unix 或 stat 失败）时退化为允许跨设备，
			// 否则整个根都会被跳过，功能直接失效
			devOK = false
		}

		walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return ctxErr
			}
			if err != nil {
				atomic.AddInt64(&errCount, 1)
				if d != nil && d.IsDir() {
					return fs.SkipDir // 没权限的目录跳过，不要中断整轮扫描
				}
				return nil
			}
			if d.IsDir() {
				if isSkipped(path, alwaysSkip) {
					return fs.SkipDir
				}
				if isSkipped(path, rootGuardedSkip) {
					return fs.SkipDir
				}
				if !opts.CrossDevice && devOK && path != root {
					if dev, ok := deviceOf(path); ok && dev != rootDev {
						return fs.SkipDir
					}
				}
				report(path, false)
				return nil
			}
			// 只统计普通文件：符号链接不跟随，设备/socket/管道没有实际占用
			if !d.Type().IsRegular() {
				return nil
			}
			info, err := d.Info()
			if err != nil {
				atomic.AddInt64(&errCount, 1)
				return nil
			}
			size := info.Size()
			atomic.AddInt64(&scannedFiles, 1)
			atomic.AddInt64(&scannedBytes, size)

			dir := filepath.Dir(path)
			agg, ok := dirAgg[dir]
			if !ok {
				agg = &DirItem{Path: dir}
				dirAgg[dir] = agg
			}
			agg.Size += size
			agg.Count++

			if size >= opts.MinSize {
				top.push(FileItem{
					Path:        path,
					Size:        size,
					ModTime:     info.ModTime(),
					Category:    Categorize(path),
					IsContainer: InContainerStore(path),
				})
			}
			return nil
		})
		if walkErr != nil {
			if errors.Is(walkErr, context.Canceled) || errors.Is(walkErr, context.DeadlineExceeded) {
				return nil, walkErr
			}
			// 单个根失败（不存在/无权限）不影响其他根
			atomic.AddInt64(&errCount, 1)
		}
	}

	report("", true)

	res.Files = top.sorted()
	res.Dirs = topDirs(dirAgg, opts.TopDirs)
	res.ScannedFiles = scannedFiles
	res.ScannedBytes = scannedBytes
	res.Errors = errCount
	res.FinishedAt = time.Now()
	return res, nil
}

func isSkipped(path string, skip []string) bool {
	for _, s := range skip {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		s = filepath.Clean(s)
		if path == s || strings.HasPrefix(path, s+string(os.PathSeparator)) {
			return true
		}
	}
	return false
}

// InContainerStore 判断路径是否落在容器存储目录下
func InContainerStore(path string) bool {
	for _, hint := range containerStoreHints {
		if strings.Contains(path, hint) {
			return true
		}
	}
	return false
}

// IsJournalInternal 判断是否为 systemd-journald 的内部文件。
// 这类文件不能 truncate 也不建议直接 rm：journald 持有打开的 fd 和内部索引，
// 清空会导致日志库损坏、空间也未必释放。正确姿势是 journalctl --vacuum-size。
func IsJournalInternal(path string) bool {
	lower := strings.ToLower(path)
	return strings.Contains(lower, "/log/journal/") || strings.HasSuffix(lower, ".journal") ||
		strings.HasSuffix(lower, ".journal~")
}

// Categorize 给文件分个类，前端据此给出「建议动作」——
// 只给一列大小的列表用户不敢动手，得告诉他这东西是什么、能不能删。
func Categorize(path string) string {
	lower := strings.ToLower(path)
	base := filepath.Base(lower)
	ext := filepath.Ext(base)

	switch {
	case InContainerStore(lower):
		return "container"
	case ext == ".log" || strings.Contains(lower, "/log/") || strings.Contains(lower, "/logs/") ||
		strings.Contains(lower, "/var/log/") || strings.HasPrefix(base, "journal"):
		return "log"
	case strings.Contains(lower, "/var/cache/") || strings.Contains(lower, "/.cache/") ||
		strings.Contains(lower, "/var/lib/apt/lists"):
		return "cache"
	case ext == ".tar" || ext == ".gz" || ext == ".tgz" || ext == ".zip" || ext == ".bz2" ||
		ext == ".xz" || ext == ".7z" || ext == ".rar":
		return "archive"
	case strings.HasPrefix(base, "core.") || base == "core" || ext == ".dmp":
		return "coredump"
	case strings.Contains(lower, "/tmp/") || ext == ".tmp" || ext == ".swp":
		return "temp"
	case ext == ".sql" || ext == ".dump" || strings.Contains(lower, "/backup"):
		return "backup"
	default:
		return "other"
	}
}

func topDirs(agg map[string]*DirItem, limit int) []DirItem {
	list := make([]DirItem, 0, len(agg))
	for _, v := range agg {
		list = append(list, *v)
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].Size != list[j].Size {
			return list[i].Size > list[j].Size
		}
		return list[i].Path < list[j].Path
	})
	if len(list) > limit {
		list = list[:limit]
	}
	return list
}

// fileHeap 最小堆：堆顶是当前 Top-N 里最小的那个，
// 新文件只要比堆顶大就替换掉它，避免把全盘文件都塞进内存再排序。
type fileHeap struct {
	items []FileItem
	limit int
}

func (h fileHeap) Len() int            { return len(h.items) }
func (h fileHeap) Less(i, j int) bool  { return h.items[i].Size < h.items[j].Size }
func (h fileHeap) Swap(i, j int)       { h.items[i], h.items[j] = h.items[j], h.items[i] }
func (h *fileHeap) Push(x interface{}) { h.items = append(h.items, x.(FileItem)) }
func (h *fileHeap) Pop() interface{} {
	old := h.items
	n := len(old)
	it := old[n-1]
	h.items = old[:n-1]
	return it
}

func (h *fileHeap) push(item FileItem) {
	if h.Len() < h.limit {
		heap.Push(h, item)
		return
	}
	if h.Len() > 0 && item.Size > h.items[0].Size {
		h.items[0] = item
		heap.Fix(h, 0)
	}
}

// sorted 返回从大到小的结果（堆内部是无序的，不能直接返回 items）
func (h *fileHeap) sorted() []FileItem {
	out := make([]FileItem, len(h.items))
	copy(out, h.items)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Size != out[j].Size {
			return out[i].Size > out[j].Size
		}
		return out[i].Path < out[j].Path
	})
	return out
}
