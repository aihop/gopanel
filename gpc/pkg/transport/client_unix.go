//go:build !windows

package transport

import (
	"bufio"
	"context"
	"encoding/json"
	"net"
	"time"

	"github.com/aihop/gopanel/gpc/pkg/proto"
)

type Client struct {
	SocketPath string
	Timeout    time.Duration
}

func (c Client) Do(ctx context.Context, req proto.Request) (proto.Response, error) {
	d := net.Dialer{Timeout: c.Timeout}
	conn, err := d.DialContext(ctx, "unix", c.SocketPath)
	if err != nil {
		return proto.Response{}, err
	}
	defer conn.Close()

	_ = conn.SetDeadline(time.Now().Add(c.Timeout))

	w := bufio.NewWriter(conn)
	r := bufio.NewReader(conn)

	b, err := json.Marshal(req)
	if err != nil {
		return proto.Response{}, err
	}
	if _, err := w.Write(append(b, '\n')); err != nil {
		return proto.Response{}, err
	}
	if err := w.Flush(); err != nil {
		return proto.Response{}, err
	}

	line, err := r.ReadBytes('\n')
	if err != nil {
		return proto.Response{}, err
	}
	var resp proto.Response
	if err := json.Unmarshal(line, &resp); err != nil {
		return proto.Response{}, err
	}
	return resp, nil
}

