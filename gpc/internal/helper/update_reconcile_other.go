//go:build !darwin && !windows

package helper

import "context"

func (s *Server) reconcilePendingPanelUpdate(context.Context) {}
