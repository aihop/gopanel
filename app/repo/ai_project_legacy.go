package repo

import (
	"errors"
	"fmt"

	"gorm.io/gorm"
)

const legacyAIProjectTable = "ai_groups"

func DropLegacyAIProjectTable(db *gorm.DB) error {
	if db == nil {
		return errors.New("数据库连接为空")
	}
	if !db.Migrator().HasTable(legacyAIProjectTable) {
		return nil
	}
	if err := db.Migrator().DropTable(legacyAIProjectTable); err != nil {
		return fmt.Errorf("删除旧项目表失败: %w", err)
	}
	return nil
}
