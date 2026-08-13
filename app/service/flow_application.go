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

type FlowUpdateInput struct {
	Name                       string                 `json:"name"`
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
		items[index].PipelineSourceType = loadFlowPipelineSourceType(s.db, items[index].PipelineID)
		for envIndex := range items[index].Environments {
			environment := &items[index].Environments[envIndex]
			environment.WebsiteName = websiteNames[environment.WebsiteID]
		}
	}
	return total, items, nil
}

func loadFlowPipelineSourceType(db *gorm.DB, id uint) string {
	var pipeline model.Pipeline
	if db.Select("source_type").First(&pipeline, id).Error != nil {
		return "git"
	}
	return pipelineSourceType(&pipeline)
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
	var project model.AIProject
	if err := s.db.First(&project, input.ProjectID).Error; err != nil {
		return nil, buserr.New(constant.ErrFlowProjectNotFound)
	}
	if !includeAll && project.CreatorID != userID {
		return nil, buserr.New(constant.ErrFlowProjectForbidden)
	}
	var pipeline model.Pipeline
	if err := s.db.First(&pipeline, input.PipelineID).Error; err != nil {
		return nil, buserr.New(constant.ErrFlowPipelineNotFound)
	}
	if pipelineSourceType(&pipeline) == "code" && pipeline.CodeProjectID != input.ProjectID {
		return nil, buserr.New(constant.ErrFlowPipelineProjectMismatch)
	}
	var count int64
	if err := s.db.Model(&model.Flow{}).Where("project_id = ?", input.ProjectID).Count(&count).Error; err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, buserr.New(constant.ErrFlowProjectExists)
	}
	environments, err := s.validateEnvironments(input.Environments)
	if err != nil {
		return nil, err
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

func (s *FlowApplicationService) Update(id uint, input FlowUpdateInput, userID uint, includeAll bool) (*model.Flow, error) {
	item, err := s.getAccessible(id, userID, includeAll)
	if err != nil {
		return nil, err
	}
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		return nil, buserr.New(constant.ErrFlowNameRequired)
	}
	if input.PipelineID == 0 {
		return nil, buserr.New(constant.ErrFlowPipelineRequired)
	}
	var pipeline model.Pipeline
	if err := s.db.First(&pipeline, input.PipelineID).Error; err != nil {
		return nil, buserr.New(constant.ErrFlowPipelineNotFound)
	}
	if pipelineSourceType(&pipeline) == "code" && pipeline.CodeProjectID != item.ProjectID {
		return nil, buserr.New(constant.ErrFlowPipelineProjectMismatch)
	}
	environments, err := s.validateEnvironments(input.Environments)
	if err != nil {
		return nil, err
	}
	item.Name = input.Name
	item.PipelineID = input.PipelineID
	item.AutoStartAfterCodeDelivery = input.AutoStartAfterCodeDelivery
	if err := s.repo.UpdateWithEnvironments(item, environments); err != nil {
		return nil, err
	}
	item.Environments = environments
	return item, nil
}

func (s *FlowApplicationService) Delete(id uint, userID uint, includeAll bool) error {
	if _, err := s.getAccessible(id, userID, includeAll); err != nil {
		return err
	}
	return s.repo.DeleteConfiguration(id)
}

func (s *FlowApplicationService) getAccessible(id uint, userID uint, includeAll bool) (*model.Flow, error) {
	item, err := s.repo.Get(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, buserr.New(constant.ErrFlowNotFound)
		}
		return nil, err
	}
	if !includeAll && item.CreatedBy != userID {
		return nil, buserr.New(constant.ErrFlowForbidden)
	}
	return item, nil
}

func (s *FlowApplicationService) validateEnvironments(inputs []FlowEnvironmentInput) ([]model.FlowEnvironment, error) {
	if len(inputs) == 0 {
		return nil, buserr.New(constant.ErrFlowEnvironmentRequired)
	}
	environments := make([]model.FlowEnvironment, 0, len(inputs))
	seen := make(map[string]bool)
	for index, raw := range inputs {
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
	return environments, nil
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
