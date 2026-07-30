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
	codeExecutionQuality     = "quality"
)

var (
	errCodeExecutionBusy     = errors.New("当前工作区已有任务正在执行")
	errCodeExecutionCapacity = errors.New("Code 执行并发已满，请稍后重试")
	errCodeExecutionStopping = errors.New("Code 执行服务正在停止")
)

type codeExecutionLease struct {
	coordinator  *codeExecutionCoordinator
	id           uint64
	kind         string
	keys         []string
	done         chan struct{}
	releaseOnce  sync.Once
	cancel       context.CancelFunc
	cancelled    bool
	slotAcquired bool
}

type codeExecutionCoordinator struct {
	mu       sync.Mutex
	nextID   uint64
	active   map[string]*codeExecutionLease
	capacity chan struct{}
	stopping bool
	stop     chan struct{}
	stopOnce sync.Once
}

var codeExecutions = newCodeExecutionCoordinator(codeExecutionConcurrency())

func codeExecutionConcurrency() int {
	value, err := strconv.Atoi(strings.TrimSpace(os.Getenv("GOPANEL_CODE_MAX_CONCURRENCY")))
	if err != nil || value < 1 {
		return 4
	}
	if value > 32 {
		return 32
	}
	return value
}

func newCodeExecutionCoordinator(capacity int) *codeExecutionCoordinator {
	if capacity < 1 {
		capacity = 1
	}
	return &codeExecutionCoordinator{
		active:   make(map[string]*codeExecutionLease),
		capacity: make(chan struct{}, capacity),
		stop:     make(chan struct{}),
	}
}

func codeExecutionWorkspaceKeys(session *model.AIDevSession) []string {
	if session == nil {
		return nil
	}
	paths := []string{session.WorkDir}
	if session.SourceWorkDir == "" && session.WorktreeBranch == "" && session.ProjectID > 0 {
		if project, err := repo.NewAIGroupRepo().GetGroupByID(session.ProjectID); err == nil {
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

func (coordinator *codeExecutionCoordinator) acquire(
	ctx context.Context,
	keys []string,
	kind string,
	wait bool,
	preemptInteractive bool,
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
		conflicts := coordinator.conflicts(keys)
		if len(conflicts) == 0 {
			coordinator.nextID++
			lease := &codeExecutionLease{
				coordinator: coordinator,
				id:          coordinator.nextID,
				kind:        kind,
				keys:        append([]string(nil), keys...),
				done:        make(chan struct{}),
			}
			for _, key := range lease.keys {
				coordinator.active[key] = lease
			}
			coordinator.mu.Unlock()
			if kind == codeExecutionInteractive {
				if err := ctx.Err(); err != nil {
					lease.Release()
					return nil, err
				}
				return lease, nil
			}
			if wait {
				select {
				case coordinator.capacity <- struct{}{}:
					coordinator.mu.Lock()
					lease.slotAcquired = true
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
			case coordinator.capacity <- struct{}{}:
				coordinator.mu.Lock()
				lease.slotAcquired = true
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
		if preemptInteractive {
			for _, conflict := range conflicts {
				conflict.CancelIfInteractive()
			}
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

func (coordinator *codeExecutionCoordinator) conflicts(keys []string) []*codeExecutionLease {
	seen := make(map[uint64]struct{})
	conflicts := make([]*codeExecutionLease, 0)
	for _, key := range keys {
		lease := coordinator.active[key]
		if lease == nil {
			continue
		}
		if _, exists := seen[lease.id]; exists {
			continue
		}
		seen[lease.id] = struct{}{}
		conflicts = append(conflicts, lease)
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

func (coordinator *codeExecutionCoordinator) cancelAndWait(ctx context.Context, keys []string) bool {
	coordinator.mu.Lock()
	leases := coordinator.conflicts(keys)
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

func (lease *codeExecutionLease) CancelIfInteractive() {
	if lease == nil || lease.kind != codeExecutionInteractive || lease.coordinator == nil {
		return
	}
	lease.coordinator.mu.Lock()
	lease.cancelled = true
	cancel := lease.cancel
	lease.coordinator.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (lease *codeExecutionLease) Release() {
	if lease == nil || lease.coordinator == nil {
		return
	}
	lease.releaseOnce.Do(func() {
		coordinator := lease.coordinator
		coordinator.mu.Lock()
		for _, key := range lease.keys {
			if coordinator.active[key] == lease {
				delete(coordinator.active, key)
			}
		}
		slotAcquired := lease.slotAcquired
		close(lease.done)
		coordinator.mu.Unlock()
		if slotAcquired {
			select {
			case <-coordinator.capacity:
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
	for _, lease := range coordinator.active {
		if _, exists := seen[lease.id]; exists {
			continue
		}
		seen[lease.id] = struct{}{}
		leases = append(leases, lease)
	}
	return leases
}

func (coordinator *codeExecutionCoordinator) isStopping() bool {
	coordinator.mu.Lock()
	defer coordinator.mu.Unlock()
	return coordinator.stopping
}
