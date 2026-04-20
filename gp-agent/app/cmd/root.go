package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/aihop/gopanel/gp-agent/global"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var (
	rootCmd = &cobra.Command{
		Use:          "agent",
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
		baseDir = strings.TrimSpace(os.Getenv("GOPANEL_BASE_DIR"))
		if baseDir == "" {
			baseDir = strings.TrimSpace(os.Getenv("GP_AGENT_BASE_DIR"))
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
			if home, err := os.UserHomeDir(); err == nil && home != "" {
				baseDir = filepath.Join(home, ".gopanel")
			}
		}
	}
	global.CONF.BaseDir = baseDir
	global.CONF.SocketPath = filepath.Join(baseDir, "agent", "run", "gp-agent.sock")

	if global.CONF.BaseDir == "" {
		panic(errors.New("base_dir is empty"))
	}
	if global.CONF.SocketPath == "" {
		panic(errors.New("socket_path is empty"))
	}

	global.CONF.RunDir = filepath.Join(global.CONF.BaseDir, "agent", "run")
	global.CONF.LogDir = filepath.Join(global.CONF.BaseDir, "agent", "log")
	global.CONF.BackupDir = filepath.Join(global.CONF.BaseDir, "agent", "backup")
	global.CONF.StartedAt = time.Now()
}

func detectBaseDir() string {
	var candidates []string
	if runtime.GOOS == "linux" {
		candidates = append(candidates, "/opt/gopanel", "/var/lib/gopanel")
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
	if _, err := os.Stat(filepath.Join(dir, "agent")); err == nil {
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
