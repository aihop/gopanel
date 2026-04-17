package gormx

import (
	"time"

	"gorm.io/gorm"
	"gorm.io/hints"
)

type ScopeType = func(*gorm.DB) *gorm.DB

var NothingScope = func(db *gorm.DB) *gorm.DB {
	return db
}

func Use(val bool, items ...ScopeType) ScopeType {
	return func(db *gorm.DB) *gorm.DB {
		if val {
			return db.Scopes(items...)
		}
		return db
	}
}

func Or(val bool, wheres string) ScopeType {
	return func(db *gorm.DB) *gorm.DB {
		if val {
			return db.Where(wheres)
		}
		return db
	}
}

func Select(val string) ScopeType {
	return func(db *gorm.DB) *gorm.DB {
		if val != "" {
			db.Select(CamelToSnake(val))
		}
		return db
	}
}

func Limit(val int) ScopeType {
	return func(db *gorm.DB) *gorm.DB {
		if val > 0 {
			db.Limit(val)
		}
		return db
	}
}

func Page(page, limit int) ScopeType {
	return func(db *gorm.DB) *gorm.DB {
		if page > 1 && limit > 0 {
			db.Offset((page - 1) * limit)
		}
		return db
	}
}

func Comment(clause, val string) ScopeType {
	return func(db *gorm.DB) *gorm.DB {
		if val != "" {
			db.Clauses(hints.CommentBefore(clause, val))
		}
		return db
	}
}

func WhereIds(ids []uint) ScopeType {
	return whereFieldContainUint("id", ids)
}

func WhereCreatedAtAfter(t time.Time) ScopeType {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("created_at > ?", t)
	}
}

func WhereCreatedAtBefore(t time.Time) ScopeType {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("created_at < ?", t)
	}
}

// 根据val长度生成不同的查询条件
func whereFieldContainUint(field string, val []uint) ScopeType {
	return func(db *gorm.DB) *gorm.DB {
		if len(val) == 1 {
			return db.Where(field+" = ?", val[0])
		}
		return db.Where(field+" in ?", val)
	}
}

func WhereParentIds(ids []uint) ScopeType {
	return whereFieldContainUint("parent_id", ids)
}
