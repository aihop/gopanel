package api

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
)

const (
	codeExecutionInteractive = "interactive"
	codeExecutionInstruction = "instruction"
	codeExecutionMutation    = "mutation"
	codeExecutionQuality     = "quality"
	codeExecutionDelivery    = "delivery"
)

var (
	errCodeExecutionBusy     = errors.New("当前工作区正在交付或执行 AI 指令，需要独占目录；请等它结束后重试。多个开发终端可以同时打开，不受此限制")
	errCodeExecutionCapacity = errors.New("Code 执行并发已满，请稍后重试")
	errCodeExecutionStopping = errors.New("Code 执行服务正在停止")
)

type codeExecutionLease struct {
	coordinator *codeExecutionCoordinator
	id          uint64
	sessionID   uint
	kind        string
	keys        []string
	done        chan struct{}
	releaseOnce sync.Once
	cancel      context.CancelFunc
	cancelled   bool
	// 记住槽位取自哪个池子，归还时才不会还错——交付和执行现在是两个池。
	slotPool chan struct{}
}

type codeExecutionCoordinator struct {
	mu     sync.Mutex
	nextID uint64
	// 一个工作区键可以同时挂多条租约：终端之间允许共存，
	// 用单值 map 的话后来的会把先来的登记覆盖掉，先来的那条一旦被
	// 遗忘，交付就看不见「还有终端在跑」了。
	active   map[string]map[uint64]*codeExecutionLease
	capacity chan struct{}
	// 交付单独一个池子。之前交付和所有会话的 AI 执行共抢同一批槽位，
	// 会话一多，槽位长期被执行占满，交付只能排队到超时——实测这是
	// 交付失败的最大单一原因。交付本身已被仓库租约按仓库串行化，
	// 再挤同一个池子只剩饿死这一个效果。
	deliveryCapacity chan struct{}
	stopping         bool
	stop             chan struct{}
	stopOnce         sync.Once
}

var codeExecutions = newCodeExecutionCoordinator(codeExecutionConcurrency(), codeDeliveryConcurrency())

func codeExecutionConcurrency() int {
	return codeConcurrencyFromEnv("GOPANEL_CODE_MAX_CONCURRENCY", 4)
}

// codeDeliveryConcurrency 是同时进行的交付上限。
// 默认给 2 而不是跟随执行并发：交付期间要跑质量检查，同样吃 CPU，
// 放开太多只会把机器压垮，而按仓库串行化之后 2 个已经够并行不同项目。
func codeDeliveryConcurrency() int {
	return codeConcurrencyFromEnv("GOPANEL_CODE_MAX_DELIVERY_CONCURRENCY", 2)
}

func codeConcurrencyFromEnv(name string, fallback int) int {
	value, err := strconv.Atoi(strings.TrimSpace(os.Getenv(name)))
	if err != nil || value < 1 {
		return fallback
	}
	if value > 32 {
		return 32
	}
	return value
}

func newCodeExecutionCoordinator(capacity, deliveryCapacity int) *codeExecutionCoordinator {
	if capacity < 1 {
		capacity = 1
	}
	if deliveryCapacity < 1 {
		deliveryCapacity = 1
	}
	return &codeExecutionCoordinator{
		active:           make(map[string]map[uint64]*codeExecutionLease),
		capacity:         make(chan struct{}, capacity),
		deliveryCapacity: make(chan struct{}, deliveryCapacity),
		stop:             make(chan struct{}),
	}
}

// slotPoolFor 选出该类执行要占的配额池。
func (coordinator *codeExecutionCoordinator) slotPoolFor(kind string) chan struct{} {
	if kind == codeExecutionDelivery {
		return coordinator.deliveryCapacity
	}
	return coordinator.capacity
}

func codeExecutionWorkspaceKeys(session *model.AIDevSession) []string {
	if session == nil {
		return nil
	}
	paths := []string{session.WorkDir}
	if session.IsolationMode == codeIsolationMultiWorktree {
		if repositories, err := loadCodeSessionRepositories(session.ID); err == nil {
			for _, repository := range repositories {
				paths = append(paths, repository.WorktreeDir)
			}
		}
	}
	if session.IsolationMode != codeIsolationMultiWorktree && session.SourceWorkDir == "" && session.WorktreeBranch == "" && session.ProjectID > 0 {
		if project, err := repo.NewAIProjectRepo().GetProjectByID(session.ProjectID); err == nil {
			paths = append(paths, project.SourceDirs...)
		}
	}
	seen := make(map[string]struct{}, len(paths))
	keys := make([]string, 0, len(paths))
	for _, candidate := range paths {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		resolved, err := filepath.EvalSymlinks(filepath.Clean(candidate))
		if err != nil {
			resolved, err = filepath.Abs(filepath.Clean(candidate))
			if err != nil {
				continue
			}
		}
		resolved = filepath.Clean(resolved)
		if _, exists := seen[resolved]; exists {
			continue
		}
		seen[resolved] = struct{}{}
		keys = append(keys, resolved)
	}
	sort.Strings(keys)
	return keys
}

func codeExecutionDeliveryKeys(session *model.AIDevSession) []string {
	if session == nil {
		return nil
	}
	keys := make([]string, 0)
	if session.IsolationMode == codeIsolationMultiWorktree {
		if repositories, err := loadCodeSessionRepositories(session.ID); err == nil {
			for _, repository := range repositories {
				keys = append(keys, repository.SourceDir)
			}
		}
	} else if strings.TrimSpace(session.SourceWorkDir) != "" {
		keys = append(keys, session.SourceWorkDir)
	}
	return normalizedCodeExecutionKeys(keys)
}

func normalizedCodeExecutionKeys(paths []string) []string {
	seen := make(map[string]struct{}, len(paths))
	keys := make([]string, 0, len(paths))
	for _, candidate := range paths {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		resolved, err := filepath.EvalSymlinks(filepath.Clean(candidate))
		if err != nil {
			resolved, err = filepath.Abs(filepath.Clean(candidate))
			if err != nil {
				continue
			}
		}
		resolved = filepath.Clean(resolved)
		if _, exists := seen[resolved]; exists {
			continue
		}
		seen[resolved] = struct{}{}
		keys = append(keys, resolved)
	}
	sort.Strings(keys)
	return keys
}

func (coordinator *codeExecutionCoordinator) acquire(
	ctx context.Context,
	keys []string,
	kind string,
	wait bool,
	_ bool,
) (*codeExecutionLease, error) {
	return coordinator.acquireOwned(ctx, 0, keys, kind, wait)
}

func (coordinator *codeExecutionCoordinator) acquireSession(
	ctx context.Context,
	session *model.AIDevSession,
	kind string,
	wait bool,
) (*codeExecutionLease, error) {
	if session == nil || session.ID == 0 {
		return nil, errors.New("Code 执行会话无效")
	}
	keys := codeExecutionWorkspaceKeys(session)
	if kind == codeExecutionDelivery {
		keys = codeExecutionDeliveryKeys(session)
	}
	return coordinator.acquireOwned(ctx, session.ID, keys, kind, wait)
}

func (coordinator *codeExecutionCoordinator) acquireOwned(
	ctx context.Context,
	sessionID uint,
	keys []string,
	kind string,
	wait bool,
) (*codeExecutionLease, error) {
	if len(keys) == 0 {
		return nil, errors.New("Code 执行工作区无效")
	}
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		coordinator.mu.Lock()
		if coordinator.stopping {
			coordinator.mu.Unlock()
			return nil, errCodeExecutionStopping
		}
		conflicts := coordinator.conflicts(keys, kind)
		if len(conflicts) == 0 {
			coordinator.nextID++
			lease := &codeExecutionLease{
				coordinator: coordinator,
				id:          coordinator.nextID,
				sessionID:   sessionID,
				kind:        kind,
				keys:        append([]string(nil), keys...),
				done:        make(chan struct{}),
			}
			for _, key := range lease.keys {
				if coordinator.active[key] == nil {
					coordinator.active[key] = make(map[uint64]*codeExecutionLease)
				}
				coordinator.active[key][lease.id] = lease
			}
			coordinator.mu.Unlock()
			if kind == codeExecutionInteractive {
				if err := ctx.Err(); err != nil {
					lease.Release()
					return nil, err
				}
				return lease, nil
			}
			pool := coordinator.slotPoolFor(kind)
			if wait {
				select {
				case pool <- struct{}{}:
					coordinator.mu.Lock()
					lease.slotPool = pool
					coordinator.mu.Unlock()
					if err := ctx.Err(); err != nil {
						lease.Release()
						return nil, err
					}
					return lease, nil
				case <-ctx.Done():
					lease.Release()
					return nil, ctx.Err()
				case <-coordinator.stop:
					lease.Release()
					return nil, errCodeExecutionStopping
				}
			}
			select {
			case pool <- struct{}{}:
				coordinator.mu.Lock()
				lease.slotPool = pool
				coordinator.mu.Unlock()
				if err := ctx.Err(); err != nil {
					lease.Release()
					return nil, err
				}
				return lease, nil
			default:
				lease.Release()
				return nil, errCodeExecutionCapacity
			}
		}
		coordinator.mu.Unlock()
		if !wait {
			return nil, errCodeExecutionBusy
		}
		select {
		case <-conflicts[0].done:
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-coordinator.stop:
			return nil, errCodeExecutionStopping
		}
	}
}

// codeExecutionCoexists 判断两类工作能不能同时占同一个工作区。
//
// 终端之间放行：两个终端在同一个目录各干各的本来就是日常操作，
// 面板在自己的界面里拦一道也挡不住风险——SSH 上去开两个终端跑同一个 CLI
// 谁也拦不住，拦只会让面板比裸终端更难用。风险由开发者自己掌握。
//
// 交付仍然独占：它要把提交合进源仓库，那是一次真正的原子写，
// 并发写坏的是 Git 对象和分支指针，不是「两个人各改各的文件」。
func codeExecutionCoexists(existing, incoming string) bool {
	return existing == codeExecutionInteractive && incoming == codeExecutionInteractive
}

func (coordinator *codeExecutionCoordinator) conflicts(keys []string, kind string) []*codeExecutionLease {
	seen := make(map[uint64]struct{})
	conflicts := make([]*codeExecutionLease, 0)
	for _, key := range keys {
		for _, lease := range coordinator.active[key] {
			if _, exists := seen[lease.id]; exists {
				continue
			}
			if codeExecutionCoexists(lease.kind, kind) {
				continue
			}
			seen[lease.id] = struct{}{}
			conflicts = append(conflicts, lease)
		}
	}
	return conflicts
}

func (lease *codeExecutionLease) SetCancel(cancel context.CancelFunc) {
	if lease == nil || lease.coordinator == nil {
		return
	}
	lease.coordinator.mu.Lock()
	lease.cancel = cancel
	cancelled := lease.cancelled || lease.coordinator.stopping
	lease.coordinator.mu.Unlock()
	if cancelled && cancel != nil {
		cancel()
	}
}

func (coordinator *codeExecutionCoordinator) cancelSessionAndWait(ctx context.Context, sessionID uint) bool {
	return coordinator.cancelSessionKindAndWait(ctx, sessionID, "")
}

func (coordinator *codeExecutionCoordinator) hasSessionKind(sessionID uint, kind string) bool {
	if sessionID == 0 {
		return false
	}
	coordinator.mu.Lock()
	defer coordinator.mu.Unlock()
	return len(coordinator.sessionLeases(sessionID, kind)) > 0
}

func (coordinator *codeExecutionCoordinator) cancelSessionKindAndWait(ctx context.Context, sessionID uint, kind string) bool {
	if sessionID == 0 {
		return false
	}
	coordinator.mu.Lock()
	leases := coordinator.sessionLeases(sessionID, kind)
	coordinator.mu.Unlock()
	for _, lease := range leases {
		coordinator.mu.Lock()
		lease.cancelled = true
		cancel := lease.cancel
		coordinator.mu.Unlock()
		if cancel != nil {
			cancel()
		}
	}
	for _, lease := range leases {
		select {
		case <-lease.done:
		case <-ctx.Done():
			return len(leases) > 0
		}
	}
	return len(leases) > 0
}

func (coordinator *codeExecutionCoordinator) sessionLeases(sessionID uint, kind string) []*codeExecutionLease {
	seen := make(map[uint64]struct{})
	leases := make([]*codeExecutionLease, 0)
	for _, holders := range coordinator.active {
		for _, lease := range holders {
			if lease.sessionID != sessionID || (kind != "" && lease.kind != kind) {
				continue
			}
			if _, exists := seen[lease.id]; exists {
				continue
			}
			seen[lease.id] = struct{}{}
			leases = append(leases, lease)
		}
	}
	return leases
}

func (lease *codeExecutionLease) Release() {
	if lease == nil || lease.coordinator == nil {
		return
	}
	lease.releaseOnce.Do(func() {
		coordinator := lease.coordinator
		coordinator.mu.Lock()
		for _, key := range lease.keys {
			if holders := coordinator.active[key]; holders != nil {
				delete(holders, lease.id)
				if len(holders) == 0 {
					delete(coordinator.active, key)
				}
			}
		}
		slotPool := lease.slotPool
		close(lease.done)
		coordinator.mu.Unlock()
		if slotPool != nil {
			select {
			case <-slotPool:
			default:
			}
		}
	})
}

func (coordinator *codeExecutionCoordinator) shutdown(ctx context.Context) error {
	coordinator.mu.Lock()
	coordinator.stopping = true
	coordinator.stopOnce.Do(func() { close(coordinator.stop) })
	leases := coordinator.conflictsMap()
	coordinator.mu.Unlock()
	for _, lease := range leases {
		coordinator.mu.Lock()
		lease.cancelled = true
		cancel := lease.cancel
		coordinator.mu.Unlock()
		if cancel != nil {
			cancel()
		}
	}
	for _, lease := range leases {
		select {
		case <-lease.done:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

func ShutdownCodeExecutions(ctx context.Context) error {
	return codeExecutions.shutdown(ctx)
}

func (coordinator *codeExecutionCoordinator) conflictsMap() []*codeExecutionLease {
	seen := make(map[uint64]struct{})
	leases := make([]*codeExecutionLease, 0, len(coordinator.active))
	for _, holders := range coordinator.active {
		for _, lease := range holders {
			if _, exists := seen[lease.id]; exists {
				continue
			}
			seen[lease.id] = struct{}{}
			leases = append(leases, lease)
		}
	}
	return leases
}

func (coordinator *codeExecutionCoordinator) isStopping() bool {
	coordinator.mu.Lock()
	defer coordinator.mu.Unlock()
	return coordinator.stopping
}
