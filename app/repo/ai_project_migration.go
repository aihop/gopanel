package repo

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/aihop/gopanel/app/model"
	"gorm.io/gorm"
)

const legacyAIProjectTable = "ai_groups"

func sameAIProject(left, right *model.AIProject) bool {
	return left.ID == right.ID && left.CreatedAt.Equal(right.CreatedAt) && left.UpdatedAt.Equal(right.UpdatedAt) &&
		left.Name == right.Name && left.Description == right.Description && left.WorkDir == right.WorkDir &&
		reflect.DeepEqual(left.SourceDirs, right.SourceDirs) && left.CreatorID == right.CreatorID &&
		left.RequireQualityGate == right.RequireQualityGate && left.MonthlyTokenBudget == right.MonthlyTokenBudget
}

func MigrateLegacyAIProjects(db *gorm.DB) error {
	if db == nil {
		return errors.New("数据库连接为空")
	}
	if !db.Migrator().HasTable(legacyAIProjectTable) {
		return nil
	}

	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.AutoMigrate(&model.AIProject{}); err != nil {
			return fmt.Errorf("创建项目表失败: %w", err)
		}

		var projects []*model.AIProject
		if err := tx.Table(legacyAIProjectTable).Order("id asc").Find(&projects).Error; err != nil {
			return fmt.Errorf("读取旧项目数据失败: %w", err)
		}
		if len(projects) > 0 {
			projectIDs := make([]uint, 0, len(projects))
			for _, project := range projects {
				projectIDs = append(projectIDs, project.ID)
			}

			var existingProjects []*model.AIProject
			if err := tx.Where("id IN ?", projectIDs).Find(&existingProjects).Error; err != nil {
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
				if !sameAIProject(existing, project) {
					return fmt.Errorf("新旧项目表的项目 %d 内容冲突，已取消自动迁移", project.ID)
				}
			}
			if len(pendingProjects) > 0 {
				if result := tx.Create(&pendingProjects); result.Error != nil {
					return fmt.Errorf("迁移旧项目数据失败: %w", result.Error)
				} else if result.RowsAffected != int64(len(pendingProjects)) {
					return fmt.Errorf("项目迁移数量不一致: 预期 %d 条，实际 %d 条", len(pendingProjects), result.RowsAffected)
				}
			}
		}

		if err := tx.Migrator().DropTable(legacyAIProjectTable); err != nil {
			return fmt.Errorf("清理旧项目表失败: %w", err)
		}
		return nil
	})
}
