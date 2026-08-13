package api

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/app/service"
	"github.com/aihop/gopanel/buserr"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/token"
	"gorm.io/gorm"
)

var errWebsiteIssueAlreadyHandedOff = errors.New("website issue already handed off")

type websiteIssueCodeRequest struct {
	Requirement string `json:"requirement"`
	AllowCode   bool   `json:"allowCode"`
	RunChecks   bool   `json:"runChecks"`
}

func handoffWebsiteIssueToCode(websiteID, issueID uint, claims *token.CustomClaims, input websiteIssueCodeRequest) (*model.WebsiteIssue, error) {
	if !input.AllowCode {
		return nil, buserr.New("ErrWebsiteDiagnosticCodeRequired")
	}
	repository := repo.NewWebsiteDiagnostic(global.DB)
	issue, err := repository.GetIssue(websiteID, issueID)
	if err != nil {
		return nil, buserr.New("ErrWebsiteDiagnosticIssueNotFound")
	}
	if issue.CodeSessionID > 0 && map[string]bool{"initializing": true, "queued": true, "running": true, "pending_approval": true}[issue.CodeStatus] {
		return issue, nil
	}
	setting, err := repository.GetByWebsiteID(websiteID)
	if err != nil || setting == nil || setting.CodeProjectID == 0 {
		return nil, buserr.New("ErrWebsiteDiagnosticProjectRequired")
	}
	project, err := repo.NewAIProjectRepo().GetProjectByID(setting.CodeProjectID)
	if err != nil {
		return nil, buserr.New("ErrWebsiteDiagnosticProjectNotFound")
	}
	if project.CreatorID != claims.UserId && claims.Role != constant.UserRoleSuper {
		return nil, buserr.New("ErrWebsiteDiagnosticProjectForbidden")
	}
	if err = validateCodeSessionPrerequisites(project, true); err != nil {
		return nil, err
	}
	executorID, err := validateCodeExecutorAvailable(setting.DefaultExecutorID, claims.Role)
	if err != nil {
		return nil, err
	}
	approvalPolicy, err := normalizeCodeApprovalPolicy(setting.ApprovalPolicy)
	if err != nil {
		return nil, err
	}
	if err = validateCodeExecutorApprovalPolicy(executorID, approvalPolicy); err != nil {
		return nil, err
	}
	if err = validateCodeExecutorConfigured(executorID, nil); err != nil {
		return nil, err
	}
	workDir, err := aiProjectSessionWorkDir(project, claims)
	if err != nil {
		return nil, err
	}
	workDir, err = normalizeAIProjectWorkDir(workDir, claims)
	if err != nil {
		return nil, err
	}
	website, err := repo.NewWebsite().GetFirst(repo.NewCommonRepo().WithByID(websiteID))
	if err != nil {
		return nil, err
	}
	detail, err := service.GetWebsiteIssueDetail(websiteID, issueID)
	if err != nil {
		return nil, err
	}
	prompt := service.BuildWebsiteIssueCodePrompt(&website, detail, input.Requirement, input.RunChecks)
	includeUncommitted := true
	session := &model.AIDevSession{
		UserID: claims.UserId, ProjectID: project.ID,
		AgentName: executorID, WorkDir: workDir, Status: codeSessionStatusInitializing, CurrentStage: codeSessionStageSyncingBase,
		ApprovalPolicy: approvalPolicy, IncludeUncommitted: &includeUncommitted,
	}
	session.Title = "网站问题 ISSUE-" + formatUint(issue.ID) + " · " + limitedWebsiteIssueTitle(issue.Title)
	if err = global.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(session).Error; err != nil {
			return err
		}
		instruction := &model.AIInstruction{
			SessionID: session.ID, UserID: claims.UserId, ProjectID: project.ID, Content: prompt,
			Status: "queued", AllowCode: true, AutoPreview: false, RequireApproval: approvalPolicy != codeApprovalPolicyFullAuto,
		}
		if err := tx.Create(instruction).Error; err != nil {
			return err
		}
		now := time.Now()
		result := tx.Model(&model.WebsiteIssue{}).
			Where("id = ? AND website_id = ? AND (code_session_id = 0 OR code_status IN ?)", issue.ID, websiteID, []string{"completed", "failed", "cancelled", "rejected"}).
			Updates(map[string]interface{}{"status": "code_processing", "code_session_id": session.ID, "code_status": "initializing", "confirmed_at": now})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errWebsiteIssueAlreadyHandedOff
		}
		return nil
	}); err != nil {
		if errors.Is(err, errWebsiteIssueAlreadyHandedOff) {
			return repository.GetIssue(websiteID, issueID)
		}
		return nil, err
	}
	enqueueCodeSessionInitialization(session.ID)
	_ = repository.AddTimeline(&model.WebsiteDiagnosticTimeline{WebsiteID: websiteID, IssueID: issue.ID, Type: "code_handoff", UserID: claims.UserId, Content: formatUint(session.ID)})
	issue, _ = repository.GetIssue(websiteID, issueID)
	return issue, nil
}

func formatUint(value uint) string { return strconv.FormatUint(uint64(value), 10) }

func limitedWebsiteIssueTitle(value string) string {
	value = strings.TrimSpace(value)
	if len([]rune(value)) > 60 {
		value = string([]rune(value)[:60])
	}
	return value
}

func reconcileWebsiteIssueCodeTasks() error {
	var issues []model.WebsiteIssue
	if err := global.DB.Where("code_session_id > 0 AND code_status IN ?", []string{"initializing", "queued", "running", "pending_approval"}).Find(&issues).Error; err != nil {
		return err
	}
	for index := range issues {
		issue := &issues[index]
		var session model.AIDevSession
		if err := global.DB.First(&session, issue.CodeSessionID).Error; err != nil {
			continue
		}
		var instruction model.AIInstruction
		_ = global.DB.Where("session_id = ?", session.ID).Order("id ASC").First(&instruction).Error
		if session.LastTaskID > 0 && instruction.TaskID == 0 {
			instruction.TaskID = session.LastTaskID
			_ = global.DB.Save(&instruction).Error
		}
		status := instruction.Status
		if status == "" {
			status = session.Status
		}
		issueStatus := issue.Status
		if status == "completed" {
			issueStatus = "fix_ready"
		}
		if status == "failed" || status == "cancelled" || status == "rejected" {
			issueStatus = "confirmed"
		}
		if issue.CodeTaskID != session.LastTaskID || issue.CodeStatus != status || issue.Status != issueStatus {
			issue.CodeTaskID, issue.CodeStatus, issue.Status = session.LastTaskID, status, issueStatus
			_ = global.DB.Save(issue).Error
			_ = global.DB.Create(&model.WebsiteDiagnosticTimeline{WebsiteID: issue.WebsiteID, IssueID: issue.ID, Type: "code_" + status, Content: formatUint(session.ID)}).Error
		}
		if session.Status == codeSessionStatusActive && instruction.ID > 0 && instruction.TaskID > 0 && instruction.Status == "queued" {
			enqueueCodeInstruction(instruction.ID)
		}
	}
	return nil
}

func autoHandoffWebsiteDiagnosticIssues() error {
	settings, err := repo.NewWebsiteDiagnostic(global.DB).ListEnabled()
	if err != nil {
		return err
	}
	for index := range settings {
		setting := &settings[index]
		if !setting.AutoAnalysis || setting.CodeProjectID == 0 || setting.ConfiguredByUserID == 0 {
			continue
		}
		var issues []model.WebsiteIssue
		windowStart := time.Now().Add(-time.Duration(setting.TriggerWindowMinutes) * time.Minute)
		if err = global.DB.Where("website_id = ? AND status IN ? AND code_session_id = 0 AND last_seen_at >= ?", setting.WebsiteID, []string{"open", "confirmed", "reopened"}, windowStart).
			Order("last_seen_at DESC").Limit(10).Find(&issues).Error; err != nil {
			return err
		}
		user, userErr := repo.NewUser(global.DB).Get(setting.ConfiguredByUserID)
		if userErr != nil {
			continue
		}
		claims := &token.CustomClaims{UserId: user.ID, Role: user.Role, FileBaseDir: user.FileBaseDir}
		for _, issue := range issues {
			var recentCount int64
			if countErr := global.DB.Model(&model.WebsiteDiagnosticEvent{}).Where("issue_id = ? AND occurred_at >= ?", issue.ID, windowStart).Count(&recentCount).Error; countErr != nil {
				return countErr
			}
			if recentCount < int64(setting.TriggerCount) {
				continue
			}
			if _, handoffErr := handoffWebsiteIssueToCode(setting.WebsiteID, issue.ID, claims, websiteIssueCodeRequest{AllowCode: true, RunChecks: true}); handoffErr != nil && global.LOG != nil {
				global.LOG.Errorf("Auto handoff website issue %d failed: %v", issue.ID, handoffErr)
			}
		}
	}
	return nil
}
