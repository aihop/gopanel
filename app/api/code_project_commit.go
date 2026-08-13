package api

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/utils/token"
	"github.com/gofiber/fiber/v3"
)

type codeProjectCommitResult struct {
	RepositoryName string `json:"repositoryName"`
	Status         string `json:"status"`
	Commit         string `json:"commit,omitempty"`
	Files          int    `json:"files"`
	ErrorMessage   string `json:"errorMessage,omitempty"`
}

const (
	codeProjectCommitStatusCommitted = "committed"
	codeProjectCommitStatusClean     = "clean"
	codeProjectCommitStatusFailed    = "failed"
)

// CommitCodeProjectChanges 把项目各仓库的未提交改动就地提交。
//
// 建会话前的"先提交"路径需要它：不提供的话，用户为了开一个会话得切出面板
// 去命令行提交，多数人于是选了"复制进隔离区"——而那条路会把人写的改动和
// AI 的产出混在一起，还会让源仓库一直脏到交付被拒。
//
// 这是面板里少有的、直接改用户源仓库的写操作，所以边界收得很紧：
// 只碰项目声明的仓库、必须显式确认、提交信息由用户给。
func CommitCodeProjectChanges(c fiber.Ctx) error {
	startedAt := time.Now()
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	projectID, err := strconv.Atoi(c.Params("id"))
	if err != nil || projectID <= 0 {
		return c.JSON(e.Fail(errors.New("项目参数无效")))
	}
	var req struct {
		Message string `json:"message"`
		Confirm bool   `json:"confirm"`
	}
	if err := c.Bind().JSON(&req); err != nil {
		return c.JSON(e.Fail(err))
	}
	if !req.Confirm {
		return c.JSON(e.Fail(errors.New("提交源仓库改动需要明确确认")))
	}
	message, err := validateCodeGitCommitMessage(req.Message)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	project, err := repo.NewAIProjectRepo().GetProjectByID(uint(projectID))
	if err != nil {
		return c.JSON(e.Fail(errors.New("项目不存在")))
	}
	if project.CreatorID != claims.UserId && claims.Role != constant.UserRoleSuper {
		return c.JSON(e.Fail(errors.New("无权操作该项目")))
	}
	if err := beginCodeRepositoryOperation(project); err != nil {
		if errors.Is(err, errCodeProjectActiveTerminal) {
			return c.JSON(e.Fail(errors.New("项目主仓仍有活动终端，请关闭后再提交")))
		}
		return c.JSON(e.Fail(err))
	}
	defer endCodeRepositoryOperation(project)
	candidates, err := discoverCodeProjectRepositoryCandidates(project, codeProjectSourceDirs(project))
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if len(candidates) == 0 {
		return c.JSON(e.Fail(errors.New("项目目录中未发现 Git 仓库")))
	}
	specs, err := codeProjectRepositorySpecsWithCandidates(project, codeProjectSourceDirs(project), candidates)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	leaseKeys := make([]string, 0, len(specs))
	for _, spec := range specs {
		leaseKeys = append(leaseKeys, spec.LeaseKey)
	}
	leaseOwner := newCodeRepositoryLeaseOwner("project-commit")
	acquired, err := acquireCodeRepositoryLeases(leaseOwner, 0, leaseKeys)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if !acquired {
		return c.JSON(e.Fail(errors.New("项目主仓正在同步或交付，请稍后重试")))
	}
	defer func() { _ = releaseCodeRepositoryLeases(leaseOwner, leaseKeys) }()
	leaseContext, cancelHeartbeat := context.WithCancel(context.Background())
	defer cancelHeartbeat()
	go heartbeatCodeRepositoryLeases(leaseContext, leaseOwner, leaseKeys)
	results := make([]codeProjectCommitResult, 0, len(candidates))
	for _, candidate := range candidates {
		results = append(results, commitCodeProjectRepository(candidate.SourceDir, message))
	}
	recordCodeAudit(claims.UserId, project.ID, 0, "project_git_commit",
		summarizeCodeProjectCommit(results), "", message, c.IP(), startedAt, nil)
	return c.JSON(e.Succ(results))
}

func summarizeCodeProjectCommit(results []codeProjectCommitResult) string {
	for _, result := range results {
		if result.Status == codeProjectCommitStatusFailed {
			return "failed"
		}
	}
	return "success"
}

func commitCodeProjectRepository(sourceDir, message string) (result codeProjectCommitResult) {
	result.RepositoryName = codeRepositoryDisplayName(sourceDir)
	status, err := runCodeGit(sourceDir, "status", "--porcelain")
	if err != nil {
		result.Status, result.ErrorMessage = codeProjectCommitStatusFailed, err.Error()
		return result
	}
	if strings.TrimSpace(status) == "" {
		result.Status = codeProjectCommitStatusClean
		return result
	}
	if err := validateCodeGitSaveFiles(sourceDir); err != nil {
		result.Status, result.ErrorMessage = codeProjectCommitStatusFailed, err.Error()
		return result
	}
	indexSnapshot, err := snapshotCodeGitIndex(sourceDir)
	if err != nil {
		result.Status, result.ErrorMessage = codeProjectCommitStatusFailed, err.Error()
		return result
	}
	restoreIndex := true
	defer func() {
		if restoreIndex {
			if restoreErr := indexSnapshot.restore(); restoreErr != nil {
				result.ErrorMessage = strings.TrimSpace(result.ErrorMessage + "；恢复原暂存区失败：" + restoreErr.Error())
			}
		}
	}()
	// 用 add -A 而不是只提交已暂存的：用户点「先提交」的意图是
	// 「把工作区清干净再开会话」，留下未暂存的改动等于没解决问题。
	// .gitignore 之外的垃圾文件本就该被 ignore，不该靠这里兜底。
	if _, err := runCodeGit(sourceDir, "add", "-A"); err != nil {
		result.Status, result.ErrorMessage = codeProjectCommitStatusFailed, err.Error()
		return result
	}
	staged, err := runCodeGit(sourceDir, "diff", "--cached", "--name-only")
	if err != nil {
		result.Status, result.ErrorMessage = codeProjectCommitStatusFailed, err.Error()
		return result
	}
	if strings.TrimSpace(staged) == "" {
		// status 非空但暂存区为空：改动全被 .gitignore 挡住了，没什么可提交的。
		result.Status = codeProjectCommitStatusClean
		return result
	}
	result.Files = len(strings.Split(strings.TrimSpace(staged), "\n"))
	if err := validateCodeGitStagedChanges(sourceDir); err != nil {
		result.Status, result.ErrorMessage = codeProjectCommitStatusFailed, err.Error()
		return result
	}
	// 用用户自己的 git 身份，不套 GoPanel Code：这是人写的代码，
	// 提交人就该是人。面板只是替他省了切终端这一步。
	if _, err := runCodeGit(sourceDir, "commit", "-m", message); err != nil {
		result.Status, result.ErrorMessage = codeProjectCommitStatusFailed, err.Error()
		return result
	}
	restoreIndex = false
	commit, err := runCodeGit(sourceDir, "rev-parse", "HEAD")
	if err != nil {
		result.Status, result.ErrorMessage = codeProjectCommitStatusFailed, err.Error()
		return result
	}
	result.Status, result.Commit = codeProjectCommitStatusCommitted, strings.TrimSpace(commit)
	return result
}

type codeGitIndexSnapshot struct {
	path   string
	data   []byte
	mode   os.FileMode
	exists bool
}

func snapshotCodeGitIndex(sourceDir string) (codeGitIndexSnapshot, error) {
	indexPath, err := runCodeGit(sourceDir, "rev-parse", "--git-path", "index")
	if err != nil {
		return codeGitIndexSnapshot{}, err
	}
	if !filepath.IsAbs(indexPath) {
		indexPath = filepath.Join(sourceDir, indexPath)
	}
	snapshot := codeGitIndexSnapshot{path: filepath.Clean(indexPath)}
	info, err := os.Stat(snapshot.path)
	if errors.Is(err, os.ErrNotExist) {
		return snapshot, nil
	}
	if err != nil {
		return codeGitIndexSnapshot{}, fmt.Errorf("读取 Git 暂存区失败：%w", err)
	}
	snapshot.data, err = os.ReadFile(snapshot.path)
	if err != nil {
		return codeGitIndexSnapshot{}, fmt.Errorf("读取 Git 暂存区失败：%w", err)
	}
	snapshot.exists, snapshot.mode = true, info.Mode().Perm()
	return snapshot, nil
}

func (snapshot codeGitIndexSnapshot) restore() error {
	if !snapshot.exists {
		if err := os.Remove(snapshot.path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		return nil
	}
	temporary, err := os.CreateTemp(filepath.Dir(snapshot.path), ".gopanel-index-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if _, err = temporary.Write(snapshot.data); err == nil {
		err = temporary.Chmod(snapshot.mode)
	}
	if closeErr := temporary.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	return os.Rename(temporaryPath, snapshot.path)
}

func codeRepositoryDisplayName(sourceDir string) string {
	trimmed := strings.TrimRight(strings.TrimSpace(sourceDir), "/")
	if index := strings.LastIndex(trimmed, "/"); index >= 0 && index+1 < len(trimmed) {
		return trimmed[index+1:]
	}
	if trimmed == "" {
		return "repository"
	}
	return trimmed
}
