package helper

import (
	"os/user"
	"path/filepath"
	"strings"
)

// buildGpAgentSystemdUnit 生成 gp-agent 的 systemd unit。
// 内容必须和 install.sh 里 install_service_gp_agent_linux 写的保持一致，
// 否则同一台机器上「安装时」和「面板里补建时」跑出来的 agent 行为会不一样：
//   - HOME 用运行用户的家目录（不是 base_dir）：rootless podman 的配置/存储都在 ~ 下
//   - 非 root 运行时补 XDG_RUNTIME_DIR / DBUS_SESSION_BUS_ADDRESS：rootless podman 必需
//   - docker.socket / podman.socket 只有 root 才依赖（非 root 用的是 user 级 socket）
func buildGpAgentSystemdUnit(baseDir, runtimeUser string) string {
	isRoot := runtimeUser == "" || runtimeUser == "root"
	home := runtimeUserHomeDir(runtimeUser, baseDir)

	b := strings.Builder{}
	b.WriteString("[Unit]\n")
	b.WriteString("Description=GoPanel Agent (gp-agent)\n")
	if isRoot {
		b.WriteString("After=network.target docker.socket podman.socket\n")
		b.WriteString("Wants=docker.socket podman.socket\n\n")
	} else {
		b.WriteString("After=network.target\n\n")
	}
	b.WriteString("[Service]\n")
	b.WriteString("Type=simple\n")
	b.WriteString("User=" + runtimeUser + "\n")
	b.WriteString("Group=" + runtimeUser + "\n")
	b.WriteString("WorkingDirectory=" + baseDir + "\n")
	b.WriteString("ExecStart=" + filepath.Join(baseDir, "gp-agent") + " service --base-dir " + baseDir + "\n")
	b.WriteString("Restart=always\n")
	b.WriteString("RestartSec=2\n\n")
	b.WriteString(`Environment="HOME=` + home + `"` + "\n")
	b.WriteString(`Environment="CADDY_DATA_DIR=` + filepath.Join(baseDir, "caddy", "data") + `"` + "\n")
	if !isRoot {
		if runtimeDir := runtimeUserRuntimeDir(runtimeUser); runtimeDir != "" {
			b.WriteString(`Environment="XDG_RUNTIME_DIR=` + runtimeDir + `"` + "\n")
			b.WriteString(`Environment="DBUS_SESSION_BUS_ADDRESS=unix:path=` + filepath.Join(runtimeDir, "bus") + `"` + "\n")
		}
	}
	b.WriteString("\n")
	b.WriteString("AmbientCapabilities=CAP_NET_BIND_SERVICE\n")
	b.WriteString("CapabilityBoundingSet=CAP_NET_BIND_SERVICE\n\n")
	b.WriteString("LimitNOFILE=65535\n")
	b.WriteString("OOMScoreAdjust=-100\n\n")
	b.WriteString("[Install]\n")
	b.WriteString("WantedBy=multi-user.target\n")
	return b.String()
}

func runtimeUserHomeDir(runtimeUser, fallback string) string {
	name := strings.TrimSpace(runtimeUser)
	if name == "" || name == "root" {
		if u, err := user.Lookup("root"); err == nil && strings.TrimSpace(u.HomeDir) != "" {
			return u.HomeDir
		}
		return "/root"
	}
	if u, err := user.Lookup(name); err == nil && strings.TrimSpace(u.HomeDir) != "" {
		return u.HomeDir
	}
	return fallback
}

// runtimeUserRuntimeDir 返回 /run/user/<uid>（rootless podman 的 XDG_RUNTIME_DIR）
func runtimeUserRuntimeDir(runtimeUser string) string {
	name := strings.TrimSpace(runtimeUser)
	if name == "" || name == "root" {
		return ""
	}
	u, err := user.Lookup(name)
	if err != nil || strings.TrimSpace(u.Uid) == "" {
		return ""
	}
	return filepath.Join("/run/user", u.Uid)
}
