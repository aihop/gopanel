package db

import "strings"

func mysqlQuotedIdentifier(name string) string {
	return "`" + strings.ReplaceAll(name, "`", "``") + "`"
}

func mysqlLiteral(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

func postgresQuotedIdentifier(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

func postgresLiteral(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}
