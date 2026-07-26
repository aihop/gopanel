package helper

import (
	"strings"
	"testing"
)

// unit 内容必须和 install.sh 的 install_service_gp_agent_linux 对齐，
// 否则「安装时生成的 unit」和「面板里补建的 unit」行为不一致：
// rootless podman 丢了 XDG_RUNTIME_DIR / DBUS_SESSION_BUS_ADDRESS 就连不上 podman，
// 表现为 agent 起来了但所有容器/网站功能全挂。
func TestBuildGpAgentSystemdUnitRoot(t *testing.T) {
	unit := buildGpAgentSystemdUnit("/opt/gopanel", "root")

	mustContain(t, unit, []string{
		"ExecStart=/opt/gopanel/gp-agent service --base-dir /opt/gopanel",
		"User=root",
		"Group=root",
		"After=network.target docker.socket podman.socket",
		"Wants=docker.socket podman.socket",
		`Environment="CADDY_DATA_DIR=/opt/gopanel/caddy/data"`,
		"WantedBy=multi-user.target",
	})
	// root 的 HOME 不该是 base_dir
	if strings.Contains(unit, `Environment="HOME=/opt/gopanel"`) {
		t.Errorf("root 的 HOME 不应该是 base_dir:\n%s", unit)
	}
	// root 不需要 user 级 runtime dir
	if strings.Contains(unit, "XDG_RUNTIME_DIR") {
		t.Errorf("root 不该带 XDG_RUNTIME_DIR:\n%s", unit)
	}
}

func TestBuildGpAgentSystemdUnitNonRoot(t *testing.T) {
	// 用一个几乎不可能存在的用户名，验证查不到用户时的兜底行为
	unit := buildGpAgentSystemdUnit("/opt/gopanel", "gopanel-no-such-user")

	mustContain(t, unit, []string{
		"User=gopanel-no-such-user",
		"ExecStart=/opt/gopanel/gp-agent service --base-dir /opt/gopanel",
	})
	// 非 root 不该依赖系统级 docker/podman socket（rootless 用的是 user 级）
	if strings.Contains(unit, "Wants=docker.socket") {
		t.Errorf("非 root 不该 Wants 系统级 socket:\n%s", unit)
	}
	if !strings.Contains(unit, "After=network.target\n") {
		t.Errorf("缺少 After=network.target:\n%s", unit)
	}
}

func mustContain(t *testing.T, unit string, wants []string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(unit, want) {
			t.Errorf("unit 缺少 %q:\n%s", want, unit)
		}
	}
}
