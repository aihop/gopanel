package api

import (
	"fmt"
	"strings"

	"github.com/aihop/gopanel/app/repo"
)

// 交付有两种落地方式：
//   - direct：把交付提交直接推到交付目标分支（默认，保持既有行为）。
//   - branch：推到本会话独占的分支，由平台的 PR/MR 负责合并。
//
// branch 模式下远端目标分支被别人推进不再阻断交付推送，
// 因为会话分支只有本次交付会写。
const (
	codeDeliveryModeDirect = "direct"
	codeDeliveryModeBranch = "branch"
)

// codeDeliverySessionBranch 是会话独占的交付分支名。
// 用会话编号而不是时间戳，保证同一会话重复交付时推的是同一个分支，PR 可以持续更新。
func codeDeliverySessionBranch(sessionID uint) string {
	return fmt.Sprintf("code/session-%d", sessionID)
}

// normalizeCodeDeliveryMode 只接受已知模式，未知或缺省一律回落到 direct，
// 保证配置写错时行为退回到既有的直推目标分支。
func normalizeCodeDeliveryMode(mode *string) string {
	if mode == nil {
		return codeDeliveryModeDirect
	}
	if strings.TrimSpace(*mode) == codeDeliveryModeBranch {
		return codeDeliveryModeBranch
	}
	return codeDeliveryModeDirect
}

func codeProjectDeliveryMode(projectID uint) string {
	if projectID == 0 {
		return codeDeliveryModeDirect
	}
	project, err := repo.NewAIProjectRepo().GetProjectByID(projectID)
	if err != nil || project == nil {
		return codeDeliveryModeDirect
	}
	if strings.TrimSpace(project.DeliveryMode) == codeDeliveryModeBranch {
		return codeDeliveryModeBranch
	}
	return codeDeliveryModeDirect
}

// codeDeliveryPushTarget 解析交付提交要推往的远端分支。
// remoteBranch/targetBranch 仍然是同步基线（从哪里拉），推送目标由交付模式决定。
func codeDeliveryPushTarget(mode string, sessionID uint, remoteBranch, targetBranch string) string {
	if mode == codeDeliveryModeBranch && sessionID != 0 {
		return codeDeliverySessionBranch(sessionID)
	}
	return deliveryRemoteBranch(remoteBranch, targetBranch)
}

// codeDeliveryPushIsolated 表示推送目标是否为本次交付独占的分支。
// 独占时远端不需要是交付基线的后代，允许在重新交付后覆盖自己上次推送的提交。
func codeDeliveryPushIsolated(mode string) bool {
	return mode == codeDeliveryModeBranch
}
