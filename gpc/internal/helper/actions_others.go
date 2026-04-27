//go:build !linux && !windows && !darwin

package helper

import (
	"context"
	"errors"
)

func (s *Server) actionChownBaseDir(ctx context.Context, params map[string]interface{}) error {
	_ = ctx
	_ = params
	return errors.New("unsupported platform")
}

func (s *Server) actionEnableForwarding(ctx context.Context, params map[string]interface{}) error {
	_ = ctx
	_ = params
	return errors.New("unsupported platform")
}

func (s *Server) actionRestartHost(ctx context.Context) error {
	_ = ctx
	return errors.New("unsupported platform")
}

func (s *Server) actionFirewallApply(ctx context.Context, params map[string]interface{}) (string, error) {
	_ = ctx
	_ = params
	return "", errors.New("unsupported platform")
}

func (s *Server) actionGoPanelService(ctx context.Context, params map[string]interface{}) (string, error) {
	_ = ctx
	_ = params
	return "", errors.New("unsupported platform")
}

func (s *Server) actionGoPanelInfo(ctx context.Context, params map[string]interface{}) (string, error) {
	_ = ctx
	_ = params
	return "", errors.New("unsupported platform")
}

func (s *Server) actionPodmanSocketRepair(ctx context.Context, params map[string]interface{}) (string, error) {
	_ = ctx
	_ = params
	return "", errors.New("unsupported platform")
}

func (s *Server) actionRepairPodmanShortName(ctx context.Context, params map[string]interface{}) (string, error) {
	_ = ctx
	_ = params
	return "", errors.New("unsupported platform")
}

func (s *Server) actionSystemdEnableLinger(ctx context.Context, params map[string]interface{}) (string, error) {
	_ = ctx
	_ = params
	return "", errors.New("unsupported platform")
}

func (s *Server) actionPodmanRegistriesGet(ctx context.Context, params map[string]interface{}) (string, error) {
	_ = ctx
	_ = params
	return "", errors.New("unsupported platform")
}

func (s *Server) actionPodmanRegistriesSet(ctx context.Context, params map[string]interface{}) (string, error) {
	_ = ctx
	_ = params
	return "", errors.New("unsupported platform")
}

func (s *Server) actionComposeInstall(ctx context.Context, params map[string]interface{}) (string, error) {
	_ = ctx
	_ = params
	return "", errors.New("unsupported platform")
}

func (s *Server) actionRepairPodmanSubuid(ctx context.Context, params map[string]interface{}) (string, error) {
	_ = ctx
	_ = params
	return "", errors.New("unsupported platform")
}

func (s *Server) actionSecurityScanSSH(ctx context.Context, params map[string]interface{}) (string, error) {
	return "", errors.New("unsupported platform")
}

func (s *Server) actionSecurityFixSSH(ctx context.Context, params map[string]interface{}) (string, error) {
	return "", errors.New("unsupported platform")
}

func (s *Server) actionSecurityScanPort(ctx context.Context, params map[string]interface{}) (string, error) {
	return "", errors.New("unsupported platform")
}

func (s *Server) actionSSHLoginLogList(ctx context.Context, params map[string]interface{}) (string, error) {
	_ = ctx
	_ = params
	return "", errors.New("unsupported platform")
}

func (s *Server) actionPodmanContainerJournalLogs(ctx context.Context, params map[string]interface{}) (string, error) {
	_ = ctx
	_ = params
	return "", errors.New("unsupported platform")
}
