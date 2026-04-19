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
	switch runtime.GOOS {
	case "darwin":
		return "/var/run/gopanel/gpc.sock"
	case "windows":
		return `\\.\pipe\gopanel-gpc`
	default:
		return "/run/gopanel/gpc.sock"
	}
}

func Do(ctx context.Context, action string, params map[string]interface{}) (*Response, error) {
	if strings.TrimSpace(action) == "" {
		return nil, errors.New("gpc action is empty")
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
		return nil, errors.New("gpc helper is not supported on windows yet")
	}

	socketPath := SocketPath()
	conn, err := (&net.Dialer{}).DialContext(ctx, "unix", socketPath)
	if err != nil {
		return nil, errors.New("需要安装/启用 gpc helper（未能连接到 gpc socket）: " + err.Error())
	}
	defer conn.Close()

	req := Request{
		ID:     strconvID(),
		Action: action,
		Params: params,
	}

	b, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	if _, err := conn.Write(append(b, '\n')); err != nil {
		return nil, err
	}

	br := bufio.NewReader(conn)
	line, err := br.ReadBytes('\n')
	if err != nil {
		return nil, err
	}

	var resp Response
	if err := json.Unmarshal(bytesTrimNL(line), &resp); err != nil {
		return nil, err
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
