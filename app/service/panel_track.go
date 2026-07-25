package service

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/shirou/gopsutil/v4/host"
)

// 面板生命周期事件（与 install.sh track_install_event 共用一套 event 命名）
const (
	TrackEventUpgradeSuccess      = "upgrade_success"
	TrackEventUpgradeFailed       = "upgrade_failed"
	TrackEventAgentUpgradeSuccess = "agent_upgrade_success"
)

var (
	installIDMu    sync.Mutex
	installIDCache string
)

// InstallID 返回本机安装标识，与 install.sh 的 ensure_install_id 共用 ${BaseDir}/install_id。
// 老版本装机脚本没有写过该文件时在这里补生成并落盘；落盘失败则只在进程内缓存，
// 避免同一次运行内反复变化。
func InstallID() string {
	installIDMu.Lock()
	defer installIDMu.Unlock()
	if installIDCache != "" {
		return installIDCache
	}
	idFile := filepath.Join(global.CONF.System.BaseDir, "install_id")
	if content, err := os.ReadFile(idFile); err == nil {
		if id := strings.TrimSpace(string(content)); id != "" {
			installIDCache = id
			return installIDCache
		}
	}
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return ""
	}
	installIDCache = hex.EncodeToString(buf)
	if err := os.WriteFile(idFile, []byte(installIDCache), 0644); err != nil {
		global.LOG.Debugf("write install_id failed: %v", err)
	}
	return installIDCache
}

// TrackEvent 向 gopanel.cn 上报面板事件。install.sh 只覆盖脚本安装/升级，
// 面板内一键更新与 gp-agent 自动更新此前完全静默，这里补齐。
// 上报失败只记 debug 日志，不影响调用方流程。
func TrackEvent(event, version string) {
	if strings.TrimSpace(constant.TrackUrl) == "" || strings.TrimSpace(event) == "" {
		return
	}
	installID := InstallID()
	if installID == "" {
		return
	}

	query := url.Values{}
	query.Set("event", event)
	query.Set("install_id", installID)
	query.Set("channel", trackChannel())
	query.Set("os", runtime.GOOS)
	query.Set("arch", runtime.GOARCH)
	// 区分上报来源：install.sh 装机脚本 vs 面板自身
	query.Set("source", "panel")
	if v := strings.TrimSpace(version); v != "" {
		query.Set("version", v)
	}
	if ip := GetOutboundIP(); ip != "" && ip != "IPNotFound" {
		query.Set("ip", ip)
	}
	if hostInfo, err := host.Info(); err == nil && hostInfo != nil {
		if distro := trackDistro(hostInfo.Platform, hostInfo.PlatformVersion); distro != "" {
			query.Set("distro", distro)
		}
		if hostInfo.KernelArch != "" {
			query.Set("machine", hostInfo.KernelArch)
		}
		if hostInfo.KernelVersion != "" {
			query.Set("kernel", hostInfo.KernelVersion)
		}
	}
	if rt := trackContainerRuntime(); rt != "" {
		query.Set("runtime", rt)
	}

	// 与 install.sh 的 curl --max-time 3 对齐：升级链路上不因上报卡住重启
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(constant.TrackUrl + "?" + query.Encode())
	if err != nil {
		global.LOG.Debugf("track event %s failed: %v", event, err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		global.LOG.Debugf("track event %s got status %d", event, resp.StatusCode)
	}
}

// TrackEventAsync 异步上报，用于不希望被网络耗时拖住的调用点。
func TrackEventAsync(event, version string) {
	go TrackEvent(event, version)
}

// trackChannel 与 install.sh 保持一致：未通过 CHANNEL 环境变量指定时上报 unknown。
// 装机脚本没有把 channel 落盘，面板侧无法还原，只能读环境变量。
func trackChannel() string {
	for _, key := range []string{"CONFIG_CHANNEL", "CHANNEL"} {
		if v := strings.TrimSpace(os.Getenv(key)); v != "" {
			return v
		}
	}
	return "unknown"
}

// trackDistro 对齐 install.sh detect_distro 的 ID_VERSION_ID 格式，仅 linux 上报。
func trackDistro(platform, platformVersion string) string {
	if runtime.GOOS != "linux" || platform == "" {
		return ""
	}
	if platformVersion == "" {
		return platform
	}
	return platform + "_" + platformVersion
}

// trackContainerRuntime 对齐 install.sh detect_runtime：优先 docker，其次 podman。
func trackContainerRuntime() string {
	if _, err := exec.LookPath("docker"); err == nil {
		return "docker"
	}
	if _, err := exec.LookPath("podman"); err == nil {
		return "podman"
	}
	return ""
}
