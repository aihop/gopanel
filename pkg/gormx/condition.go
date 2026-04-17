package gormx

import (
	"regexp"
	"strings"

	"github.com/duke-git/lancet/v2/strutil"
)

func Condition(ws []*WhereOne, table string) (where string) {
	for _, w := range ws {
		reg, _ := regexp.Compile("[^a-zA-Z0-9_]+")
		w.Field = strutil.SnakeCase(reg.ReplaceAllString(w.Field, ""))
		if w.Field == "" {
			continue
		}
		if table != "" {
			w.Field = table + "." + w.Field
		}
		if where != "" {
			where += " AND "
		}
		switch w.Rule {
		case WhereRuleEq:
			w.Val = strings.Trim(w.Val, " ")
			where += w.Field + " = '" + w.Val + "'"
		case WhereRuleNEq:
			w.Val = strings.Trim(w.Val, " ")
			where += w.Field + " != '" + w.Val + "'"
		case WhereRuleEqTrue:
			where += w.Field + " = '1'"
		case WhereRuleEqFalse:
			where += w.Field + " = '0'"
		case WhereRuleNull:
			where += w.Field + " = ''"
		case WhereRuleNNull:
			where += w.Field + " != ''"
		case WhereRuleGt:
			where += w.Field + " > " + w.Val
		case WhereRuleGtE:
			where += w.Field + " >= " + w.Val
		case WhereRuleLt:
			where += w.Field + " < " + w.Val
		case WhereRuleLtE:
			where += w.Field + " <= " + w.Val
		case WhereRuleIn:
			var vals string
			valArr := strings.Split(w.Val, ",")
			for _, v := range valArr {
				vals += "'" + v + "',"
			}
			vals = strings.TrimRight(vals, ",")
			where += w.Field + " IN (" + vals + ")"
		case WhereRuleInInt:
			where += w.Field + " IN (" + w.Val + ")"
		case WhereRuleLikes:
			vals := strings.Split(w.Val, ",")
			for _, vv := range vals {
				vv = strings.Trim(vv, " ")
				where += w.Field + " LIKE '%" + vv + "%' OR "
			}
			where = "(" + strings.Trim(where, "OR ") + ")"
		case WhereRuleNLikes:
			vals := strings.Split(w.Val, ",")
			for _, vv := range vals {
				vv = strings.Trim(vv, " ")
				where += w.Field + " NOT LIKE '%" + vv + "%' OR "
			}
			where = "(" + strings.Trim(where, "OR ") + ")"
		case WhereRuleLike:
			w.Val = strings.Trim(w.Val, " ")
			where += w.Field + " LIKE '%" + w.Val + "%'"
		case WhereRuleLikeBf:
			w.Val = strings.Trim(w.Val, " ")
			where += w.Field + " LIKE '" + w.Val + "%'"
		case WhereRuleLikeAf:
			w.Val = strings.Trim(w.Val, " ")
			where += w.Field + " LIKE '%" + w.Val + "'"
		case WhereRuleBtw:
			timeArr := strings.Split(w.Val, ",")
			where += w.Field + " BETWEEN '" + timeArr[0] + "' AND '" + timeArr[1] + "'"
		case WhereRuleNBtw:
			timeArr := strings.Split(w.Val, ",")
			where += w.Field + " NOT BETWEEN '" + timeArr[0] + "' AND '" + timeArr[1] + "'"
		default:
			where += w.Field + " LIKE '%" + w.Val + "%'"
		}
	}
	return where
}

func Conditions(ws [][]*WhereOne, table string) (resWhere string) {
	for _, where := range ws {
		where := Condition(where, table)
		if where != "" {
			resWhere += "(" + strings.Trim(where, "AND ") + ") OR "
		}
	}
	return strings.Trim(resWhere, "OR ")
}
