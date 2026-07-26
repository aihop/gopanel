package service

import (
	"fmt"
	sysLog "log"
	"runtime/debug"
	"strings"

	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/gpc"
)

// gpcResponseOutput 安全地取 gpc 响应里的 output（错误分支里也能放心调）
func gpcResponseOutput(resp *gpc.Response) string {
	if resp == nil {
		return ""
	}
	return strings.TrimSpace(resp.Output)
}

// SafeGo 启动一个带 panic 兜底的后台任务。
//
// 面板里有大量 `go func(){...}()` 形式的后台任务（安装、更新、修复…），
// 而 fiber 的 recover 中间件只能兜住「请求处理协程」里的 panic：
// 后台 goroutine 里一旦 panic，整个面板进程会直接退出（表现就是"点一下按钮，
// 面板/系统就崩了"）。这里统一兜住，把「进程死」降级成「这次任务失败」。
//
// onPanic 可选，用于把失败状态写回任务自己的日志（例如 update logger 标记 failed）。
func SafeGo(name string, fn func(), onPanic ...func(err error)) {
	go func() {
		defer func() {
			r := recover()
			if r == nil {
				return
			}
			err := fmt.Errorf("后台任务 %s panic: %v", name, r)
			// 兜底日志本身不能再炸：global.LOG 在 log.Init 之前是 nil，
			// 在这里直接 global.LOG.Errorf 会二次 panic，那就白兜了
			if global.LOG != nil {
				global.LOG.Errorf("%v\n%s", err, debug.Stack())
			} else {
				sysLog.Printf("%v\n%s", err, debug.Stack())
			}
			for _, handler := range onPanic {
				if handler == nil {
					continue
				}
				// 兜底处理本身再 panic 也不能让进程挂掉
				func() {
					defer func() { _ = recover() }()
					handler(err)
				}()
			}
		}()
		fn()
	}()
}
