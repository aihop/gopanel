package db

import (
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	_ "github.com/glebarez/go-sqlite"
)

func isRecoverableSQLiteOpenError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "database disk image is malformed") ||
		strings.Contains(message, "malformed database schema") ||
		strings.Contains(message, "already exists (11)") ||
		strings.Contains(message, "already exists (1)")
}

func repairSQLiteSchema(dbPath string, cause error) error {
	info, err := os.Stat(dbPath)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("database path is a directory: %s", dbPath)
	}

	backupPath, err := backupSQLiteFile(dbPath)
	if err != nil {
		return err
	}

	dsn := fmt.Sprintf("file:%s?_pragma=writable_schema(ON)", dbPath)
	rawDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return err
	}
	defer rawDB.Close()

	if _, err := rawDB.Exec("PRAGMA writable_schema = ON"); err != nil {
		return fmt.Errorf("enable writable_schema failed: %w", err)
	}

	duplicateNames, err := loadDuplicateSQLiteTableNames(rawDB)
	if err != nil {
		return fmt.Errorf("scan duplicate schema entries failed: %w", err)
	}
	targetNames := extractMalformedSchemaTargets(cause)
	if len(duplicateNames) == 0 && len(targetNames) == 0 {
		return fmt.Errorf("no repairable schema entries found, backup saved to %s", backupPath)
	}

	for _, name := range duplicateNames {
		if _, err := rawDB.Exec(`
			DELETE FROM sqlite_schema
			WHERE (
				(type = 'table' AND name = ?)
				OR tbl_name = ?
			)
			  AND rowid NOT IN (
				SELECT MIN(rowid)
				FROM sqlite_schema
				WHERE type = 'table' AND name = ?
			  )
		`, name, name, name); err != nil {
			return fmt.Errorf("remove duplicate schema entries for %s failed: %w", name, err)
		}
	}

	for _, name := range targetNames {
		if _, err := rawDB.Exec(`
			DELETE FROM sqlite_schema
			WHERE name = ?
			   OR tbl_name = ?
		`, name, name); err != nil {
			return fmt.Errorf("remove malformed schema entries for %s failed: %w", name, err)
		}
	}

	if _, err := rawDB.Exec("PRAGMA writable_schema = OFF"); err != nil {
		return fmt.Errorf("disable writable_schema failed: %w", err)
	}
	return nil
}

func loadDuplicateSQLiteTableNames(rawDB *sql.DB) ([]string, error) {
	rows, err := rawDB.Query(`
		SELECT name
		FROM sqlite_schema
		WHERE type = 'table'
		  AND name NOT LIKE 'sqlite_%'
		GROUP BY name
		HAVING COUNT(*) > 1
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		names = append(names, name)
	}
	return names, rows.Err()
}

func backupSQLiteFile(dbPath string) (string, error) {
	source, err := os.Open(dbPath)
	if err != nil {
		return "", err
	}
	defer source.Close()

	backupPath := filepath.Join(filepath.Dir(dbPath), fmt.Sprintf("%s.bak.%s", filepath.Base(dbPath), time.Now().Format("20060102-150405")))
	target, err := os.OpenFile(backupPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return "", err
	}
	defer target.Close()

	if _, err := io.Copy(target, source); err != nil {
		return "", err
	}
	return backupPath, nil
}

func extractMalformedSchemaTargets(cause error) []string {
	if cause == nil {
		return nil
	}
	message := cause.Error()
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`malformed database schema \(([^)]+)\)`),
		regexp.MustCompile("table `([^`]+)` already exists"),
		regexp.MustCompile(`table "([^"]+)" already exists`),
	}
	var names []string
	for _, pattern := range patterns {
		matches := pattern.FindAllStringSubmatch(message, -1)
		for _, match := range matches {
			if len(match) > 1 {
				names = appendUniqueSchemaNames(names, strings.TrimSpace(match[1]))
			}
		}
	}
	return names
}

func appendUniqueSchemaNames(base []string, names ...string) []string {
	seen := make(map[string]struct{}, len(base))
	for _, name := range base {
		seen[name] = struct{}{}
	}
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		base = append(base, name)
	}
	return base
}
