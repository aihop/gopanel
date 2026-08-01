package repo

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/aihop/gopanel/app/model"
	"gorm.io/gorm"
)

const (
	legacyAIProjectTable       = "ai_groups"
	legacyAIProjectBackupTable = "ai_groups_legacy_backup"
)

type legacyAIProjectColumns struct {
	workDir            bool
	sourceDirs         bool
	requireQualityGate bool
	monthlyTokenBudget bool
}

func MigrateLegacyAIProjects(db *gorm.DB) error {
	if db == nil {
		return errors.New("数据库连接为空")
	}
	if !db.Migrator().HasTable(legacyAIProjectTable) {
		return nil
	}
	if db.Migrator().HasTable(legacyAIProjectBackupTable) {
		return fmt.Errorf("旧项目备份表 %s 已存在，无法安全迁移", legacyAIProjectBackupTable)
	}

	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.AutoMigrate(&model.AIProject{}); err != nil {
			return fmt.Errorf("创建项目表失败: %w", err)
		}

		columns := inspectLegacyAIProjectColumns(tx)
		projects, err := loadLegacyAIProjects(tx, columns)
		if err != nil {
			return err
		}
		if err := copyLegacyAIProjects(tx, projects, columns); err != nil {
			return err
		}
		if err := tx.Migrator().RenameTable(legacyAIProjectTable, legacyAIProjectBackupTable); err != nil {
			return fmt.Errorf("备份旧项目表失败: %w", err)
		}
		return nil
	})
}

func inspectLegacyAIProjectColumns(db *gorm.DB) legacyAIProjectColumns {
	return legacyAIProjectColumns{
		workDir:            db.Migrator().HasColumn(legacyAIProjectTable, "work_dir"),
		sourceDirs:         db.Migrator().HasColumn(legacyAIProjectTable, "source_dirs"),
		requireQualityGate: db.Migrator().HasColumn(legacyAIProjectTable, "require_quality_gate"),
		monthlyTokenBudget: db.Migrator().HasColumn(legacyAIProjectTable, "monthly_token_budget"),
	}
}

func loadLegacyAIProjects(db *gorm.DB, columns legacyAIProjectColumns) ([]*model.AIProject, error) {
	selectedColumns := []string{"id", "created_at", "updated_at", "name", "description", "creator_id"}
	if columns.workDir {
		selectedColumns = append(selectedColumns, "work_dir")
	}
	if columns.sourceDirs {
		selectedColumns = append(selectedColumns, "source_dirs")
	}
	if columns.requireQualityGate {
		selectedColumns = append(selectedColumns, "require_quality_gate")
	}
	if columns.monthlyTokenBudget {
		selectedColumns = append(selectedColumns, "monthly_token_budget")
	}

	var projects []*model.AIProject
	if err := db.Table(legacyAIProjectTable).Select(strings.Join(selectedColumns, ", ")).Order("id asc").Find(&projects).Error; err != nil {
		return nil, fmt.Errorf("读取旧项目数据失败: %w", err)
	}
	return projects, nil
}

func copyLegacyAIProjects(db *gorm.DB, projects []*model.AIProject, columns legacyAIProjectColumns) error {
	if len(projects) == 0 {
		return nil
	}

	projectIDs := make([]uint, 0, len(projects))
	for _, project := range projects {
		projectIDs = append(projectIDs, project.ID)
	}
	var existingProjects []*model.AIProject
	if err := db.Where("id IN ?", projectIDs).Find(&existingProjects).Error; err != nil {
		return fmt.Errorf("检查项目数据冲突失败: %w", err)
	}
	existingByID := make(map[uint]*model.AIProject, len(existingProjects))
	for _, project := range existingProjects {
		existingByID[project.ID] = project
	}

	pendingProjects := make([]*model.AIProject, 0, len(projects))
	for _, project := range projects {
		existing := existingByID[project.ID]
		if existing == nil {
			pendingProjects = append(pendingProjects, project)
			continue
		}
		if !sameLegacyAIProject(existing, project, columns) {
			return fmt.Errorf("新旧项目表的项目 %d 内容冲突，已取消自动迁移", project.ID)
		}
	}
	if len(pendingProjects) == 0 {
		return nil
	}
	result := db.Create(&pendingProjects)
	if result.Error != nil {
		return fmt.Errorf("迁移旧项目数据失败: %w", result.Error)
	}
	if result.RowsAffected != int64(len(pendingProjects)) {
		return fmt.Errorf("项目迁移数量不一致: 预期 %d 条，实际 %d 条", len(pendingProjects), result.RowsAffected)
	}
	return nil
}

func sameLegacyAIProject(current, legacy *model.AIProject, columns legacyAIProjectColumns) bool {
	if current.ID != legacy.ID || !current.CreatedAt.Equal(legacy.CreatedAt) || !current.UpdatedAt.Equal(legacy.UpdatedAt) ||
		current.Name != legacy.Name || current.Description != legacy.Description || current.CreatorID != legacy.CreatorID {
		return false
	}
	if columns.workDir && current.WorkDir != legacy.WorkDir {
		return false
	}
	if columns.sourceDirs && !reflect.DeepEqual(current.SourceDirs, legacy.SourceDirs) {
		return false
	}
	if columns.requireQualityGate && current.RequireQualityGate != legacy.RequireQualityGate {
		return false
	}
	return !columns.monthlyTokenBudget || current.MonthlyTokenBudget == legacy.MonthlyTokenBudget
}
