package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
)

type codeDeliveryConflictRepository struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Branch          string   `json:"branch"`
	TargetBranch    string   `json:"targetBranch"`
	Files           []string `json:"files"`
	UnresolvedFiles []string `json:"unresolvedFiles"`
	Resolved        int      `json:"resolved"`
	Total           int      `json:"total"`
}

type codeDeliveryConflictFile struct {
	RepositoryID  string `json:"repositoryId"`
	Path          string `json:"path"`
	BaseContent   string `json:"baseContent,omitempty"`
	MainContent   string `json:"mainContent,omitempty"`
	TaskContent   string `json:"taskContent,omitempty"`
	ResultContent string `json:"resultContent,omitempty"`
	BaseExists    bool   `json:"baseExists"`
	MainExists    bool   `json:"mainExists"`
	TaskExists    bool   `json:"taskExists"`
	ResultExists  bool   `json:"resultExists"`
	Binary        bool   `json:"binary"`
	Resolved      bool   `json:"resolved"`
	Version       string `json:"version"`
}

type codeDeliveryConflictContext struct {
	RepositoryID string
	Name         string
	Branch       string
	TargetBranch string
	SourceDir    string
	WorkDir      string
	SourceCommit string
	TaskCommit   string
	Files        []string
	Delivery     *model.AICodeDelivery
	Repository   *model.AIDevSessionRepository
}

func loadCodeDeliveryConflictContexts(sessionID uint) (*model.AICodeDeliveryJob, []codeDeliveryConflictContext, error) {
	var job model.AICodeDeliveryJob
	if err := global.DB.Where("session_id = ?", sessionID).First(&job).Error; err != nil {
		return nil, nil, err
	}
	if job.Status != codeDeliveryJobConflict {
		return nil, nil, errors.New("当前交付任务没有等待处理的冲突")
	}
	var storedResults []codeRepositoryDeliveryResult
	_ = json.Unmarshal([]byte(job.RepositoryResults), &storedResults)
	filesByRepository := make(map[string][]string, len(storedResults))
	for _, result := range storedResults {
		if len(result.ConflictFiles) > 0 {
			filesByRepository[result.RepositoryID] = normalizedCodeConflictFiles(result.ConflictFiles)
		}
	}
	var jobConflictFiles []string
	_ = json.Unmarshal([]byte(job.ConflictFiles), &jobConflictFiles)
	jobConflictFiles = normalizedCodeConflictFiles(jobConflictFiles)
	var repositories []model.AIDevSessionRepository
	if err := global.DB.Where("session_id = ?", sessionID).Order("id asc").Find(&repositories).Error; err != nil {
		return nil, nil, err
	}
	contexts := make([]codeDeliveryConflictContext, 0)
	if len(repositories) > 0 {
		conflictRepositoryID := ""
		for index := range repositories {
			if repositories[index].Status == codeDeliveryJobConflict {
				if conflictRepositoryID != "" {
					conflictRepositoryID = ""
					break
				}
				conflictRepositoryID = codeSessionRepositoryID(repositories[index].ID)
			}
		}
		if conflictRepositoryID != "" && len(filesByRepository[conflictRepositoryID]) == 0 {
			filesByRepository[conflictRepositoryID] = jobConflictFiles
		}
		for index := range repositories {
			repository := &repositories[index]
			if strings.TrimSpace(repository.IntegrationWorkDir) == "" {
				continue
			}
			files := mergeCodeConflictFiles(
				filesByRepository[codeSessionRepositoryID(repository.ID)],
				discoverCodeDeliveryConflictFiles(repository.IntegrationWorkDir),
			)
			if len(files) == 0 {
				continue
			}
			contexts = append(contexts, codeDeliveryConflictContext{
				RepositoryID: codeSessionRepositoryID(repository.ID), Name: repository.LinkName,
				Branch: repository.Branch, TargetBranch: repository.TargetBranch,
				SourceDir: repository.SourceDir, WorkDir: repository.IntegrationWorkDir, SourceCommit: repository.SourceCommit,
				TaskCommit: repository.WorktreeCommit, Files: files, Repository: repository,
			})
		}
	} else {
		var delivery model.AICodeDelivery
		if err := global.DB.Where("session_id = ?", sessionID).First(&delivery).Error; err != nil {
			return nil, nil, err
		}
		files := jobConflictFiles
		if len(files) > 0 && strings.TrimSpace(delivery.DeliveryWorkDir) != "" {
			contexts = append(contexts, codeDeliveryConflictContext{
				RepositoryID: "session", Name: filepath.Base(delivery.SourceWorkDir),
				Branch: delivery.WorktreeBranch, TargetBranch: delivery.TargetBranch,
				SourceDir: delivery.SourceWorkDir, WorkDir: delivery.DeliveryWorkDir, SourceCommit: delivery.SourceCommit,
				TaskCommit: delivery.WorktreeCommit, Files: files, Delivery: &delivery,
			})
		}
	}
	if len(contexts) == 0 {
		return nil, nil, errors.New("交付冲突现场不可用，请重新发起交付")
	}
	for index := range contexts {
		if err := validateCodeDeliveryConflictWorktree(sessionID, job.UserID, &contexts[index]); err != nil {
			return nil, nil, err
		}
	}
	return &job, contexts, nil
}

func normalizedCodeConflictFiles(files []string) []string {
	seen := make(map[string]struct{}, len(files))
	result := make([]string, 0, len(files))
	for _, file := range files {
		clean := filepath.ToSlash(filepath.Clean(strings.TrimSpace(file)))
		if clean == "." || clean == "" || filepath.IsAbs(clean) || strings.HasPrefix(clean, "../") {
			continue
		}
		if _, exists := seen[clean]; exists {
			continue
		}
		seen[clean] = struct{}{}
		result = append(result, clean)
	}
	sort.Strings(result)
	return result
}

func mergeCodeConflictFiles(groups ...[]string) []string {
	files := make([]string, 0)
	for _, group := range groups {
		files = append(files, group...)
	}
	return normalizedCodeConflictFiles(files)
}

func discoverCodeDeliveryConflictFiles(workDir string) []string {
	files := codeGitConflictFiles(workDir)
	changed, err := runCodeGitBytes(workDir, nil, "diff", "--name-only", "--diff-filter=ACMR", "-z", "HEAD")
	if err != nil {
		return normalizedCodeConflictFiles(files)
	}
	for _, rawPath := range bytes.Split(changed, []byte{0}) {
		file := strings.TrimSpace(string(rawPath))
		if file == "" {
			continue
		}
		content, exists, readErr := readCodeConflictResultFile(workDir, filepath.ToSlash(file))
		if readErr == nil && exists && hasCodeConflictMarkerLines(content) {
			files = append(files, file)
		}
	}
	return normalizedCodeConflictFiles(files)
}

func validateCodeDeliveryConflictWorktree(sessionID, userID uint, context *codeDeliveryConflictContext) error {
	if context == nil || strings.TrimSpace(context.WorkDir) == "" {
		return errors.New("交付冲突 Worktree 不可用")
	}
	if context.Delivery != nil {
		if !isManagedAIDeliveryWorkDir(context.WorkDir, userID, sessionID) {
			return errors.New("交付冲突 Worktree 不在 GoPanel 管理目录中")
		}
	} else if !isPathInside(context.WorkDir, filepath.Join(aiProjectWorktreeRoot(userID), fmt.Sprintf("delivery_%d_multi", sessionID))) {
		return errors.New("仓库冲突 Worktree 不在 GoPanel 管理目录中")
	}
	topLevel, err := runCodeGit(context.WorkDir, "rev-parse", "--show-toplevel")
	if err != nil || filepath.Clean(topLevel) != filepath.Clean(context.WorkDir) {
		return errors.New("交付冲突 Worktree 不是有效 Git 工作区")
	}
	return nil
}

func codeDeliveryConflictRepositoryViews(contexts []codeDeliveryConflictContext) []codeDeliveryConflictRepository {
	views := make([]codeDeliveryConflictRepository, 0, len(contexts))
	for index := range contexts {
		context := &contexts[index]
		unresolved := discoverCodeDeliveryConflictFiles(context.WorkDir)
		unresolvedSet := make(map[string]struct{}, len(unresolved))
		for _, file := range unresolved {
			unresolvedSet[filepath.ToSlash(file)] = struct{}{}
		}
		resolved := 0
		for _, file := range context.Files {
			if _, exists := unresolvedSet[file]; !exists {
				resolved++
			}
		}
		views = append(views, codeDeliveryConflictRepository{
			ID: context.RepositoryID, Name: context.Name, Branch: context.Branch,
			TargetBranch: context.TargetBranch, Files: context.Files,
			UnresolvedFiles: normalizedCodeConflictFiles(unresolved), Resolved: resolved, Total: len(context.Files),
		})
	}
	return views
}

func findCodeDeliveryConflictContext(contexts []codeDeliveryConflictContext, repositoryID, file string) (*codeDeliveryConflictContext, string, error) {
	file = filepath.ToSlash(filepath.Clean(strings.TrimSpace(file)))
	if file == "." || file == "" || filepath.IsAbs(file) || strings.HasPrefix(file, "../") {
		return nil, "", errors.New("冲突文件路径无效")
	}
	for index := range contexts {
		context := &contexts[index]
		if context.RepositoryID != strings.TrimSpace(repositoryID) {
			continue
		}
		for _, allowed := range context.Files {
			if allowed == file {
				return context, file, nil
			}
		}
		return nil, "", errors.New("该文件不属于当前交付冲突")
	}
	return nil, "", errors.New("冲突仓库不存在或不属于当前会话")
}

func readCodeDeliveryConflictFile(context *codeDeliveryConflictContext, file string) (codeDeliveryConflictFile, error) {
	baseCommit, _ := runCodeGit(context.WorkDir, "merge-base", context.SourceCommit, context.TaskCommit)
	base, baseExists, err := readCodeConflictCommitFile(context.WorkDir, baseCommit, file)
	if err != nil {
		return codeDeliveryConflictFile{}, err
	}
	main, mainExists, err := readCodeConflictCommitFile(context.WorkDir, context.SourceCommit, file)
	if err != nil {
		return codeDeliveryConflictFile{}, err
	}
	task, taskExists, err := readCodeConflictCommitFile(context.WorkDir, context.TaskCommit, file)
	if err != nil {
		return codeDeliveryConflictFile{}, err
	}
	result, resultExists, err := readCodeConflictResultFile(context.WorkDir, file)
	if err != nil {
		return codeDeliveryConflictFile{}, err
	}
	binary := isCodeConflictBinary(base, baseExists) || isCodeConflictBinary(main, mainExists) ||
		isCodeConflictBinary(task, taskExists) || isCodeConflictBinary(result, resultExists)
	unresolved := false
	for _, conflict := range discoverCodeDeliveryConflictFiles(context.WorkDir) {
		if filepath.ToSlash(conflict) == file {
			unresolved = true
			break
		}
	}
	response := codeDeliveryConflictFile{
		RepositoryID: context.RepositoryID, Path: file,
		BaseExists: baseExists, MainExists: mainExists, TaskExists: taskExists,
		ResultExists: resultExists, Binary: binary, Resolved: !unresolved,
		Version: codeConflictResultVersion(result, resultExists),
	}
	if !binary {
		response.BaseContent, response.MainContent = string(base), string(main)
		response.TaskContent, response.ResultContent = string(task), string(result)
	}
	return response, nil
}

func readCodeConflictCommitFile(workDir, commit, file string) ([]byte, bool, error) {
	commit = strings.TrimSpace(commit)
	if commit == "" {
		return nil, false, nil
	}
	if _, err := runCodeGit(workDir, "cat-file", "-e", commit+":"+file); err != nil {
		return nil, false, nil
	}
	content, err := runCodeGitBytes(workDir, nil, "show", commit+":"+file)
	if err != nil {
		return nil, false, err
	}
	if len(content) > maxAISessionFileSize {
		return nil, false, errors.New("冲突文件超过 2 MB，无法在网页中处理")
	}
	return content, true, nil
}

func readCodeConflictResultFile(workDir, file string) ([]byte, bool, error) {
	target, err := resolveCodeConflictResultPath(workDir, file, false)
	if err != nil {
		return nil, false, err
	}
	content, err := os.ReadFile(target)
	if errors.Is(err, os.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if len(content) > maxAISessionFileSize {
		return nil, false, errors.New("冲突文件超过 2 MB，无法在网页中处理")
	}
	return content, true, nil
}

func resolveCodeConflictResultPath(workDir, file string, allowMissing bool) (string, error) {
	root := filepath.Clean(workDir)
	target := filepath.Join(root, filepath.FromSlash(file))
	if !isPathInside(target, root) {
		return "", errors.New("冲突文件路径越界")
	}
	current := root
	parts := strings.Split(filepath.FromSlash(file), string(filepath.Separator))
	for index, part := range parts {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if errors.Is(err, os.ErrNotExist) && allowMissing {
			if index < len(parts)-1 {
				if err := os.Mkdir(current, 0750); err != nil {
					return "", err
				}
			}
			continue
		}
		if errors.Is(err, os.ErrNotExist) && index == len(parts)-1 {
			return target, nil
		}
		if err != nil {
			return "", err
		}
		if info.Mode()&os.ModeSymlink != 0 || (index < len(parts)-1 && !info.IsDir()) || (index == len(parts)-1 && info.IsDir()) {
			return "", errors.New("冲突文件路径包含目录或符号链接")
		}
	}
	return target, nil
}

func isCodeConflictBinary(content []byte, exists bool) bool {
	return exists && (!utf8.Valid(content) || bytes.IndexByte(content, 0) >= 0)
}

func codeConflictResultVersion(content []byte, exists bool) string {
	prefix := []byte("missing\x00")
	if exists {
		prefix = []byte("exists\x00")
	}
	return fileContentVersion(append(prefix, content...))
}

func saveCodeDeliveryConflictFile(context *codeDeliveryConflictContext, file, resolution, content, version string) (codeDeliveryConflictFile, error) {
	current, exists, err := readCodeConflictResultFile(context.WorkDir, file)
	if err != nil {
		return codeDeliveryConflictFile{}, err
	}
	if strings.TrimSpace(version) == "" || codeConflictResultVersion(current, exists) != version {
		return codeDeliveryConflictFile{}, errors.New("冲突文件已被其他操作修改，请刷新后重试")
	}
	var next []byte
	deleted := false
	switch resolution {
	case "content":
		next = []byte(content)
		if len(next) > maxAISessionFileSize || !utf8.Valid(next) || bytes.IndexByte(next, 0) >= 0 {
			return codeDeliveryConflictFile{}, errors.New("文件内容必须是 2 MB 以内的 UTF-8 文本")
		}
	case "main":
		next, exists, err = readCodeConflictCommitFile(context.WorkDir, context.SourceCommit, file)
		deleted = !exists
	case "task":
		next, exists, err = readCodeConflictCommitFile(context.WorkDir, context.TaskCommit, file)
		deleted = !exists
	case "delete":
		deleted = true
	default:
		return codeDeliveryConflictFile{}, errors.New("冲突解决方式无效")
	}
	if err != nil {
		return codeDeliveryConflictFile{}, err
	}
	if !deleted && hasCodeConflictMarkerLines(next) {
		return codeDeliveryConflictFile{}, errors.New("最终结果仍包含 Git 冲突标记，请处理 <<<<<<<、======= 和 >>>>>>> 后再保存")
	}
	target, err := resolveCodeConflictResultPath(context.WorkDir, file, !deleted)
	if err != nil {
		return codeDeliveryConflictFile{}, err
	}
	if deleted {
		if err := os.Remove(target); err != nil && !errors.Is(err, os.ErrNotExist) {
			return codeDeliveryConflictFile{}, err
		}
	} else if err := writeCodeConflictResultFile(target, next); err != nil {
		return codeDeliveryConflictFile{}, err
	}
	if _, err := runCodeGit(context.WorkDir, "add", "-A", "--", file); err != nil {
		return codeDeliveryConflictFile{}, err
	}
	return readCodeDeliveryConflictFile(context, file)
}

func hasCodeConflictMarkerLines(content []byte) bool {
	for _, line := range bytes.Split(content, []byte{'\n'}) {
		line = bytes.TrimSuffix(line, []byte{'\r'})
		if bytes.HasPrefix(line, []byte("<<<<<<<")) || bytes.Equal(line, []byte("=======")) ||
			bytes.HasPrefix(line, []byte(">>>>>>>")) {
			return true
		}
	}
	return false
}

func writeCodeConflictResultFile(target string, content []byte) error {
	mode := os.FileMode(0644)
	if info, err := os.Stat(target); err == nil {
		mode = info.Mode().Perm()
	}
	temporary, err := os.CreateTemp(filepath.Dir(target), ".gopanel-conflict-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(mode); err != nil {
		_ = temporary.Close()
		return err
	}
	if _, err := temporary.Write(content); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporaryPath, target)
}
