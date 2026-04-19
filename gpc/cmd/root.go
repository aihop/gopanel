package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
	cfgFile string
	cfg     Config
	log     *zap.Logger
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

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file")
	rootCmd.PersistentFlags().String("socket-path", "", "unix domain socket path (linux/macos)")
	rootCmd.PersistentFlags().String("windows-pipe", "", "windows named pipe")
	rootCmd.PersistentFlags().String("base-dir", "", "base dir (align with gopanel system.base_dir)")
	rootCmd.PersistentFlags().String("gopanel-service-name", "", "gopanel service name/label")
	rootCmd.PersistentFlags().String("gopanel-binary-path", "", "gopanel binary path (fallback mode)")
	rootCmd.PersistentFlags().String("gopanel-config-path", "", "gopanel config path (for info)")
	rootCmd.PersistentFlags().String("gopanel-pidfile-path", "", "gopanel pidfile path (fallback mode)")
	rootCmd.PersistentFlags().Int("action-timeout-ms", 0, "action timeout in ms")
	rootCmd.PersistentFlags().Int("lock-timeout-ms", 0, "lock timeout in ms")

	_ = viper.BindPFlag("gpc.socket_path", rootCmd.PersistentFlags().Lookup("socket-path"))
	_ = viper.BindPFlag("gpc.windows_pipe", rootCmd.PersistentFlags().Lookup("windows-pipe"))
	_ = viper.BindPFlag("gpc.base_dir", rootCmd.PersistentFlags().Lookup("base-dir"))
	_ = viper.BindPFlag("gpc.gopanel_service_name", rootCmd.PersistentFlags().Lookup("gopanel-service-name"))
	_ = viper.BindPFlag("gpc.gopanel_binary_path", rootCmd.PersistentFlags().Lookup("gopanel-binary-path"))
	_ = viper.BindPFlag("gpc.gopanel_config_path", rootCmd.PersistentFlags().Lookup("gopanel-config-path"))
	_ = viper.BindPFlag("gpc.gopanel_pidfile_path", rootCmd.PersistentFlags().Lookup("gopanel-pidfile-path"))
	_ = viper.BindPFlag("gpc.action_timeout_ms", rootCmd.PersistentFlags().Lookup("action-timeout-ms"))
	_ = viper.BindPFlag("gpc.lock_timeout_ms", rootCmd.PersistentFlags().Lookup("lock-timeout-ms"))

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
	viper.SetEnvPrefix("GPC")
	viper.AutomaticEnv()

	defaultBaseDir := ""
	homeDir, _ := os.UserHomeDir()
	switch runtime.GOOS {
	case "windows":
		if homeDir != "" {
			defaultBaseDir = filepath.Join(homeDir, ".gopanel")
		}
	default:
		if homeDir != "" {
			defaultBaseDir = filepath.Join(homeDir, ".gopanel")
		}
	}

	viper.SetDefault("gpc.socket_path", "/run/gopanel/gpc.sock")
	if runtime.GOOS == "darwin" {
		viper.SetDefault("gpc.socket_path", "/var/run/gopanel/gpc.sock")
	}
	viper.SetDefault("gpc.windows_pipe", `\\.\pipe\gopanel-gpc`)
	viper.SetDefault("gpc.base_dir", defaultBaseDir)
	viper.SetDefault("gpc.gopanel_service_name", "gopanel.service")
	viper.SetDefault("gpc.action_timeout_ms", 30000)
	viper.SetDefault("gpc.lock_timeout_ms", 30000)
	viper.SetDefault("gpc.file_roots", []string{defaultBaseDir})
	viper.SetDefault("gpc.allow_rootfs", false)
	viper.SetDefault("gpc.max_file_read_bytes", int64(1048576))
	viper.SetDefault("gpc.max_file_write_bytes", int64(1048576))

	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else if homeDir != "" {
		viper.AddConfigPath(filepath.Join(homeDir, ".gpc"))
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
	}

	_ = viper.ReadInConfig()
	_ = viper.Unmarshal(&cfg)

	if cfg.GoPanelPidfilePath == "" && cfg.BaseDir != "" {
		cfg.GoPanelPidfilePath = filepath.Join(cfg.BaseDir, "run", "gopanel.pid")
	}
}

func actionTimeout() time.Duration {
	if cfg.ActionTimeoutMs <= 0 {
		return 30 * time.Second
	}
	return time.Duration(cfg.ActionTimeoutMs) * time.Millisecond
}
