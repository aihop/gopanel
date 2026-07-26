package service

import (
	"context"
	"fmt"
	"net"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/gpc"
)

// GpcStatus gpc helper 的可用性诊断。
// 面板以非 root 运行时，全盘扫描/清理都要靠 gpc 提权；它不可用不算错误，
// 但必须把「为什么退化、怎么修」直接告诉用户，而不是只在结果里标一句"可能不完整"。
type GpcStatus struct {
	Needed     bool     `json:"needed"`     // 面板非 root 才需要 gpc
	Available  bool     `json:"available"`  // socket 可连通
	Installed  bool     `json:"installed"`  // gpc 二进制是否存在
	SocketPath string   `json:"socketPath"` //
	Hint       string   `json:"hint"`       // 一句话结论，直接展示
	Commands   []string `json:"commands"`   // 建议在服务器上执行的命令（需要 root）
}

const gpcBinaryPath = "/usr/local/bin/gpc"
const gpcDarwinPlist = "/Library/LaunchDaemons/io.aihop.gpc.plist"

// DiagnoseGpc 探测 gpc 状态并给出修复指引。
// 探测只做 socket 拨号：gpc 协议是单请求单响应，拨通即说明服务活着；
// 发真实 action 反而会占用它的串行锁。
func DiagnoseGpc() GpcStatus {
	st := GpcStatus{
		Needed:     os.Geteuid() != 0,
		SocketPath: gpc.SocketPath(),
	}
	if !st.Needed {
		st.Available = true
		st.Hint = "面板以 root 运行，无需 gpc 即可全盘扫描与清理"
		return st
	}
	if _, err := os.Stat(gpcBinaryPath); err == nil {
		st.Installed = true
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	conn, err := (&net.Dialer{}).DialContext(ctx, "unix", st.SocketPath)
	if err == nil {
		_ = conn.Close()
		st.Available = true
		st.Hint = "gpc helper 正常，扫描与清理可覆盖全盘"
		return st
	}

	// 三种失败各有各的修法，混在一起提示只会让用户乱试
	errText := strings.ToLower(err.Error())
	baseDir := strings.TrimSpace(global.CONF.System.BaseDir)
	switch {
	case strings.Contains(errText, "permission denied"):
		// socket 是 0660 root:<面板用户组>；连不上说明面板换过运行用户，
		// 重启 gpc 会按当前面板用户重新授权
		st.Hint = "gpc socket 存在但面板无权访问（面板运行用户可能变过），重启 gpc 服务可自动修复授权"
		st.Commands = restartGpcCommands()
	case st.Installed:
		st.Hint = "gpc 已安装但服务未运行，请在服务器上以 root 启动它"
		st.Commands = restartGpcCommands()
		if runtime.GOOS == "darwin" {
			if _, err := os.Stat(gpcDarwinPlist); err != nil {
				// 开发/手装场景没有 LaunchDaemon，只能前台拉起
				st.Commands = []string{fmt.Sprintf("sudo %s --base-dir %s service", gpcBinaryPath, baseDir)}
			}
		}
	default:
		st.Hint = "未检测到 gpc helper。重新运行官方安装脚本会自动安装并配置开机自启"
		st.Commands = []string{"bash <(curl -fsSL https://gopanel.run)"}
	}
	return st
}

func restartGpcCommands() []string {
	if runtime.GOOS == "darwin" {
		return []string{
			"sudo launchctl bootout system " + gpcDarwinPlist + " 2>/dev/null; sudo launchctl bootstrap system " + gpcDarwinPlist,
		}
	}
	return []string{"sudo systemctl enable --now gpc.service", "sudo systemctl restart gpc.service"}
}
