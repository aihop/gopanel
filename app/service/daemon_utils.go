package service

import (
	"bufio"
	"fmt"
	"github.com/aihop/gopanel/global"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type ProcCfg struct {
	Name         string `json:"name" validate:"required"`
	Command      string `json:"command"  validate:"required"`
	Directory    string `json:"directory"`
	ProcessName  string `json:"process_name"`
	NumProcs     int    `json:"numprocs"`
	Priority     int    `json:"priority"`
	AutoStart    bool   `json:"autostart"`
	AutoRestart  string `json:"autorestart"`
	StartSecs    int    `json:"startsecs"`
	StartRetries int    `json:"startretries"`
	ExitCodes    []int  `json:"exitcodes"` // 基本配置
	// 启动失败重试次数 (默认为3)

	StopWaitSecs      int    `json:"stopwaitsecs"`
	StopAsGroup       bool   `json:"stopasgroup"`
	KillAsGroup       bool   `json:"killasgroup"`
	StopSignal        string `json:"stopsignal"`
	StdoutLogfile     string `json:"stdout_logfile"`
	StderrLogfile     string `json:"stderr_logfile"`
	StdoutLogMaxBytes string `json:"stdout_logfile_maxbytes"`
	StdoutLogBackups  int    `json:"stdout_logfile_backups"`
	RedirectStderr    bool   `json:"redirect_stderr"`
	Environment       map[   // 被认为是正常退出的退出码 (默认为0,2)
	// 环境和工作目录
	string]string `json:"environment"`
	User          string `json:"user"`
	Umask         string `json:"umask"`
	ServerURL     string `json:"serverurl"`
	Eventlistener bool   `json:"eventlistener"`
}

func (p *ProcCfg) ToConfigString() string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("[program:%s]", p.Name))
	builder.WriteString("\n")
	if p.Command != "" {
		builder.WriteString(fmt.Sprintf("command=%s\n", p.Command))
	}
	if p.Directory != "" {
		builder.WriteString(fmt.Sprintf("directory=%s\n", p.Directory))
	}
	if p.NumProcs != 0 {
		builder.WriteString(fmt.Sprintf("numprocs=%d\n", p.NumProcs))
		if p.NumProcs > 1 {
			builder.WriteString("process_name=%(program_name)s_%(process_num)s\n")
		}
	}
	if p.Priority != 0 {
		builder.WriteString(fmt.Sprintf("priority=%d\n", p.Priority))
	}
	builder.WriteString(fmt.Sprintf("autostart=%v\n", p.AutoStart))
	if p.AutoRestart != "" {
		builder.WriteString(fmt.Sprintf("autorestart=%s\n", p.AutoRestart))
	}
	if p.StartSecs != 0 {
		builder.WriteString(fmt.Sprintf("startsecs=%d\n", p.StartSecs))
	}
	if p.StartRetries != 0 {
		builder.WriteString(fmt.Sprintf("startretries=%d\n", p.StartRetries))
	}
	if len(p.ExitCodes) > 0 {
		codes := make([]string, // 环境变量
			// 是否为事件监听器
			len(p.ExitCodes))
		for i, code := range p.ExitCodes {
			codes[i] = fmt.Sprintf("%d", code)
		}
		builder.WriteString(fmt.Sprintf("exitcodes=%s\n", strings.Join(codes, ",")))
	}
	if p.StopWaitSecs != 0 {
		builder.WriteString(fmt.Sprintf("stopwaitsecs=%d\n", p.StopWaitSecs))
	}
	if p.StopAsGroup {
		builder.WriteString("stopasgroup=true\n")
	}
	if p.KillAsGroup {
		builder.WriteString("killasgroup=true\n")
	}
	if p.StopSignal != "" {
		builder.WriteString(fmt.Sprintf("stopsignal=%s\n", p.StopSignal))
	}
	if p.StdoutLogfile != "" {
		builder.WriteString(fmt.Sprintf("stdout_logfile=%s\n", p.StdoutLogfile))
	}
	if p.StderrLogfile != "" {
		builder.WriteString(fmt.Sprintf("stderr_logfile=%s\n", p.StderrLogfile))
	}
	if p.StdoutLogMaxBytes != "" {
		builder.WriteString(fmt.Sprintf("stdout_logfile_maxbytes=%s\n", p.StdoutLogMaxBytes))
	}
	if p.StdoutLogBackups != 0 {
		builder.WriteString(fmt.Sprintf("stdout_logfile_backups=%d\n", p.StdoutLogBackups))
	}
	if p.RedirectStderr {
		builder.WriteString("redirect_stderr=true\n")
	}
	if len(p.Environment) > 0 {
		envVars := make([]string, 0, len(p.Environment))
		for k, v := range p.Environment {
			envVars = append(envVars, fmt.Sprintf("%s=%s", k, v))
		}
		builder.WriteString(fmt.Sprintf("environment=%s\n", strings.Join(envVars, ",")))
	}
	if p.User != "" {
		builder.WriteString(fmt.Sprintf("user=%s\n", p.User))
	}
	if p.Umask != "" {
		builder.WriteString(fmt.Sprintf("umask=%s\n", p.Umask))
	}
	if p.ServerURL != "" {
		builder.WriteString(fmt.Sprintf("serverurl=%s\n", p.ServerURL))
	}
	if p.Eventlistener {
		builder.WriteString("eventlistener=true\n")
	}
	return builder.String()
}

type DaemonConfigManager struct{ FilePath string }

func NewDaemonConfigManager() *DaemonConfigManager {
	return &DaemonConfigManager{FilePath: filepath.Join(global.CONF.System.BaseDir, "supervisord.ini")}
}
func (m *DaemonConfigManager) GetConfig() ([]*ProcCfg, error) {
	file, err := os.Open(m.FilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %v", err)
	}
	defer file.Close()
	var configs []*ProcCfg
	var currentCfg *ProcCfg
	scanner := bufio.NewScanner(file)
	sectionRegex := regexp.MustCompile(`^\[program:([^\]]+)\]$`)
	keyValueRegex := regexp.MustCompile(`^([^=]+)\s*=\s*(.*)$`)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
			continue
		}
		if matches := sectionRegex.FindStringSubmatch(line); matches != nil {
			if currentCfg != nil {
				configs = append(configs, currentCfg)
			}
			currentCfg = &ProcCfg{Name: matches[1], AutoStart: true}
			continue
		}
		if currentCfg != nil {
			if matches := keyValueRegex.FindStringSubmatch(line); matches != nil {
				key := strings.TrimSpace(matches[1])
				value := strings.TrimSpace(matches[2])
				m.parseConfigKeyValue(currentCfg, key, value)
			}
		}
	}
	if currentCfg != nil {
		configs = append(configs, currentCfg)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading config file: %v", err)
	}
	return configs, nil
}
func (m *DaemonConfigManager) AddConfig(cfg *ProcCfg) error {
	configs, err := m.GetConfig()
	if err != nil {
		return err
	}
	for _, c := range configs {
		if c.Name == cfg.Name {
			return fmt.Errorf("program %s already exists", cfg.Name)
		}
	}
	if cfg.StdoutLogfile == "" {
		cfg.StdoutLogfile = filepath.Join(global.CONF.System.LogPath, fmt.Sprintf("stdout_%s.log", cfg.Name))
		cfg.StdoutLogMaxBytes = "50MB"
	}
	if cfg.StderrLogfile == "" {
		cfg.RedirectStderr = true
	}
	configs = append(configs, cfg)
	return m.saveConfigs(configs)
}
func (m *DaemonConfigManager) UpdateConfig(cfg *ProcCfg) error {
	configs, err := m.GetConfig()
	if err != nil {
		return err
	}
	found := false
	for i, c := range configs {
		if c.Name == cfg.Name {
			configs[i] = cfg
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("program %s not found", cfg.Name)
	}
	return m.saveConfigs(configs)
}
func (m *DaemonConfigManager) DeleteConfig(name string) error {
	configs, err := m.GetConfig()
	if err != nil {
		return err
	}
	newConfigs := make([]*ProcCfg, 0, len(configs))
	found := false
	for _, c := range configs {
		if c.Name == name {
			found = true
		} else {
			newConfigs = append(newConfigs, c)
		}
	}
	if !found {
		return fmt.Errorf("program %s not found", name)
	}
	return m.saveConfigs(newConfigs)
}
func (m *DaemonConfigManager) saveConfigs(configs []*ProcCfg) error {
	file, err := os.Create(m.FilePath)
	if err != nil {
		return fmt.Errorf("failed to create config file: %v", err)
	}
	defer file.Close()
	writer := bufio.NewWriter(file)
	for _, cfg := range configs {
		_, err := writer.WriteString(cfg.ToConfigString() + "\n\n")
		if err != nil {
			return fmt.Errorf("failed to write config: %v", err)
		}
	}
	return writer.Flush()
}
func (m *DaemonConfigManager) parseConfigKeyValue(cfg *ProcCfg, key, value string) {
	switch key {
	case "command":
		cfg.Command = value
	case "directory":
		cfg.Directory = value
	case "process_name":
		cfg.ProcessName = value
	case "numprocs":
		cfg.NumProcs = parseInt(value, 1)
	case "priority":
		cfg.Priority = parseInt(value, 999)
	case "autostart":
		cfg.AutoStart = parseBool(value, true)
	case "autorestart":
		cfg.AutoRestart = value
	case "startsecs":
		cfg.StartSecs = parseInt(value, 1)
	case "startretries":
		cfg.StartRetries = parseInt(value, 3)
	case "exitcodes":
		cfg.ExitCodes = parseIntSlice(value)
	case "stopwaitsecs":
		cfg.StopWaitSecs = parseInt(value, 10)
	case "stopasgroup":
		cfg.StopAsGroup = parseBool(value, false)
	case "killasgroup":
		cfg.KillAsGroup = parseBool(value, false)
	case "stopsignal":
		cfg.StopSignal = value
	case "stdout_logfile":
		cfg.StdoutLogfile = value
	case "stderr_logfile":
		cfg.StderrLogfile = value
	case "stdout_logfile_maxbytes":
		cfg.StdoutLogMaxBytes = value
	case "stdout_logfile_backups":
		cfg.StdoutLogBackups = parseInt(value, 0)
	case "redirect_stderr":
		cfg.RedirectStderr = parseBool(value, false)
	case "environment":
		cfg.Environment = parseEnvironment(value)
	case "user":
		cfg.User = value
	case "umask":
		cfg.Umask = value
	case "serverurl":
		cfg.ServerURL = value
	case "eventlistener":
		cfg.Eventlistener = parseBool(value, false)
	}
}
