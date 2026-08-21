package service

import (
	"testing"

	"github.com/aihop/gopanel/app/model"
)

func TestDatabaseUserAccessScope(t *testing.T) {
	tests := []struct {
		name       string
		serverType model.DatabaseType
		host       string
		want       model.DatabaseUserAccessScope
	}{
		{name: "all hosts", serverType: model.DatabaseTypeMysql, host: "%", want: model.DatabaseUserAccessScopeAll},
		{name: "localhost", serverType: model.DatabaseTypeMysql, host: "localhost", want: model.DatabaseUserAccessScopeLocal},
		{name: "loopback address", serverType: model.DatabaseTypeMysql, host: "127.0.0.1", want: model.DatabaseUserAccessScopeLocal},
		{name: "specific subnet", serverType: model.DatabaseTypeMysql, host: "10.0.%", want: model.DatabaseUserAccessScopeSpecific},
		{name: "postgres does not use host identity", serverType: model.DatabaseTypePostgresql, host: "%", want: model.DatabaseUserAccessScopeUnknown},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := databaseUserAccessScope(test.serverType, test.host); got != test.want {
				t.Fatalf("databaseUserAccessScope(%q, %q) = %q, want %q", test.serverType, test.host, got, test.want)
			}
		})
	}
}
