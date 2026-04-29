//go:build !windows

package helper

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/aihop/gopanel/gpc/pkg/proto"
)

type Server struct {
	cfg   Config
	mu    sync.Mutex
	locks sync.Map
}

func NewServer(cfg Config) *Server {
	return &Server{cfg: cfg}
}

func (s *Server) Serve(ctx context.Context) error {
	if os.Geteuid() != 0 {
		return errors.New("permission denied: gpc helper must run as root")
	}
	if s.cfg.SocketPath == "" {
		return errors.New("socket_path is empty")
	}

	if err := os.MkdirAll(filepath.Dir(s.cfg.SocketPath), 0755); err != nil {
		return err
	}
	_ = os.RemoveAll(s.cfg.SocketPath)

	l, err := net.Listen("unix", s.cfg.SocketPath)
	if err != nil {
		return err
	}
	defer l.Close()
	go func() {
		<-ctx.Done()
		_ = l.Close()
	}()

	_ = os.Chmod(s.cfg.SocketPath, 0660)
	s.tryChgrpSocketToBaseDir()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		c, err := l.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				time.Sleep(100 * time.Millisecond)
				continue
			}
			return err
		}
		go s.handleConn(c)
	}
}

func (s *Server) tryChgrpSocketToBaseDir() {
	if strings.TrimSpace(s.cfg.BaseDir) == "" {
		return
	}
	bi, err := os.Stat(s.cfg.BaseDir)
	if err != nil {
		return
	}
	st, ok := bi.Sys().(*syscall.Stat_t)
	if !ok {
		return
	}
	_ = os.Chown(s.cfg.SocketPath, 0, int(st.Gid))
}

func (s *Server) handleConn(c net.Conn) {
	defer c.Close()

	uc, ok := c.(*net.UnixConn)
	if ok {
		_ = s.authorize(uc)
	}

	r := bufio.NewReader(c)
	w := bufio.NewWriter(c)
	line, err := r.ReadBytes('\n')
	if err != nil {
		return
	}
	line = bytes.TrimSpace(line)
	if len(line) == 0 {
		return
	}

	var req proto.Request
	if err := json.Unmarshal(line, &req); err != nil {
		_ = writeResp(w, proto.Response{ID: req.ID, OK: false, Code: proto.CodeInvalidParams, Error: err.Error()})
		return
	}

	resp := s.dispatch(req)
	_ = writeResp(w, resp)
}

func (s *Server) authorize(c *net.UnixConn) error {
	return authorizeUnixConn(c)
}

func (s *Server) dispatch(req proto.Request) proto.Response {
	ctx, cancel := context.WithTimeout(context.Background(), s.cfg.ActionTimeout)
	defer cancel()

	lockKey := lockKeyForAction(req.Action)
	if lockKey != "" {
		out, err := s.withLock(ctx, lockKey, func() (string, error) {
			return s.doAction(ctx, req)
		})
		if err != nil {
			return proto.Response{ID: req.ID, OK: false, Code: codeFromErr(err), Error: err.Error()}
		}
		return proto.Response{ID: req.ID, OK: true, Code: proto.CodeOK, Output: out}
	}

	out, err := s.doAction(ctx, req)
	if err != nil {
		return proto.Response{ID: req.ID, OK: false, Code: codeFromErr(err), Error: err.Error()}
	}
	return proto.Response{ID: req.ID, OK: true, Code: proto.CodeOK, Output: out}
}

func (s *Server) doAction(ctx context.Context, req proto.Request) (string, error) {
	switch req.Action {
	case "CHOWN_BASE_DIR":
		return "ok", s.actionChownBaseDir(ctx, req.Params)
	case "ENABLE_FORWARDING":
		return "ok", s.actionEnableForwarding(ctx, req.Params)
	case "RESTART_HOST":
		return "ok", s.actionRestartHost(ctx)
	case "FIREWALL_APPLY":
		out, err := s.actionFirewallApply(ctx, req.Params)
		return out, err
	case "GOPANEL_SERVICE_ACTION":
		out, err := s.actionGoPanelService(ctx, req.Params)
		return out, err
	case "GOPANEL_INFO":
		out, err := s.actionGoPanelInfo(ctx, req.Params)
		return out, err
	case "GOPANEL_UNINSTALL":
		out, err := s.actionGoPanelUninstall(ctx, req.Params)
		return out, err
	case "GOPANEL_USER_INFO":
		out, err := s.actionGoPanelUserInfo(ctx, req.Params)
		return out, err
	case "GOPANEL_AGENT_ENSURE":
		out, err := s.actionGoPanelAgentEnsure(ctx, req.Params)
		return out, err
	case "GOPANEL_AGENT_INSTALL":
		out, err := s.actionGoPanelAgentInstall(ctx, req.Params)
		return out, err
	case "GOPANEL_GPC_INSTALL":
		out, err := s.actionGoPanelGPCInstall(ctx, req.Params)
		return out, err
	case "PODMAN_SOCKET_REPAIR":
		out, err := s.actionPodmanSocketRepair(ctx, req.Params)
		return out, err
	case "REPAIR_PODMAN_SHORT_NAME":
		out, err := s.actionRepairPodmanShortName(ctx, req.Params)
		return out, err
	case "SYSTEMD_ENABLE_LINGER":
		out, err := s.actionSystemdEnableLinger(ctx, req.Params)
		return out, err
	case "REPAIR_PODMAN_SUBUID":
		out, err := s.actionRepairPodmanSubuid(ctx, req.Params)
		return out, err
	case "PODMAN_REGISTRIES_GET":
		out, err := s.actionPodmanRegistriesGet(ctx, req.Params)
		return out, err
	case "PODMAN_REGISTRIES_SET":
		out, err := s.actionPodmanRegistriesSet(ctx, req.Params)
		return out, err
	case "COMPOSE_INSTALL":
		out, err := s.actionComposeInstall(ctx, req.Params)
		return out, err
	case "MYSQL_CLIENT_INSTALL":
		out, err := s.actionMysqlClientInstall(ctx, req.Params)
		return out, err
	case "SECURITY_SCAN_SSH":
		return s.actionSecurityScanSSH(ctx, req.Params)
	case "SECURITY_FIX_SSH":
		return s.actionSecurityFixSSH(ctx, req.Params)
	case "SECURITY_SCAN_PORT":
		return s.actionSecurityScanPort(ctx, req.Params)
	case "SSH_LOGIN_LOG_LIST":
		return s.actionSSHLoginLogList(ctx, req.Params)
	case "PODMAN_CONTAINER_JOURNAL_LOGS":
		return s.actionPodmanContainerJournalLogs(ctx, req.Params)
	case "FILE_STAT":
		return s.actionFileStat(ctx, req.Params)
	case "FILE_LIST":
		return s.actionFileList(ctx, req.Params)
	case "FILE_READ":
		return s.actionFileRead(ctx, req.Params)
	case "FILE_WRITE":
		return s.actionFileWrite(ctx, req.Params)
	case "FILE_MKDIR":
		return s.actionFileMkdir(ctx, req.Params)
	case "FILE_CREATE":
		return s.actionFileCreate(ctx, req.Params)
	case "FILE_REMOVE":
		return s.actionFileRemove(ctx, req.Params)
	case "FILE_CHMOD":
		return s.actionFileChmod(ctx, req.Params)
	case "FILE_CHOWN":
		return s.actionFileChown(ctx, req.Params)
	default:
		return "", errors.New("unknown action")
	}
}

func lockKeyForAction(action string) string {
	switch action {
	case "FIREWALL_APPLY":
		return "firewall"
	case "CHOWN_BASE_DIR":
		return "chown"
	case "GOPANEL_SERVICE_ACTION":
		return "gopanel_service"
	case "GOPANEL_UNINSTALL":
		return "gopanel_uninstall"
	case "GOPANEL_AGENT_ENSURE", "GOPANEL_AGENT_INSTALL":
		return "gp_agent_install"
	case "GOPANEL_GPC_INSTALL":
		return "gpc_install"
	case "PODMAN_SOCKET_REPAIR":
		return "podman_socket_repair"
	case "REPAIR_PODMAN_SHORT_NAME":
		return "repair_podman_short_name"
	case "REPAIR_PODMAN_SUBUID":
		return "repair_podman_subuid"
	case "SYSTEMD_ENABLE_LINGER":
		return "systemd_enable_linger"
	case "COMPOSE_INSTALL":
		return "compose_install"
	case "MYSQL_CLIENT_INSTALL":
		return "mysql_client_install"
	case "RESTART_HOST":
		return "restart_host"
	case "FILE_WRITE", "FILE_MKDIR", "FILE_CREATE", "FILE_REMOVE", "FILE_CHMOD", "FILE_CHOWN":
		return "file_mutate"
	default:
		return ""
	}
}

func (s *Server) withLock(ctx context.Context, key string, fn func() (string, error)) (string, error) {
	ch := make(chan struct{}, 1)
	ch <- struct{}{}
	actual, _ := s.locks.LoadOrStore(key, ch)
	lock := actual.(chan struct{})

	timer := time.NewTimer(s.cfg.LockTimeout)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case <-timer.C:
		return "", errors.New("lock timeout")
	case <-lock:
	}
	defer func() { lock <- struct{}{} }()
	return fn()
}

func writeResp(w *bufio.Writer, resp proto.Response) error {
	b, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	if _, err := w.Write(append(b, '\n')); err != nil {
		return err
	}
	return w.Flush()
}

func codeFromErr(err error) string {
	if err == nil {
		return proto.CodeOK
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return proto.CodeTimeout
	}
	if strings.Contains(strings.ToLower(err.Error()), "permission denied") {
		return proto.CodePermissionDenied
	}
	if strings.Contains(strings.ToLower(err.Error()), "timeout") {
		return proto.CodeTimeout
	}
	if strings.Contains(strings.ToLower(err.Error()), "unknown action") {
		return proto.CodeNotFound
	}
	if strings.Contains(strings.ToLower(err.Error()), "unsupported platform") {
		return proto.CodeUnsupportedPlatform
	}
	return proto.CodeInternal
}
