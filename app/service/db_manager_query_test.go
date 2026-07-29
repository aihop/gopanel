package service

import (
	"reflect"
	"testing"

	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/app/model"
)

func TestBuildTableSearchClausePostgresql(t *testing.T) {
	conditions := []request.SearchCondition{
		{Column: "email", Operator: "LIKE", Value: "Admin"},
		{Column: "status", Operator: "NOT LIKE", Value: "%disabled"},
		{Column: "age", Operator: ">=", Value: "18"},
		{Column: "deleted_at", Operator: "IS NULL"},
	}

	whereClause, args := buildTableSearchClause(model.DatabaseTypePostgresql, "id", "42", conditions)
	wantClause := ` WHERE CAST("id" AS TEXT) ILIKE $1 AND CAST("email" AS TEXT) ILIKE $2 AND CAST("status" AS TEXT) NOT ILIKE $3 AND "age" >= $4 AND "deleted_at" IS NULL`
	wantArgs := []interface{}{"%42%", "%Admin%", "%disabled", "18"}
	if whereClause != wantClause {
		t.Fatalf("unexpected PostgreSQL WHERE clause\ngot:  %s\nwant: %s", whereClause, wantClause)
	}
	if !reflect.DeepEqual(args, wantArgs) {
		t.Fatalf("unexpected PostgreSQL args\ngot:  %#v\nwant: %#v", args, wantArgs)
	}
}

func TestBuildTableSearchClauseMysql(t *testing.T) {
	conditions := []request.SearchCondition{
		{Column: "display_name", Operator: "LIKE", Value: "Admin"},
		{Column: "external_id", Operator: "NOT LIKE", Value: "test"},
		{Column: "enabled", Operator: "=", Value: "1"},
	}

	whereClause, args := buildTableSearchClause(model.DatabaseTypeMysql, "id", "42", conditions)
	wantClause := " WHERE LOWER(CAST(`id` AS CHAR)) LIKE LOWER(?) AND LOWER(CAST(`display_name` AS CHAR)) LIKE LOWER(?) AND LOWER(CAST(`external_id` AS CHAR)) NOT LIKE LOWER(?) AND `enabled` = ?"
	wantArgs := []interface{}{"%42%", "%Admin%", "%test%", "1"}
	if whereClause != wantClause {
		t.Fatalf("unexpected MySQL WHERE clause\ngot:  %s\nwant: %s", whereClause, wantClause)
	}
	if !reflect.DeepEqual(args, wantArgs) {
		t.Fatalf("unexpected MySQL args\ngot:  %#v\nwant: %#v", args, wantArgs)
	}
}

func TestBuildTableSearchClauseMariaDBUsesMysqlSemantics(t *testing.T) {
	whereClause, args := buildTableSearchClause(model.DatabaseTypeMariaDB, "name", "GoPanel", nil)
	wantClause := " WHERE LOWER(CAST(`name` AS CHAR)) LIKE LOWER(?)"
	if whereClause != wantClause {
		t.Fatalf("unexpected MariaDB WHERE clause\ngot:  %s\nwant: %s", whereClause, wantClause)
	}
	if !reflect.DeepEqual(args, []interface{}{"%GoPanel%"}) {
		t.Fatalf("unexpected MariaDB args: %#v", args)
	}
}

func TestBuildTableSearchClauseSkipsEmptyColumns(t *testing.T) {
	conditions := []request.SearchCondition{
		{Column: " `\" ", Operator: "LIKE", Value: "ignored"},
	}

	whereClause, args := buildTableSearchClause(model.DatabaseTypePostgresql, "", "unused", conditions)
	if whereClause != "" || len(args) != 0 {
		t.Fatalf("empty columns should not produce filters: clause=%q args=%#v", whereClause, args)
	}
}
