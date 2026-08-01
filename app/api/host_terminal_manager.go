package api

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
)

type hostTerminal struct {
	mu               sync.Mutex
	record           *model.HostTerminalSession
	command          *exec.Cmd
	pty              nativeTerminal
	sequence         uint64
	history          []byte
	historyTruncated bool
	subscribers      map[string]*hostTerminalSubscription
	controllerID     string
	controlExpiresAt time.Time
	controlTimer     *time.Timer
	stopRequested    bool
	finished         bool
	done             chan struct{}
	readDone         chan struct{}
}

type hostTerminalManager struct {
	mu       sync.Mutex
	sessions map[uint]*hostTerminal
}

var (
	hostTerminals             = &hostTerminalManager{sessions: make(map[uint]*hostTerminal)}
	hostTerminalSubscriberSeq uint64
)

func (manager *hostTerminalManager) create(req createHostTerminalRequest, userID uint, ip string) (*model.HostTerminalSession, error) {
	workDir, err := resolveHostTerminalWorkDir(req.WorkDir)
	if err != nil {
		return nil, err
	}
	codeHostTerminalLifecycle.Lock()
	defer codeHostTerminalLifecycle.Unlock()
	if err := validateHostTerminalDevelopmentOpen(workDir); err != nil {
		return nil, err
	}
	command, shellName, err := buildHostTerminalCommand(req.Shell, workDir)
	if err != nil {
		return nil, err
	}
	configureNativeTerminalEnvironment(command)
	if req.Cols == 0 {
		req.Cols = 120
	}
	if req.Rows == 0 {
		req.Rows = 32
	}
	record := &model.HostTerminalSession{
		UserID: userID, Status: "starting", Shell: shellName, WorkDir: workDir,
		ClientIP: ip, StartedAt: time.Now(),
	}
	if err := global.DB.Create(record).Error; err != nil {
		return nil, err
	}
	pty, err := startNativeTerminal(command, req.Cols, req.Rows)
	if err != nil {
		manager.failStart(record, err)
		return nil, err
	}
	record.Status = "running"
	record.PID = command.Process.Pid
	if err := global.DB.Model(record).Updates(map[string]any{"status": record.Status, "pid": record.PID}).Error; err != nil {
		_ = stopHostTerminalProcess(command)
		_ = pty.Close()
		_, _ = command.Process.Wait()
		return nil, err
	}
	session := &hostTerminal{
		record: record, command: command, pty: pty,
		subscribers: make(map[string]*hostTerminalSubscription), done: make(chan struct{}), readDone: make(chan struct{}),
	}
	manager.mu.Lock()
	manager.sessions[record.ID] = session
	manager.mu.Unlock()
	go session.readOutput()
	go session.wait(manager)
	recordHostTerminalAudit(record.ID, userID, "create", "success", ip, fmt.Sprintf("shell=%s workDir=%s", shellName, workDir))
	return record, nil
}

func (manager *hostTerminalManager) resume(id uint) (*model.HostTerminalSession, error) {
	session := manager.get(id)
	if session == nil {
		return nil, errors.New("终端进程已结束或服务已重启")
	}
	session.mu.Lock()
	record := *session.record
	finished := session.finished
	session.mu.Unlock()
	if finished {
		return nil, errors.New("终端进程已结束或服务已重启")
	}
	return &record, nil
}

func resolveHostTerminalWorkDir(requested string) (string, error) {
	requested = strings.TrimSpace(requested)
	if requested == "" {
		var err error
		requested, err = os.UserHomeDir()
		if err != nil {
			requested = "."
		}
	}
	requested, err := filepath.Abs(filepath.Clean(requested))
	if err != nil {
		return "", errors.New("终端工作目录无效")
	}
	if resolvedPath, resolveErr := filepath.EvalSymlinks(requested); resolveErr == nil {
		requested = resolvedPath
	}
	resolved, err := os.Stat(requested)
	if err != nil || !resolved.IsDir() {
		return "", errors.New("终端工作目录不存在或不可访问")
	}
	return requested, nil
}

func (manager *hostTerminalManager) failStart(record *model.HostTerminalSession, startErr error) {
	now := time.Now()
	message := truncateHostTerminalDetail(startErr.Error())
	_ = global.DB.Model(record).Updates(map[string]any{"status": "failed", "ended_at": now, "error_message": message}).Error
	recordHostTerminalAudit(record.ID, record.UserID, "create", "failed", record.ClientIP, message)
}

func (session *hostTerminal) readOutput() {
	defer close(session.readDone)
	buffer := make([]byte, 32*1024)
	for {
		count, err := session.pty.Read(buffer)
		if count > 0 {
			session.publish(buffer[:count])
		}
		if err != nil {
			if !errors.Is(err, io.EOF) && !errors.Is(err, os.ErrClosed) {
				global.LOG.Errorf("Read host terminal %d failed: %v", session.record.ID, err)
			}
			return
		}
	}
}

func (session *hostTerminal) publish(output []byte) {
	session.mu.Lock()
	session.sequence++
	session.history = append(session.history, output...)
	if len(session.history) > hostTerminalHistoryLimit {
		session.historyTruncated = true
		session.history = append([]byte(nil), session.history[len(session.history)-hostTerminalHistoryLimit:]...)
	}
	session.record.OutputBytes += int64(len(output))
	event := hostTerminalEvent{Type: "output", Sequence: session.sequence, Data: string(output)}
	for _, subscriber := range session.subscribers {
		select {
		case subscriber.Events <- event:
		default:
			drainHostTerminalEvents(subscriber.Events)
			subscriber.Events <- hostTerminalEvent{Type: "resync_required", Sequence: session.sequence}
		}
	}
	session.mu.Unlock()
}

func drainHostTerminalEvents(events chan hostTerminalEvent) {
	for {
		select {
		case <-events:
		default:
			return
		}
	}
}

func (session *hostTerminal) wait(manager *hostTerminalManager) {
	err := session.command.Wait()
	_ = session.pty.Close()
	<-session.readDone
	now := time.Now()
	exitCode := 0
	if session.command.ProcessState != nil {
		exitCode = session.command.ProcessState.ExitCode()
	}
	session.mu.Lock()
	status := "exited"
	if session.stopRequested {
		status = "stopped"
	} else if err != nil {
		status = "failed"
	}
	session.record.Status, session.record.ExitCode, session.record.EndedAt = status, exitCode, &now
	session.finished = true
	if err != nil && !session.stopRequested {
		session.record.ErrorMessage = truncateHostTerminalDetail(err.Error())
	}
	if session.controlTimer != nil {
		session.controlTimer.Stop()
	}
	session.mu.Unlock()
	_ = global.DB.Model(session.record).Updates(map[string]any{
		"status": status, "exit_code": exitCode, "ended_at": now,
		"output_bytes": session.record.OutputBytes, "error_message": session.record.ErrorMessage,
	}).Error
	manager.mu.Lock()
	delete(manager.sessions, session.record.ID)
	manager.mu.Unlock()
	session.mu.Lock()
	for id, subscriber := range session.subscribers {
		select {
		case subscriber.Events <- hostTerminalEvent{Type: "closed", Sequence: session.sequence}:
		default:
		}
		delete(session.subscribers, id)
		close(subscriber.Events)
	}
	session.mu.Unlock()
	recordHostTerminalAudit(session.record.ID, session.record.UserID, "exit", status, session.record.ClientIP, session.record.ErrorMessage)
	close(session.done)
}

func (manager *hostTerminalManager) get(id uint) *hostTerminal {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	return manager.sessions[id]
}

func (manager *hostTerminalManager) stop(id uint) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return manager.stopAndWait(ctx, id)
}

func (manager *hostTerminalManager) stopAndWait(ctx context.Context, id uint) bool {
	session := manager.get(id)
	if session == nil {
		return false
	}
	session.mu.Lock()
	if session.finished {
		session.mu.Unlock()
		return true
	}
	session.stopRequested = true
	process := session.command.Process
	session.mu.Unlock()
	if process == nil {
		return false
	}
	_ = stopHostTerminalProcess(session.command)
	select {
	case <-session.done:
		return true
	case <-ctx.Done():
		return false
	}
}

func (session *hostTerminal) subscribe(userID uint, ip string, readOnly bool) (*hostTerminalSubscription, hostTerminalEvent) {
	subscriber := &hostTerminalSubscription{ID: nextHostTerminalSubscriberID(), Events: make(chan hostTerminalEvent, 128), UserID: userID, IP: ip}
	session.mu.Lock()
	session.subscribers[subscriber.ID] = subscriber
	now := time.Now()
	if !readOnly && (session.controllerID == "" || !now.Before(session.controlExpiresAt)) {
		session.controllerID = subscriber.ID
		session.renewControlLocked(now)
	}
	baseline := hostTerminalEvent{
		Type: "baseline", Sequence: session.sequence, Data: string(session.history),
		HasControl: session.controllerID == subscriber.ID, Truncated: session.historyTruncated,
		LeaseExpiresAt: session.controlExpiresAt.UnixMilli(),
	}
	session.mu.Unlock()
	return subscriber, baseline
}

func (session *hostTerminal) unsubscribe(subscriber *hostTerminalSubscription) {
	if subscriber == nil {
		return
	}
	session.mu.Lock()
	registered := session.subscribers[subscriber.ID]
	if registered != nil {
		delete(session.subscribers, subscriber.ID)
		close(registered.Events)
	}
	controlChanged := session.controllerID == subscriber.ID
	if controlChanged {
		session.controllerID = ""
		session.controlExpiresAt = time.Time{}
		if session.controlTimer != nil {
			session.controlTimer.Stop()
			session.controlTimer = nil
		}
	}
	session.mu.Unlock()
	if controlChanged {
		session.broadcastControl()
	}
}

func (session *hostTerminal) takeControl(subscriberID string) (bool, string) {
	session.mu.Lock()
	now := time.Now()
	if session.subscribers[subscriberID] == nil {
		session.mu.Unlock()
		return false, "连接不存在"
	}
	if session.controllerID != "" && session.controllerID != subscriberID && now.Before(session.controlExpiresAt) {
		session.mu.Unlock()
		return false, "其他设备正在控制终端"
	}
	session.controllerID = subscriberID
	session.renewControlLocked(now)
	session.mu.Unlock()
	session.broadcastControl()
	return true, ""
}

func (session *hostTerminal) releaseControl(subscriberID string) bool {
	session.mu.Lock()
	if session.controllerID != subscriberID {
		session.mu.Unlock()
		return false
	}
	session.controllerID = ""
	session.controlExpiresAt = time.Time{}
	if session.controlTimer != nil {
		session.controlTimer.Stop()
		session.controlTimer = nil
	}
	session.mu.Unlock()
	session.broadcastControl()
	return true
}

func (session *hostTerminal) write(subscriberID string, data []byte) error {
	session.mu.Lock()
	defer session.mu.Unlock()
	if session.controllerID != subscriberID || !time.Now().Before(session.controlExpiresAt) {
		return errors.New("当前连接没有终端输入权")
	}
	session.renewControlLocked(time.Now())
	_, err := session.pty.Write(data)
	return err
}

func (session *hostTerminal) resize(subscriberID string, cols, rows uint16) error {
	session.mu.Lock()
	allowed := session.controllerID == subscriberID && time.Now().Before(session.controlExpiresAt)
	if allowed {
		session.renewControlLocked(time.Now())
	}
	session.mu.Unlock()
	if !allowed {
		return errors.New("当前连接没有终端控制权")
	}
	return session.pty.Resize(cols, rows)
}

func (session *hostTerminal) renewControlLocked(now time.Time) {
	session.controlExpiresAt = now.Add(hostTerminalControlLease)
	if session.controlTimer != nil {
		session.controlTimer.Stop()
	}
	controllerID := session.controllerID
	session.controlTimer = time.AfterFunc(hostTerminalControlLease, func() {
		session.mu.Lock()
		if session.controllerID != controllerID || time.Now().Before(session.controlExpiresAt) {
			session.mu.Unlock()
			return
		}
		session.controllerID = ""
		session.controlExpiresAt = time.Time{}
		session.controlTimer = nil
		session.mu.Unlock()
		session.broadcastControl()
	})
}

func (session *hostTerminal) broadcastControl() {
	session.mu.Lock()
	defer session.mu.Unlock()
	for id, subscriber := range session.subscribers {
		event := hostTerminalEvent{Type: "control", Sequence: session.sequence, HasControl: session.controllerID == id, LeaseExpiresAt: session.controlExpiresAt.UnixMilli()}
		select {
		case subscriber.Events <- event:
		default:
		}
	}
}

func nextHostTerminalSubscriberID() string {
	return fmt.Sprintf("host-%d", atomic.AddUint64(&hostTerminalSubscriberSeq, 1))
}
