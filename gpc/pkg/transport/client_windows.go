//go:build windows

package transport

import (
	"context"
	"errors"
	"time"

	"github.com/aihop/gopanel/gpc/pkg/proto"
)

type Client struct {
	SocketPath string
	Timeout    time.Duration
}

func (c Client) Do(ctx context.Context, req proto.Request) (proto.Response, error) {
	_ = ctx
	_ = req
	return proto.Response{}, errors.New("windows transport not implemented")
}

