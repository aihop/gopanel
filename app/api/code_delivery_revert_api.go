package api

import (
	"errors"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/token"
	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

type codeDeliveryRevertResponse struct {
	Status       string                       `json:"status"`
	Repositories []codeRepositoryRevertResult `json:"repositories"`
}

// RevertCodeSessionDelivery 撤销会话最近一次交付。
//
// 只允许撤最近一次：交付之间可能互相依赖，撤销中间某一笔会让后面的
// 交付建立在一段并不存在的历史上，那种混乱不是一个按钮该制造的。
func RevertCodeSessionDelivery(c fiber.Ctx) error {
	startedAt := time.Now()
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	var req struct {
		Confirm bool `json:"confirm"`
	}
	if err := c.Bind().JSON(&req); err != nil {
		return c.JSON(e.Fail(err))
	}
	if !req.Confirm {
		return c.JSON(e.Fail(errors.New("撤销交付需要明确确认")))
	}
	session, err := getCodeDeliverySessionContext(c)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if session.UserID != claims.UserId && claims.Role != constant.UserRoleSuper {
		return c.JSON(e.Fail(errors.New("无权撤销该会话的交付")))
	}
	if err := validateCodeDeliveryRevertable(session); err != nil {
		return c.JSON(e.Fail(err))
	}
	response, err := runCodeDeliveryRevert(session, claims.UserId)
	if err != nil {
		recordCodeAudit(claims.UserId, session.ProjectID, session.ID, "delivery_revert", "failed",
			session.WorktreeBranch, err.Error(), c.IP(), startedAt, nil)
		return c.JSON(e.Fail(err))
	}
	recordCodeAudit(claims.UserId, session.ProjectID, session.ID, "delivery_revert", response.Status,
		session.WorktreeBranch, response.Status, c.IP(), startedAt, nil)
	return c.JSON(e.Succ(response))
}

// validateCodeDeliveryRevertable 拦住不该撤的情况。
// 交付还在跑的时候撤销会和交付作业抢同一批仓库，必须先等它结束。
func validateCodeDeliveryRevertable(session *model.AIDevSession) error {
	if global.DB == nil {
		return errors.New("交付记录不可用")
	}
	var job model.AICodeDeliveryJob
	err := global.DB.Where("session_id = ?", session.ID).First(&job).Error
	if err == nil && (job.Status == codeDeliveryJobQueued || job.Status == codeDeliveryJobRunning) {
		return errors.New("该会话的交付正在进行，请等交付结束后再撤销")
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return nil
}

// runCodeDeliveryRevert 取得仓库租约后逐仓库撤销。
//
// 租约用的键与交付一致，撤销和交付因此天然互斥——否则一边在合入、
// 一边在撤销，目标分支会被两个方向的写入撕开。
func runCodeDeliveryRevert(session *model.AIDevSession, userID uint) (codeDeliveryRevertResponse, error) {
	requests, err := loadCodeDeliveryRevertTargets(session)
	if err != nil {
		return codeDeliveryRevertResponse{}, err
	}
	if len(requests) == 0 {
		return codeDeliveryRevertResponse{}, errors.New("该会话没有可撤销的交付")
	}
	keys := make([]string, 0, len(requests))
	for _, request := range requests {
		keys = append(keys, codeDeliveryRepositoryKey(request.SourceDir, request.RemoteName, request.TargetBranch))
	}
	owner := newCodeRepositoryLeaseOwner("delivery-revert")
	acquired, err := acquireCodeRepositoryLeases(owner, 0, keys)
	if err != nil {
		return codeDeliveryRevertResponse{}, err
	}
	if !acquired {
		return codeDeliveryRevertResponse{}, errors.New("目标仓库正被其他交付操作占用，请稍后重试")
	}
	defer func() { _ = releaseCodeRepositoryLeases(owner, keys) }()

	response := codeDeliveryRevertResponse{Repositories: make([]codeRepositoryRevertResult, 0, len(requests))}
	for _, request := range requests {
		result, revertErr := revertCodeDeliveryInRepository(request, session)
		if revertErr != nil {
			result.Status, result.ErrorMessage = "failed", revertErr.Error()
		}
		if result.Status == codeRevertStatusReverted {
			pushed, pushErr := pushCodeDeliveryRevert(request, result.RevertCommit)
			result.Pushed = pushed
			if pushErr != nil {
				// 本地已经撤销成功，推送失败不该把它报成整体失败：
				// 撤销提交就在本地分支上，用户可以重新推送。
				result.ErrorMessage = "本地已撤销，但推送远端失败：" + pushErr.Error()
			}
			persistCodeDeliveryRevert(session, request, result)
		}
		response.Repositories = append(response.Repositories, result)
	}
	response.Status = summarizeCodeRevertStatus(response.Repositories)
	return response, nil
}

func summarizeCodeRevertStatus(results []codeRepositoryRevertResult) string {
	reverted, failed, conflict := 0, 0, 0
	for _, result := range results {
		switch result.Status {
		case codeRevertStatusReverted:
			reverted++
		case codeRevertStatusConflict:
			conflict++
		case codeRevertStatusSkipped:
		default:
			failed++
		}
	}
	switch {
	case failed > 0 || conflict > 0:
		if reverted > 0 {
			return "partial"
		}
		if conflict > 0 {
			return codeRevertStatusConflict
		}
		return "failed"
	case reverted > 0:
		return codeRevertStatusReverted
	default:
		return codeRevertStatusSkipped
	}
}

// loadCodeDeliveryRevertTargets 汇总本次要撤销的仓库。
// 已经撤过的直接排除：反向提交撤销后原合并提交仍在分支上，
// 再撤一次等于把改动重新加回去。
func loadCodeDeliveryRevertTargets(session *model.AIDevSession) ([]codeDeliveryRevertRequest, error) {
	credentialID := codeProjectGitCredentialID(session.ProjectID)
	repositories, err := loadCodeSessionRepositories(session.ID)
	if err != nil {
		return nil, err
	}
	if len(repositories) > 0 {
		requests := make([]codeDeliveryRevertRequest, 0, len(repositories))
		for index := range repositories {
			repository := &repositories[index]
			if repository.RevertedAt != nil || strings.TrimSpace(repository.MergeCommit) == "" {
				continue
			}
			requests = append(requests, codeDeliveryRevertRequest{
				SourceDir: repository.SourceDir, TargetBranch: repository.TargetBranch,
				MergeCommit: repository.MergeCommit, RemoteName: repository.RemoteName,
				RemoteBranch: repository.RemoteBranch, PushStatus: repository.PushStatus,
				CredentialID: credentialID, Label: repository.LinkName,
			})
		}
		return requests, nil
	}
	var delivery model.AICodeDelivery
	if err := global.DB.Where("session_id = ?", session.ID).First(&delivery).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	if delivery.RevertedAt != nil {
		return nil, errors.New("该交付已经撤销过")
	}
	if strings.TrimSpace(delivery.MergeCommit) == "" {
		return nil, nil
	}
	return []codeDeliveryRevertRequest{{
		SourceDir: delivery.SourceWorkDir, TargetBranch: delivery.TargetBranch,
		MergeCommit: delivery.MergeCommit, RemoteName: delivery.RemoteName,
		RemoteBranch: delivery.RemoteBranch, PushStatus: delivery.PushStatus,
		CredentialID: credentialID,
	}}, nil
}

func persistCodeDeliveryRevert(
	session *model.AIDevSession,
	request codeDeliveryRevertRequest,
	result codeRepositoryRevertResult,
) {
	now := time.Now()
	updates := map[string]any{"revert_commit": result.RevertCommit, "reverted_at": now}
	if request.Label != "" {
		if err := global.DB.Model(&model.AIDevSessionRepository{}).
			Where("session_id = ? AND link_name = ?", session.ID, request.Label).
			Updates(updates).Error; err != nil {
			global.LOG.Errorf("Persist Code delivery revert for session %d repository %s failed: %v",
				session.ID, request.Label, err)
		}
		return
	}
	if err := global.DB.Model(&model.AICodeDelivery{}).
		Where("session_id = ?", session.ID).Updates(updates).Error; err != nil {
		global.LOG.Errorf("Persist Code delivery revert for session %d failed: %v", session.ID, err)
	}
}
