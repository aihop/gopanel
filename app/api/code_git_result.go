package api

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
	"gorm.io/gorm"
)

func discoverCodeGitResultRepositories(session *model.AIDevSession, excludedRepositories []string) []codeGitRepository {
	if session == nil {
		return nil
	}
	if session.IsolationMode == codeIsolationMultiWorktree || (global.DB != nil && hasCodeMultiRepositoryDelivery(session.ID)) {
		rows, err := loadCodeSessionRepositories(session.ID)
		if err != nil {
			return nil
		}
		result := make([]codeGitRepository, 0, len(rows))
		for _, row := range rows {
			repository, ok := inspectCodeGitResultRepository(
				codeSessionRepositoryID(row.ID), row.LinkName, row.WorktreeDir, row.SourceDir, row.LinkName,
				row.BaseCommit, row.WorktreeCommit, row.Status,
			)
			if ok {
				result = append(result, repository)
			}
		}
		return result
	}
	if global.DB != nil {
		var delivery model.AICodeDelivery
		deliveryErr := global.DB.Where("session_id = ?", session.ID).First(&delivery).Error
		if deliveryErr == nil {
			repository, ok := inspectCodeGitResultRepository(
				"session", filepath.Base(delivery.SourceWorkDir), delivery.WorkDir, delivery.SourceWorkDir, "",
				delivery.BaseCommit, delivery.WorktreeCommit, delivery.Status,
			)
			if ok {
				return []codeGitRepository{repository}
			}
		} else if !errors.Is(deliveryErr, gorm.ErrRecordNotFound) {
			return nil
		}
	}
	if strings.TrimSpace(session.BaseCommit) == "" || strings.TrimSpace(session.WorktreeBranch) == "" {
		return nil
	}
	repository, ok := inspectCodeGitResultRepository(
		"session", filepath.Base(session.SourceWorkDir), session.WorkDir, session.SourceWorkDir, "",
		session.BaseCommit, "", session.Status,
	)
	if !ok {
		return nil
	}
	return []codeGitRepository{repository}
}

func inspectCodeGitResultRepository(
	id, name, worktreeDir, sourceDir, workspacePrefix, baseCommit, storedCommit, deliveryStatus string,
) (codeGitRepository, bool) {
	root, live := worktreeDir, true
	repository, ok := inspectCodeGitRepository(id, name, root, workspacePrefix)
	if !ok {
		root, live = sourceDir, false
		repository, ok = inspectCodeGitRepository(id, name, root, workspacePrefix)
	}
	if !ok || strings.TrimSpace(baseCommit) == "" {
		return codeGitRepository{}, false
	}
	resultCommit := strings.TrimSpace(storedCommit)
	if live {
		resultCommit, _ = runCodeGit(repository.root, "rev-parse", "HEAD")
		resultCommit = strings.TrimSpace(resultCommit)
	}
	if resultCommit == "" {
		return codeGitRepository{}, false
	}
	if _, err := runCodeGit(repository.root, "cat-file", "-e", strings.TrimSpace(baseCommit)+"^{commit}"); err != nil {
		return codeGitRepository{}, false
	}
	if _, err := runCodeGit(repository.root, "cat-file", "-e", resultCommit+"^{commit}"); err != nil {
		return codeGitRepository{}, false
	}
	repository.Isolated = true
	repository.BaseCommit = strings.TrimSpace(baseCommit)
	repository.ResultCommit = resultCommit
	repository.HeadCommit = shortCodeGitCommit(resultCommit)
	repository.DeliveryStatus = deliveryStatus
	repository.resultLive = live
	repository.ReviewState = "saved"
	if live {
		repository.ReviewState = "live"
	}
	if !live || deliveryStatus == codeDeliveryCompleted || deliveryStatus == codeDeliveryWorktreeCleaned {
		repository.ReviewState = "delivered"
	}
	return repository, true
}

func shortCodeGitCommit(commit string) string {
	commit = strings.TrimSpace(commit)
	if len(commit) <= 8 {
		return commit
	}
	return commit[:8]
}

func loadCodeGitResultStatus(session *model.AIDevSession, excludedRepositories []string) (codeGitStatus, error) {
	repositories := discoverCodeGitResultRepositories(session, excludedRepositories)
	result := codeGitStatus{
		Available: len(repositories) > 0, Reason: "result_unavailable", Scope: "result",
		ReviewReady: len(repositories) > 0, Repositories: make([]codeGitRepository, 0, len(repositories)),
	}
	if result.Available {
		result.Reason = ""
	}
	for _, repository := range repositories {
		loaded, clean, err := loadCodeGitResultRepositoryStatus(repository)
		if err != nil {
			return codeGitStatus{}, err
		}
		result.Repositories = append(result.Repositories, loaded)
		result.Files += len(loaded.Files)
		result.Changed += loaded.ChangedCount
		result.Additions += loaded.Additions
		result.Deletions += loaded.Deletions
		result.ReviewReady = result.ReviewReady && clean
	}
	if result.ReviewReady {
		result.ReviewRevision = codeGitReviewRevision(result.Repositories)
	}
	return result, nil
}

func loadCodeGitResultRepositoryStatus(repository codeGitRepository) (codeGitRepository, bool, error) {
	clean := true
	if repository.resultLive {
		status, _, err := runCodeGitReviewCommand(
			repository.root, false, 4*codeGitDiffOutputLimit, "status", "--porcelain=v1", "-z", "--untracked-files=all",
		)
		if err != nil {
			return repository, false, err
		}
		clean = strings.TrimSpace(status) == ""
	}
	ref := repository.BaseCommit + ".." + repository.ResultCommit
	nameStatus, truncated, err := runCodeGitReviewCommand(
		repository.root, false, 4*codeGitDiffOutputLimit,
		"--literal-pathspecs", "diff", "--name-status", "-z", "--find-renames", ref,
	)
	if err != nil {
		return repository, clean, err
	}
	if truncated {
		return repository, clean, errors.New("Git 任务差异输出过大，请减少单次任务变更")
	}
	repository.Files = parseCodeGitResultFiles(nameStatus, repository.workspacePrefix)
	if len(repository.Files) > codeGitStatusFileLimit {
		repository.Files = repository.Files[:codeGitStatusFileLimit]
		repository.Truncated = true
	}
	repository.ChangedCount = len(repository.Files)
	numstat, _, _ := runCodeGitReviewCommand(
		repository.root, false, codeGitDiffOutputLimit, "--literal-pathspecs", "diff", "--numstat", ref,
	)
	repository.Additions, repository.Deletions = parseCodeGitNumstat(numstat)
	return repository, clean, nil
}

func parseCodeGitResultFiles(output, workspacePrefix string) []codeGitFile {
	records := strings.Split(output, "\x00")
	files := make([]codeGitFile, 0, len(records)/2)
	for index := 0; index < len(records); {
		status := strings.TrimSpace(records[index])
		index++
		if status == "" || index >= len(records) {
			continue
		}
		oldPath, filePath := "", records[index]
		index++
		if strings.HasPrefix(status, "R") || strings.HasPrefix(status, "C") {
			oldPath = filePath
			if index >= len(records) {
				break
			}
			filePath = records[index]
			index++
		}
		files = append(files, codeGitFile{
			Path: filePath, OldPath: oldPath, WorkspacePath: path.Join(workspacePrefix, filepath.ToSlash(filePath)),
			ResultStatus: string(status[0]), Changed: true,
		})
	}
	return files
}

func codeGitReviewRevision(repositories []codeGitRepository) string {
	parts := make([]string, 0, len(repositories))
	for _, repository := range repositories {
		parts = append(parts, repository.ID+"="+repository.BaseCommit+".."+repository.ResultCommit)
	}
	sort.Strings(parts)
	sum := sha256.Sum256([]byte(strings.Join(parts, "\n")))
	return fmt.Sprintf("%x", sum[:])
}

func validateCodeGitReviewRevision(session *model.AIDevSession, expected string) error {
	status, err := loadCodeGitResultStatus(session, nil)
	if err != nil {
		return err
	}
	if !status.ReviewReady || status.ReviewRevision != strings.TrimSpace(expected) {
		return errors.New("任务变更已发生变化，请重新评审后再交付")
	}
	return nil
}

func loadCodeGitResultFileDiff(
	session *model.AIDevSession, excludedRepositories []string, repositoryID, filePath string,
) (string, bool, error) {
	repository, err := findCodeGitRepository(
		discoverCodeGitResultRepositories(session, excludedRepositories), strings.TrimSpace(repositoryID),
	)
	if err != nil {
		return "", false, err
	}
	cleanPath := filepath.ToSlash(path.Clean(strings.TrimSpace(filePath)))
	if cleanPath == "." || path.IsAbs(cleanPath) || strings.HasPrefix(cleanPath, "../") {
		return "", false, errors.New("Git 文件路径无效")
	}
	loaded, _, err := loadCodeGitResultRepositoryStatus(*repository)
	if err != nil {
		return "", false, err
	}
	var file *codeGitFile
	for index := range loaded.Files {
		if filepath.ToSlash(loaded.Files[index].Path) == cleanPath {
			file = &loaded.Files[index]
			break
		}
	}
	if file == nil {
		return "", false, errors.New("文件不在当前任务变更中")
	}
	ref := repository.BaseCommit + ".." + repository.ResultCommit
	return runCodeGitReviewCommand(
		repository.root, false, codeGitDiffOutputLimit,
		"--literal-pathspecs", "diff", "--no-ext-diff", "--unified=3", ref, "--", file.Path,
	)
}
