package service

import (
	"fmt"
	"strings"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/init/db"
	"github.com/aihop/gopanel/utils/mysql"
	clientMysql "github.com/aihop/gopanel/utils/mysql/client"
	"github.com/aihop/gopanel/utils/postgresql"
	clientPostgresql "github.com/aihop/gopanel/utils/postgresql/client"
)

func checkServer(server *model.DatabaseServer) bool {
	switch server.Type {
	case model.DatabaseTypeMysql:
		mysql, err := db.NewMySQL(server.Username, server.Password, fmt.Sprintf("%s:%d", server.Host, server.Port))
		if err == nil {
			_ = mysql.Close()
			server.Status = model.DatabaseServerStatusValid
			return true
		}
	case model.DatabaseTypePostgresql:
		postgres, err := db.NewPostgres(server.Username, server.Password, server.Host, server.Port)
		if err == nil {
			_ = postgres.Close()
			server.Status = model.DatabaseServerStatusValid
			return true
		}
	case "sqlite":
		sqliteDb, err := db.NewSQLite(server.Host)
		if err == nil {
			_ = sqliteDb.Close()
			server.Status = model.DatabaseServerStatusValid
			return true
		}
	}
	server.Status = model.DatabaseServerStatusInvalid
	return false
}

func LoadMysqlClientByFrom(server *model.DatabaseServer) (mysql.MysqlClient, string, error) {
	var (
		dbInfo  clientMysql.DBInfo
		version string
		err     error
	)

	dbInfo.Port = server.Port
	dbInfo.Username = server.Username
	dbInfo.Password = server.Password
	dbInfo.Timeout = 300
	dbInfo.Address = server.Host
	dbInfo.From = string(server.Mode)

	cli, err := mysql.NewMysqlClient(dbInfo)
	if err != nil {
		return nil, "", err
	}

	if v, ok := inferMysqlVersion(cli); ok {
		version = v
	}
	return cli, version, nil
}

func inferMysqlVersion(cli mysql.MysqlClient) (string, bool) {
	execer, ok := cli.(interface {
		ExecSQLForRows(command string, timeout uint) ([]string, error)
	})
	if !ok {
		return "", false
	}
	lines, err := execer.ExecSQLForRows("select version();", 30)
	if err != nil {
		return "", false
	}
	for i := len(lines) - 1; i >= 0; i-- {
		s := strings.TrimSpace(lines[i])
		if s == "" {
			continue
		}
		if strings.EqualFold(s, "version()") || strings.HasPrefix(strings.ToLower(s), "version") {
			continue
		}
		v := trimVersionPrefix(s)
		if v != "" {
			return v, true
		}
	}
	return "", false
}

func trimVersionPrefix(s string) string {
	s = strings.TrimSpace(s)
	start := -1
	for i, r := range s {
		if (r >= '0' && r <= '9') {
			start = i
			break
		}
	}
	if start < 0 {
		return ""
	}
	end := start
	dot := 0
	for end < len(s) {
		r := s[end]
		if (r >= '0' && r <= '9') || r == '.' {
			if r == '.' {
				dot++
			}
			end++
			continue
		}
		break
	}
	if dot == 0 {
		return ""
	}
	return s[start:end]
}

func LoadPostgresqlClientByFrom(database string) (postgresql.PostgresqlClient, error) {
	var (
		dbInfo clientPostgresql.DBInfo
		err    error
	)
	dbInfo.Timeout = 300
	cli, err := postgresql.NewPostgresqlClient(dbInfo)
	if err != nil {
		return nil, err
	}
	return cli, nil
}
