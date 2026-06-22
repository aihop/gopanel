package service

import (
	"reflect"
	"testing"
)

func TestSplitSQLStatements(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "simple statements",
			input:    "SELECT 1; SELECT 2;",
			expected: []string{"SELECT 1", "SELECT 2"},
		},
		{
			name:     "semicolon inside single-quoted string",
			input:    "INSERT INTO t VALUES ('hello; world'); SELECT 1",
			expected: []string{"INSERT INTO t VALUES ('hello; world')", "SELECT 1"},
		},
		{
			name:     "escaped single quote inside string",
			input:    "INSERT INTO t VALUES ('O''Brien'); SELECT 1",
			expected: []string{"INSERT INTO t VALUES ('O''Brien')", "SELECT 1"},
		},
		{
			name:     "semicolon inside backtick",
			input:    "CREATE TABLE `test;table` (id INT); SELECT 1",
			expected: []string{"CREATE TABLE `test;table` (id INT)", "SELECT 1"},
		},
		{
			name:     "semicolon inside double-quoted string",
			input:    `INSERT INTO t VALUES ("hello; world"); SELECT 1`,
			expected: []string{"INSERT INTO t VALUES (\"hello; world\")", "SELECT 1"},
		},
		{
			name:     "line comment with semicolon",
			input:    "SELECT 1; -- comment; here\nSELECT 2;",
			expected: []string{"SELECT 1", "-- comment; here\nSELECT 2"},
		},
		{
			name:     "block comment with semicolon",
			input:    "SELECT 1; /* block; comment */ SELECT 2;",
			expected: []string{"SELECT 1", "/* block; comment */ SELECT 2"},
		},
		{
			name:     "no trailing semicolon",
			input:    "SELECT 1; SELECT 2",
			expected: []string{"SELECT 1", "SELECT 2"},
		},
		{
			name:     "empty content",
			input:    "",
			expected: nil,
		},
		{
			name:     "only comments and whitespace",
			input:    "-- comment\n  \n/* another */",
			expected: []string{"-- comment\n  \n/* another */"},
		},
		{
			name:     "mixed real-world SQL",
			input:    "CREATE TABLE `test` (`id` int, `name` varchar(255) DEFAULT 'hello; world'); INSERT INTO `test` VALUES (1, 'foo;bar');",
			expected: []string{"CREATE TABLE `test` (`id` int, `name` varchar(255) DEFAULT 'hello; world')", "INSERT INTO `test` VALUES (1, 'foo;bar')"},
		},
		{
			name:     "double dash without space is not a comment",
			input:    "SELECT 1--2; SELECT 3",
			expected: []string{"SELECT 1--2", "SELECT 3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitSQLStatements(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("splitSQLStatements()\ngot:  %#v\nwant: %#v", result, tt.expected)
			}
		})
	}
}
