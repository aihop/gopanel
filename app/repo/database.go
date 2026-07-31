package repo

import (
	"fmt"
	"net/url"
	"slices"
	"strings"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/init/db"
	"github.com/aihop/gopanel/pkg/gormx"
	"gorm.io/gorm"
)

type DatabaseRepo struct {
	db *gorm.DB
}

func NewDatabase() *DatabaseRepo {
	return &DatabaseRepo{
		db: global.DB,
	}
}

func (r *DatabaseRepo) MigrateTable() error {
	return r.db.AutoMigrate(&model.Database{})
}

func (r *DatabaseRepo) Create(item *model.DatabaseServer) (err error) {
	return r.db.Model(&model.DatabaseServer{}).Create(item).Error
}

func (r *DatabaseRepo) List(ctx *gormx.Contextx) (*model.DatabaseListResult, error) {
	var databaseServer []*model.DatabaseServer
	query := r.db.Model(&model.DatabaseServer{}).Order("id desc")

	if ctx.Wheres != nil {
		for _, w := range ctx.Wheres {
			if w.Field == "server_id" {
				query = query.Where("id = ?", w.Val)
			}
		}
	}

	if err := query.Find(&databaseServer).Error; err != nil {
		return nil, err
	}
	if err := decryptDatabaseServerPasswords(databaseServer); err != nil {
		return nil, err
	}

	database := make([]*model.Database, 0)
	warnings := make([]model.DatabaseListWarning, 0)
	for _, server := range databaseServer {
		switch server.Type {
		case model.DatabaseTypeMysql, model.DatabaseTypeMariaDB:
			mysql, err := db.NewMySQL(server.Username, server.Password, fmt.Sprintf("%s:%d", server.Host, server.Port))
			if err != nil {
				warnings = append(warnings, databaseListWarning(server, "connection_failed", err))
				continue
			}
			databases, err := mysql.Databases()
			_ = mysql.Close()
			if err != nil {
				warnings = append(warnings, databaseListWarning(server, "query_failed", err))
				continue
			}
			for item := range slices.Values(databases) {
				database = append(database, &model.Database{
					Type:     server.Type,
					Name:     item.Name,
					Server:   server.Name,
					ServerID: server.ID,
					Encoding: item.CharSet,
				})
			}
		case model.DatabaseTypePostgresql:
			postgres, err := db.NewPostgres(server.Username, server.Password, server.Host, server.Port)
			if err != nil {
				warnings = append(warnings, databaseListWarning(server, "connection_failed", err))
				continue
			}
			databases, err := postgres.Databases()
			_ = postgres.Close()
			if err != nil {
				warnings = append(warnings, databaseListWarning(server, "query_failed", err))
				continue
			}
			for item := range slices.Values(databases) {
				database = append(database, &model.Database{
					Type:     model.DatabaseTypePostgresql,
					Name:     item.Name,
					Server:   server.Name,
					ServerID: server.ID,
					Encoding: item.Encoding,
					Comment:  item.Comment,
				})
			}
		case "sqlite":
			// SQLite 是单文件，自身就是一个 Server 也是一个 Database
			database = append(database, &model.Database{
				Type:     model.DatabaseSQLite,
				Name:     server.Name,
				Server:   server.Host, // 借用 server 字段展示路径
				ServerID: server.ID,
				Encoding: "UTF-8",
				Comment:  server.Remark,
			})
		}
	}

	start := (ctx.Page - 1) * ctx.Limit
	if start > len(database) {
		start = len(database)
	}
	end := start + ctx.Limit
	if end > len(database) {
		end = len(database)
	}

	return &model.DatabaseListResult{
		Total:    int64(len(database)),
		Items:    database[start:end],
		Warnings: warnings,
	}, nil
}

func databaseListWarning(server *model.DatabaseServer, code string, err error) model.DatabaseListWarning {
	message := err.Error()
	if server.Password != "" {
		message = strings.ReplaceAll(message, server.Password, "******")
		message = strings.ReplaceAll(message, url.QueryEscape(server.Password), "******")
	}
	return model.DatabaseListWarning{
		ServerID:   server.ID,
		ServerName: server.Name,
		ServerType: server.Type,
		Code:       code,
		Message:    message,
	}
}

func (r *DatabaseRepo) CountByWhere(where *gormx.Wherex) (int64, error) {
	var databaseServer []*model.DatabaseServer
	query := r.db.Model(&model.DatabaseServer{})

	if where != nil {
		for _, w := range where.Wheres {
			if w.Field == "server_id" {
				query = query.Where("id = ?", w.Val)
			}
		}
	}

	if err := query.Find(&databaseServer).Error; err != nil {
		return 0, err
	}
	if err := decryptDatabaseServerPasswords(databaseServer); err != nil {
		return 0, err
	}

	var res int64
	for _, server := range databaseServer {
		switch server.Type {
		case model.DatabaseTypeMysql, model.DatabaseTypeMariaDB:
			mysql, err := db.NewMySQL(server.Username, server.Password, fmt.Sprintf("%s:%d", server.Host, server.Port))
			if err == nil {
				if dbs, err := mysql.Databases(); err == nil {
					res += int64(len(dbs))
				}
				_ = mysql.Close()
			}
		case model.DatabaseTypePostgresql:
			postgres, err := db.NewPostgres(server.Username, server.Password, server.Host, server.Port)
			if err == nil {
				if dbs, err := postgres.Databases(); err == nil {
					res += int64(len(dbs))
				}
				_ = postgres.Close()
			}
		case model.DatabaseSQLite:
			res++
		}
	}
	return res, nil
}

func decryptDatabaseServerPasswords(servers []*model.DatabaseServer) error {
	for _, server := range servers {
		password, err := decryptDatabaseSecret(server.Password)
		if err != nil {
			return err
		}
		server.Password = password
	}
	return nil
}

// 删除
func (r *DatabaseRepo) Delete(id uint) error {
	return r.db.Delete(&model.Database{}, id).Error
}

// 删除指定 Server 下的指定 Database
func (r *DatabaseRepo) DeleteByServerIdName(serverID uint, name string) error {
	return r.db.Delete(&model.Database{}, "server_id = ? AND name = ?", serverID, name).Error
}
