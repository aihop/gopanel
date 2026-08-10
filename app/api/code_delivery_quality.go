package api

import (
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/global"
)

type codeDeliveryQualityRoot struct {
	WorkDir     string
	IdentityDir string
	RuntimeDir  string
	Commit      string
	Label       string
}

func runCodeDeliveryQualityGate(session *model.AIDevSession, userID uint, lease *codeExecutionLease, report codeDeliveryProgressReporter) error {
	enabled, err := codeDeliveryQualityGateEnabled(session)
	if err != nil || !enabled {
		return err
	}
	if session.IsolationMode == codeIsolationMultiWorktree || hasCodeMultiRepositoryDelivery(session.ID) {
		roots, cleanup, err := prepareCodeMultiRepositoryQualityRoots(session)
		if err != nil {
			return err
		}
		qualityErr := runCodeDeliveryQualityGateAtRoots(session, userID, lease, report, roots)
		return errors.Join(qualityErr, cleanup())
	}
	roots, err := codeDeliveryQualityRoots(session)
	if err != nil {
		return err
	}
	return runCodeDeliveryQualityGateAtRoots(session, userID, lease, report, roots)
}

func codeDeliveryQualityGateEnabled(session *model.AIDevSession) (bool, error) {
	if session == nil || session.ProjectID == 0 {
		return false, nil
	}
	project, err := repo.NewAIProjectRepo().GetProjectByID(session.ProjectID)
	if err != nil {
		return false, err
	}
	return project.RequireQualityGate, nil
}

func runCodeDeliveryQualityGateAtRoots(session *model.AIDevSession, userID uint, lease *codeExecutionLease, report codeDeliveryProgressReporter, roots []codeDeliveryQualityRoot) (returnErr error) {
	if len(roots) == 0 {
		return errors.New("项目已启用质量门禁，但交付快照不可用")
	}
	if err := validateCodeDeliveryQualityRoots(roots); err != nil {
		return err
	}
	cleanup, err := prepareCodeDeliveryQualityEnvironment(roots)
	if err != nil {
		return err
	}
	defer func() { returnErr = errors.Join(returnErr, cleanup()) }()
	checks := detectCodeDeliveryQualityChecks(session.ProjectID, roots)
	if len(checks) == 0 {
		return errors.New("项目已启用质量门禁，但未识别到可执行检查")
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
		if changedRoot := changedCodeDeliveryQualityRoot(check, roots); changedRoot != nil {
			return errors.Join(
				fmt.Errorf("质量检查修改了交付快照：%s", check.Label),
				restoreCodeDeliveryQualityRoot(*changedRoot),
			)
		}
		currentRevision, currentErr := codeQualityRevision(check.workDirPath)
		result.Revision = revision
		result.Current = currentErr == nil && currentRevision == revision
		if err := persistCodeDeliveryQualityResult(session, userID, check, result, roots); err != nil {
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
	if err := validateCodeDeliveryQualityRoots(roots); err != nil {
		return err
	}
	loadCodeQualityResults(session.ID, checks)
	return validateCodeQualityCheckResults(checks)
}

func changedCodeDeliveryQualityRoot(check codeQualityCheck, roots []codeDeliveryQualityRoot) *codeDeliveryQualityRoot {
	for index := range roots {
		root := &roots[index]
		if !pathWithinDirectory(root.WorkDir, check.workDirPath) {
			continue
		}
		status, err := runCodeGit(root.WorkDir, "status", "--porcelain", "--untracked-files=no")
		if err == nil && strings.TrimSpace(status) != "" {
			return root
		}
		return nil
	}
	return nil
}

func restoreCodeDeliveryQualityRoot(root codeDeliveryQualityRoot) error {
	if _, err := runCodeGit(root.WorkDir, "reset", "--hard", root.Commit); err != nil {
		return err
	}
	_, err := runCodeGit(root.WorkDir, "clean", "-d", "-f")
	return err
}

func persistCodeDeliveryQualityResult(
	session *model.AIDevSession,
	userID uint,
	check codeQualityCheck,
	result codeQualityCheckResult,
	roots []codeDeliveryQualityRoot,
) error {
	if err := persistCodeQualityResult(session, userID, check, result); err != nil {
		return err
	}
	for _, root := range roots {
		identityDir := strings.TrimSpace(root.IdentityDir)
		if identityDir == "" {
			identityDir = root.WorkDir
		}
		if filepath.Clean(identityDir) != filepath.Clean(root.WorkDir) {
			continue
		}
		relative, err := filepath.Rel(root.WorkDir, check.workDirPath)
		if err != nil || filepath.IsAbs(relative) || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			continue
		}
		alias := check
		alias.ID = codeQualityCheckID(filepath.Join(identityDir, relative), check.Kind, check.Command)
		if alias.ID == check.ID {
			return nil
		}
		aliasResult := result
		aliasResult.CheckID = alias.ID
		return persistCodeQualityResult(session, userID, alias, aliasResult)
	}
	return nil
}

func codeDeliveryQualityRoots(session *model.AIDevSession) ([]codeDeliveryQualityRoot, error) {
	if session.IsolationMode == codeIsolationMultiWorktree || hasCodeMultiRepositoryDelivery(session.ID) {
		repositories, err := loadCodeSessionRepositories(session.ID)
		if err != nil || len(repositories) == 0 {
			return nil, errors.New("会话多仓库交付快照不可用")
		}
		roots := make([]codeDeliveryQualityRoot, 0, len(repositories))
		for _, repository := range repositories {
			if repository.Status == codeDeliveryCompleted {
				continue
			}
			roots = append(roots, codeDeliveryQualityRoot{
				WorkDir: repository.WorktreeDir, IdentityDir: repository.WorktreeDir,
				RuntimeDir: repository.SourceDir,
				Commit:     repository.WorktreeCommit, Label: "仓库 " + repository.LinkName,
			})
		}
		return roots, nil
	}
	var delivery model.AICodeDelivery
	if err := global.DB.Where("session_id = ?", session.ID).First(&delivery).Error; err != nil {
		return nil, err
	}
	workDir := strings.TrimSpace(session.WorkDir)
	commit, label := strings.TrimSpace(delivery.WorktreeCommit), "Worktree"
	if workDir == strings.TrimSpace(delivery.DeliveryWorkDir) && workDir != "" {
		commit, label = strings.TrimSpace(delivery.MergeCommit), "交付 Worktree"
	} else if workDir != strings.TrimSpace(delivery.WorkDir) {
		return nil, errors.New("质量检查目录与交付快照不一致")
	}
	return []codeDeliveryQualityRoot{{
		WorkDir: workDir, IdentityDir: delivery.WorkDir, RuntimeDir: delivery.SourceWorkDir,
		Commit: commit, Label: label,
	}}, nil
}

func detectCodeDeliveryQualityChecks(projectID uint, roots []codeDeliveryQualityRoot) []codeQualityCheck {
	paths := make([]string, 0, len(roots))
	for _, root := range roots {
		paths = append(paths, root.WorkDir)
	}
	checks := detectCodeQualityChecksInRoots(paths)
	checks = mergeCodeQualityChecks(checks, loadConfiguredCodeQualityChecks(projectID, roots))
	disableCodeDeliveryFlutterPub(checks)
	deliveryFingerprint := codeDeliveryQualityFingerprint(roots)
	for index := range checks {
		check := &checks[index]
		for _, root := range roots {
			relative, err := filepath.Rel(root.WorkDir, check.workDirPath)
			if err != nil || filepath.IsAbs(relative) || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
				continue
			}
			identityDir := strings.TrimSpace(root.IdentityDir)
			if identityDir == "" {
				identityDir = root.WorkDir
			}
			applyDartQualityToolchain(checks[index:index+1], identityDir)
			check.ID = codeQualityCheckID(
				filepath.Join(identityDir, relative), check.Kind, check.Command+"\x00"+deliveryFingerprint,
			)
			break
		}
	}
	return checks
}

func disableCodeDeliveryFlutterPub(checks []codeQualityCheck) {
	for index := range checks {
		check := &checks[index]
		if filepath.Base(check.Executable) != "flutter" || check.LocalScript || len(check.Args) == 0 {
			continue
		}
		if check.Args[0] != "analyze" && check.Args[0] != "test" {
			continue
		}
		check.Args = append(check.Args, "--no-pub")
		check.Command += " --no-pub"
	}
}

func codeDeliveryQualityFingerprint(roots []codeDeliveryQualityRoot) string {
	commits := make([]string, 0, len(roots))
	for _, root := range roots {
		identityDir := strings.TrimSpace(root.IdentityDir)
		if identityDir == "" {
			identityDir = root.WorkDir
		}
		commits = append(commits, filepath.Clean(identityDir)+"\x00"+strings.TrimSpace(root.Commit))
	}
	sort.Strings(commits)
	return codeQualityCheckID(strings.Join(commits, "\x00"), "delivery", "")
}

func validateCodeDeliveryQualityRoots(roots []codeDeliveryQualityRoot) error {
	for _, root := range roots {
		if strings.TrimSpace(root.WorkDir) == "" || strings.TrimSpace(root.Commit) == "" {
			return fmt.Errorf("%s 交付快照不可用", root.Label)
		}
		if err := verifyCodeDeliveryCommit(root.WorkDir, root.Commit, root.Label); err != nil {
			return err
		}
	}
	return nil
}
