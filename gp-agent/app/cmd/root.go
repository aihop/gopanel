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
	"go.uber.org/zap"
)

var (
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

	rootCmd.PersistentFlags().String("base-dir", "", "base dir (align with gopanel system.base_dir)")
	rootCmd.PersistentFlags().Bool("enable-caddy", true, "enable embedded caddy")
	rootCmd.PersistentFlags().Bool("enable-daemon", true, "enable supervisor daemon")

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
	baseDir, _ := rootCmd.PersistentFlags().GetString("base-dir")
	baseDir = filepath.Clean(baseDir)
	if baseDir == "." || baseDir == string(os.PathSeparator) {
		baseDir = ""
	}
	if baseDir == "" {
		baseDir = "/opt/gopanel"
		if runtime.GOOS != "linux" || os.Geteuid() != 0 {
			if home, err := os.UserHomeDir(); err == nil && home != "" {
				baseDir = filepath.Join(home, ".gopanel")
			}
		}
	}

	global.CONF.BaseDir = baseDir
	global.CONF.SocketPath = filepath.Join(baseDir, "gp-agent", "run", "gp-agent.sock")

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
