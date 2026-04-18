package service

import (
	"errors"
	"fmt"

	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/init/db"
	"github.com/aihop/gopanel/pkg/gormx"
)

func NewDatabase() *DatabaseService {
	return &DatabaseService{
		repo: repo.NewDatabase(),
	}
}

type DatabaseService struct {
	repo *repo.DatabaseRepo
}

func (s *DatabaseService) List(ctx *gormx.Contextx) (res []*model.Database, err error) {
	res, err = s.repo.List(ctx)
	return
}

func (s DatabaseService) Create(req *request.DatabaseCreate) error {
	server, err := NewDatabaseServer().Get(req.ServerID)
	if err != nil {
		return errors.New("获取数据库服务器失败: " + err.Error())
	}
	switch server.Type {
	case model.DatabaseTypeMysql:
		mysql, err := db.NewMySQL(server.Username, server.Password, fmt.Sprintf("%s:%d", server.Host, server.Port))
		if err != nil {
			return errors.New("获取数据库服务器失败: " + err.Error())
		}
		defer func(mysql *db.MySQL) {
			_ = mysql.Close()
		}(mysql)
		if req.CreateUser {
			if err = NewDatabaseUser().Create(&request.DatabaseUserCreate{
				ServerID: req.ServerID,
				Username: req.Username,
				Password: req.Password,
				Host:     req.Host,
			}); err != nil {
				return errors.New("创建用户失败: " + err.Error())
			}
		}
		if err = mysql.DatabaseCreate(req.Name); err != nil {
			return errors.New("创建数据库失败: " + err.Error())
		}
		if req.Username != "" {
			if err = mysql.PrivilegesGrant(req.Username, req.Name, req.Host); err != nil {
				return errors.New("授权用户失败: " + err.Error())
			}
		}
	case model.DatabaseTypePostgresql:
		postgres, err := db.NewPostgres(server.Username, server.Password, server.Host, server.Port)
		if err != nil {
			return errors.New("获取数据库服务器失败: " + err.Error())
		}
		defer func(postgres *db.Postgres) {
			_ = postgres.Close()
		}(postgres)
		if req.CreateUser {
			if err = NewDatabaseUser().Create(&request.DatabaseUserCreate{
				ServerID: req.ServerID,
				Username: req.Username,
				Password: req.Password,
				Host:     req.Host,
			}); err != nil {
				return errors.New("创建用户失败: " + err.Error())
			}
		}
		if err = postgres.DatabaseCreate(req.Name); err != nil {
			return errors.New("创建数据库失败: " + err.Error())
		}
		if req.Username != "" {
			if err = postgres.PrivilegesGrant(req.Username, req.Name); err != nil {
				return err
			}
		}
		if err = postgres.DatabaseComment(req.Name, req.Comment); err != nil {
			return err
		}
	case "sqlite":
		// SQLite 是单文件，"创建数据库" 实际上在 checkServer / connect 阶段只要文件存在就 ok
		// 这里可以验证一下路径是否有效
		sqliteDb, err := db.NewSQLite(req.Host)
		if err != nil {
			return errors.New("无法访问 SQLite 数据库文件: " + err.Error())
		}
		_ = sqliteDb.Close()
	}

	return nil
}

func (r DatabaseService) Delete(req *request.DatabaseDelete) error {
	server, err := NewDatabaseServer().Get(req.ServerID)
	if err != nil {
		return err
	}
	switch server.Type {
	case model.DatabaseTypeMysql:
		mysql, err := db.NewMySQL(server.Username, server.Password, fmt.Sprintf("%s:%d", server.Host, server.Port))
		if err != nil {
			return err
		}
		defer func(mysql *db.MySQL) {
			_ = mysql.Close()
		}(mysql)
		return mysql.DatabaseDrop(req.Name)
	case model.DatabaseTypePostgresql:
		postgres, err := db.NewPostgres(server.Username, server.Password, server.Host, server.Port)
		if err != nil {
			return err
		}
		defer func(postgres *db.Postgres) {
			_ = postgres.Close()
		}(postgres)
		return postgres.DatabaseDrop(req.Name)
	}

	return nil
}

func (r DatabaseService) Comment(req *request.DatabaseComment) error {
	server, err := NewDatabaseServer().Get(req.ServerID)
	if err != nil {
		return err
	}

	switch server.Type {
	case model.DatabaseTypeMysql:
		return errors.New("mysql not support database comment")
	case model.DatabaseTypePostgresql:
		postgres, err := db.NewPostgres(server.Username, server.Password, server.Host, server.Port)
		if err != nil {
			return err
		}
		defer func(postgres *db.Postgres) {
			_ = postgres.Close()
		}(postgres)
		return postgres.DatabaseComment(req.Name, req.Comment)
	}

	return nil
}
