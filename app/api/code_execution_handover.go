package api

import (
	"context"

	"github.com/aihop/gopanel/app/model"
)

func (coordinator *codeExecutionCoordinator) acquireInteractiveSession(
	ctx context.Context,
	session *model.AIDevSession,
) (*codeExecutionLease, error) {
	return coordinator.acquireSession(ctx, session, codeExecutionInteractive, false)
}
