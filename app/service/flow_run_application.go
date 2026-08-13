package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/buserr"
	"github.com/aihop/gopanel/constant"
	"gorm.io/gorm"
)

const (
	flowRunQueued            = "queued"
	flowRunRunning           = "running"
	flowRunFailed            = "failed"
	flowRunWaitingDeployment = "waiting_deployment"
)

var flowVersionPattern = regexp.MustCompile(`^[0-9A-Za-z][0-9A-Za-z._+-]{0,49}$`)

type FlowRunCreateInput struct {
	FlowID             uint   `json:"flowId"`
	CodeDeliveryJobID  uint   `json:"codeDeliveryJobId"`
	UseProjectBaseline bool   `json:"useProjectBaseline"`
	SourceCommit       string `json:"sourceCommit"`
	Version            string `json:"version"`
	SessionID          uint   `json:"sessionId"`
	TaskID             uint   `json:"taskId"`
}

type FlowRunApplicationService struct {
	db            *gorm.DB
	repo          *repo.FlowRepo
	recordRepo    *repo.PipelineRecordRepo
	runPipeline   func(uint, string, string, PipelineRunSource) (uint, error)
	publishRecord func(uint) (*model.Release, error)
	pollInterval  time.Duration
	autoStart     bool
}

func NewFlowRunApplication(db *gorm.DB) *FlowRunApplicationService {
	pipeline := NewPipelineService(db)
	publisher := NewPipelineApplication(db)
	return &FlowRunApplicationService{
		db: db, repo: repo.NewFlow(db), recordRepo: repo.NewPipelineRecord(db),
		runPipeline: pipeline.RunPipelineForSource, publishRecord: publisher.PublishRecord,
		pollInterval: time.Second, autoStart: true,
	}
}

func (s *FlowRunApplicationService) Create(input FlowRunCreateInput, userID uint, includeAll bool) (*model.FlowRun, error) {
	flow, err := s.repo.Get(input.FlowID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, buserr.New(constant.ErrFlowNotFound)
	}
	if err != nil {
		return nil, err
	}
	if !includeAll && flow.CreatedBy != userID {
		return nil, buserr.New(constant.ErrFlowForbidden)
	}
	if !flow.Enabled {
		return nil, buserr.New(constant.ErrFlowDisabled)
	}
	pipeline, err := repo.NewPipeline(s.db).Get(flow.PipelineID)
	if err != nil {
		return nil, buserr.New(constant.ErrFlowPipelineNotFound)
	}
	if strings.TrimSpace(pipeline.RepoUrl) == "" && pipelineSourceType(pipeline) != "code" {
		return nil, buserr.New(constant.ErrPipelineExpectedCommitRepo)
	}
	if pipelineSourceType(pipeline) == "code" && pipeline.CodeProjectID != flow.ProjectID {
		return nil, buserr.New(constant.ErrFlowPipelineProjectMismatch)
	}
	sourceType, sourceDigest, sourceManifest := "git", "", ""
	commit, sessionID, taskID, codeDeliveryJobID := "", input.SessionID, input.TaskID, uint(0)
	var lockedCodeManifest flowSourceManifest
	if pipelineSourceType(pipeline) == "code" {
		var manifest flowSourceManifest
		var digest string
		if input.UseProjectBaseline {
			var available bool
			manifest, digest, _, available, err = s.resolveFlowProjectBaseline(flow.ProjectID)
			if err != nil {
				return nil, err
			}
			if !available {
				return nil, buserr.New(constant.ErrFlowCodeBaselineUnavailable)
			}
			sourceType = "code_baseline"
		} else {
			var job *model.AICodeDeliveryJob
			manifest, digest, job, err = s.resolveFlowCodeDelivery(input.CodeDeliveryJobID, flow.ProjectID)
			if err != nil {
				return nil, err
			}
			sourceType, sessionID, taskID, codeDeliveryJobID = "code_delivery", job.SessionID, job.TaskID, job.ID
		}
		encodedManifest, marshalErr := json.Marshal(manifest)
		if marshalErr != nil {
			return nil, marshalErr
		}
		sourceDigest, sourceManifest = digest, string(encodedManifest)
		commit = flowManifestPrimaryCommit(manifest)
		lockedCodeManifest = manifest
	} else {
		commit, err = normalizePipelineExpectedCommit(input.SourceCommit)
		if err != nil || commit == "" {
			return nil, buserr.New(constant.ErrFlowCommitRequired)
		}
	}
	version := strings.TrimSpace(input.Version)
	if version != "" && !flowVersionPattern.MatchString(version) {
		return nil, buserr.New(constant.ErrFlowVersionInvalid)
	}
	if version == "" {
		version = nextFlowPatchVersion(pipeline.Version)
	}
	for attempt := 0; attempt < 100; attempt++ {
		if attempt > 0 {
			version = nextFlowPatchVersion(version)
		}
		exists, checkErr := s.repo.VersionExists(flow.ID, version)
		if checkErr != nil {
			return nil, checkErr
		}
		if exists {
			if strings.TrimSpace(input.Version) != "" {
				return nil, buserr.New(constant.ErrFlowVersionExists)
			}
			continue
		}
		now := time.Now()
		sourceRepository := pipeline.RepoUrl
		if pipelineSourceType(pipeline) == "code" {
			sourceRepository = sourceType
			if sourceType == "code_delivery" {
				sourceRepository = fmt.Sprintf("code-delivery:%d", codeDeliveryJobID)
			}
		}
		item := &model.FlowRun{
			FlowID: flow.ID, ProjectID: flow.ProjectID, PipelineID: flow.PipelineID,
			Version: version, SourceRepository: sourceRepository, SourceType: sourceType, SourceBranch: pipeline.Branch,
			SourceCommit: commit, SourceDigest: sourceDigest, SourceManifest: sourceManifest,
			SessionID: sessionID, TaskID: taskID, CodeDeliveryJobID: codeDeliveryJobID,
			CurrentStage: "created", Status: flowRunQueued, RequestedBy: userID,
		}
		stage := &model.FlowStageRun{
			Stage: "created", Attempt: 1, Status: "success",
			Summary: "flow run created", StartedAt: &now, CompletedAt: &now,
		}
		if createErr := s.repo.CreateRun(item, stage); createErr != nil {
			if isFlowRunVersionDuplicate(createErr) && strings.TrimSpace(input.Version) == "" {
				continue
			}
			if isFlowRunVersionDuplicate(createErr) {
				return nil, buserr.New(constant.ErrFlowVersionExists)
			}
			return nil, createErr
		}
		if isFlowCodeSourceType(sourceType) {
			if retainErr := retainFlowSourceCommits(item.ID, lockedCodeManifest); retainErr != nil {
				_ = s.repo.DeleteRun(item.ID)
				return nil, buserr.New(constant.ErrFlowCodeSourceInvalid)
			}
		}
		if s.autoStart {
			go s.Advance(item.ID)
		}
		return item, nil
	}
	return nil, buserr.New(constant.ErrFlowVersionExists)
}

func nextFlowPatchVersion(current string) string {
	parts := strings.Split(strings.TrimSpace(current), ".")
	if len(parts) != 3 {
		return "1.0.0"
	}
	major, majorErr := strconv.Atoi(parts[0])
	minor, minorErr := strconv.Atoi(parts[1])
	patch, patchErr := strconv.Atoi(parts[2])
	if majorErr != nil || minorErr != nil || patchErr != nil || major < 0 || minor < 0 || patch < 0 {
		return "1.0.0"
	}
	return fmt.Sprintf("%d.%d.%d", major, minor, patch+1)
}

func isFlowRunVersionDuplicate(err error) bool {
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "unique") || strings.Contains(message, "duplicate")
}

func (s *FlowRunApplicationService) Page(flowID, userID uint, includeAll bool, page, limit int) (int64, []model.FlowRun, error) {
	total, items, err := s.repo.PageRuns(flowID, userID, includeAll, page, limit)
	if err != nil {
		return 0, nil, err
	}
	if err := s.fillRunSummaries(items); err != nil {
		return 0, nil, err
	}
	return total, items, nil
}

func (s *FlowRunApplicationService) Get(id, userID uint, includeAll bool) (*model.FlowRun, error) {
	item, err := s.repo.GetRun(id, userID, includeAll)
	if err != nil {
		return nil, err
	}
	items := []model.FlowRun{*item}
	if err := s.fillRunSummaries(items); err != nil {
		return nil, err
	}
	items[0].Stages = item.Stages
	return &items[0], nil
}

func (s *FlowRunApplicationService) Resume(id, userID uint, includeAll bool) (*model.FlowRun, error) {
	identity, err := s.repo.GetRunInternal(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, buserr.New(constant.ErrFlowNotFound)
	}
	if err != nil {
		return nil, err
	}
	flow, err := s.repo.Get(identity.FlowID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, buserr.New(constant.ErrFlowNotFound)
	}
	if err != nil {
		return nil, err
	}
	if !includeAll && flow.CreatedBy != userID {
		return nil, buserr.New(constant.ErrFlowForbidden)
	}
	if !flow.Enabled {
		return nil, buserr.New(constant.ErrFlowDisabled)
	}
	run, err := s.repo.GetRun(id, userID, includeAll)
	if err != nil {
		return nil, err
	}
	if run.Status != flowRunFailed {
		return nil, buserr.New(constant.ErrFlowRunNotFailed)
	}
	failedStage := flowFailedStage(run.Stages)
	if failedStage == "" && run.FailureCode == "pipeline_record_unavailable" {
		failedStage = "building"
	}
	if failedStage != "building" && failedStage != "publishing" {
		return nil, buserr.New(constant.ErrFlowRunResumeUnsupported)
	}
	if failedStage == "publishing" {
		record, recordErr := s.recordRepo.Get(run.PipelineRecordID)
		if recordErr != nil || record.Status != "success" {
			return nil, buserr.New(constant.ErrFlowRunResumeUnsupported)
		}
	}
	attempt, err := s.repo.NextStageAttempt(run.ID, failedStage)
	if err != nil {
		return nil, err
	}
	values := map[string]any{
		"status": flowRunQueued, "current_stage": failedStage,
		"failure_code": "", "error_summary": "", "completed_at": nil,
		"lease_owner": "", "lease_expires_at": nil,
	}
	if failedStage == "building" {
		values["pipeline_record_id"] = 0
	}
	stage := &model.FlowStageRun{
		FlowRunID: run.ID, Stage: failedStage, Attempt: attempt, Status: "pending",
		IdempotencyKey: fmt.Sprintf("flow:%d:%s:%d", run.ID, failedStage, attempt),
		Summary:        fmt.Sprintf("delivery resumed by user #%d", userID),
	}
	if failedStage == "publishing" {
		stage.ResourceType = "pipeline_record"
		stage.ResourceID = run.PipelineRecordID
	}
	resumed, err := s.repo.ResumeFailedRun(run.ID, values, stage)
	if err != nil {
		return nil, err
	}
	if !resumed {
		return nil, buserr.New(constant.ErrFlowRunNotFailed)
	}
	if s.autoStart {
		go s.Advance(run.ID)
	}
	return s.Get(run.ID, userID, includeAll)
}

func flowFailedStage(stages []model.FlowStageRun) string {
	for index := len(stages) - 1; index >= 0; index-- {
		if stages[index].Status == "failed" {
			return stages[index].Stage
		}
	}
	return ""
}

func (s *FlowRunApplicationService) fillRunSummaries(items []model.FlowRun) error {
	for index := range items {
		var flow model.Flow
		if err := s.db.Select("name").First(&flow, items[index].FlowID).Error; err != nil {
			return err
		}
		items[index].FlowName = flow.Name
		items[index].ProjectName = loadFlowRunName(s.db, "ai_projects", items[index].ProjectID)
		items[index].PipelineName = loadFlowRunName(s.db, "pipelines", items[index].PipelineID)
		if isFlowCodeSourceType(items[index].SourceType) && strings.TrimSpace(items[index].SourceManifest) != "" {
			var manifest flowSourceManifest
			if json.Unmarshal([]byte(items[index].SourceManifest), &manifest) == nil {
				items[index].SourceTaskTitle = manifest.TaskTitle
				items[index].SourceRepositories = flowPublicSourceRepositories(manifest)
			}
		}
		if items[index].ReleaseID > 0 {
			var release model.Release
			if err := s.db.Select("artifact_digest").First(&release, items[index].ReleaseID).Error; err == nil {
				items[index].ArtifactDigest = release.ArtifactDigest
			}
		}
	}
	return nil
}

func isFlowCodeSourceType(sourceType string) bool {
	return sourceType == "code_delivery" || sourceType == "code_baseline"
}

func flowManifestPrimaryCommit(manifest flowSourceManifest) string {
	for _, repository := range manifest.Repositories {
		if repository.WorkspacePath == "." {
			return repository.Commit
		}
	}
	if len(manifest.Repositories) > 0 {
		return manifest.Repositories[0].Commit
	}
	return ""
}

func loadFlowRunName(db *gorm.DB, table string, id uint) string {
	var row struct{ Name string }
	_ = db.Table(table).Select("name").Where("id = ?", id).Scan(&row).Error
	return row.Name
}

func flowWorkerOwner(runID uint) string {
	host, _ := os.Hostname()
	return fmt.Sprintf("%s:%d:%d:%d", host, os.Getpid(), runID, time.Now().UnixNano())
}
