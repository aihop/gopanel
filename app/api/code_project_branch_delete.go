package api

import (
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/token"
	"github.com/gofiber/fiber/v3"
)

const (
	codeBranchDeleteBlockRemote   = "remote"
	codeBranchDeleteBlockCurrent  = "current"
	codeBranchDeleteBlockDelivery = "delivery"
	codeBranchDeleteBlockWorktree = "worktree"
	codeBranchDeleteBlockSession  = "session"
)

type codeProjectProtectedBranches struct {
	Worktrees map[string]struct{}
	Sessions  map[string]struct{}
}

func inspectCodeProjectProtectedBranches(root string) (codeProjectProtectedBranches, error) {
	result := codeProjectProtectedBranches{Worktrees: map[string]struct{}{}, Sessions: map[string]struct{}{}}
	worktrees, err := runCodeGit(root, "worktree", "list", "--porcelain")
	if err != nil {
		return result, err
	}
	for _, line := range strings.Split(worktrees, "\n") {
		if branch, found := strings.CutPrefix(strings.TrimSpace(line), "branch refs/heads/"); found {
			result.Worktrees[branch] = struct{}{}
		}
	}
	if global.DB == nil {
		return result, nil
	}
	var sessionBranches, repositoryBranches []string
	if err := global.DB.Model(&model.AIDevSession{}).
		Where("source_work_dir = ? AND worktree_branch <> '' AND status <> ?", root, codeSessionStatusDelivered).
		Pluck("worktree_branch", &sessionBranches).Error; err != nil {
		return result, err
	}
	if err := global.DB.Model(&model.AIDevSessionRepository{}).
		Joins("JOIN ai_dev_sessions ON ai_dev_sessions.id = ai_dev_session_repositories.session_id").
		Where("ai_dev_session_repositories.source_dir = ? AND ai_dev_session_repositories.branch <> '' AND ai_dev_sessions.status <> ?",
			root, codeSessionStatusDelivered).
		Pluck("ai_dev_session_repositories.branch", &repositoryBranches).Error; err != nil {
		return result, err
	}
	for _, branch := range append(sessionBranches, repositoryBranches...) {
		result.Sessions[strings.TrimSpace(branch)] = struct{}{}
	}
	return result, nil
}

func codeProjectBranchDeleteBlockReason(
	deliveryBranch string,
	branch codeProjectBranch,
	protected codeProjectProtectedBranches,
) string {
	if branch.Scope != "local" {
		return codeBranchDeleteBlockRemote
	}
	if branch.Current {
		return codeBranchDeleteBlockCurrent
	}
	if strings.TrimSpace(deliveryBranch) == branch.Name {
		return codeBranchDeleteBlockDelivery
	}
	if _, exists := protected.Worktrees[branch.Name]; exists {
		return codeBranchDeleteBlockWorktree
	}
	if _, exists := protected.Sessions[branch.Name]; exists {
		return codeBranchDeleteBlockSession
	}
	return ""
}

func DeleteCodeProjectBranch(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	projectID, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil || projectID == 0 {
		return c.JSON(e.Fail(errors.New("项目参数无效")))
	}
	project, err := repo.NewAIProjectRepo().GetProjectByID(uint(projectID))
	if err != nil {
		return c.JSON(e.Fail(errors.New("项目不存在")))
	}
	if !canManageAIProject(project, claims) {
		return c.JSON(e.Fail(errors.New("无权访问该项目")))
	}
	repositoryPath := strings.TrimSpace(c.Query("repositoryPath"))
	branch := strings.TrimSpace(c.Query("branch"))
	force, _ := strconv.ParseBool(c.Query("force"))
	startedAt := time.Now()
	err = deleteCodeProjectLocalBranch(project, repositoryPath, branch, force)
	status, detail := "success", "项目本地分支已删除"
	if err != nil {
		status, detail = "failed", err.Error()
	}
	recordCodeAudit(claims.UserId, project.ID, 0, "project_branch_delete", status, branch, detail, c.IP(), startedAt,
		codeAuditMeta{"repositoryPath": repositoryPath, "force": force})
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ())
}

func deleteCodeProjectLocalBranch(project *model.AIProject, repositoryPath, branch string, force bool) error {
	if project == nil {
		return errors.New("项目不可用")
	}
	specs, err := codeProjectRepositorySpecs(project)
	if err != nil {
		return err
	}
	keys := make([]string, 0, len(specs))
	for _, spec := range specs {
		keys = append(keys, spec.LeaseKey)
	}
	owner := newCodeRepositoryLeaseOwner("branch-delete")
	acquired, err := acquireCodeRepositoryLeases(owner, 0, keys)
	if err != nil {
		return err
	}
	if !acquired {
		return errors.New("项目仓库正在同步或交付，请稍后重试")
	}
	defer func() { _ = releaseCodeRepositoryLeases(owner, keys) }()

	root, err := resolveCodeProjectBranchRepository(project, repositoryPath)
	if err != nil {
		return err
	}
	if branch == "" {
		return errors.New("分支名称不能为空")
	}
	if _, err := runCodeGit(root, "check-ref-format", "--branch", branch); err != nil {
		return errors.New("分支名称无效")
	}
	if _, err := runCodeGit(root, "show-ref", "--verify", "refs/heads/"+branch); err != nil {
		return errors.New("本地分支不存在或已删除")
	}
	repository, err := inspectCodeProjectBranchRepository(project, root)
	if err != nil {
		return err
	}
	var target *codeProjectBranch
	for index := range repository.Branches {
		if repository.Branches[index].Scope == "local" && repository.Branches[index].Name == branch {
			target = &repository.Branches[index]
			break
		}
	}
	if target == nil {
		return errors.New("本地分支不存在或已删除")
	}
	if target.DeleteBlockReason != "" {
		return codeProjectBranchDeleteBlockedError(target.DeleteBlockReason)
	}
	if !target.Merged && !force {
		return errors.New("分支尚未合并；确认不再需要后可强制删除")
	}
	if _, err := runCodeGit(root, "branch", "-D", "--", branch); err != nil {
		return err
	}
	return nil
}

func resolveCodeProjectBranchRepository(project *model.AIProject, requested string) (string, error) {
	requested, err := filepath.EvalSymlinks(filepath.Clean(requested))
	if err != nil {
		return "", errors.New("项目仓库不可访问")
	}
	roots, err := discoverCodeProjectBranchRepositories(codeProjectSourceDirs(project))
	if err != nil {
		return "", err
	}
	for _, root := range roots {
		if filepath.Clean(root) == requested {
			return root, nil
		}
	}
	return "", errors.New("仓库不属于当前项目")
}

func codeProjectBranchDeleteBlockedError(reason string) error {
	messages := map[string]string{
		codeBranchDeleteBlockRemote:   "远端分支当前仅支持查看，请在远端仓库中管理",
		codeBranchDeleteBlockCurrent:  "不能删除仓库当前所在分支",
		codeBranchDeleteBlockDelivery: "不能删除项目交付目标分支",
		codeBranchDeleteBlockWorktree: "分支仍被 Git Worktree 使用",
		codeBranchDeleteBlockSession:  "分支仍被 GoPanel Code 会话引用，请先处理对应会话",
	}
	if message := messages[reason]; message != "" {
		return errors.New(message)
	}
	return fmt.Errorf("分支当前不可删除：%s", reason)
}
