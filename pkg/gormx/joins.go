package gormx

import (
	"strings"

	"gorm.io/gorm"
)

func Joins(query string, db *gorm.DB, val bool, wss []*WhereOne) *gorm.DB {
	if val {
		if strings.Contains(query, ".") {
			query = strings.Replace(query, ".", "__", -1)
		}
		var sts []ScopeType
		for _, w := range wss {
			sts = append(sts, Where(w, query))
		}
		return db.Scopes(sts...)
	}
	return db

}
