//go:build !windows

package transport

import (
	"bufio"
	"context"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/aihop/gopanel/gp-agent/app/api"
	"github.com/aihop/gopanel/gp-agent/global"
	"github.com/aihop/gopanel/gp-agent/pkg/proto"
	"go.uber.org/zap"
)

func Serve(ctx context.Context, socketPath string) error {
	if err := os.MkdirAll(filepath.Dir(socketPath), 0755); err != nil {
		return err
	}
	_ = os.RemoveAll(socketPath)

	l, err := net.Listen("unix", socketPath)
	if err != nil {
		return err
	}
	defer l.Close()
	go func() {
		<-ctx.Done()
		_ = l.Close()
	}()

	_ = os.Chmod(socketPath, 0660)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		conn, err := l.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				time.Sleep(50 * time.Millisecond)
				continue
			}
			return err
		}
		go handleConn(conn)
	}
}

func handleConn(c net.Conn) {
	defer c.Close()
	_ = c.SetDeadline(time.Now().Add(30 * time.Second))

	r := bufio.NewReader(c)
	w := bufio.NewWriter(c)

	line, err := r.ReadBytes('\n')
	if err != nil {
		return
	}
	var req proto.Request
	if err := json.Unmarshal(line, &req); err != nil {
		resp := proto.Response{ID: "", OK: false, Code: proto.CodeInvalidParams, Error: err.Error()}
		_ = writeResp(w, resp)
		return
	}

	resp := api.Handle(context.Background(), req)
	if err := writeResp(w, resp); err != nil {
		if global.LOG != nil {
			global.LOG.Warn("write resp failed", zap.Error(err))
		}
	}
}

func writeResp(w *bufio.Writer, resp proto.Response) error {
	b, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	if _, err := w.Write(append(b, '\n')); err != nil {
		return err
	}
	return w.Flush()
}
