package service

import "testing"

func TestValidateCodeReadOnlySQL(t *testing.T) {
	allowed := []string{
		"SELECT id, name FROM users LIMIT 10",
		"SHOW TABLES",
		"EXPLAIN SELECT * FROM users",
		"WITH active AS (SELECT id FROM users) SELECT * FROM active;",
		"PRAGMA table_info(users)",
	}
	for _, statement := range allowed {
		if err := ValidateCodeReadOnlySQL(statement); err != nil {
			t.Errorf("expected query to be allowed: %q: %v", statement, err)
		}
	}
	blocked := []string{
		"UPDATE users SET admin = 1",
		"SELECT 1; DELETE FROM users",
		"WITH changed AS (UPDATE users SET admin=1 RETURNING *) SELECT * FROM changed",
		"SELECT * FROM users FOR UPDATE",
		"SELECT pg_sleep(10)",
		"PRAGMA writable_schema=ON",
		"PRAGMA journal_mode(WAL)",
		"SELECT 1 -- bypass",
		"/* comment */ SELECT 1",
	}
	for _, statement := range blocked {
		if err := ValidateCodeReadOnlySQL(statement); err == nil {
			t.Errorf("expected query to be blocked: %q", statement)
		}
	}
}
