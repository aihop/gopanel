package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/aihop/gopanel/gp-agent/global"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var (
	cfgFile string
	rootCmd = &cobra.Command{
		Use:          "gp-agent",
		SilenceUsage: true,
	}
)

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig, initLogger)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file")
	rootCmd.PersistentFlags().String("base-dir", "", "base dir (align with gopanel system.base_dir)")
	rootCmd.PersistentFlags().String("socket-path", "", "uds socket path (linux/macos)")
	rootCmd.PersistentFlags().Bool("enable-caddy", true, "enable embedded caddy")
	rootCmd.PersistentFlags().Bool("enable-daemon", true, "enable supervisor daemon")

	_ = viper.BindPFlag("base_dir", rootCmd.PersistentFlags().Lookup("base-dir"))
	_ = viper.BindPFlag("socket_path", rootCmd.PersistentFlags().Lookup("socket-path"))
	_ = viper.BindPFlag("enable_caddy", rootCmd.PersistentFlags().Lookup("enable-caddy"))
	_ = viper.BindPFlag("enable_daemon", rootCmd.PersistentFlags().Lookup("enable-daemon"))

	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(statusCmd)
}

func initLogger() {
	l, err := zap.NewProduction()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	global.LOG = l
}

func initConfig() {
	viper.SetEnvPrefix("GP_AGENT")
	viper.AutomaticEnv()

	baseDir := "/opt/gopanel"
	if runtime.GOOS != "linux" || os.Geteuid() != 0 {
		if home, err := os.UserHomeDir(); err == nil && home != "" {
			baseDir = filepath.Join(home, ".gopanel")
		}
	}
	viper.SetDefault("base_dir", baseDir)

	socketPath := filepath.Join(baseDir, "gp-agent", "run", "gp-agent.sock")
	viper.SetDefault("socket_path", socketPath)

	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
		_ = viper.ReadInConfig()
	} else {
		_ = viper.ReadInConfig()
	}

	global.CONF.BaseDir = viper.GetString("base_dir")
	global.CONF.SocketPath = viper.GetString("socket_path")

	if global.CONF.BaseDir == "" {
		panic(errors.New("base_dir is empty"))
	}
	if global.CONF.SocketPath == "" {
		panic(errors.New("socket_path is empty"))
	}

	global.CONF.RunDir = filepath.Join(global.CONF.BaseDir, "gp-agent", "run")
	global.CONF.LogDir = filepath.Join(global.CONF.BaseDir, "gp-agent", "log")
	global.CONF.BackupDir = filepath.Join(global.CONF.BaseDir, "gp-agent", "backup")
	global.CONF.StartedAt = time.Now()
}
