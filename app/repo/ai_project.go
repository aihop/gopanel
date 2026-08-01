package repo

import (
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
)

type IAIProjectRepo interface {
	CreateProject(project *model.AIProject) error
	GetProjectByID(id uint) (*model.AIProject, error)
	GetProjects(userID uint, includeAll bool, page, limit int) ([]*model.AIProject, int64, error)
	LoadExecutionSummaries(projects []*model.AIProject, userID uint, includeAll bool) error
	UpdateProject(project *model.AIProject) error
	DeleteProject(id uint) error
}

type aiProjectRepo struct{}

func NewAIProjectRepo() IAIProjectRepo {
	return &aiProjectRepo{}
}

func (r *aiProjectRepo) CreateProject(project *model.AIProject) error {
	return global.DB.Create(project).Error
}

func (r *aiProjectRepo) GetProjectByID(id uint) (*model.AIProject, error) {
	var project model.AIProject
	err := global.DB.Where("id = ?", id).First(&project).Error
	return &project, err
}

func (r *aiProjectRepo) GetProjects(userID uint, includeAll bool, page, limit int) ([]*model.AIProject, int64, error) {
	var projects []*model.AIProject
	var total int64
	db := global.DB.Model(&model.AIProject{})
	if !includeAll {
		db = db.Where("creator_id = ?", userID)
	}
	db.Count(&total)
	err := db.Order("created_at desc").Offset((page - 1) * limit).Limit(limit).Find(&projects).Error
	return projects, total, err
}

func (r *aiProjectRepo) LoadExecutionSummaries(projects []*model.AIProject, userID uint, includeAll bool) error {
	if len(projects) == 0 {
		return nil
	}
	projectIDs := make([]uint, 0, len(projects))
	summaries := make(map[uint]*model.AIProjectExecutionSummary, len(projects))
	currentPriorities := make(map[uint]int, len(projects))
	for _, project := range projects {
		projectIDs = append(projectIDs, project.ID)
		updatedAt := project.UpdatedAt
		project.ExecutionSummary = &model.AIProjectExecutionSummary{Status: "idle", UpdatedAt: &updatedAt}
		summaries[project.ID] = project.ExecutionSummary
	}

	type projectTaskCount struct {
		ProjectID uint
		TaskCount int64
	}
	var taskCounts []projectTaskCount
	taskQuery := global.DB.Model(&model.AITask{}).Where("project_id IN ?", projectIDs)
	if !includeAll {
		taskQuery = taskQuery.Where("user_id = ?", userID)
	}
	if err := taskQuery.Select("project_id, COUNT(*) AS task_count").Group("project_id").Scan(&taskCounts).Error; err != nil {
		return err
	}
	projectsByID := make(map[uint]*model.AIProject, len(projects))
	for _, project := range projects {
		projectsByID[project.ID] = project
	}
	for _, count := range taskCounts {
		if project := projectsByID[count.ProjectID]; project != nil {
			project.TaskCount = count.TaskCount
		}
	}

	var activeTasks []*model.AITask
	activeQuery := global.DB.Where("project_id IN ? AND status IN ?", projectIDs, []string{"queued", "running", "pending_approval", "delivering"})
	if !includeAll {
		activeQuery = activeQuery.Where("user_id = ?", userID)
	}
	if err := activeQuery.Order("updated_at desc").Find(&activeTasks).Error; err != nil {
		return err
	}
	sessionIDs := make([]uint, 0, len(activeTasks))
	for _, task := range activeTasks {
		if task.SessionID > 0 {
			sessionIDs = append(sessionIDs, task.SessionID)
		}
	}
	sessionsByID := make(map[uint]*model.AIDevSession, len(sessionIDs))
	if len(sessionIDs) > 0 {
		var sessions []*model.AIDevSession
		if err := global.DB.Where("id IN ?", sessionIDs).Find(&sessions).Error; err != nil {
			return err
		}
		for _, session := range sessions {
			sessionsByID[session.ID] = session
		}
	}
	for _, task := range activeTasks {
		summary := summaries[task.ProjectID]
		if summary == nil {
			continue
		}
		summary.ActiveTaskCount++
		if task.Status == "pending_approval" {
			summary.PendingApprovalCount++
		}
		priority := map[string]int{"queued": 1, "running": 2, "delivering": 3, "pending_approval": 4}[task.Status]
		if priority <= currentPriorities[task.ProjectID] {
			continue
		}
		currentPriorities[task.ProjectID] = priority
		summary.Status = task.Status
		summary.CurrentSessionID = task.SessionID
		summary.CurrentTaskID = task.ID
		summary.CurrentTaskTitle = task.Title
		updatedAt := task.UpdatedAt
		if session := sessionsByID[task.SessionID]; session != nil {
			summary.CurrentStage = session.CurrentStage
			if session.UpdatedAt.After(updatedAt) {
				updatedAt = session.UpdatedAt
			}
		}
		summary.UpdatedAt = &updatedAt
	}
	return nil
}

func (r *aiProjectRepo) UpdateProject(project *model.AIProject) error {
	return global.DB.Save(project).Error
}

func (r *aiProjectRepo) DeleteProject(id uint) error {
	// 连带删除关联的 AI 任务及消息
	global.DB.Where("project_id = ?", id).Delete(&model.AITask{})
	return global.DB.Delete(&model.AIProject{}, id).Error
}
