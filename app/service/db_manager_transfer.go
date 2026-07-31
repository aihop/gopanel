package service

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"strings"

	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/app/model"
)

func (s *DBManagerService) ImportTable(req request.ImportTableReq) (int, error) {
	db, err := s.getDBConn(req.ServerID, req.DatabaseName)
	if err != nil {
		return 0, err
	}
	defer db.Close()

	server, _ := s.serverRepo.Get(req.ServerID)
	tableName := sanitizeIdent(req.TableName)
	switch req.Format {
	case "sql":
		return execSQLImport(db, req.Content)
	case "csv":
		return importCSV(db, server.Type, tableName, req.Content)
	default:
		return 0, fmt.Errorf("unsupported import format: %s", req.Format)
	}
}

func importCSV(db *sql.DB, dbType model.DatabaseType, tableName, content string) (int, error) {
	reader := csv.NewReader(strings.NewReader(strings.TrimPrefix(content, "\ufeff")))
	headers, err := reader.Read()
	if err != nil {
		return 0, fmt.Errorf("read CSV header: %w", err)
	}

	quotedTable := quoteTable(dbType, tableName)
	stringColumns := tableStringColumns(db, quotedTable)
	columns := make([]string, 0, len(headers))
	columnNames := make([]string, 0, len(headers))
	placeholders := make([]string, 0, len(headers))
	for _, header := range headers {
		column := sanitizeIdent(header)
		if column == "" {
			return 0, fmt.Errorf("CSV header contains an empty column")
		}
		columnNames = append(columnNames, column)
		columns = append(columns, quoteIdent(dbType, column))
		if dbType == model.DatabaseTypePostgresql {
			placeholders = append(placeholders, fmt.Sprintf("$%d", len(placeholders)+1))
		} else {
			placeholders = append(placeholders, "?")
		}
	}

	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}
	statement := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", quotedTable, strings.Join(columns, ", "), strings.Join(placeholders, ", "))
	imported := 0
	for rowNumber := 2; ; rowNumber++ {
		values, readErr := reader.Read()
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			_ = tx.Rollback()
			return imported, fmt.Errorf("read CSV row %d: %w", rowNumber, readErr)
		}
		args := make([]interface{}, len(values))
		for index, value := range values {
			args[index] = emptyStringToNull(value, columnNames[index], stringColumns)
		}
		if _, execErr := tx.Exec(statement, args...); execErr != nil {
			_ = tx.Rollback()
			return imported, fmt.Errorf("import CSV row %d: %w", rowNumber, execErr)
		}
		imported++
	}
	if imported == 0 {
		_ = tx.Rollback()
		return 0, fmt.Errorf("csv must have at least one data row")
	}
	if err := tx.Commit(); err != nil {
		return imported, fmt.Errorf("commit CSV import: %w", err)
	}
	return imported, nil
}

func (s *DBManagerService) ImportSQLContent(serverID uint, databaseName, content string) (int, error) {
	db, err := s.getDBConn(serverID, databaseName)
	if err != nil {
		return 0, err
	}
	defer db.Close()
	return execSQLImport(db, content)
}

func (s *DBManagerService) ExportTable(req request.ExportTableReq) (string, string, error) {
	db, err := s.getDBConn(req.ServerID, req.DatabaseName)
	if err != nil {
		return "", "", err
	}
	defer db.Close()

	server, _ := s.serverRepo.Get(req.ServerID)
	tableName := sanitizeIdent(req.TableName)
	q := string(quoteChar(server.Type))
	quote := func(name string) string {
		return q + name + q
	}

	var columnList string
	var exportColumns []string
	if len(req.Columns) > 0 {
		quotedColumns := make([]string, len(req.Columns))
		for index, column := range req.Columns {
			quotedColumns[index] = quote(sanitizeIdent(column))
		}
		columnList = strings.Join(quotedColumns, ", ")
		exportColumns = req.Columns
	} else {
		columnList = "*"
		columnRows, err := db.Query(fmt.Sprintf("SELECT %s FROM %s LIMIT 0", columnList, quote(tableName)))
		if err != nil {
			return "", "", err
		}
		exportColumns, err = columnRows.Columns()
		columnRows.Close()
		if err != nil {
			return "", "", err
		}
	}

	dataSQL := fmt.Sprintf("SELECT %s FROM %s", columnList, quote(tableName))
	if req.Where != "" {
		dataSQL += " WHERE " + req.Where
	}
	rows, err := db.Query(dataSQL)
	if err != nil {
		return "", "", err
	}
	defer rows.Close()

	if req.Format == "sql" {
		dump := generateSQLDump(db, server.Type, tableName, req, exportColumns, rows, quote)
		return dump, fmt.Sprintf("%s_%s.sql", req.DatabaseName, tableName), nil
	}
	return generateCSV(exportColumns, rows), fmt.Sprintf("%s_%s.csv", req.DatabaseName, tableName), nil
}
