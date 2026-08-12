package api

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
)

func aiSessionMultiDeliveryWorktreeDir(userID, sessionID uint, linkName string) string {
	return filepath.Join(aiProjectWorktreeRoot(userID), fmt.Sprintf("delivery_%d_multi", sessionID), linkName)
}

func codeMultiDeliveryRootDir(session *model.AIDevSession) string {
	if session == nil {
		return ""
	}
	return filepath.Join(aiProjectWorktreeRoot(session.UserID), fmt.Sprintf("delivery_%d_multi", session.ID))
}

func codeRepositoryIntegrationWorkDir(
	session *model.AIDevSession,
	repository *model.AIDevSessionRepository,
	repositories []model.AIDevSessionRepository,
) (string, error) {
	if session == nil || repository == nil {
		return "", errors.New("仓库集成交付元数据不可用")
	}
	workDir := aiSessionMultiDeliveryWorktreeDir(session.UserID, session.ID, repository.LinkName)
	if parentSource := strings.TrimSpace(repository.ParentSourceDir); parentSource != "" {
		var parent *model.AIDevSessionRepository
		for index := range repositories {
			if filepath.Clean(repositories[index].SourceDir) == filepath.Clean(parentSource) {
				parent = &repositories[index]
				break
			}
		}
		path := filepath.Clean(filepath.FromSlash(strings.TrimSpace(repository.GitlinkPath)))
		if parent == nil || strings.TrimSpace(parent.IntegrationWorkDir) == "" || filepath.IsAbs(path) ||
			path == "." || path == ".." || strings.HasPrefix(path, ".."+string(filepath.Separator)) {
			return "", fmt.Errorf("仓库 %s 的父仓库集成路径不可用", repository.LinkName)
		}
		workDir = filepath.Join(parent.IntegrationWorkDir, path)
		if !isPathInside(workDir, parent.IntegrationWorkDir) {
			return "", fmt.Errorf("仓库 %s 的集成交付路径越界", repository.LinkName)
		}
	}
	root := codeMultiDeliveryRootDir(session)
	if !isPathInside(workDir, root) {
		return "", errors.New("仓库集成交付 Worktree 不在 GoPanel 管理目录中")
	}
	return filepath.Clean(workDir), nil
}

func codeDeliveryRepositoriesInOrder(
	repositories []model.AIDevSessionRepository,
	parentFirst bool,
) ([]model.AIDevSessionRepository, error) {
	bySource := make(map[string]int, len(repositories))
	for index := range repositories {
		source := filepath.Clean(strings.TrimSpace(repositories[index].SourceDir))
		if source == "." || source == "" {
			return nil, fmt.Errorf("仓库 %s 的源目录不可用", repositories[index].LinkName)
		}
		if _, exists := bySource[source]; exists {
			return nil, fmt.Errorf("仓库 %s 的源目录重复", repositories[index].LinkName)
		}
		bySource[source] = index
	}
	children := make([][]int, len(repositories))
	roots := make([]int, 0, len(repositories))
	for index := range repositories {
		parentSource := strings.TrimSpace(repositories[index].ParentSourceDir)
		if parentSource == "" {
			roots = append(roots, index)
			continue
		}
		parentIndex, exists := bySource[filepath.Clean(parentSource)]
		if !exists {
			return nil, fmt.Errorf("仓库 %s 的父仓库不存在", repositories[index].LinkName)
		}
		if _, err := codeRepositoryGitlinkPath(&repositories[index]); err != nil {
			return nil, err
		}
		children[parentIndex] = append(children[parentIndex], index)
	}
	less := func(left, right int) bool {
		if repositories[left].LinkName != repositories[right].LinkName {
			return repositories[left].LinkName < repositories[right].LinkName
		}
		return filepath.Clean(repositories[left].SourceDir) < filepath.Clean(repositories[right].SourceDir)
	}
	sortCodeRepositoryIndexes(roots, less)
	for index := range children {
		sortCodeRepositoryIndexes(children[index], less)
	}
	state := make([]uint8, len(repositories))
	ordered := make([]model.AIDevSessionRepository, 0, len(repositories))
	var visit func(int) error
	visit = func(index int) error {
		if state[index] == 1 {
			return fmt.Errorf("仓库 %s 的父子关系存在循环", repositories[index].LinkName)
		}
		if state[index] == 2 {
			return nil
		}
		state[index] = 1
		if parentFirst {
			ordered = append(ordered, repositories[index])
		}
		for _, childIndex := range children[index] {
			if err := visit(childIndex); err != nil {
				return err
			}
		}
		if !parentFirst {
			ordered = append(ordered, repositories[index])
		}
		state[index] = 2
		return nil
	}
	for _, rootIndex := range roots {
		if err := visit(rootIndex); err != nil {
			return nil, err
		}
	}
	for index := range repositories {
		if state[index] == 0 {
			if err := visit(index); err != nil {
				return nil, err
			}
		}
	}
	return ordered, nil
}

func sortCodeRepositoryIndexes(indexes []int, less func(int, int) bool) {
	sort.SliceStable(indexes, func(left, right int) bool { return less(indexes[left], indexes[right]) })
}

func codeRepositoryGitlinkPath(repository *model.AIDevSessionRepository) (string, error) {
	path := filepath.Clean(filepath.FromSlash(strings.TrimSpace(repository.GitlinkPath)))
	if filepath.IsAbs(path) || path == "." || path == ".." || strings.HasPrefix(path, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("仓库 %s 的子仓库路径不可用", repository.LinkName)
	}
	return filepath.ToSlash(path), nil
}

func effectiveCodeRepositoryCommit(repository *model.AIDevSessionRepository) string {
	if repository == nil {
		return ""
	}
	for _, commit := range []string{repository.MergeCommit, repository.WorktreeCommit, repository.BaseCommit} {
		if commit = strings.TrimSpace(commit); commit != "" {
			return commit
		}
	}
	return ""
}

func materializeCompletedCodeRepositoryIntegration(repository *model.AIDevSessionRepository) error {
	commit := effectiveCodeRepositoryCommit(repository)
	if strings.TrimSpace(repository.IntegrationWorkDir) == "" || commit == "" {
		return fmt.Errorf("仓库 %s 的最终集成提交不可用", repository.LinkName)
	}
	if _, err := runCodeGit(repository.IntegrationWorkDir, "reset", "--hard", commit); err != nil {
		return fmt.Errorf("恢复仓库 %s 的最终集成树失败：%w", repository.LinkName, err)
	}
	if _, err := runCodeGit(repository.IntegrationWorkDir, "clean", "-d", "-f"); err != nil {
		return err
	}
	return verifyCodeDeliveryCommit(
		repository.IntegrationWorkDir, commit, "仓库 "+repository.LinkName+" 的最终集成提交",
	)
}

func restoreCodeMultiRepositoryIntegrationWorktrees(
	session *model.AIDevSession,
	repositories []model.AIDevSessionRepository,
) error {
	working, err := codeDeliveryRepositoriesInOrder(repositories, true)
	if err != nil {
		return err
	}
	for index := range working {
		if err := ensureCodeRepositoryIntegrationWorktree(session, &working[index], working); err != nil {
			return err
		}
		if err := materializeCompletedCodeRepositoryIntegration(&working[index]); err != nil {
			return err
		}
	}
	return nil
}

func ensureCodeRepositoryIntegrationWorktree(
	session *model.AIDevSession,
	repository *model.AIDevSessionRepository,
	repositories []model.AIDevSessionRepository,
) error {
	if session == nil || repository == nil {
		return errors.New("仓库集成交付基线不可用")
	}
	workDir, err := codeRepositoryIntegrationWorkDir(session, repository, repositories)
	if err != nil {
		return err
	}
	if stored := strings.TrimSpace(repository.IntegrationWorkDir); stored != "" && filepath.Clean(stored) != workDir {
		if _, statErr := os.Lstat(stored); statErr == nil {
			return errors.New("仓库集成交付 Worktree 路径与仓库关系不一致")
		}
	}
	exists, err := validateExistingCodeIntegrationWorktree(repository, workDir)
	if err != nil {
		return err
	}
	if !exists {
		baseCommit := ""
		if repository.Status == codeDeliveryCompleted {
			baseCommit = effectiveCodeRepositoryCommit(repository)
		} else {
			baseCommit, err = codeRepositoryIntegrationBaseCommit(repository.SourceCommit)
		}
		if err != nil || strings.TrimSpace(baseCommit) == "" {
			return errors.New("仓库集成交付基线不可用")
		}
		if err := os.MkdirAll(filepath.Dir(workDir), 0750); err != nil {
			return err
		}
		if _, err := runCodeGit(repository.SourceDir, "worktree", "prune", "--expire", "now"); err != nil {
			return err
		}
		if _, err := runCodeGit(repository.SourceDir, "worktree", "add", "--detach", workDir, baseCommit); err != nil {
			return fmt.Errorf("创建仓库 %s 的集成交付 Worktree 失败：%w", repository.LinkName, err)
		}
		if err := installCodeManagedPushGuard(workDir); err != nil {
			_, _ = runCodeGit(repository.SourceDir, "worktree", "remove", "--force", workDir)
			return err
		}
	}
	if repository.IntegrationWorkDir != workDir {
		if err := global.DB.Model(repository).Update("integration_work_dir", workDir).Error; err != nil {
			return err
		}
		repository.IntegrationWorkDir = workDir
	}
	return nil
}

func validateExistingCodeIntegrationWorktree(
	repository *model.AIDevSessionRepository,
	workDir string,
) (bool, error) {
	info, err := os.Lstat(workDir)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return false, errors.New("仓库集成交付目录不是有效目录")
	}
	topLevel, topLevelErr := runCodeGit(workDir, "rev-parse", "--show-toplevel")
	if topLevelErr != nil || filepath.Clean(topLevel) != filepath.Clean(workDir) {
		entries, readErr := os.ReadDir(workDir)
		if readErr != nil || len(entries) != 0 {
			return false, errors.New("仓库集成交付目录不是有效 Git Worktree")
		}
		if err := os.Remove(workDir); err != nil {
			return false, err
		}
		return false, nil
	}
	worktreeCommonDir, err := resolveCodeGitPath(workDir, "--git-common-dir")
	if err != nil {
		return false, err
	}
	sourceCommonDir, err := resolveCodeGitPath(repository.SourceDir, "--git-common-dir")
	if err != nil || worktreeCommonDir != sourceCommonDir {
		return false, errors.New("仓库集成交付 Worktree 与源仓库不匹配")
	}
	return true, nil
}

func resetCodeRepositoryIntegrationWorktree(
	session *model.AIDevSession,
	repository *model.AIDevSessionRepository,
	repositories []model.AIDevSessionRepository,
) error {
	if err := ensureCodeRepositoryIntegrationWorktree(session, repository, repositories); err != nil {
		return err
	}
	baseCommit, err := codeRepositoryIntegrationBaseCommit(repository.SourceCommit)
	if err != nil {
		return err
	}
	_, _ = runCodeGit(repository.IntegrationWorkDir, "merge", "--abort")
	if _, err := runCodeGit(repository.IntegrationWorkDir, "reset", "--hard", baseCommit); err != nil {
		return fmt.Errorf("重置仓库 %s 的集成交付 Worktree 失败：%w", repository.LinkName, err)
	}
	_, err = runCodeGit(repository.IntegrationWorkDir, "clean", "-d", "-f")
	return err
}

func prepareCodeRepositoryIntegration(
	session *model.AIDevSession,
	repository *model.AIDevSessionRepository,
	repositories []model.AIDevSessionRepository,
) (codeRepositoryDeliveryResult, error) {
	result := codeStoredRepositoryDeliveryResult(repository)
	if err := resetCodeRepositoryIntegrationWorktree(session, repository, repositories); err != nil {
		return result, err
	}
	if _, err := runCodeGit(
		repository.IntegrationWorkDir,
		codeGitAuthoredArgs(
			"-c", "commit.gpgsign=false", "-c", "core.commitGraph=false",
			"merge", "--no-ff",
			"-m", codeDeliveryMergeMessage(session, repository.LinkName), repository.WorktreeCommit,
		)...,
	); err != nil {
		conflicts := codeGitConflictFiles(repository.IntegrationWorkDir)
		if len(conflicts) == 0 {
			return result, fmt.Errorf("合并仓库 %s 的交付快照失败：%w", repository.LinkName, err)
		}
		if err := resolveCodeIntegrationGitlinkConflicts(repository, repositories, conflicts); err != nil {
			return result, err
		}
		conflicts = codeGitConflictFiles(repository.IntegrationWorkDir)
		if len(conflicts) > 0 {
			result.Status, result.ConflictFiles = codeDeliveryJobConflict, conflicts
			repository.Status, repository.ErrorMessage = codeDeliveryJobConflict, strings.Join(conflicts, ", ")
			_ = global.DB.Model(repository).Updates(map[string]any{
				"status": repository.Status, "error_message": repository.ErrorMessage,
			}).Error
			return result, nil
		}
		if _, err := runCodeGit(
			repository.IntegrationWorkDir, codeResolvedMergeCommitArgs()...,
		); err != nil {
			return result, fmt.Errorf("提交仓库 %s 的受管子仓库合并结果失败：%w", repository.LinkName, err)
		}
	}
	if err := commitCodeIntegrationGitlinkUpdates(repository, repositories); err != nil {
		return result, err
	}
	commit, err := runCodeGit(repository.IntegrationWorkDir, "rev-parse", "HEAD")
	if err != nil {
		return result, err
	}
	now := time.Now()
	if err := global.DB.Model(repository).Updates(map[string]any{
		"status": codeDeliveryMerged, "merge_commit": commit, "merged_at": now, "error_message": "",
	}).Error; err != nil {
		return result, err
	}
	repository.Status, repository.MergeCommit, repository.MergedAt = codeDeliveryMerged, commit, &now
	result.Status, result.Commit, result.ErrorMessage = codeDeliveryMerged, commit, ""
	return result, nil
}

func resolveCodeIntegrationGitlinkConflicts(
	repository *model.AIDevSessionRepository,
	repositories []model.AIDevSessionRepository,
	conflicts []string,
) error {
	unmerged := make(map[string]struct{}, len(conflicts))
	for _, path := range conflicts {
		unmerged[filepath.ToSlash(strings.TrimSpace(path))] = struct{}{}
	}
	for index := range repositories {
		child := &repositories[index]
		path, err := codeRepositoryGitlinkPath(child)
		if err != nil || filepath.Clean(child.ParentSourceDir) != filepath.Clean(repository.SourceDir) {
			continue
		}
		commit := effectiveCodeRepositoryCommit(child)
		if commit == "" {
			continue
		}
		if _, exists := unmerged[path]; !exists {
			continue
		}
		if err := updateCodeGitlink(repository.IntegrationWorkDir, path, commit); err != nil {
			return fmt.Errorf("解析仓库 %s 的受管子仓库冲突失败：%w", repository.LinkName, err)
		}
	}
	return nil
}

func commitCodeIntegrationGitlinkUpdates(
	repository *model.AIDevSessionRepository,
	repositories []model.AIDevSessionRepository,
) error {
	expected := make(map[string]struct{})
	for index := range repositories {
		child := &repositories[index]
		commit := effectiveCodeRepositoryCommit(child)
		if filepath.Clean(child.ParentSourceDir) != filepath.Clean(repository.SourceDir) || child.GitlinkPath == "" || commit == "" {
			continue
		}
		path, err := codeRepositoryGitlinkPath(child)
		if err != nil {
			return err
		}
		if err := updateCodeGitlink(repository.IntegrationWorkDir, path, commit); err != nil {
			return fmt.Errorf("同步父仓库 %s 的子仓库指针失败：%w", repository.LinkName, err)
		}
		expected[path] = struct{}{}
	}
	if len(expected) == 0 {
		return nil
	}
	staged, err := runCodeGit(repository.IntegrationWorkDir, "diff", "--cached", "--name-only")
	if err != nil || strings.TrimSpace(staged) == "" {
		return err
	}
	for _, path := range strings.Split(staged, "\n") {
		if _, allowed := expected[filepath.ToSlash(strings.TrimSpace(path))]; !allowed {
			return fmt.Errorf("仓库 %s 包含非预期的暂存修改：%s", repository.LinkName, path)
		}
	}
	_, err = runCodeGit(
		repository.IntegrationWorkDir,
		codeGitAuthoredArgs(
			"-c", "commit.gpgsign=false", "commit", "-m", "chore: update nested repository pointers",
		)...,
	)
	return err
}
