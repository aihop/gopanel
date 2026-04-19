package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

type Config struct {
	SocketPath         string   `mapstructure:"socket_path"`
	WindowsPipe        string   `mapstructure:"windows_pipe"`
	BaseDir            string   `mapstructure:"base_dir"`
	GoPanelServiceName string   `mapstructure:"gopanel_service_name"`
	GoPanelBinaryPath  string   `mapstructure:"gopanel_binary_path"`
	GoPanelConfigPath  string   `mapstructure:"gopanel_config_path"`
	GoPanelPidfilePath string   `mapstructure:"gopanel_pidfile_path"`
	ActionTimeoutMs    int      `mapstructure:"action_timeout_ms"`
	LockTimeoutMs      int      `mapstructure:"lock_timeout_ms"`
	FileRoots          []string `mapstructure:"file_roots"`
	AllowRootFS        bool     `mapstructure:"allow_rootfs"`
	MaxFileReadBytes   int64    `mapstructure:"max_file_read_bytes"`
	MaxFileWriteBytes  int64    `mapstructure:"max_file_write_bytes"`
}

var (
	cfg Config
	log *zap.Logger
)

var rootCmd = &cobra.Command{
	Use:          "gpc",
	SilenceUsage: true,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(exitCodeFromErr(err))
	}
}

func exitCodeFromErr(err error) int {
	if err == nil {
		return 0
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "unsupported platform"):
		return 3
	case strings.Contains(msg, "permission denied"):
		return 4
	case strings.Contains(msg, "timeout"):
		return 6
	case strings.Contains(msg, "lock timeout"):
		return 7
	case strings.Contains(msg, "invalid params"):
		return 2
	default:
		return 1
	}
}

func init() {
	cobra.OnInitialize(initConfig, initLogger)

	rootCmd.PersistentFlags().String("base-dir", "", "base dir (align with gopanel system.base_dir)")

	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(helperCmd)
	rootCmd.AddCommand(goPanelCmd)
	rootCmd.AddCommand(serverCmd)
}

func initLogger() {
	l, err := zap.NewProduction()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	log = l
}

func initConfig() {
	baseDir, _ := rootCmd.PersistentFlags().GetString("base-dir")
	baseDir = filepath.Clean(strings.TrimSpace(baseDir))
	if baseDir == "." || baseDir == string(os.PathSeparator) {
		baseDir = ""
	}
	if baseDir == "" {
		baseDir = strings.TrimSpace(os.Getenv("GOPANEL_BASE_DIR"))
		if baseDir == "" {
			baseDir = strings.TrimSpace(os.Getenv("GPC_BASE_DIR"))
		}
		if baseDir != "" {
			baseDir = filepath.Clean(baseDir)
		}
	}
	if baseDir == "" {
		baseDir = detectBaseDir()
	}
	if baseDir == "" {
		baseDir = "/opt/gopanel"
		if runtime.GOOS != "linux" || os.Geteuid() != 0 {
			if homeDir, err := os.UserHomeDir(); err == nil && homeDir != "" {
				baseDir = filepath.Join(homeDir, ".gopanel")
			}
		}
	}
	cfg.BaseDir = baseDir
	cfg.SocketPath = filepath.Join(baseDir, "gpc.sock")
	cfg.WindowsPipe = `\\.\pipe\gopanel-gpc`
	cfg.ActionTimeoutMs = 30000
	cfg.LockTimeoutMs = 30000
	cfg.FileRoots = []string{baseDir}
	cfg.AllowRootFS = false
	cfg.MaxFileReadBytes = int64(1048576)
	cfg.MaxFileWriteBytes = int64(1048576)

	switch runtime.GOOS {
	case "darwin":
		cfg.GoPanelServiceName = "dev.gopanel.server"
	case "windows":
		cfg.GoPanelServiceName = "GoPanel"
	default:
		cfg.GoPanelServiceName = "gopanel.service"
	}

	cfg.GoPanelConfigPath = filepath.Join(baseDir, "conf.yaml")
	cfg.GoPanelPidfilePath = filepath.Join(baseDir, "run", "gopanel.pid")
}

func detectBaseDir() string {
	var candidates []string
	if runtime.GOOS == "linux" {
		candidates = append(candidates, "/opt/gopanel")
	} else {
		if homeDir, err := os.UserHomeDir(); err == nil && homeDir != "" {
			candidates = append(candidates, filepath.Join(homeDir, ".gopanel"))
		}
	}
	if exe, err := os.Executable(); err == nil && exe != "" {
		d := filepath.Dir(exe)
		for i := 0; i < 4; i++ {
			candidates = append(candidates, d)
			nd := filepath.Dir(d)
			if nd == d {
				break
			}
			d = nd
		}
	}
	for _, c := range candidates {
		c = filepath.Clean(strings.TrimSpace(c))
		if c == "" {
			continue
		}
		if isLikelyBaseDir(c) {
			return c
		}
	}
	return ""
}

func isLikelyBaseDir(dir string) bool {
	if dir == "" {
		return false
	}
	st, err := os.Stat(dir)
	if err != nil || !st.IsDir() {
		return false
	}
	if _, err := os.Stat(filepath.Join(dir, "db")); err == nil {
		return true
	}
	if _, err := os.Stat(filepath.Join(dir, "gp-agent")); err == nil {
		return true
	}
	if _, err := os.Stat(filepath.Join(dir, "conf.yaml")); err == nil {
		return true
	}
	if _, err := os.Stat(filepath.Join(dir, "conf.yml")); err == nil {
		return true
	}
	return false
}

func actionTimeout() time.Duration {
	if cfg.ActionTimeoutMs <= 0 {
		return 30 * time.Second
	}
	return time.Duration(cfg.ActionTimeoutMs) * time.Millisecond
}
