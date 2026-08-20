package api

import (
	"context"
	"errors"
	"time"

	"github.com/aihop/gopanel/app/model"
)

func (coordinator *codeExecutionCoordinator) acquireInteractiveSession(
	ctx context.Context,
	session *model.AIDevSession,
) (*codeExecutionLease, error) {
	if session == nil || session.ID == 0 {
		return nil, errors.New("Code 执行会话无效")
	}
	keys := codeExecutionWorkspaceKeys(session)
	if len(keys) == 0 {
		return nil, errors.New("Code 执行工作区无效")
	}
	coordinator.mu.Lock()
	for _, conflict := range coordinator.conflicts(keys, codeExecutionInteractive) {
		if conflict.sessionID != session.ID || conflict.kind != codeExecutionInstruction {
			coordinator.mu.Unlock()
			return nil, errCodeExecutionBusy
		}
	}
	coordinator.mu.Unlock()
	if coordinator.hasSessionKind(session.ID, codeExecutionInstruction) {
		if coordinator.cancelSessionKindAndWait(ctx, session.ID, codeExecutionInstruction) && ctx.Err() != nil {
			return nil, ctx.Err()
		}
	}
	return coordinator.acquireOwned(ctx, session.ID, keys, codeExecutionInteractive, false)
}

func handoverCodeSessionToConversation(ctx context.Context, sessionID uint) error {
	if sessionID == 0 {
		return errors.New("Code 执行会话无效")
	}
	backgroundCodeRunner.setInteractive(sessionID, false)
	if codeExecutions.cancelSessionKindAndWait(ctx, sessionID, codeExecutionInteractive) && ctx.Err() != nil {
		return ctx.Err()
	}
	return nil
}

func acquireCodeInstructionSession(ctx context.Context, session *model.AIDevSession) (*codeExecutionLease, error) {
	if session == nil || session.ID == 0 {
		return nil, errors.New("Code 执行会话无效")
	}
	stopContext, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	if err := handoverCodeSessionToConversation(stopContext, session.ID); err != nil {
		return nil, err
	}
	return codeExecutions.acquireSession(ctx, session, codeExecutionInstruction, true)
}
