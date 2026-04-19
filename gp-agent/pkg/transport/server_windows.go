//go:build windows

package transport

import (
	"context"
	"errors"
)

func Serve(ctx context.Context, socketPath string) error {
	_ = ctx
	_ = socketPath
	return errors.New("unsupported platform")
}

