package api

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"time"

	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/token"
	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

type codeSessionRunner struct {
	mu      sync.Mutex
	locks   map[uint]*sync.Mutex
	cancels map[uint]context.CancelFunc
	queued  map[uint]struct{}
}

var backgroundCodeRunner = &codeSessionRunner{
	locks:   make(map[uint]*sync.Mutex),
	cancels: make(map[uint]context.CancelFunc),
	queued:  make(map[uint]struct{}),
}

func (r *codeSessionRunner) sessionLock(sessionID uint) *sync.Mutex {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.locks[sessionID] == nil {
		r.locks[sessionID] = &sync.Mutex{}
	}
	return r.locks[sessionID]
}

func enqueueCodeInstruction(instructionID uint) {
	if codeExecutions.isStopping() {
		return
	}
	backgroundCodeRunner.mu.Lock()
	if _, exists := backgroundCodeRunner.queued[instructionID]; exists {
		backgroundCodeRunner.mu.Unlock()
		return
	}
	backgroundCodeRunner.queued[instructionID] = struct{}{}
	backgroundCodeRunner.mu.Unlock()
	go backgroundCodeRunner.run(instructionID)
}

var codeInstructionRecoveryOnce sync.Once

func StartCodeInstructionRecovery() {
	codeInstructionRecoveryOnce.Do(func() {
		if err := recoverInterruptedCodeInstructions(); err != nil {
			global.LOG.Errorf("Recover interrupted Code instructions failed: %v", err)
		}
		enqueuePersistedCodeInstructions()
		go func() {
			ticker := time.NewTicker(15 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					enqueuePersistedCodeInstructions()
				case <-codeExecutions.stop:
					return
				}
			}
		}()
	})
}

func enqueuePersistedCodeInstructions() {
	ids, err := repo.NewAIDevSessionRepo().GetQueuedInstructionIDs(500)
	if err != nil {
		global.LOG.Errorf("Load queued Code instructions failed: %v", err)
		return
	}
	for _, id := range ids {
		enqueueCodeInstruction(id)
	}
}

func (r *codeSessionRunner) run(instructionID uint) {
	defer func() {
		r.mu.Lock()
		delete(r.queued, instructionID)
		r.mu.Unlock()
	}()
	sessionRepo := repo.NewAIDevSessionRepo()
	instruction, err := sessionRepo.GetInstructionByID(instructionID)
	if err != nil {
		global.LOG.Errorf("Failed to load Code instruction %d: %v", instructionID, err)
		return
	}
	lock := r.sessionLock(instruction.SessionID)
	lock.Lock()
	defer lock.Unlock()

	for {
		pending, err := sessionRepo.GetPendingInstructionsBySessionID(instruction.SessionID)
		if err != nil || len(pending) == 0 {
			return
		}
		current := pending[0]
		session, err := sessionRepo.GetSessionByID(current.SessionID)
		if err != nil {
			global.LOG.Errorf("Failed to load Code session %d: %v", current.SessionID, err)
			return
		}
		taskRepo := repo.NewAITaskRepo()
		task, err := taskRepo.GetTaskByID(current.TaskID)
		if err != nil {
			global.LOG.Errorf("Failed to load Code task %d: %v", current.TaskID, err)
			return
		}
		ctx, cancel := context.WithCancel(context.Background())
		r.mu.Lock()
		r.cancels[session.ID] = cancel
		r.mu.Unlock()
		result := executeCodeInstruction(ctx, sessionRepo, taskRepo, session, task, current, true)
		cancel()
		r.mu.Lock()
		delete(r.cancels, session.ID)
		r.mu.Unlock()
		if errors.Is(result.Err, context.Canceled) {
			return
		}
		if errors.Is(result.Err, errCodeExecutionStopping) {
			return
		}
		if result.Err != nil {
			global.LOG.Errorf("Code instruction %d failed: %v", current.ID, result.Err)
		}
	}
}

func (r *codeSessionRunner) cancel(sessionID uint) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	cancel := r.cancels[sessionID]
	if cancel == nil {
		return false
	}
	cancel()
	return true
}

func (r *codeSessionRunner) wait(ctx context.Context, sessionID uint) bool {
	lock := r.sessionLock(sessionID)
	done := make(chan struct{})
	go func() {
		lock.Lock()
		lock.Unlock()
		close(done)
	}()
	select {
	case <-done:
		return true
	case <-ctx.Done():
		return false
	}
}

func cancelQueuedCodeInstructions(session *model.AIDevSession) (int64, error) {
	var cancelled int64
	err := global.DB.Transaction(func(tx *gorm.DB) error {
		lockedSession, err := lockCodeSessionForDevelopment(tx, session.ID)
		if err != nil {
			return err
		}
		session = lockedSession
		result := tx.Model(&model.AIInstruction{}).
			Where("session_id = ? AND status = ?", session.ID, "queued").
			Update("status", "cancelled")
		if result.Error != nil {
			return result.Error
		}
		cancelled = result.RowsAffected
		if cancelled == 0 || session.LastTaskID == 0 {
			return nil
		}
		var task model.AITask
		if err := tx.First(&task, session.LastTaskID).Error; err != nil {
			return err
		}
		return reconcileCodeTaskState(tx, session, &task, "cancelled", "cancelled")
	})
	return cancelled, err
}

func StopCodeSessionExecution(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	sessionID, _ := strconv.Atoi(c.Params("id"))
	session, err := getAISessionWithPermission(uint(sessionID), claims)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	unlockLifecycle := codeSessionLifecycles.lock(session.ID)
	defer unlockLifecycle()
	cancelledQueued, err := cancelQueuedCodeInstructions(session)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	stoppingRunning := backgroundCodeRunner.cancel(uint(sessionID))
	stopContext, cancelStop := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelStop()
	stoppingWorkspace := codeExecutions.cancelSessionAndWait(stopContext, session.ID)
	if stopContext.Err() != nil {
		return c.JSON(e.Fail(errors.New("停止 Code 会话超时，请稍后重试")))
	}
	if cancelledQueued == 0 && !stoppingRunning && !stoppingWorkspace {
		return c.JSON(e.Fail(errors.New("当前会话没有正在执行的任务")))
	}
	return c.JSON(e.Succ(fiber.Map{
		"stopping":        stoppingRunning || stoppingWorkspace,
		"stoppingRunning": stoppingRunning || stoppingWorkspace,
		"cancelledQueued": cancelledQueued,
	}))
}

func RetryCodeInstruction(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	instructionID, _ := strconv.Atoi(c.Params("id"))
	sessionRepo := repo.NewAIDevSessionRepo()
	instruction, err := sessionRepo.GetInstructionByID(uint(instructionID))
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if _, err := getAISessionWithPermission(instruction.SessionID, claims); err != nil {
		return c.JSON(e.Fail(err))
	}
	if instruction.Status != "failed" && instruction.Status != "cancelled" {
		return c.JSON(e.Fail(errors.New("只有失败或已停止的指令可以重试")))
	}
	session, err := sessionRepo.GetSessionByID(instruction.SessionID)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if err := validateCodeSessionDevelopmentOpen(session); err != nil {
		return c.JSON(e.Fail(err))
	}
	unlockLifecycle := codeSessionLifecycles.lock(session.ID)
	defer unlockLifecycle()
	task, err := repo.NewAITaskRepo().GetTaskByID(instruction.TaskID)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if err := global.DB.Transaction(func(tx *gorm.DB) error {
		lockedSession, err := lockCodeSessionForDevelopment(tx, session.ID)
		if err != nil {
			return err
		}
		session = lockedSession
		result := tx.Model(&model.AIInstruction{}).
			Where("id = ? AND status IN ?", instruction.ID, []string{"failed", "cancelled"}).
			Update("status", "queued")
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errors.New("该指令状态已变化，请刷新后重试")
		}
		instruction.Status = "queued"
		if err := reconcileCodeTaskState(tx, session, task, "queued", "instruction_queued"); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return c.JSON(e.Fail(err))
	}
	enqueueCodeInstruction(instruction.ID)
	return c.JSON(e.Succ(instruction))
}
