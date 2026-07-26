package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/common"
	"github.com/aihop/gopanel/utils/diskscan"
	"github.com/aihop/gopanel/utils/gpc"
)

// 磁盘扫描任务状态
const (
	DiskScanStatusRunning  = "running"
	DiskScanStatusSuccess  = "success"
	DiskScanStatusFailed   = "failed"
	DiskScanStatusCanceled = "canceled"
)

const (
	// diskTaskTTL 扫描结果保留时长。磁盘状态时效性很强，隔夜的结果没有参考价值，
	// 所以只放内存 + 过期即弃，不落库（落库还得配清理逻辑，得不偿失）。
	// 这个值必须 ≤ gpc 侧的 diskScanTTL，否则面板还认为任务有效、gpc 那边授权已过期。
	diskTaskTTL = 30 * time.Minute
	// diskScanTimeout 单次扫描硬上限
	diskScanTimeout = 10 * time.Minute
)

// DiskScanTask 一次扫描任务。结果只在内存里，面板重启即失效。
type DiskScanTask struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
	ViaGpc bool   `json:"viaGpc"` // 是否由 gpc 执行（rootless 场景）
	// Degraded 非 root 且 gpc 不可用时退回本机权限扫描，结果不完整
	Degraded       bool   `json:"degraded"`
	DegradedReason string `json:"degradedReason,omitempty"`
	// ProgressLive 是否有实时进度。走 gpc 时为 false——gpc 协议是单请求单响应
	// （proto.Response 只有一个 Output 字段），扫描期间没有任何中间输出，
	// Progress 会一直是零值。前端据此改成不确定态提示，别显示"已扫 0 个文件"。
	ProgressLive bool               `json:"progressLive"`
	GpcScanID    string             `json:"-"` // gpc 侧的授权 ID，删除时必须带上
	Progress     diskscan.Progress  `json:"progress"`
	Result       *diskscan.Result   `json:"result,omitempty"`
	CreatedAt    time.Time          `json:"createdAt"`
	ExpiresAt    time.Time          `json:"expiresAt"`
	paths        map[string]int64   // 结果里的路径 -> 体积，删除时做归属校验
	cancel       context.CancelFunc `json:"-"`
}

type diskScanManager struct {
	mu      sync.Mutex
	tasks   map[string]*DiskScanTask
	running string // 同时只允许一个扫描任务：并发扫描只会互相抢 IO
}

var diskScanMgr = &diskScanManager{tasks: make(map[string]*DiskScanTask)}

// DiskScanRequest 启动扫描的参数
type DiskScanRequest struct {
	Roots       []string `json:"roots"`
	MinSize     int64    `json:"minSize"`
	TopN        int      `json:"topN"`
	CrossDevice bool     `json:"crossDevice"`
}

// StartDiskScan 启动一次扫描，立即返回 taskId，进度通过 GetDiskScanTask / SSE 获取。
func StartDiskScan(req DiskScanRequest) (*DiskScanTask, error) {
	diskScanMgr.mu.Lock()
	diskScanMgr.gcLocked()
	if diskScanMgr.running != "" {
		if t, ok := diskScanMgr.tasks[diskScanMgr.running]; ok && t.Status == DiskScanStatusRunning {
			diskScanMgr.mu.Unlock()
			return nil, fmt.Errorf("已有扫描任务正在运行（%s），请先等待或取消", t.ID)
		}
		diskScanMgr.running = ""
	}

	ctx, cancel := context.WithTimeout(context.Background(), diskScanTimeout)
	task := &DiskScanTask{
		ID:        common.RandStr(16),
		Status:    DiskScanStatusRunning,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(diskTaskTTL),
		// root 走本地遍历，有逐步回调；非 root 要过 gpc，全程没有中间输出
		ProgressLive: os.Geteuid() == 0,
		paths:        make(map[string]int64),
		cancel:       cancel,
	}
	diskScanMgr.tasks[task.ID] = task
	diskScanMgr.running = task.ID
	diskScanMgr.mu.Unlock()

	go runDiskScan(ctx, task, req)
	return task, nil
}

func runDiskScan(ctx context.Context, task *DiskScanTask, req DiskScanRequest) {
	defer task.cancel()
	defer func() {
		diskScanMgr.mu.Lock()
		if diskScanMgr.running == task.ID {
			diskScanMgr.running = ""
		}
		diskScanMgr.mu.Unlock()
	}()

	native := func() (*diskscan.Result, error) {
		return diskscan.Scan(ctx, diskscan.Options{
			Roots:       req.Roots,
			MinSize:     req.MinSize,
			TopN:        req.TopN,
			CrossDevice: req.CrossDevice,
			SkipDirs:    []string{global.CONF.System.TmpDir},
		}, func(p diskscan.Progress) {
			diskScanMgr.mu.Lock()
			task.Progress = p
			diskScanMgr.mu.Unlock()
		})
	}

	// 面板以 root 运行时直接本地扫，不必绕 gpc——每次 socket 往返都是纯开销。
	if os.Geteuid() == 0 {
		res, err := native()
		finishDiskScan(task, res, err, false, "")
		return
	}

	// 非 root：优先让 gpc 扫（它是 root，能看到全盘，还能顺带颁发删除授权）。
	res, scanID, err := scanViaGpc(ctx, req)
	if err == nil {
		finishDiskScan(task, res, err, true, scanID)
		return
	}
	if ctx.Err() != nil {
		finishDiskScan(task, nil, ctx.Err(), true, "")
		return
	}
	// gpc 不可用时退回本地扫描：当前用户读得到的部分照样能扫、能删，
	// 总比整个功能直接不可用强。读不到的目录会计入 Result.Errors，
	// 前端据此提示「结果可能不完整」。
	global.LOG.Warnf("[Disk] gpc 不可用，退回本机权限扫描（结果可能不完整）: %v", err)
	diskScanMgr.mu.Lock()
	task.ProgressLive = true // 退回本地遍历后又有逐步回调了
	diskScanMgr.mu.Unlock()
	nres, nerr := native()
	finishDiskScan(task, nres, nerr, false, "")
	if nerr == nil {
		diskScanMgr.mu.Lock()
		task.Degraded = true
		task.DegradedReason = "gpc helper 不可用，仅扫描了当前用户有权限访问的目录，结果可能不完整"
		diskScanMgr.mu.Unlock()
	}
}

// scanViaGpc rootless 场景：让 gpc 做扫描，同时拿到删除授权用的 scanId。
// 授权必须由 gpc 自己颁发——它得亲眼见过这些路径确实是大文件，才肯在删除时放行。
func scanViaGpc(ctx context.Context, req DiskScanRequest) (*diskscan.Result, string, error) {
	params := map[string]interface{}{
		"roots":       req.Roots,
		"minSize":     req.MinSize,
		"topN":        req.TopN,
		"crossDevice": req.CrossDevice,
	}
	resp, err := gpc.Do(ctx, "DISK_SCAN", params)
	if err != nil {
		return nil, "", err
	}
	var out struct {
		ScanID       string `json:"scanId"`
		Roots        []string
		MinSize      int64 `json:"minSize"`
		ScannedFiles int64 `json:"scannedFiles"`
		ScannedBytes int64 `json:"scannedBytes"`
		Errors       int64 `json:"errors"`
		Files        []struct {
			Path    string    `json:"path"`
			Size    int64     `json:"size"`
			ModTime time.Time `json:"modTime"`
		} `json:"files"`
	}
	if err := json.Unmarshal([]byte(resp.Output), &out); err != nil {
		return nil, "", fmt.Errorf("解析 gpc 扫描结果失败: %w", err)
	}
	res := &diskscan.Result{
		Roots:        out.Roots,
		MinSize:      out.MinSize,
		ScannedFiles: out.ScannedFiles,
		ScannedBytes: out.ScannedBytes,
		Errors:       out.Errors,
		FinishedAt:   time.Now(),
	}
	for _, f := range out.Files {
		res.Files = append(res.Files, diskscan.FileItem{
			Path:        f.Path,
			Size:        f.Size,
			ModTime:     f.ModTime,
			Category:    diskscan.Categorize(f.Path),
			IsContainer: diskscan.InContainerStore(f.Path),
		})
	}
	return res, out.ScanID, nil
}

func finishDiskScan(task *DiskScanTask, res *diskscan.Result, err error, viaGpc bool, gpcScanID string) {
	diskScanMgr.mu.Lock()
	defer diskScanMgr.mu.Unlock()
	task.ViaGpc = viaGpc
	task.GpcScanID = gpcScanID
	if err != nil {
		if errors.Is(err, context.Canceled) {
			task.Status = DiskScanStatusCanceled
		} else {
			task.Status = DiskScanStatusFailed
			task.Error = err.Error()
		}
		return
	}
	// 标注每条结果在当前运行条件下能不能清理。canElevate 表示删除能否以 root 执行：
	// 面板自己就是 root，或 rootless 下拿到了 gpc 授权。
	canElevate := os.Geteuid() == 0 || (viaGpc && strings.TrimSpace(gpcScanID) != "")
	diskscan.AnnotateRemovable(res.Files, global.CONF.System.BaseDir, os.Geteuid(), canElevate)

	task.Result = res
	task.Status = DiskScanStatusSuccess
	for _, f := range res.Files {
		task.paths[f.Path] = f.Size
	}
	global.LOG.Infof("[Disk] 扫描完成 task=%s roots=%v 文件=%d 命中=%d viaGpc=%v",
		task.ID, res.Roots, res.ScannedFiles, len(res.Files), viaGpc)
}

// GetDiskScanTask 取任务快照。返回的是拷贝，调用方不会看到并发写入的中间状态。
func GetDiskScanTask(id string) (*DiskScanTask, bool) {
	diskScanMgr.mu.Lock()
	defer diskScanMgr.mu.Unlock()
	task, ok := diskScanMgr.tasks[id]
	if !ok || time.Now().After(task.ExpiresAt) {
		return nil, false
	}
	snapshot := *task
	return &snapshot, true
}

// CancelDiskScan 取消正在跑的扫描
func CancelDiskScan(id string) error {
	diskScanMgr.mu.Lock()
	task, ok := diskScanMgr.tasks[id]
	diskScanMgr.mu.Unlock()
	if !ok {
		return errors.New("扫描任务不存在或已过期")
	}
	if task.Status != DiskScanStatusRunning {
		return nil
	}
	task.cancel()
	return nil
}

func (m *diskScanManager) gcLocked() {
	now := time.Now()
	for id, t := range m.tasks {
		if now.After(t.ExpiresAt) && t.Status != DiskScanStatusRunning {
			delete(m.tasks, id)
		}
	}
}

// DiskCleanResult 单个路径的处理结果
type DiskCleanResult struct {
	Path    string `json:"path"`
	OK      bool   `json:"ok"`
	Freed   int64  `json:"freed"`
	Message string `json:"message,omitempty"`
}

// CleanDiskPaths 删除或清空扫描结果中的文件。
//
// 三层校验，缺一不可：
//  1. 路径必须属于该次扫描结果——用户不能随便传一个路径进来
//  2. 保护名单（面板侧这份管 root 直删的场景）
//  3. 必须是普通文件
//
// rootless 场景下真正执行删除的是 gpc，它会用 scanId 再独立校验一遍，
// 面板这层挡不住的，gpc 那层还会挡。
func CleanDiskPaths(taskID string, paths []string, truncate bool) ([]DiskCleanResult, error) {
	diskScanMgr.mu.Lock()
	task, ok := diskScanMgr.tasks[taskID]
	if !ok || time.Now().After(task.ExpiresAt) {
		diskScanMgr.mu.Unlock()
		return nil, errors.New("扫描结果不存在或已过期，请重新扫描")
	}
	if task.Status != DiskScanStatusSuccess {
		diskScanMgr.mu.Unlock()
		return nil, errors.New("扫描尚未成功完成")
	}
	known := make(map[string]int64, len(task.paths))
	for k, v := range task.paths {
		known[k] = v
	}
	viaGpc, gpcScanID := task.ViaGpc, task.GpcScanID
	diskScanMgr.mu.Unlock()

	baseDir := global.CONF.System.BaseDir
	results := make([]DiskCleanResult, 0, len(paths))
	var totalFreed int64

	for _, p := range paths {
		size, inScan := known[p]
		if !inScan {
			results = append(results, DiskCleanResult{Path: p, Message: "不在本次扫描结果中，拒绝操作"})
			continue
		}
		if diskscan.IsProtected(p, baseDir) {
			results = append(results, DiskCleanResult{Path: p, Message: "系统关键路径，禁止操作"})
			continue
		}
		if !diskscan.IsRegularFile(p) {
			results = append(results, DiskCleanResult{Path: p, Message: "不是普通文件，拒绝操作"})
			continue
		}
		if diskscan.InContainerStore(p) && !truncate {
			results = append(results, DiskCleanResult{Path: p, Message: "容器存储目录下的文件，请到容器页面执行清理（prune）"})
			continue
		}
		if diskscan.IsJournalInternal(p) {
			results = append(results, DiskCleanResult{Path: p, Message: "journald 内部文件，请用 journalctl --vacuum-size 清理"})
			continue
		}

		var err error
		if truncate {
			err = truncatePath(p, viaGpc, gpcScanID)
		} else {
			err = removePath(p, viaGpc, gpcScanID)
		}
		if err != nil {
			results = append(results, DiskCleanResult{Path: p, Message: err.Error()})
			global.LOG.Warnf("[Disk] 清理失败 path=%s truncate=%v err=%v", p, truncate, err)
			continue
		}
		totalFreed += size
		results = append(results, DiskCleanResult{Path: p, OK: true, Freed: size})
		// 审计：磁盘清理是不可逆操作，必须留痕
		global.LOG.Warnf("[Disk][审计] %s path=%s size=%d task=%s viaGpc=%v",
			map[bool]string{true: "清空", false: "删除"}[truncate], p, size, taskID, viaGpc)
	}

	global.LOG.Warnf("[Disk][审计] 本次清理完成 task=%s 处理=%d 释放=%d 字节", taskID, len(results), totalFreed)
	return results, nil
}

func removePath(path string, viaGpc bool, scanID string) error {
	if !viaGpc {
		if err := os.Remove(path); err != nil {
			if isPermissionErr(err) && scanID != "" {
				return removeViaGpcScan(path, scanID)
			}
			return err
		}
		return nil
	}
	return removeViaGpcScan(path, scanID)
}

func truncatePath(path string, viaGpc bool, scanID string) error {
	if !viaGpc {
		if err := os.Truncate(path, 0); err != nil {
			if isPermissionErr(err) && scanID != "" {
				return truncateViaGpcScan(path, scanID)
			}
			return err
		}
		return nil
	}
	return truncateViaGpcScan(path, scanID)
}

func removeViaGpcScan(path string, scanID string) error {
	if scanID == "" {
		return errors.New("缺少扫描授权，无法删除该路径")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_, err := gpc.Do(ctx, "FILE_REMOVE", map[string]interface{}{"path": path, "scanId": scanID})
	return err
}

func truncateViaGpcScan(path string, scanID string) error {
	if scanID == "" {
		return errors.New("缺少扫描授权，无法清空该文件")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_, err := gpc.Do(ctx, "FILE_TRUNCATE", map[string]interface{}{"path": path, "scanId": scanID})
	return err
}
