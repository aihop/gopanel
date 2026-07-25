package service

import (
	"context"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/gpagent"
	"github.com/aihop/gopanel/utils/gpc"
)

// gp-agent 自动更新只解码需要的字段
type agentStatusLite struct {
	Version     string `json:"version"`
	VersionCode string `json:"version_code"`
}

func gpAgentServiceNameForOS() string {
	if runtime.GOOS == "darwin" {
		return "io.aihop.gp-agent"
	}
	return "gp-agent.service"
}

// AutoUpdateGpAgent 面板启动时自动检测并更新 gp-agent：
// 仅当 agent 在线、且升级服务器上有更高 version_code 时，才强制替换并重启。
// 与主包（面板本体）无关——主包仍是手动更新。
func AutoUpdateGpAgent() {
	if runtime.GOOS == "windows" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// 1. 取 agent 当前版本（必须在线；离线交给手动 ensure，不在这里装）
	r, err := gpagent.Do(ctx, "AGENT_STATUS", nil)
	if err != nil {
		global.LOG.Infof("auto-update gp-agent: agent 离线或不可达，跳过 (%v)", err)
		return
	}
	st, err := gpagent.DecodeOutput[agentStatusLite](r)
	if err != nil {
		global.LOG.Infof("auto-update gp-agent: 解析 agent 状态失败，跳过 (%v)", err)
		return
	}
	cur, err := strconv.ParseInt(strings.TrimSpace(st.VersionCode), 10, 64)
	if err != nil {
		global.LOG.Infof("auto-update gp-agent: 当前版本码 %q 无法解析，跳过（agent 可能未注入版本）", st.VersionCode)
		return
	}

	// 2. 选源（国内 gitcode / 海外 github），查升级服务器上 gp-agent 的最新版
	source := "github"
	if IsChinaMainlandServer() {
		source = "gitcode"
	}
	info, err := NewIAppVersionService().GetUpdateInfo(constant.UpgradeUrl, &dto.SettingUpgradeVersion{
		VersionName: st.Version,
		VersionCode: cur,
		OS:          runtime.GOOS,
		Arch:        runtime.GOARCH,
		Lang:        "zh",
		AppBrand:    constant.AppBrand,
		Source:      source,
		Package:     "gp-agent",
	})
	if err != nil || info == nil || strings.TrimSpace(info.DownloadUrl) == "" {
		return
	}
	if info.LatestVersionCode <= cur {
		return // 已是最新
	}

	// 3. 强制替换并重启（gpc INSTALL 内部 bootout+bootstrap，会用新二进制重启 agent）
	global.LOG.Infof("auto-update gp-agent: 发现新版 %d -> %d，开始更新", cur, info.LatestVersionCode)
	out, err := gpc.Do(ctx, "GOPANEL_AGENT_INSTALL", map[string]interface{}{
		"download_url": info.DownloadUrl,
		"base_dir":     global.CONF.System.BaseDir,
		"service_name": gpAgentServiceNameForOS(),
	})
	if err != nil {
		global.LOG.Errorf("auto-update gp-agent 失败: %v, out=%s", err, out.Output)
		return
	}
	global.LOG.Infof("auto-update gp-agent 完成（%d）: %s", info.LatestVersionCode, out.Output)
}
