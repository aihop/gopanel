package service

import (
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
	FlowID       uint   `json:"flowId"`
	SourceCommit string `json:"sourceCommit"`
	Version      string `json:"version"`
	SessionID    uint   `json:"sessionId"`
	TaskID       uint   `json:"taskId"`
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
	commit, err := normalizePipelineExpectedCommit(input.SourceCommit)
	if err != nil || commit == "" {
		return nil, buserr.New(constant.ErrFlowCommitRequired)
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
			sourceRepository = fmt.Sprintf("code-project:%d", pipeline.CodeProjectID)
		}
		item := &model.FlowRun{
			FlowID: flow.ID, ProjectID: flow.ProjectID, PipelineID: flow.PipelineID,
			Version: version, SourceRepository: sourceRepository, SourceBranch: pipeline.Branch,
			SourceCommit: commit, SessionID: input.SessionID, TaskID: input.TaskID,
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

func (s *FlowRunApplicationService) fillRunSummaries(items []model.FlowRun) error {
	for index := range items {
		var flow model.Flow
		if err := s.db.Select("name").First(&flow, items[index].FlowID).Error; err != nil {
			return err
		}
		items[index].FlowName = flow.Name
		items[index].ProjectName = loadFlowRunName(s.db, "ai_projects", items[index].ProjectID)
		items[index].PipelineName = loadFlowRunName(s.db, "pipelines", items[index].PipelineID)
		if items[index].ReleaseID > 0 {
			var release model.Release
			if err := s.db.Select("artifact_digest").First(&release, items[index].ReleaseID).Error; err == nil {
				items[index].ArtifactDigest = release.ArtifactDigest
			}
		}
	}
	return nil
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
