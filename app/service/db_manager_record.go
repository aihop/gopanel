package service

import (
	"fmt"
	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/app/model"
	"strings"
)

func (s *DBManagerService) InsertRecord(req request.InsertRecordReq) error {
	db, err := s.getDBConn(req.ServerID, req.DatabaseName)
	if err != nil {
		return err
	}
	defer db.Close()
	server, _ := s.serverRepo.Get(req.ServerID)
	dbType := server.Type
	tableName := sanitizeIdent(req.TableName)
	removeVirtualColumns(req.Data)
	var cols []string
	var placeholders []string
	var args []interface{}
	paramOffset := 1
	for k, v := range req.Data {
		col := sanitizeIdent(k)
		if col == "" {
			continue
		}
		cols = append(cols, quoteIdent(dbType, col))
		if dbType == model.DatabaseTypePostgresql {
			placeholders = append(placeholders, fmt.Sprintf("$%d", paramOffset))
		} else {
			placeholders = append(placeholders, "?")
		}
		args = append(args, v)
		paramOffset++
	}
	sqlStr := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", quoteTable(dbType, tableName), strings.Join(cols, ", "), strings.Join(placeholders, ", "))
	_, err = db.Exec(sqlStr, args...)
	return err
}
func (s *DBManagerService) UpdateRecord(req request.UpdateRecordReq) error {
	db, err := s.getDBConn(req.ServerID, req.DatabaseName)
	if err != nil {
		return err
	}
	defer db.Close()
	server, _ := s.serverRepo.Get(req.ServerID)
	dbType := server.Type
	tableName := sanitizeIdent(req.TableName)
	removeVirtualColumns(req.Data)
	var setCols []string
	var args []interface{}
	paramOffset := 1
	stripComplexConditions(req.Conditions)
	rowidValue, hasRowid := popRowidCondition(req.Conditions)
	idValue, hasID := popCondition(req.Conditions, "id")
	if hasID && idValue != nil {
		req.Conditions = map[string]interface{}{"id": idValue}
	}
	if dbType == model.DatabaseSQLite {
		if hasRowid && rowidValue != nil {
			if rowid, ok := normalizeRowid(rowidValue); ok {
				for k, val := range req.Data {
					col := sanitizeIdent(k)
					if col == "" {
						continue
					}
					setCols = append(setCols, fmt.Sprintf("%s = ?", quoteIdent(dbType, col)))
					args = append(args, val)
				}
				args = append(args, rowid)
				sqlStr := fmt.Sprintf("UPDATE %s SET %s WHERE rowid = ?", quoteTable(dbType, tableName), strings.Join(setCols, ", "))
				_, err = db.Exec(sqlStr, args...)
				return err
			}
		}
	}
	for k, v := range req.Data {
		col := sanitizeIdent(k)
		if col == "" {
			continue
		}
		if dbType == model.DatabaseTypePostgresql {
			setCols = append(setCols, fmt.Sprintf("%s = $%d", quoteIdent(dbType, col), paramOffset))
		} else {
			setCols = append(setCols, fmt.Sprintf("%s = ?", quoteIdent(dbType, col)))
		}
		args = append(args, v)
		paramOffset++
	}
	whereSql, whereArgs := buildWhereClause(req.Conditions, paramOffset, dbType)
	args = append(args, whereArgs...)
	sqlStr := fmt.Sprintf("UPDATE %s SET %s WHERE %s", quoteTable(dbType, tableName), strings.Join(setCols, ", "), whereSql)
	_, err = db.Exec(sqlStr, args...)
	return err
}
func (s *DBManagerService) DeleteRecord(req request.DeleteRecordReq) error {
	db, err := s.getDBConn(req.ServerID, req.DatabaseName)
	if err != nil {
		return err
	}
	defer db.Close()
	server, _ := s.serverRepo.Get(req.ServerID)
	dbType := server.Type
	tableName := sanitizeIdent(req.TableName)
	stripComplexConditions(req.Conditions)
	rowidValue, hasRowid := popRowidCondition(req.Conditions)
	idValue, hasID := popCondition(req.Conditions, "id")
	if hasID && idValue != nil {
		req.Conditions = map[string]interface{}{"id": idValue}
	}
	if dbType == model.DatabaseSQLite {
		if hasRowid && rowidValue != nil {
			if rowid, ok := normalizeRowid(rowidValue); ok {
				_, err = db.Exec(fmt.Sprintf("DELETE FROM %s WHERE rowid = ?", quoteTable(dbType, tableName)), rowid)
				return err
			}
		}
	}
	whereSql, args := buildWhereClause(req.Conditions, 1, dbType)
	sqlStr := fmt.Sprintf("DELETE FROM %s WHERE %s", quoteTable(dbType, tableName), whereSql)
	_, err = db.Exec(sqlStr, args...)
	return err
}
