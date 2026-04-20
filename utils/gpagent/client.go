package gpagent

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/aihop/gopanel/global"
)

type Request struct {
	ID     string                 `json:"id"`
	Action string                 `json:"action"`
	Params map[string]interface{} `json:"params"`
}

type Response struct {
	ID     string `json:"id"`
	OK     bool   `json:"ok"`
	Code   string `json:"code"`
	Output string `json:"output"`
	Error  string `json:"error"`
}

func SocketPath() string {
	if runtimeSock := os.Getenv("GP_AGENT_SOCKET_PATH"); runtimeSock != "" {
		return runtimeSock
	}
	if s := strings.TrimSpace(global.CONF.System.GpAgentSocketPath); s != "" {
		return s
	}
	baseDir := strings.TrimSpace(global.CONF.System.BaseDir)
	if baseDir == "" {
		if homeDir, err := os.UserHomeDir(); err == nil && homeDir != "" {
			baseDir = filepath.Join(homeDir, ".gopanel")
		}
	}
	sock := filepath.Join(baseDir, "agent", "run", "gp-agent.sock")
	return sock
}

func Do(ctx context.Context, action string, params map[string]interface{}) (Response, error) {
	if action == "" {
		return Response{}, errors.New("action is empty")
	}
	if params == nil {
		params = map[string]interface{}{}
	}

	timeout := 10 * time.Second
	if ctx != nil {
		if dl, ok := ctx.Deadline(); ok {
			if d := time.Until(dl); d > 0 {
				timeout = d
			}
		}
	}

	d := net.Dialer{Timeout: timeout}
	conn, err := d.DialContext(ctx, "unix", SocketPath())
	if err != nil {
		return Response{}, err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(timeout))

	req := Request{
		ID:     strconv.FormatInt(time.Now().UnixNano(), 10),
		Action: action,
		Params: params,
	}
	b, err := json.Marshal(req)
	if err != nil {
		return Response{}, err
	}

	w := bufio.NewWriter(conn)
	if _, err := w.Write(append(b, '\n')); err != nil {
		return Response{}, err
	}
	if err := w.Flush(); err != nil {
		return Response{}, err
	}

	r := bufio.NewReader(conn)
	line, err := r.ReadBytes('\n')
	if err != nil {
		return Response{}, err
	}

	var resp Response
	if err := json.Unmarshal(line, &resp); err != nil {
		return Response{}, err
	}
	if resp.OK {
		return resp, nil
	}
	if resp.Error != "" {
		return resp, errors.New(resp.Error)
	}
	if resp.Code != "" {
		return resp, errors.New(resp.Code)
	}
	return resp, errors.New("gp-agent request failed")
}

func DecodeOutput[T any](resp Response) (T, error) {
	var zero T
	s := strings.TrimSpace(resp.Output)
	if s == "" {
		return zero, nil
	}
	if err := json.Unmarshal([]byte(s), &zero); err != nil {
		return zero, err
	}
	return zero, nil
}
