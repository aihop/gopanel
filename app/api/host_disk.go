package api

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/service"
	"github.com/gofiber/fiber/v3"
)

// HostDiskOverview 磁盘容量概览
func HostDiskOverview(c fiber.Ctx) error {
	return c.JSON(e.Succ(service.DiskOverview()))
}

// HostDiskGpcStatus gpc helper 可用性诊断。
// 非 root 面板缺了 gpc 只能做退化扫描，这个接口把原因和修复命令直接给到前端。
func HostDiskGpcStatus(c fiber.Ctx) error {
	return c.JSON(e.Succ(service.DiagnoseGpc()))
}

// HostDiskScanStart 启动大文件扫描，立即返回 taskId，进度走 SSE
func HostDiskScanStart(c fiber.Ctx) error {
	req, err := e.BodyToStruct[service.DiskScanRequest](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	task, err := service.StartDiskScan(*req)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(task))
}

// HostDiskScanResult 拉取任务状态与结果（SSE 断开时的兜底轮询入口）
func HostDiskScanResult(c fiber.Ctx) error {
	id := strings.TrimSpace(c.Query("taskId"))
	if id == "" {
		return c.JSON(e.Fail(errors.New("taskId is required")))
	}
	task, ok := service.GetDiskScanTask(id)
	if !ok {
		return c.JSON(e.Fail(errors.New("扫描任务不存在或已过期")))
	}
	return c.JSON(e.Succ(task))
}

// HostDiskScanCancel 取消扫描
func HostDiskScanCancel(c fiber.Ctx) error {
	req, err := e.BodyToStruct[struct {
		TaskID string `json:"taskId"`
	}](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if err := service.CancelDiskScan(strings.TrimSpace(req.TaskID)); err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(nil))
}

// HostDiskScanStream SSE 推送扫描进度，完成时推最终结果。
// 全盘扫描可能跑几分钟，没有进度反馈用户会以为卡死。
func HostDiskScanStream(c fiber.Ctx) error {
	id := strings.TrimSpace(c.Query("taskId"))
	if id == "" {
		return c.JSON(e.Fail(errors.New("taskId is required")))
	}

	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("Transfer-Encoding", "chunked")

	c.RequestCtx().SetBodyStreamWriter(func(w *bufio.Writer) {
		writeEvent := func(event string, payload interface{}) {
			b, err := json.Marshal(payload)
			if err != nil {
				return
			}
			_, _ = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, string(b))
			_ = w.Flush()
		}

		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		// 扫描上限 10 分钟，这里留出余量后强制收尾，避免连接被永久挂住
		deadline := time.After(12 * time.Minute)

		for {
			select {
			case <-c.Context().Done():
				return
			case <-deadline:
				writeEvent("eof", fiber.Map{"reason": "timeout"})
				return
			case <-ticker.C:
				task, ok := service.GetDiskScanTask(id)
				if !ok {
					writeEvent("eof", fiber.Map{"reason": "not_found"})
					return
				}
				if task.Status == service.DiskScanStatusRunning {
					// 推整个任务快照而不是只推 Progress：progressLive / viaGpc / degraded
					// 这些标记会在扫描过程中变化（比如 gpc 不可用退回本地扫描），
					// 只推进度数字的话前端永远停在启动时那一份，会显示错误的状态文案。
					// 运行期间 Result 为 nil，整包很小，不存在带宽问题。
					writeEvent("progress", task)
					continue
				}
				writeEvent("done", task)
				writeEvent("eof", fiber.Map{"reason": "finished"})
				return
			}
		}
	})
	return nil
}

// HostDiskClean 删除或清空扫描结果中的文件。
// truncate=true 时是清空（保留 inode），日志类文件必须用这个——
// 直接删掉正在被写入的日志，空间不会释放。
func HostDiskClean(c fiber.Ctx) error {
	req, err := e.BodyToStruct[struct {
		TaskID   string   `json:"taskId"`
		Paths    []string `json:"paths"`
		Truncate bool     `json:"truncate"`
	}](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if len(req.Paths) == 0 {
		return c.JSON(e.Fail(errors.New("未选择要处理的文件")))
	}
	results, err := service.CleanDiskPaths(strings.TrimSpace(req.TaskID), req.Paths, req.Truncate)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(results))
}
