package service

import (
	"errors"
	"fmt"
	"log/slog"
	"slices"

	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/init/db"
	"github.com/aihop/gopanel/pkg/gormx"
)

func NewDatabaseServer() *DatabaseServerService {
	return &DatabaseServerService{
		repo: repo.NewDatabaseServer(),
	}
}

type DatabaseServerService struct {
	repo *repo.DatabaseServerRepo
}

func (s DatabaseServerService) Create(req *request.DatabaseServerCreate, mode model.DatabaseMode) error {
	databaseServer := &model.DatabaseServer{
		Name:     req.Name,
		Type:     model.DatabaseType(req.Type),
		Host:     req.Host,
		Port:     req.Port,
		Username: req.Username,
		Password: req.Password,
		Remark:   req.Remark,
		Mode:     mode,
	}

	if !checkServer(databaseServer) {
		return errors.New("check server connection failed")
	}

	return s.repo.Create(databaseServer)
}

func (s DatabaseServerService) Get(id uint) (res *model.DatabaseServer, err error) {
	if res, err = s.repo.Get(id); err != nil {
		return nil, err
	}
	checkServer(res)
	return res, nil
}

func (s DatabaseServerService) Update(req *request.DatabaseServerUpdate) error {
	server, err := s.Get(req.ID)
	if err != nil {
		return err
	}
	server.Name = req.Name
	server.Host = req.Host
	server.Port = req.Port
	server.Username = req.Username
	server.Password = req.Password
	server.Remark = req.Remark

	if !checkServer(server) {
		return errors.New("check server connection failed")
	}
	return s.repo.Update(server)
}

func (s DatabaseServerService) Delete(id uint) error {
	if _, err := s.Get(id); err != nil {
		return err
	}
	return s.repo.Delete(id)
}

func (s *DatabaseServerService) List(ctx *gormx.Contextx) (res []*model.DatabaseServer, err error) {
	res, err = s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	for server := range slices.Values(res) {
		checkServer(server)
	}
	return
}

func (s *DatabaseServerService) CountByWhere(where *gormx.Wherex) (res int64, err error) {
	return s.repo.CountByWhere(where)
}

func (s DatabaseServerService) Sync(id uint) error {
	server, err := s.Get(id)
	if err != nil {
		return err
	}
	users, err := NewDatabaseUser().ListByServerId(&gormx.Contextx{}, server.ID)
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
		allUsers, err := mysql.Users()
		if err != nil {
			return err
		}
		for user := range slices.Values(allUsers) {
			if !slices.ContainsFunc(users, func(a *model.DatabaseUser) bool {
				return a.Username == user.User && a.Host == user.Host
			}) && !slices.Contains([]string{"root", "mysql.sys", "mysql.session", "mysql.infoschema"}, user.User) {
				newUser := &model.DatabaseUser{
					ServerID: id,
					Username: user.User,
					Host:     user.Host,
					Remark:   "sync from server " + server.Name,
				}
				if err = repo.NewDatabaseUser().Create(newUser); err != nil {
					slog.Warn("sync user failed", slog.String("user", user.User), slog.String("host", user.Host), slog.String("error", err.Error()))
				}
			}
		}
	case "sqlite":
		// SQLite 是单文件，不需要像 MySQL/PG 那样拉取所有内部用户列表
		return nil
	case model.DatabaseTypePostgresql:
		postgres, err := db.NewPostgres(server.Username, server.Password, server.Host, server.Port)
		if err != nil {
			return err
		}
		defer func(postgres *db.Postgres) {
			_ = postgres.Close()
		}(postgres)
		allUsers, err := postgres.Users()
		if err != nil {
			return err
		}
		for user := range slices.Values(allUsers) {
			if !slices.ContainsFunc(users, func(a *model.DatabaseUser) bool {
				return a.Username == user.Role
			}) && !slices.Contains([]string{"postgres"}, user.Role) {
				newUser := &model.DatabaseUser{
					ServerID: id,
					Username: user.Role,
					Remark:   "sync from server " + server.Name,
				}
				repo.NewDatabaseUser().Create(newUser)
			}
		}
	}

	return nil
}
