//go:build !linux && !darwin

package helper

import (
	"context"
	"errors"
)

func (s *Server) actionGoPanelAgentEnsure(ctx context.Context, params map[string]interface{}) (string, error) {
	_ = ctx
	_ = params
	return "", errors.New("gp-agent ensure is only supported on linux/darwin")
}

func (s *Server) actionGoPanelAgentInstall(ctx context.Context, params map[string]interface{}) (string, error) {
	_ = ctx
	_ = params
	return "", errors.New("gp-agent install is only supported on linux/darwin")
}
