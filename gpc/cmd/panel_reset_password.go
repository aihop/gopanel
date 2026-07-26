package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

// gpc panel reset-password [email]
//
// 这里刻意做成一层薄壳：真正改密码的是面板二进制的 --reset-password
// （bcrypt 算法、user 表结构、base_dir 解析都只在面板里维护一份，避免逻辑漂移）。
//
// 密码的读取整个交给子进程 —— 当前终端直接透传给 gopanel，所以：
//   - 不经过 argv（不会出现在 ps / shell history 里）
//   - 不经过 gpc helper 的 socket 和 zap 日志
//   - gpc 自己从头到尾看不到明文
//
// 权限沿用操作系统本身的判断：能不能改由数据库文件的读写权限决定，
// 一般需要 sudo。这样就不必新增一个"能连 socket 就能重置超管密码"的特权入口。
var goPanelResetPasswordCmd = &cobra.Command{
	Use:          "reset-password [email]",
	Short:        "Reset a panel account password (delegates to the gopanel binary)",
	Long:         "Reset a panel account password. Without an email the panel resets its super admin account.\nThe password is typed directly into the gopanel process; gpc never sees it.",
	Args:         cobra.MaximumNArgs(1),
	SilenceUsage: true,
	RunE: func(c *cobra.Command, args []string) error {
		bin, err := resolveGoPanelBinary()
		if err != nil {
			return err
		}

		argv := []string{"--reset-password"}
		if cfgPath := strings.TrimSpace(cfg.GoPanelConfigPath); cfgPath != "" {
			argv = append(argv, "-c", cfgPath)
		}
		if len(args) == 1 && strings.TrimSpace(args[0]) != "" {
			argv = append(argv, strings.TrimSpace(args[0]))
		}

		child := exec.Command(bin, argv...)
		child.Stdin = os.Stdin
		child.Stdout = os.Stdout
		child.Stderr = os.Stderr
		// 不注入 GOPANEL_BASE_DIR：面板要以自己 -c 指定的 conf 为准，
		// 否则 gpc 探测到的 base_dir 与配置不一致时会静默改错库
		child.Env = os.Environ()

		if err := child.Run(); err != nil {
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				// 子进程已经把原因打到 stderr 了，这里不再重复包装
				return fmt.Errorf("gopanel --reset-password exited with code %d", exitErr.ExitCode())
			}
			return fmt.Errorf("run %s failed: %w", bin, err)
		}
		return nil
	},
}

// resolveGoPanelBinary 定位面板二进制：先看配置，再看 base_dir（与当前 conf 属于同一份安装），
// 最后才回退到 PATH。
func resolveGoPanelBinary() (string, error) {
	name := "gopanel"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}

	if configured := strings.TrimSpace(cfg.GoPanelBinaryPath); configured != "" {
		if isExecutableFile(configured) {
			return configured, nil
		}
		return "", errors.New("gopanel binary not found: " + configured)
	}

	baseDir := strings.TrimSpace(cfg.BaseDir)
	candidates := []string{
		filepath.Join(baseDir, name),
		filepath.Join(baseDir, "bin", name),
		filepath.Join(baseDir, "server", name),
	}
	for _, candidate := range candidates {
		if isExecutableFile(candidate) {
			return candidate, nil
		}
	}
	if p, err := exec.LookPath(name); err == nil {
		return p, nil
	}
	return "", fmt.Errorf("gopanel binary not found (looked in %s and PATH)", baseDir)
}

func isExecutableFile(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}
