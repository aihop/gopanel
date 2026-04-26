//go:build !linux

package helper

import (
	"context"
	"errors"
)

func (s *Server) actionMysqlClientInstall(ctx context.Context, params map[string]interface{}) (string, error) {
	_ = ctx
	_ = params
	return "", errors.New("mysql client install is only supported on linux")
}

