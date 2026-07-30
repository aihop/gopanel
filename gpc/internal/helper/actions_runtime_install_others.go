//go:build !linux && !darwin && !windows

package helper

import (
	"context"
	"errors"
)

func (s *Server) actionContainerRuntimeInstall(ctx context.Context, params map[string]interface{}) (string, error) {
	_ = ctx
	_ = params
	return "", errors.New("unsupported platform")
}
