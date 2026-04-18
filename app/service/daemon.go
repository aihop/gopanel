package service

import (
	"bufio"
	"fmt"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/aihop/gopanel/global"
)

type ProcCfg struct {
	// 基本配置
	Name        string `json:"name" validate:"required"`     // 程序名称 (对应 [program:xxx] 中的 xxx)
	Command     string `json:"command"  validate:"required"` // 要执行的命令
	Directory   string `json:"directory"`                    // 运行前切换到的目录
	ProcessName string `json:"process_name"`                 // 进程名称 (默认为 %(program_name)s)
	NumProcs    int    `json:"numprocs"`                     // 进程数量 (默认为1)
	Priority    int    `json:"priority"`                     // 启动优先级 (数字越小优先级越高)

	// 启动控制
	AutoStart    bool   `json:"autostart"`    // 是否自动启动 (默认为true)
	AutoRestart  string `json:"autorestart"`  // 自动重启策略 (false/true/unexpected)
	StartSecs    int    `json:"startsecs"`    // 启动后多少秒认为启动成功 (默认为1)
	StartRetries int    `json:"startretries"` // 启动失败重试次数 (默认为3)
	ExitCodes    []int  `json:"exitcodes"`    // 被认为是正常退出的退出码 (默认为0,2)

	// 停止控制
	StopWaitSecs int    `json:"stopwaitsecs"` // 发送停止信号后等待的时间(秒)
	StopAsGroup  bool   `json:"stopasgroup"`  // 是否停止整个进程组
	KillAsGroup  bool   `json:"killasgroup"`  // 是否杀死整个进程组
	StopSignal   string `json:"stopsignal"`   // 停止信号 (TERM, HUP, INT等)

	// 日志配置
	StdoutLogfile     string `json:"stdout_logfile"`          // stdout日志文件路径
	StderrLogfile     string `json:"stderr_logfile"`          // stderr日志文件路径
	StdoutLogMaxBytes string `json:"stdout_logfile_maxbytes"` // 日志文件最大大小 (如50MB)
	StdoutLogBackups  int    `json:"stdout_logfile_backups"`  // 保留的日志备份数量
	RedirectStderr    bool   `json:"redirect_stderr"`         // 是否将stderr重定向到stdout

	// 环境和工作目录
	Environment map[string]string `json:"environment"` // 环境变量
	User        string            `json:"user"`        // 运行用户
	Umask       string            `json:"umask"`       // umask值 (如022)

	// 高级选项
	ServerURL     string `json:"serverurl"`     // 覆盖默认的server URL
	Eventlistener bool   `json:"eventlistener"` // 是否为事件监听器
}

// ToConfigString 将 ProcCfg 转换为配置文件字符串
func (p *ProcCfg) ToConfigString() string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("[program:%s]", p.Name))
	builder.WriteString("\n")

	// 基本配置
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

	// 启动控制
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
		codes := make([]string, len(p.ExitCodes))
		for i, code := range p.ExitCodes {
			codes[i] = fmt.Sprintf("%d", code)
		}
		builder.WriteString(fmt.Sprintf("exitcodes=%s\n", strings.Join(codes, ",")))
	}

	// 停止控制
	if p.StopWaitSecs != 0 {
		builder.WriteString(fmt.Sprintf("stopwaitsecs=%d\n", p.StopWaitSecs))
	}
	// 默认为false，不需要设置
	if p.StopAsGroup {
		builder.WriteString("stopasgroup=true\n")
	}
	if p.KillAsGroup {
		builder.WriteString("killasgroup=true\n")
	}
	if p.StopSignal != "" {
		builder.WriteString(fmt.Sprintf("stopsignal=%s\n", p.StopSignal))
	}

	// 日志配置
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

	// 环境和工作目录
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

	// 高级选项
	if p.ServerURL != "" {
		builder.WriteString(fmt.Sprintf("serverurl=%s\n", p.ServerURL))
	}
	if p.Eventlistener {
		builder.WriteString("eventlistener=true\n")
	}

	return builder.String()
}

type DaemonConfigManager struct {
	// 配置文件路径
	FilePath string
}

func NewDaemonConfigManager() *DaemonConfigManager {
	return &DaemonConfigManager{
		FilePath: global.CONF.System.ConfigSupervisorFile,
	}
}

// 从配置文件中读取并解析 Supervisord 进程配置
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

		// 跳过空行和注释
		if line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
			continue
		}

		// 检查是否是新的 [program:xxx] 部分
		if matches := sectionRegex.FindStringSubmatch(line); matches != nil {
			if currentCfg != nil {
				configs = append(configs, currentCfg)
			}
			currentCfg = &ProcCfg{
				Name:      matches[1],
				AutoStart: true, // 默认值
			}
			continue
		}

		// 解析键值对
		if currentCfg != nil {
			if matches := keyValueRegex.FindStringSubmatch(line); matches != nil {
				key := strings.TrimSpace(matches[1])
				value := strings.TrimSpace(matches[2])
				m.parseConfigKeyValue(currentCfg, key, value)
			}
		}
	}

	// 添加最后一个配置
	if currentCfg != nil {
		configs = append(configs, currentCfg)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading config file: %v", err)
	}

	return configs, nil
}

// 添加到配置文件
func (m *DaemonConfigManager) AddConfig(cfg *ProcCfg) error {
	configs, err := m.GetConfig()
	if err != nil {
		return err
	}

	// 检查是否已存在同名配置
	for _, c := range configs {
		if c.Name == cfg.Name {
			return fmt.Errorf("program %s already exists", cfg.Name)
		}
	}

	// 检查一下日志配置,是否设置，如果没设置，默认添加日志配置
	if cfg.StdoutLogfile == "" {
		cfg.StdoutLogfile = path.Join(global.CONF.System.LogPath, fmt.Sprintf("stdout_%s.log", cfg.Name))
		cfg.StdoutLogMaxBytes = "50MB"
	}
	if cfg.StderrLogfile == "" {
		cfg.RedirectStderr = true
	}

	configs = append(configs, cfg)
	return m.saveConfigs(configs)
}

// UpdateConfig 更新现有配置
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

// DeleteConfig 删除配置
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

// saveConfigs 保存所有配置到文件
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

// parseConfigKeyValue 解析单个键值对并设置到配置结构体中
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

// 辅助函数
func parseInt(s string, defaultValue int) int {
	var result int
	_, err := fmt.Sscanf(s, "%d", &result)
	if err != nil {
		return defaultValue
	}
	return result
}

func parseBool(s string, defaultValue bool) bool {
	switch strings.ToLower(s) {
	case "true", "yes", "on", "1":
		return true
	case "false", "no", "off", "0":
		return false
	default:
		return defaultValue
	}
}

func parseIntSlice(s string) []int {
	parts := strings.Split(s, ",")
	var result []int
	for _, part := range parts {
		if num := parseInt(strings.TrimSpace(part), -999); num != -999 {
			result = append(result, num)
		}
	}
	return result
}

func parseEnvironment(s string) map[string]string {
	env := make(map[string]string)
	pairs := strings.Split(s, ",")
	for _, pair := range pairs {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) == 2 {
			env[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
		}
	}
	return env
}
