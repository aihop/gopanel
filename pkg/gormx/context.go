package gormx

import (
	"strings"

	"github.com/duke-git/lancet/v2/convertor"
	"github.com/duke-git/lancet/v2/slice"
	"gorm.io/gorm"
)

func Context(ctx *Contextx, scope ...ScopeType) ScopeType {
	if ctx == nil && len(scope) == 0 {
		return NothingScope
	}
	if ctx == nil {
		ctx = &Contextx{}
	}
	ctx.Scope = append(ctx.Scope, scope...)
	if ctx.Order != "" {
		by := []string{"DESC", "desc", "ASC", "asc"}
		orders := strings.Split(ctx.Order, ",")
		var newOrder string
		for _, v := range orders {
			if strings.Contains(v, ".") {
				orderArr := strings.Split(v, ".")
				if len(orderArr) == 2 {
					ad := strings.Split(orderArr[1], " ")
					if len(ad) == 2 {
						if slice.Contain(by, ad[1]) {
							newOrder += "JSON_EXTRACT(" + orderArr[0] + ", '$." + ad[0] + "') " + ad[1] + ","
						}
					}
				}
			} else {
				ad := strings.Split(v, " ")
				if len(ad) == 2 {
					if slice.Contain(by, ad[1]) {
						newOrder += CamelToSnake(ad[0]) + " " + ad[1] + ","
					}
				}
			}
		}
		ctx.Order = strings.Trim(newOrder, ",")
	}

	if ctx.Limit <= 20 && ctx.Page > 5000 {
		return fastOffset(ctx)
	} else if ctx.Limit <= 50 && ctx.Page > 2000 {
		return fastOffset(ctx)
	} else if ctx.Limit <= 100 && ctx.Page > 1000 {
		return fastOffset(ctx)
	} else if ctx.Limit <= 200 && ctx.Page > 500 {
		return fastOffset(ctx)
	} else if ctx.Limit <= 300 && ctx.Page > 250 {
		return fastOffset(ctx)
	}

	return func(db *gorm.DB) *gorm.DB {
		return db.Scopes(Select(ctx.Select), base(ctx))
	}
}

func Wheres(ctx *Wherex) ScopeType {
	if ctx == nil {
		ctx = &Wherex{}
	}
	return func(db *gorm.DB) *gorm.DB {
		var wheres []ScopeType
		if ctx.Wheres != nil {
			for _, ws := range ctx.Wheres {
				wheres = append(wheres, Where(ws, ctx.Table))
			}
		}
		if ctx.Joins != nil {
			for _, join := range ctx.Joins {
				db.Joins(join.Table, Joins(join.Table, db, join.Wheres != nil, join.Wheres))
			}
		}
		var condition string
		if ctx.Conditions != nil {
			condition = Conditions(ctx.Conditions, ctx.Table)
		}
		return db.Scopes(
			Use(ctx.Wheres != nil, wheres...),
			Or(condition != "", condition),
		)
	}
}

func wherePreload(ctx *Contextx) ScopeType {
	return func(db *gorm.DB) *gorm.DB {
		var wheres []ScopeType
		if ctx.Wheres != nil {
			for _, ws := range ctx.Wheres {
				wheres = append(wheres, Where(ws, ctx.Table))
			}
		}
		if ctx.Joins != nil {
			for _, join := range ctx.Joins {
				db.Joins(join.Table, Joins(join.Table, db, join.Wheres != nil, join.Wheres))
			}
		}
		if ctx.Preloads != nil {
			for _, preload := range ctx.Preloads {
				db.Preload(preload.Table, Preload(preload.Wheres != nil, preload.Wheres, preload.Order))
			}
		}
		var condition string
		if ctx.Conditions != nil {
			condition = Conditions(ctx.Conditions, ctx.Table)
		}
		return db.Scopes(
			Use(ctx.Wheres != nil, wheres...),
			Or(condition != "", condition),
		).Scopes(ctx.Scope...).Order(ctx.Order)
	}
}

func base(ctx *Contextx) ScopeType {
	return func(db *gorm.DB) *gorm.DB {
		var wheres []ScopeType
		if ctx.Wheres != nil {
			for _, ws := range ctx.Wheres {
				wheres = append(wheres, Where(ws, ctx.Table))
			}
		}
		if ctx.Joins != nil {
			for _, join := range ctx.Joins {
				db.Joins(join.Table, Joins(join.Table, db, join.Wheres != nil, join.Wheres))
			}
		}
		if ctx.Preloads != nil {
			for _, preload := range ctx.Preloads {
				db.Preload(preload.Table, Preload(preload.Wheres != nil, preload.Wheres, preload.Order))
			}
		}
		var condition string
		if ctx.Conditions != nil {
			condition = Conditions(ctx.Conditions, ctx.Table)
		}
		if ctx.ExcludeIds != nil {
			var idsStr string
			for _, id := range ctx.ExcludeIds {
				idsStr += convertor.ToString(id) + ","
			}
			idsStr = strings.Trim(idsStr, ",")
			wheres = append(wheres, Where(&WhereOne{Field: "id", Rule: WhereRuleNEq, Val: idsStr}, ctx.Table))
		}
		return db.Scopes(
			Limit(ctx.Limit),
			Page(ctx.Page, ctx.Limit),
			Comment("select", ctx.Comment),
			Use(ctx.Wheres != nil, wheres...),
			Or(condition != "", condition),
		).Scopes(ctx.Scope...).Order(ctx.Order)
	}
}

func fastOffset(ctx *Contextx) ScopeType {
	return func(db *gorm.DB) *gorm.DB {
		var ids []int
		tx := db.Session(&gorm.Session{})
		if err := tx.Scopes(base(ctx)).Pluck("id", &ids).Error; err != nil {
			_ = db.AddError(err)
			return db
		}
		if len(ids) == 0 {
			_ = db.AddError(gorm.ErrRecordNotFound)
			return db
		}
		return db.Where("id in ?", ids).Scopes(Select(ctx.Select), wherePreload(ctx)).Order(ctx.Order)
	}
}

func NewContext(limit int, order string) *Contextx {
	return &Contextx{Limit: limit, Order: order}
}
