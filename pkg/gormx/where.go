package gormx

import (
	"regexp"
	"strings"

	"github.com/duke-git/lancet/v2/slice"

	"gorm.io/gorm"
)

func Where(w *WhereOne, table string) ScopeType {
	return func(db *gorm.DB) *gorm.DB {
		reg, _ := regexp.Compile("[^a-zA-Z0-9_]+")
		w.Field = CamelToSnake(reg.ReplaceAllString(w.Field, ""))
		if w.Field == "" {
			return db
		}
		if table != "" {
			w.Field = table + "." + w.Field
		}
		switch w.Rule {
		case WhereRuleEq:
			return db.Where(w.Field+"=?", w.Val)
		case WhereRuleNEq:
			return db.Where(w.Field+"!=?", w.Val)
		case WhereRuleEqTrue:
			return db.Where(w.Field + "=true")
		case WhereRuleEqFalse:
			return db.Where(w.Field + "=false")
		case WhereRuleNull:
			return db.Where(w.Field + "=''")
		case WhereRuleNNull:
			return db.Where(w.Field + "!=''")
		case WhereRuleGt:
			return db.Where(w.Field+"> ?", w.Val)
		case WhereRuleGtE:
			return db.Where(w.Field+">= ?", w.Val)
		case WhereRuleLt:
			return db.Where(w.Field+"< ?", w.Val)
		case WhereRuleLtE:
			return db.Where(w.Field+"<= ?", w.Val)
		case WhereRuleIn:
			return db.Where(w.Field+" IN ?", strings.Split(w.Val, ","))
		case WhereRuleNIn:
			return db.Where(w.Field+" NOT IN ?", strings.Split(w.Val, ","))
		case WhereRuleInInt:
			return db.Where(w.Field+" IN ?", slice.IntSlice(strings.Split(w.Val, ",")))
		case WhereRuleNInInt:
			return db.Where(w.Field+" NOT IN ?", slice.IntSlice(strings.Split(w.Val, ",")))
		case WhereRuleLikes:
			vals := strings.Split(w.Val, ",")
			var condParts []string
			var args []interface{}
			for _, v := range vals {
				if v != "" {
					condParts = append(condParts, w.Field+" LIKE ?")
					args = append(args, "%"+v+"%")
				}
			}
			if len(condParts) > 0 {
				condition := "(" + strings.Join(condParts, " OR ") + ")"
				return db.Where(condition, args...)
			}
			return db
		case WhereRuleNLikes:
			vals := strings.Split(w.Val, ",")
			var condParts []string
			var args []interface{}
			for _, v := range vals {
				if v != "" {
					condParts = append(condParts, w.Field+" NOT LIKE ?")
					args = append(args, "%"+v+"%")
				}
			}
			if len(condParts) > 0 {
				condition := "(" + strings.Join(condParts, " AND ") + ")"
				return db.Where(condition, args...)
			}
			return db
		case WhereRuleLike:
			return db.Where(w.Field+" LIKE ?", "%"+w.Val+"%")
		case WhereRuleLikeBf:
			return db.Where(w.Field+" LIKE ?", w.Val+"%")
		case WhereRuleLikeAf:
			return db.Where(w.Field+" LIKE ?", "%"+w.Val)
		case WhereRuleBtw:
			timeArr := strings.Split(w.Val, ",")
			return db.Where(w.Field+" BETWEEN ? AND ?", timeArr[0], timeArr[1])
		case WhereRuleNBtw:
			return db.Where(w.Field+" IN ?", slice.IntSlice(strings.Split(w.Val, ",")))
		case WhereRuleJArr:
			if db.Dialector.Name() == "postgres" {
				// 假设 w.Val 是逗号分隔的数组值，例如 "1,2,3"
				// 此处直接拼接字符串，注意 SQL 注入风险，必要时应做参数化处理
				return db.Where(w.Field + "::jsonb @> json_build_array(" + w.Val + ")")
			} else {
				// MySQL 使用 JSON_CONTAINS 和 JSON_ARRAY()
				return db.Where("JSON_CONTAINS(`" + w.Field + "`, JSON_ARRAY(" + w.Val + "))")
			}
		case WhereRuleJObj:
			nameVal := strings.Split(w.Val, ",")
			if len(nameVal) < 2 {
				return db // 或返回错误：参数格式不正确
			}
			if db.Dialector.Name() == "postgres" {
				// PostgreSQL 版使用 jsonb_build_object
				return db.Where(w.Field+"::jsonb @> jsonb_build_object(?, ?)", nameVal[0], nameVal[1])
			} else {
				// MySQL 版使用 JSON_CONTAINS 和 JSON_OBJECT
				return db.Where("JSON_CONTAINS(`"+w.Field+"`, JSON_OBJECT(?, ?))", nameVal[0], nameVal[1])
			}
		case WhereRuleJLikes:
			vals := strings.Split(w.Val, ",")
			var condParts []string
			var args []interface{}
			for _, v := range vals {
				if v != "" {
					if db.Dialector.Name() == "postgres" {
						condParts = append(condParts, w.Field+"::text LIKE ?")
					} else {
						condParts = append(condParts, "JSON_UNQUOTE("+w.Field+") LIKE ?")
					}
					args = append(args, "%"+v+"%")
				}
			}
			if len(condParts) > 0 {
				condition := "(" + strings.Join(condParts, " OR ") + ")"
				return db.Where(condition, args...)
			}
			return db
		default:
			return db.Where(w.Field+" LIKE ?", "%"+w.Val+"%")
		}
	}
}
