package api

import (
	"context"
	"errors"

	"github.com/aihop/gopanel/app/model"
)

func (coordinator *codeExecutionCoordinator) acquireInteractiveSession(
	ctx context.Context,
	session *model.AIDevSession,
) (*codeExecutionLease, error) {
	lease, err := coordinator.acquireSession(ctx, session, codeExecutionInteractive, false)
	if !errors.Is(err, errCodeExecutionBusy) || !codeSessionUsesDirectWorkspace(session) {
		return lease, err
	}
	keys := codeExecutionWorkspaceKeys(session)
	holderIDs, canHandover := coordinator.interactiveConflictSessionIDs(keys, session.ID)
	if !canHandover {
		return nil, err
	}
	for _, holderID := range holderIDs {
		if !coordinator.cancelSessionKindAndWait(ctx, holderID, codeExecutionInteractive) {
			return nil, errCodeExecutionBusy
		}
	}
	return coordinator.acquireSession(ctx, session, codeExecutionInteractive, false)
}

func codeSessionUsesDirectWorkspace(session *model.AIDevSession) bool {
	if session == nil {
		return false
	}
	if session.IsolationMode == codeIsolationDirect {
		return true
	}
	return session.ProjectID > 0 && session.IsolationMode == "" && session.SourceWorkDir == "" && session.WorktreeBranch == ""
}

func (coordinator *codeExecutionCoordinator) interactiveConflictSessionIDs(keys []string, targetSessionID uint) ([]uint, bool) {
	coordinator.mu.Lock()
	defer coordinator.mu.Unlock()
	holders := coordinator.conflicts(keys)
	if len(holders) == 0 {
		return nil, false
	}
	seen := make(map[uint]struct{}, len(holders))
	holderIDs := make([]uint, 0, len(holders))
	for _, holder := range holders {
		if holder.kind != codeExecutionInteractive || holder.sessionID == 0 || holder.sessionID == targetSessionID {
			return nil, false
		}
		if _, exists := seen[holder.sessionID]; exists {
			continue
		}
		seen[holder.sessionID] = struct{}{}
		holderIDs = append(holderIDs, holder.sessionID)
	}
	return holderIDs, true
}
