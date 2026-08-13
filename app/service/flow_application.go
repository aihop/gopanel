package service

import (
	"strings"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/buserr"
	"github.com/aihop/gopanel/constant"
	"gorm.io/gorm"
)

type FlowEnvironmentInput struct {
	Name             string `json:"name"`
	WebsiteID        uint   `json:"websiteId"`
	AutoDeploy       bool   `json:"autoDeploy"`
	ApprovalRequired bool   `json:"approvalRequired"`
}

type FlowCreateInput struct {
	Name                       string                 `json:"name"`
	ProjectID                  uint                   `json:"projectId"`
	PipelineID                 uint                   `json:"pipelineId"`
	AutoStartAfterCodeDelivery bool                   `json:"autoStartAfterCodeDelivery"`
	Environments               []FlowEnvironmentInput `json:"environments"`
}

type FlowApplicationService struct {
	db   *gorm.DB
	repo *repo.FlowRepo
}

func NewFlowApplication(db *gorm.DB) *FlowApplicationService {
	return &FlowApplicationService{db: db, repo: repo.NewFlow(db)}
}

func (s *FlowApplicationService) Page(userID uint, includeAll bool, page, limit int) (int64, []model.Flow, error) {
	total, items, err := s.repo.Page(userID, includeAll, page, limit)
	if err != nil || len(items) == 0 {
		return total, items, err
	}
	projectIDs := make([]uint, 0, len(items))
	pipelineIDs := make([]uint, 0, len(items))
	websiteIDs := make([]uint, 0)
	for _, item := range items {
		projectIDs = append(projectIDs, item.ProjectID)
		pipelineIDs = append(pipelineIDs, item.PipelineID)
		for _, environment := range item.Environments {
			websiteIDs = append(websiteIDs, environment.WebsiteID)
		}
	}
	projectNames, err := loadFlowNames(s.db, "ai_projects", projectIDs)
	if err != nil {
		return 0, nil, err
	}
	pipelineNames, err := loadFlowNames(s.db, "pipelines", pipelineIDs)
	if err != nil {
		return 0, nil, err
	}
	websiteNames, err := loadFlowWebsiteNames(s.db, websiteIDs)
	if err != nil {
		return 0, nil, err
	}
	for index := range items {
		items[index].ProjectName = projectNames[items[index].ProjectID]
		items[index].PipelineName = pipelineNames[items[index].PipelineID]
		for envIndex := range items[index].Environments {
			environment := &items[index].Environments[envIndex]
			environment.WebsiteName = websiteNames[environment.WebsiteID]
		}
	}
	return total, items, nil
}

func loadFlowNames(db *gorm.DB, table string, ids []uint) (map[uint]string, error) {
	result := make(map[uint]string)
	if len(ids) == 0 {
		return result, nil
	}
	var rows []struct {
		ID   uint
		Name string
	}
	if err := db.Table(table).Select("id, name").Where("id IN ?", ids).Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		result[row.ID] = row.Name
	}
	return result, nil
}

func loadFlowWebsiteNames(db *gorm.DB, ids []uint) (map[uint]string, error) {
	result := make(map[uint]string)
	if len(ids) == 0 {
		return result, nil
	}
	var rows []struct {
		ID            uint
		Alias         string
		PrimaryDomain string
	}
	if err := db.Table("website").Select("id, alias, primary_domain").Where("id IN ?", ids).Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		name := strings.TrimSpace(row.Alias)
		if name == "" {
			name = strings.TrimSpace(row.PrimaryDomain)
		}
		result[row.ID] = name
	}
	return result, nil
}

func (s *FlowApplicationService) Create(input FlowCreateInput, userID uint, includeAll bool) (*model.Flow, error) {
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		return nil, buserr.New(constant.ErrFlowNameRequired)
	}
	if input.ProjectID == 0 {
		return nil, buserr.New(constant.ErrFlowProjectRequired)
	}
	if input.PipelineID == 0 {
		return nil, buserr.New(constant.ErrFlowPipelineRequired)
	}
	if len(input.Environments) == 0 {
		return nil, buserr.New(constant.ErrFlowEnvironmentRequired)
	}
	var project model.AIProject
	if err := s.db.First(&project, input.ProjectID).Error; err != nil {
		return nil, buserr.New(constant.ErrFlowProjectNotFound)
	}
	if !includeAll && project.CreatorID != userID {
		return nil, buserr.New(constant.ErrFlowProjectForbidden)
	}
	if err := s.db.First(&model.Pipeline{}, input.PipelineID).Error; err != nil {
		return nil, buserr.New(constant.ErrFlowPipelineNotFound)
	}
	var count int64
	if err := s.db.Model(&model.Flow{}).Where("project_id = ?", input.ProjectID).Count(&count).Error; err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, buserr.New(constant.ErrFlowProjectExists)
	}
	environments := make([]model.FlowEnvironment, 0, len(input.Environments))
	seen := make(map[string]bool)
	for index, raw := range input.Environments {
		name := strings.ToLower(strings.TrimSpace(raw.Name))
		if (name != "preview" && name != "production") || seen[name] || raw.WebsiteID == 0 {
			return nil, buserr.New(constant.ErrFlowEnvironmentInvalid)
		}
		seen[name] = true
		if err := s.db.First(&model.Website{}, raw.WebsiteID).Error; err != nil {
			return nil, buserr.New(constant.ErrFlowWebsiteNotFound)
		}
		environments = append(environments, defaultFlowEnvironment(name, raw, index))
	}
	item := &model.Flow{
		ProjectID: input.ProjectID, Name: input.Name, PipelineID: input.PipelineID, Enabled: true,
		AutoStartAfterCodeDelivery: input.AutoStartAfterCodeDelivery, CreatedBy: userID, Environments: environments,
	}
	if err := s.repo.CreateWithEnvironments(item); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return nil, buserr.New(constant.ErrFlowProjectExists)
		}
		return nil, err
	}
	return item, nil
}

func defaultFlowEnvironment(name string, input FlowEnvironmentInput, sortOrder int) model.FlowEnvironment {
	return model.FlowEnvironment{
		Name: name, WebsiteID: input.WebsiteID, AutoDeploy: input.AutoDeploy,
		ApprovalRequired: input.ApprovalRequired, HealthCheckSuccessCount: 2,
		ExternalVerifyTimeoutSeconds: 60, StabilizationMinutes: 5, RuntimeMonitorEnabled: true,
		RuntimeFailureThreshold: 3, RuntimeRecoveryThreshold: 2, AutoRollbackDuringStabilization: true,
		RetainPreviousMinutes: 30, Sort: sortOrder, Enabled: true,
	}
}
