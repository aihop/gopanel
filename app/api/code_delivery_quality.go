package api

import (
	"errors"
	"fmt"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/global"
)

func runCodeDeliveryQualityGate(session *model.AIDevSession, userID uint, lease *codeExecutionLease, report codeDeliveryProgressReporter) error {
	if session == nil || session.ProjectID == 0 {
		return nil
	}
	_, err := repo.NewAIProjectRepo().GetProjectByID(session.ProjectID)
	if err != nil {
		return err
	}
	checks, err := detectCodeQualityChecks(session)
	if err != nil {
		return err
	}
	if len(checks) == 0 {
		return errors.New("项目已启用质量门禁，但未识别到可执行检查")
	}
	if err := validateCodeDeliverySnapshotCurrent(session); err != nil {
		return err
	}
	loadCodeQualityResults(session.ID, checks)
	for index := range checks {
		check := checks[index]
		if check.LastResult != nil && check.LastResult.Current && check.LastResult.Status == "passed" {
			continue
		}
		if report != nil {
			report(codeDeliveryStageQualityCheck, 60+(index*8/max(1, len(checks))))
		}
		revision, err := codeQualityRevision(check.workDirPath)
		if err != nil {
			return err
		}
		result := executeCodeQualityCheck(lease, check)
		currentRevision, currentErr := codeQualityRevision(check.workDirPath)
		result.Revision = revision
		result.Current = currentErr == nil && currentRevision == revision
		if err := persistCodeQualityResult(session, userID, check, result); err != nil {
			return err
		}
		if result.Status != "passed" {
			summary := codeQualityFailureSummary(result.Output)
			if summary != "" {
				return fmt.Errorf("质量门禁未通过：%s：%s", check.Label, summary)
			}
			return fmt.Errorf("质量门禁未通过：%s", check.Label)
		}
		if !result.Current {
			return fmt.Errorf("质量检查期间提交发生变化：%s", check.Label)
		}
	}
	if err := validateCodeDeliverySnapshotCurrent(session); err != nil {
		return err
	}
	return validateCodeQualityGate(session)
}

func validateCodeDeliverySnapshotCurrent(session *model.AIDevSession) error {
	if session.IsolationMode == codeIsolationMultiWorktree || hasCodeMultiRepositoryDelivery(session.ID) {
		repositories, err := loadCodeSessionRepositories(session.ID)
		if err != nil || len(repositories) == 0 {
			return errors.New("会话多仓库交付快照不可用")
		}
		for index := range repositories {
			repository := &repositories[index]
			if err := verifyCodeDeliveryCommit(repository.WorktreeDir, repository.WorktreeCommit, "仓库 "+repository.LinkName); err != nil {
				return err
			}
		}
		return nil
	}
	var delivery model.AICodeDelivery
	if err := global.DB.Where("session_id = ?", session.ID).First(&delivery).Error; err != nil {
		return err
	}
	return verifyCodeDeliveryCommit(delivery.WorkDir, delivery.WorktreeCommit, "Worktree")
}
