package gpc

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/aihop/gopanel/global"
)

type Request struct {
	ID     string                 `json:"id"`
	Action string                 `json:"action"`
	Params map[string]interface{} `json:"params,omitempty"`
}

type Response struct {
	ID     string `json:"id"`
	OK     bool   `json:"ok"`
	Code   string `json:"code"`
	Output string `json:"output,omitempty"`
	Error  string `json:"error,omitempty"`
}

func SocketPath() string {
	if v := strings.TrimSpace(os.Getenv("GPC_SOCKET_PATH")); v != "" {
		return v
	}
	if v := strings.TrimSpace(global.CONF.System.GpcSocketPath); v != "" {
		return v
	}
	if baseDir := strings.TrimSpace(os.Getenv("GOPANEL_BASE_DIR")); baseDir != "" {
		return filepath.Join(filepath.Clean(baseDir), "gpc.sock")
	}
	if baseDir := strings.TrimSpace(os.Getenv("GPC_BASE_DIR")); baseDir != "" {
		return filepath.Join(filepath.Clean(baseDir), "gpc.sock")
	}
	switch runtime.GOOS {
	case "darwin":
		return "/var/run/gopanel/gpc.sock"
	case "windows":
		return `\\.\pipe\gopanel-gpc`
	default:
		return "/run/gopanel/gpc.sock"
	}
}

// Do 调用 gpc helper。
//
// 契约：返回的 *Response 永远不为 nil，即使同时返回了 error。
// 原因：调用方在错误分支里读 resp.Output 打日志是很自然的写法（例如
// api.AgentEnsure / service.UpdateGpAgent），而这里以前在 socket 连不上等 7 条
// 失败路径上返回 nil —— 那些代码跑在 fire-and-forget 的 goroutine 里，
// goroutine 里的 panic 无法被 fiber 的 recover 中间件兜住，
// 一次 nil 解引用就会让整个面板进程退出。宁可返回空 Response。
func Do(ctx context.Context, action string, params map[string]interface{}) (*Response, error) {
	if strings.TrimSpace(action) == "" {
		return &Response{}, errors.New("gpc action is empty")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
	}

	if runtime.GOOS == "windows" {
		return &Response{}, errors.New("gpc helper is not supported on windows yet")
	}

	socketPath := SocketPath()
	conn, err := (&net.Dialer{}).DialContext(ctx, "unix", socketPath)
	if err != nil {
		return &Response{}, errors.New("需要安装/启用 gpc helper（未能连接到 gpc socket）: " + err.Error())
	}
	defer conn.Close()

	req := Request{
		ID:     strconvID(),
		Action: action,
		Params: params,
	}

	b, err := json.Marshal(req)
	if err != nil {
		return &Response{}, err
	}
	if _, err := conn.Write(append(b, '\n')); err != nil {
		return &Response{}, err
	}

	br := bufio.NewReader(conn)
	line, err := br.ReadBytes('\n')
	if err != nil {
		return &Response{}, err
	}

	var resp Response
	if err := json.Unmarshal(bytesTrimNL(line), &resp); err != nil {
		return &Response{}, err
	}
	if !resp.OK {
		if strings.TrimSpace(resp.Error) != "" {
			return &resp, errors.New(resp.Error)
		}
		return &resp, errors.New("gpc helper failed")
	}
	return &resp, nil
}

func bytesTrimNL(b []byte) []byte {
	return bytes.TrimSpace(b)
}

func strconvID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
