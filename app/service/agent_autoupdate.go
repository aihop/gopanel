package service

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"

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

// 手动更新只做防重入（避免连点触发两次替换二进制），不做节流：
// 用户点一次就必须真的执行一次，否则会以为按钮没反应
var updateRunning atomic.Bool

// ErrAgentUpdateBusy 已有一个更新在跑
var ErrAgentUpdateBusy = errors.New("gp-agent 更新正在进行中，请稍候")

func gpAgentServiceNameForOS() string {
	if runtime.GOOS == "darwin" {
		return "io.aihop.gp-agent"
	}
	return "gp-agent.service"
}

// UpdateGpAgent 更新 gp-agent：仅当 agent 在线、且升级服务器上有更高 version_code 时，
// 才强制替换并重启。与主包（面板本体）无关——主包仍是手动更新。
//
// 只由用户在面板里点「更新 gp-agent」触发（api.AgentUpdate）。
// 早先是「面板启动 / 进入面板」自动跑，等于运行即升级：用户没法选择时机，
// 升级失败或新版有问题时 agent 会被直接换掉，所以改成显式手动。
//
// writeLog 用于把过程写进更新日志（前端 SSE 实时展示）；传 nil 表示只写系统日志。
func UpdateGpAgent(ctx context.Context, writeLog func(string, interface{})) error {
	log := func(text string, param interface{}) {
		if writeLog != nil {
			writeLog(text, param)
		}
	}

	if runtime.GOOS == "windows" {
		log("unsupported platform", runtime.GOOS)
		return errors.New("Windows 暂不支持 gp-agent 更新")
	}
	// 防重入：同一时刻只跑一个
	if !updateRunning.CompareAndSwap(false, true) {
		log("another update in progress", "skip")
		return ErrAgentUpdateBusy
	}
	defer updateRunning.Store(false)

	// 1. 取 agent 当前版本（必须在线；离线请用「一键初始化」安装，不在这里装）
	log("check gp-agent status", gpagent.SocketPath())
	r, err := gpagent.Do(ctx, "AGENT_STATUS", nil)
	if err != nil {
		log("gp-agent offline", err)
		return fmt.Errorf("gp-agent 离线或不可达，请先执行「一键初始化」: %w", err)
	}
	st, err := gpagent.DecodeOutput[agentStatusLite](r)
	if err != nil {
		log("decode agent status error", err)
		return fmt.Errorf("解析 agent 状态失败: %w", err)
	}
	cur, err := strconv.ParseInt(strings.TrimSpace(st.VersionCode), 10, 64)
	if err != nil {
		log("invalid version code", st.VersionCode)
		return fmt.Errorf("当前版本码 %q 无法解析（agent 可能未注入版本），请执行「一键初始化」重装", st.VersionCode)
	}
	log("current version", fmt.Sprintf("%s (%d)", st.Version, cur))

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
	if err != nil {
		log("fetch upgrade info error", err)
		return fmt.Errorf("获取升级信息失败: %w", err)
	}
	if info == nil || strings.TrimSpace(info.DownloadUrl) == "" {
		log("invalid upgrade info", info)
		return errors.New("升级服务器没有返回可用的 gp-agent 下载地址")
	}
	if info.LatestVersionCode <= cur {
		log("already latest", fmt.Sprintf("%s (%d)", st.Version, cur))
		return nil // 已是最新，不算失败
	}

	// 3. 强制替换并重启（gpc INSTALL 内部 bootout+bootstrap，会用新二进制重启 agent）
	global.LOG.Infof("update gp-agent: 发现新版 %d -> %d，开始更新", cur, info.LatestVersionCode)
	log("start update", fmt.Sprintf("%d -> %d", cur, info.LatestVersionCode))
	out, err := gpc.Do(ctx, "GOPANEL_AGENT_INSTALL", map[string]interface{}{
		"download_url": info.DownloadUrl,
		"base_dir":     global.CONF.System.BaseDir,
		"service_name": gpAgentServiceNameForOS(),
	})
	// 注意：gpc.Do 失败时曾返回 nil，在错误分支里裸取 out.Output 就是 nil 解引用，
	// 而这段跑在后台 goroutine 里 —— panic 会直接带走整个面板进程。
	// 现在 gpc.Do 保证非 nil，取值也统一走 gpcResponseOutput。
	if err != nil {
		output := gpcResponseOutput(out)
		global.LOG.Errorf("update gp-agent 失败: %v, out=%s", err, output)
		log("download url", info.DownloadUrl)
		log("gpc install error", err)
		if output != "" {
			log("gpc install output", output)
		}
		return fmt.Errorf("更新失败: %w", err)
	}
	if output := gpcResponseOutput(out); output != "" {
		log("gpc install output", output)
	}
	global.LOG.Infof("update gp-agent 完成（%d）: %s", info.LatestVersionCode, out.Output)
	log("update finished", info.LatestVersionName)
	TrackEvent(TrackEventAgentUpgradeSuccess, info.LatestVersionName)
	return nil
}
