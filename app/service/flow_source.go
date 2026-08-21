package service

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/buserr"
	"github.com/aihop/gopanel/constant"
	"gorm.io/gorm"
)

const (
	flowSourceManifestLegacySchemaVersion = 1
	flowSourceManifestSchemaVersion       = 2
)

type FlowCodeDeliverySource struct {
	JobID        uint                         `json:"jobId"`
	SessionID    uint                         `json:"sessionId"`
	TaskID       uint                         `json:"taskId"`
	TaskTitle    string                       `json:"taskTitle"`
	CompletedAt  *time.Time                   `json:"completedAt,omitempty"`
	SourceDigest string                       `json:"sourceDigest"`
	Repositories []model.FlowSourceRepository `json:"repositories"`
}

type flowSourceManifest struct {
	SchemaVersion int                            `json:"schemaVersion"`
	SourceType    string                         `json:"sourceType,omitempty"`
	DeliveryJobID uint                           `json:"deliveryJobId"`
	SessionID     uint                           `json:"sessionId"`
	TaskID        uint                           `json:"taskId"`
	TaskTitle     string                         `json:"taskTitle"`
	Repositories  []flowSourceManifestRepository `json:"repositories"`
}

type flowSourceManifestRepository struct {
	Name          string `json:"name"`
	SourceDir     string `json:"-"`
	WorkspacePath string `json:"workspacePath"`
	TargetBranch  string `json:"targetBranch"`
	Commit        string `json:"commit"`
}

type flowStoredDeliveryRepository struct {
	RepositoryName string `json:"repositoryName"`
	RepositoryPath string `json:"repositoryPath"`
	Status         string `json:"status"`
	TargetBranch   string `json:"targetBranch"`
	Commit         string `json:"commit"`
}

func (s *FlowRunApplicationService) CodeDeliverySources(flowID, userID uint, includeAll bool, limit int) ([]FlowCodeDeliverySource, error) {
	flow, err := s.repo.Get(flowID)
	if err != nil {
		return nil, buserr.New(constant.ErrFlowNotFound)
	}
	if !includeAll && flow.CreatedBy != userID {
		return nil, buserr.New(constant.ErrFlowForbidden)
	}
	pipeline, err := repoPipeline(s.db, flow.PipelineID)
	if err != nil {
		return nil, buserr.New(constant.ErrFlowPipelineNotFound)
	}
	if pipelineSourceType(pipeline) != "code" || pipeline.CodeProjectID != flow.ProjectID {
		return []FlowCodeDeliverySource{}, nil
	}
	if limit < 1 || limit > 100 {
		limit = 30
	}
	var jobs []model.AICodeDeliveryJob
	if err := s.db.Where("project_id = ? AND status = ?", flow.ProjectID, "completed").
		Order("completed_at desc, id desc").Limit(limit).Find(&jobs).Error; err != nil {
		return nil, err
	}
	var project model.AIProject
	if err := s.db.First(&project, flow.ProjectID).Error; err != nil {
		return nil, err
	}
	items := make([]FlowCodeDeliverySource, 0, len(jobs))
	for index := range jobs {
		manifest, digest, resolveErr := s.resolveFlowCodeDeliveryManifest(&project, &jobs[index])
		if resolveErr != nil {
			continue
		}
		items = append(items, flowCodeDeliverySource(&jobs[index], manifest, digest))
	}
	return items, nil
}

func repoPipeline(db *gorm.DB, pipelineID uint) (*model.Pipeline, error) {
	var pipeline model.Pipeline
	err := db.First(&pipeline, pipelineID).Error
	return &pipeline, err
}

func (s *FlowRunApplicationService) resolveFlowCodeDelivery(jobID, projectID uint) (flowSourceManifest, string, *model.AICodeDeliveryJob, error) {
	if jobID == 0 {
		return flowSourceManifest{}, "", nil, buserr.New(constant.ErrFlowCodeDeliveryRequired)
	}
	var job model.AICodeDeliveryJob
	if err := s.db.Where("id = ? AND project_id = ?", jobID, projectID).First(&job).Error; err != nil || job.Status != "completed" {
		return flowSourceManifest{}, "", nil, buserr.New(constant.ErrFlowCodeDeliveryInvalid)
	}
	var project model.AIProject
	if err := s.db.First(&project, projectID).Error; err != nil {
		return flowSourceManifest{}, "", nil, buserr.New(constant.ErrFlowProjectNotFound)
	}
	manifest, digest, err := s.resolveFlowCodeDeliveryManifest(&project, &job)
	if err != nil {
		return flowSourceManifest{}, "", nil, buserr.New(constant.ErrFlowCodeDeliveryInvalid)
	}
	return manifest, digest, &job, nil
}

func (s *FlowRunApplicationService) resolveFlowCodeDeliveryManifest(project *model.AIProject, job *model.AICodeDeliveryJob) (flowSourceManifest, string, error) {
	repositories, err := s.loadFlowDeliveryRepositories(job)
	if err != nil || len(repositories) == 0 {
		return flowSourceManifest{}, "", buserr.New(constant.ErrFlowDeliveryNoCommit)
	}
	var task model.AITask
	if job.TaskID > 0 {
		_ = s.db.Select("title").First(&task, job.TaskID).Error
	}
	manifest := flowSourceManifest{
		SchemaVersion: flowSourceManifestSchemaVersion, SourceType: "code_delivery", DeliveryJobID: job.ID,
		SessionID: job.SessionID, TaskID: job.TaskID, TaskTitle: strings.TrimSpace(task.Title),
		Repositories: make([]flowSourceManifestRepository, 0, len(repositories)),
	}
	usedPaths := make(map[string]struct{}, len(repositories))
	for _, repository := range repositories {
		commit, err := normalizePipelineExpectedCommit(repository.Commit)
		if err != nil || commit == "" || strings.TrimSpace(repository.RepositoryPath) == "" {
			return flowSourceManifest{}, "", buserr.New(constant.ErrFlowDeliveryInvalidCommit)
		}
		sourceDir, workspacePath, err := flowRepositoryWorkspacePath(project.SourceDirs, repository.RepositoryPath)
		if err != nil {
			return flowSourceManifest{}, "", err
		}
		if _, exists := usedPaths[workspacePath]; exists {
			return flowSourceManifest{}, "", buserr.New(constant.ErrFlowDeliveryDuplicateMapping)
		}
		usedPaths[workspacePath] = struct{}{}
		if !flowGitCommitExists(sourceDir, commit) {
			return flowSourceManifest{}, "", fmt.Errorf("Code 交付提交不可用: %s", repository.RepositoryName)
		}
		manifest.Repositories = append(manifest.Repositories, flowSourceManifestRepository{
			Name: strings.TrimSpace(repository.RepositoryName), SourceDir: sourceDir,
			WorkspacePath: workspacePath, TargetBranch: strings.TrimSpace(repository.TargetBranch), Commit: commit,
		})
	}
	sort.Slice(manifest.Repositories, func(left, right int) bool {
		return manifest.Repositories[left].WorkspacePath < manifest.Repositories[right].WorkspacePath
	})
	digest, err := flowSourceManifestDigest(manifest)
	return manifest, digest, err
}

func (s *FlowRunApplicationService) loadFlowDeliveryRepositories(job *model.AICodeDeliveryJob) ([]flowStoredDeliveryRepository, error) {
	var repositories []flowStoredDeliveryRepository
	if strings.TrimSpace(job.RepositoryResults) != "" && json.Unmarshal([]byte(job.RepositoryResults), &repositories) == nil && len(repositories) > 0 {
		var stored []model.AIDevSessionRepository
		if err := s.db.Where("session_id = ?", job.SessionID).Find(&stored).Error; err != nil {
			return nil, err
		}
		storedByPath := make(map[string]model.AIDevSessionRepository, len(stored))
		for _, repository := range stored {
			storedByPath[filepath.Clean(repository.SourceDir)] = repository
		}
		for index := range repositories {
			if strings.TrimSpace(repositories[index].Commit) != "" {
				continue
			}
			storedRepository, exists := storedByPath[filepath.Clean(repositories[index].RepositoryPath)]
			if !exists {
				return nil, buserr.New(constant.ErrFlowDeliveryBaselineUnavailable)
			}
			repositories[index].Commit = firstFlowCommit(
				storedRepository.MergeCommit, storedRepository.WorktreeCommit, storedRepository.BaseCommit,
			)
		}
		return repositories, nil
	}
	var multi []model.AIDevSessionRepository
	if err := s.db.Where("session_id = ? AND status = ?", job.SessionID, "completed").Order("link_name asc").Find(&multi).Error; err != nil {
		return nil, err
	}
	for _, repository := range multi {
		if strings.TrimSpace(repository.MergeCommit) == "" {
			continue
		}
		repositories = append(repositories, flowStoredDeliveryRepository{
			RepositoryName: repository.LinkName, RepositoryPath: repository.SourceDir,
			Status: repository.Status, TargetBranch: repository.TargetBranch, Commit: repository.MergeCommit,
		})
	}
	if len(repositories) > 0 {
		return repositories, nil
	}
	var single model.AICodeDelivery
	if err := s.db.Where("session_id = ? AND status = ?", job.SessionID, "completed").First(&single).Error; err != nil {
		return nil, err
	}
	return []flowStoredDeliveryRepository{{
		RepositoryName: filepath.Base(single.SourceWorkDir), RepositoryPath: single.SourceWorkDir,
		Status: single.Status, TargetBranch: single.TargetBranch, Commit: single.MergeCommit,
	}}, nil
}

func firstFlowCommit(values ...string) string {
	for _, value := range values {
		if commit := strings.TrimSpace(value); commit != "" {
			return commit
		}
	}
	return ""
}

func flowRepositoryWorkspacePath(sourceDirs []string, repositoryPath string) (string, string, error) {
	resolvedRepository, err := filepath.EvalSymlinks(filepath.Clean(repositoryPath))
	if err != nil {
		return "", "", err
	}
	usedNames := make(map[string]struct{}, len(sourceDirs))
	for _, rawSource := range sourceDirs {
		resolvedSource, resolveErr := filepath.EvalSymlinks(filepath.Clean(rawSource))
		if resolveErr != nil {
			return "", "", resolveErr
		}
		rootName := ""
		if len(sourceDirs) > 1 {
			rootName = uniqueCodeSnapshotName(filepath.Base(resolvedSource), usedNames)
		}
		relative, relativeErr := filepath.Rel(resolvedSource, resolvedRepository)
		if relativeErr != nil || filepath.IsAbs(relative) || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			continue
		}
		workspacePath := filepath.ToSlash(filepath.Clean(filepath.Join(rootName, relative)))
		if workspacePath == "" {
			workspacePath = "."
		}
		return resolvedRepository, workspacePath, nil
	}
	return "", "", buserr.New(constant.ErrFlowDeliveryOutsideSource)
}

func flowGitCommitExists(repository, commit string) bool {
	command := exec.Command("git", "cat-file", "-e", commit+"^{commit}")
	command.Dir = repository
	command.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	return command.Run() == nil
}

func retainFlowSourceCommits(flowRunID uint, manifest flowSourceManifest) error {
	retained := make([]flowSourceManifestRepository, 0, len(manifest.Repositories))
	for index, repository := range manifest.Repositories {
		ref := fmt.Sprintf("refs/gopanel/flows/%d/repositories/%d", flowRunID, index+1)
		command := exec.Command("git", "update-ref", ref, repository.Commit)
		command.Dir = repository.SourceDir
		command.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
		if output, err := command.CombinedOutput(); err != nil {
			for retainedIndex := range retained {
				cleanup := exec.Command("git", "update-ref", "-d", fmt.Sprintf(
					"refs/gopanel/flows/%d/repositories/%d", flowRunID, retainedIndex+1,
				))
				cleanup.Dir = retained[retainedIndex].SourceDir
				_ = cleanup.Run()
			}
			return fmt.Errorf("保留 Flow 正式版本提交失败: %s", strings.TrimSpace(string(output)))
		}
		retained = append(retained, repository)
	}
	return nil
}

func flowSourceManifestDigest(manifest flowSourceManifest) (string, error) {
	identity := struct {
		SchemaVersion int                          `json:"schemaVersion"`
		SourceType    string                       `json:"sourceType,omitempty"`
		DeliveryJobID uint                         `json:"deliveryJobId"`
		SessionID     uint                         `json:"sessionId"`
		TaskID        uint                         `json:"taskId"`
		Repositories  []model.FlowSourceRepository `json:"repositories"`
	}{
		SchemaVersion: manifest.SchemaVersion, SourceType: manifest.SourceType, DeliveryJobID: manifest.DeliveryJobID,
		SessionID: manifest.SessionID, TaskID: manifest.TaskID,
		Repositories: flowPublicSourceRepositories(manifest),
	}
	content, err := json.Marshal(identity)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(content)
	return "sha256:" + hex.EncodeToString(digest[:]), nil
}

func flowPublicSourceRepositories(manifest flowSourceManifest) []model.FlowSourceRepository {
	result := make([]model.FlowSourceRepository, 0, len(manifest.Repositories))
	for _, repository := range manifest.Repositories {
		result = append(result, model.FlowSourceRepository{
			Name: repository.Name, WorkspacePath: repository.WorkspacePath,
			TargetBranch: repository.TargetBranch, Commit: repository.Commit,
		})
	}
	return result
}

func flowCodeDeliverySource(job *model.AICodeDeliveryJob, manifest flowSourceManifest, digest string) FlowCodeDeliverySource {
	return FlowCodeDeliverySource{
		JobID: job.ID, SessionID: job.SessionID, TaskID: job.TaskID, TaskTitle: manifest.TaskTitle,
		CompletedAt: job.CompletedAt, SourceDigest: digest, Repositories: flowPublicSourceRepositories(manifest),
	}
}
