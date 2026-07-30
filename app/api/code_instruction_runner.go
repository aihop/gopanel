package api

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"time"

	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/token"
	"github.com/gofiber/fiber/v3"
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
			for range ticker.C {
				enqueuePersistedCodeInstructions()
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

func (r *codeSessionRunner) wait(sessionID uint) {
	lock := r.sessionLock(sessionID)
	lock.Lock()
	lock.Unlock()
}

func StopCodeSessionExecution(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	sessionID, _ := strconv.Atoi(c.Params("id"))
	if _, err := getAISessionWithPermission(uint(sessionID), claims); err != nil {
		return c.JSON(e.Fail(err))
	}
	cancelledQueued, err := repo.NewAIDevSessionRepo().CancelQueuedInstructions(uint(sessionID))
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	stoppingRunning := backgroundCodeRunner.cancel(uint(sessionID))
	if cancelledQueued == 0 && !stoppingRunning {
		return c.JSON(e.Fail(errors.New("当前会话没有正在执行的任务")))
	}
	return c.JSON(e.Succ(fiber.Map{
		"stopping":        stoppingRunning,
		"stoppingRunning": stoppingRunning,
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
	instruction.Status = "queued"
	if err := sessionRepo.UpdateInstruction(instruction); err != nil {
		return c.JSON(e.Fail(err))
	}
	enqueueCodeInstruction(instruction.ID)
	return c.JSON(e.Succ(instruction))
}
