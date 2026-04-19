//go:build windows

package helper

import (
	"context"
	"errors"
)

type Server struct {
	cfg Config
}

func NewServer(cfg Config) *Server {
	return &Server{cfg: cfg}
}

func (s *Server) Serve(ctx context.Context) error {
	_ = ctx
	return errors.New("unsupported platform")
}
