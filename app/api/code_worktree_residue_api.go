package api

import (
	"errors"
	"time"

	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/token"
	"github.com/gofiber/fiber/v3"
)

type codeWorktreeResidueSummary struct {
	Residues       []codeWorktreeResidue `json:"residues"`
	ReclaimableIDs []uint                `json:"reclaimableIds"`
	ReclaimBytes   int64                 `json:"reclaimBytes"`
}

type codeResidueCleanupOutcome struct {
	SessionID uint   `json:"sessionId"`
	Cleaned   bool   `json:"cleaned"`
	Reason    string `json:"reason,omitempty"`
}

// GetCodeWorktreeResidues 列出当前用户的会话残留。
// 管理目录本身按用户分隔，扫描范围天然被限制在自己的目录里。
func GetCodeWorktreeResidues(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	residues, err := scanCodeWorktreeResidues(claims.UserId)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	summary := codeWorktreeResidueSummary{Residues: residues, ReclaimableIDs: []uint{}}
	for _, residue := range residues {
		if residue.State != codeResidueStateSafe {
			continue
		}
		summary.ReclaimableIDs = append(summary.ReclaimableIDs, residue.SessionID)
		summary.ReclaimBytes += residue.DiskBytes
	}
	return c.JSON(e.Succ(summary))
}

// CleanupCodeWorktreeResidues 清理指定会话的残留。
//
// 客户端传来的会话号只用来圈定范围，能不能删由服务端重新判定——
// 列表是在某个时刻拍的快照，用户点下按钮时会话可能已经重新活跃起来了。
func CleanupCodeWorktreeResidues(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	var req struct {
		SessionIDs []uint `json:"sessionIds"`
	}
	if err := c.Bind().JSON(&req); err != nil {
		return c.JSON(e.Fail(err))
	}
	if len(req.SessionIDs) == 0 {
		return c.JSON(e.Fail(errors.New("请选择要清理的会话残留")))
	}
	if len(req.SessionIDs) > 200 {
		return c.JSON(e.Fail(errors.New("单次最多清理 200 个会话残留")))
	}
	residues, err := scanCodeWorktreeResidues(claims.UserId)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	current := make(map[uint]codeWorktreeResidue, len(residues))
	for _, residue := range residues {
		current[residue.SessionID] = residue
	}
	outcomes := make([]codeResidueCleanupOutcome, 0, len(req.SessionIDs))
	for _, sessionID := range req.SessionIDs {
		outcomes = append(outcomes, cleanupOneCodeResidue(claims, current, sessionID))
	}
	return c.JSON(e.Succ(outcomes))
}

func cleanupOneCodeResidue(
	claims *token.CustomClaims,
	current map[uint]codeWorktreeResidue,
	sessionID uint,
) codeResidueCleanupOutcome {
	startedAt := time.Now()
	outcome := codeResidueCleanupOutcome{SessionID: sessionID}
	residue, exists := current[sessionID]
	if !exists {
		outcome.Reason = "该会话已无残留"
		return outcome
	}
	if residue.State != codeResidueStateSafe && residue.State != codeResidueStateOrphan {
		outcome.Reason = residue.Reason
		return outcome
	}
	var session model.AIDevSession
	if err := global.DB.First(&session, sessionID).Error; err != nil {
		// 会话记录已经没了，剩下的就是纯目录残留，直接按目录删。
		if err := removeCodeResidueDirectories(claims.UserId, residue.Directories); err != nil {
			outcome.Reason = err.Error()
			return outcome
		}
		outcome.Cleaned = true
		recordCodeAudit(claims.UserId, 0, sessionID, "worktree_residue_cleanup", "success",
			"", "orphan", "", startedAt, nil)
		return outcome
	}
	if session.UserID != claims.UserId && claims.Role != constant.UserRoleSuper {
		outcome.Reason = "无权清理该会话残留"
		return outcome
	}
	// 真正的删除交给既有的守卫函数：它会再检查一遍目录归属、worktree 有效性、
	// 未提交变更和分支合并状态，任何一条不满足就拒绝并保留目录。
	if err := cleanupDeliveredCodeSessionWorktrees(&session); err != nil {
		outcome.Reason = err.Error()
		recordCodeAudit(claims.UserId, session.ProjectID, sessionID, "worktree_residue_cleanup", "failed",
			session.WorktreeBranch, err.Error(), "", startedAt, nil)
		return outcome
	}
	outcome.Cleaned = true
	recordCodeAudit(claims.UserId, session.ProjectID, sessionID, "worktree_residue_cleanup", "success",
		session.WorktreeBranch, residue.State, "", startedAt, nil)
	return outcome
}
