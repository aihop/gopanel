package helper

import (
	"context"
	"errors"
)

func (s *Server) actionSecurityScanSSH(ctx context.Context, params map[string]interface{}) (string, error) {
	return "", errors.New("unsupported platform")
}

func (s *Server) actionSecurityFixSSH(ctx context.Context, params map[string]interface{}) (string, error) {
	return "", errors.New("unsupported platform")
}

func (s *Server) actionSecurityScanPort(ctx context.Context, params map[string]interface{}) (string, error) {
	return "", errors.New("unsupported platform")
}
