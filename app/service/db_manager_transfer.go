package service

import (
	"fmt"
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
	q := string(quoteChar(server.Type))
	quote := func(name string) string {
		return q + name + q
	}

	switch req.Format {
	case "sql":
		return execSQLImport(db, req.Content)
	case "csv":
		normalized := strings.ReplaceAll(req.Content, "\r\n", "\n")
		normalized = strings.ReplaceAll(normalized, "\r", "")
		rows := strings.Split(strings.TrimSpace(normalized), "\n")
		if len(rows) < 2 {
			return 0, fmt.Errorf("csv must have at least a header row and one data row")
		}

		headers := parseCSVFields(strings.TrimSpace(rows[0]))
		if len(headers) == 0 {
			return 0, fmt.Errorf("no columns found in CSV header")
		}

		imported := 0
		for _, line := range rows[1:] {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			values := parseCSVFields(line)
			if len(values) != len(headers) {
				continue
			}

			var columns, placeholders []string
			var args []interface{}
			for index, header := range headers {
				column := sanitizeIdent(header)
				if column == "" {
					continue
				}
				columns = append(columns, quote(column))
				if server.Type == model.DatabaseTypePostgresql {
					placeholders = append(placeholders, fmt.Sprintf("$%d", len(placeholders)+1))
				} else {
					placeholders = append(placeholders, "?")
				}
				if values[index] == "" && isNumericTypeGuess(server.Type, column) {
					args = append(args, nil)
				} else {
					args = append(args, values[index])
				}
			}
			if len(columns) == 0 {
				continue
			}

			sqlStr := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", quote(tableName), strings.Join(columns, ", "), strings.Join(placeholders, ", "))
			if _, err := db.Exec(sqlStr, args...); err == nil {
				imported++
			}
		}
		return imported, nil
	default:
		return 0, fmt.Errorf("unsupported import format: %s", req.Format)
	}
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
