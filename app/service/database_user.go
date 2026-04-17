package service

import (
	"errors"
	"fmt"
	"slices"

	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/init/db"
	"github.com/aihop/gopanel/pkg/gormx"
)

func NewDatabaseUser() *DatabaseUserService {
	return &DatabaseUserService{
		repo: repo.NewDatabaseUser(),
	}
}

type DatabaseUserService struct {
	repo *repo.DatabaseUserRepo
}

func (s *DatabaseUserService) Create(req *request.DatabaseUserCreate) error {
	server, err := NewDatabaseServer().Get(req.ServerID)
	if err != nil {
		return err
	}

	user := new(model.DatabaseUser)
	switch server.Type {
	case model.DatabaseTypeMysql:
		mysql, err := db.NewMySQL(server.Username, server.Password, fmt.Sprintf("%s:%d", server.Host, server.Port))
		if err != nil {
			return err
		}
		defer func(mysql *db.MySQL) {
			_ = mysql.Close()
		}(mysql)
		if err = mysql.UserCreate(req.Username, req.Password, req.Host); err != nil {
			return err
		}
		for name := range slices.Values(req.Privileges) {
			if err = mysql.DatabaseCreate(name); err != nil {
				return err
			}
			if err = mysql.PrivilegesGrant(req.Username, name, req.Host); err != nil {
				return err
			}
		}
		user = &model.DatabaseUser{
			ServerID: req.ServerID,
			Username: req.Username,
			Host:     req.Host,
		}
	case model.DatabaseTypePostgresql:
		postgres, err := db.NewPostgres(server.Username, server.Password, server.Host, server.Port)
		if err != nil {
			return err
		}
		defer func(postgres *db.Postgres) {
			_ = postgres.Close()
		}(postgres)
		if err = postgres.UserCreate(req.Username, req.Password); err != nil {
			return err
		}
		for name := range slices.Values(req.Privileges) {
			if err = postgres.DatabaseCreate(name); err != nil {
				return err
			}
			if err = postgres.PrivilegesGrant(req.Username, name); err != nil {
				return err
			}
		}
		user = &model.DatabaseUser{
			ServerID: req.ServerID,
			Username: req.Username,
		}
	}

	if err := s.repo.FirstOrInit(user, user); err != nil {
		return err
	}
	user.Password = req.Password
	user.Remark = req.Remark
	return s.repo.Save(user)
}

func (s *DatabaseUserService) Update(req *request.DatabaseUserUpdate) error {
	var (
		user *model.DatabaseUser
		err  error
	)

	if req.ID > 0 {
		user, err = s.Get(req.ID)
		if err != nil {
			return err
		}
	} else {
		if req.ServerID == 0 || req.Username == "" {
			return errors.New("id or (serverId and username) is required")
		}
		user = &model.DatabaseUser{
			ServerID: req.ServerID,
			Username: req.Username,
			Host:     req.Host,
		}
		if err = s.repo.FirstOrInit(user, user); err != nil {
			return err
		}
	}

	server, err := NewDatabaseServer().Get(user.ServerID)
	if err != nil {
		return err
	}

	targetPrivileges := uniqueStrings(req.Privileges)

	switch server.Type {
	case model.DatabaseTypeMysql:
		mysql, err := db.NewMySQL(server.Username, server.Password, fmt.Sprintf("%s:%d", server.Host, server.Port))
		if err != nil {
			return err
		}
		defer func(mysql *db.MySQL) {
			_ = mysql.Close()
		}(mysql)

		if user.Host == "" {
			user.Host = req.Host
		}
		if user.Host == "" {
			user.Host = "localhost"
		}

		if req.Password != "" {
			if err = mysql.UserPassword(user.Username, req.Password, user.Host); err != nil {
				return err
			}
		}

		currentPrivileges, err := mysql.UserPrivileges(user.Username, user.Host)
		if err != nil {
			currentPrivileges = []string{}
		}
		for _, name := range currentPrivileges {
			if !slices.Contains(targetPrivileges, name) {
				if err = mysql.PrivilegesRevoke(user.Username, name, user.Host); err != nil {
					return err
				}
			}
		}
		for _, name := range targetPrivileges {
			if err = mysql.DatabaseCreate(name); err != nil {
				return err
			}
			if !slices.Contains(currentPrivileges, name) {
				if err = mysql.PrivilegesGrant(user.Username, name, user.Host); err != nil {
					return err
				}
			}
		}
	case model.DatabaseTypePostgresql:
		postgres, err := db.NewPostgres(server.Username, server.Password, server.Host, server.Port)
		if err != nil {
			return err
		}
		defer func(postgres *db.Postgres) {
			_ = postgres.Close()
		}(postgres)
		if req.Password != "" {
			if err = postgres.UserPassword(user.Username, req.Password); err != nil {
				return err
			}
		}

		currentPrivileges, err := postgres.UserPrivileges(user.Username)
		if err != nil {
			currentPrivileges = []string{}
		}
		for _, name := range currentPrivileges {
			if !slices.Contains(targetPrivileges, name) {
				if err = postgres.PrivilegesRevoke(user.Username, name); err != nil {
					return err
				}
			}
		}
		for _, name := range targetPrivileges {
			if err = postgres.DatabaseCreate(name); err != nil {
				return err
			}
			if !slices.Contains(currentPrivileges, name) {
				if err = postgres.PrivilegesGrant(user.Username, name); err != nil {
					return err
				}
			}
		}
	default:
		return errors.New("unsupported database server type")
	}

	if req.Password != "" {
		user.Password = req.Password
	}
	user.Remark = req.Remark
	if req.Host != "" {
		user.Host = req.Host
	}

	return s.repo.Save(user)
}

func (s DatabaseUserService) Get(id uint) (res *model.DatabaseUser, err error) {
	if res, err = s.repo.Get(id); err != nil {
		return nil, err
	}
	s.fillUser(res)
	return res, nil
}

func (s DatabaseUserService) GetByIdentity(serverID uint, username, host string) (res *model.DatabaseUser, err error) {
	if serverID == 0 || username == "" {
		return nil, errors.New("serverId and username is required")
	}
	res = &model.DatabaseUser{
		ServerID: serverID,
		Username: username,
		Host:     host,
	}
	if err = s.repo.FirstOrInit(res, res); err != nil {
		return nil, err
	}
	if server, err := NewDatabaseServer().Get(serverID); err == nil {
		res.Server = server
		if server.Type == model.DatabaseTypeMysql && res.Host == "" {
			res.Host = "localhost"
		}
		if server.Type == model.DatabaseTypePostgresql {
			res.Host = ""
		}
	}
	s.fillUser(res)
	return res, nil
}

func (r DatabaseUserService) fillUser(user *model.DatabaseUser) {
	server, err := NewDatabaseServer().Get(user.ServerID)
	if err == nil {
		switch server.Type {
		case model.DatabaseTypeMysql:
			mysql, err := db.NewMySQL(server.Username, server.Password, fmt.Sprintf("%s:%d", server.Host, server.Port))
			if err == nil {
				defer func(mysql *db.MySQL) {
					_ = mysql.Close()
				}(mysql)
				privileges, _ := mysql.UserPrivileges(user.Username, user.Host)
				user.Privileges = privileges
			}
			if mysql2, err := db.NewMySQL(user.Username, user.Password, fmt.Sprintf("%s:%d", server.Host, server.Port)); err == nil {
				_ = mysql2.Close()
				user.Status = model.DatabaseUserStatusValid
			} else {
				user.Status = model.DatabaseUserStatusInvalid
			}
		case model.DatabaseTypePostgresql:
			postgres, err := db.NewPostgres(server.Username, server.Password, server.Host, server.Port)
			if err == nil {
				defer func(postgres *db.Postgres) {
					_ = postgres.Close()
				}(postgres)
				privileges, _ := postgres.UserPrivileges(user.Username)
				user.Privileges = privileges
			}
			if postgres2, err := db.NewPostgres(user.Username, user.Password, server.Host, server.Port); err == nil {
				_ = postgres2.Close()
				user.Status = model.DatabaseUserStatusValid
			} else {
				user.Status = model.DatabaseUserStatusInvalid
			}
		}
	}
	// 初始化，防止 nil
	if user.Privileges == nil {
		user.Privileges = make([]string, 0)
	}
}

func (s DatabaseUserService) Delete(id uint) error {
	if _, err := s.Get(id); err != nil {
		return err
	}
	return s.repo.Delete(id)
}

func (s *DatabaseUserService) List(ctx *gormx.Contextx) (res []*model.DatabaseUser, err error) {
	res, err = s.repo.List(ctx)
	for u := range slices.Values(res) {
		s.fillUser(u)
	}
	return
}

func (s *DatabaseUserService) ListByServerId(ctx *gormx.Contextx, serverId uint) (res []*model.DatabaseUser, err error) {
	res, err = s.repo.ListByServerId(ctx, serverId)
	if err != nil || len(res) == 0 {
		return nil, errors.New("no database user found")
	}
	return
}

func (s *DatabaseUserService) ClearUsers(serverID uint) (err error) {
	err = s.repo.ClearUsers(serverID)
	return
}

func (s *DatabaseUserService) CountByWhere(where *gormx.Wherex) (res int64, err error) {
	return s.repo.CountByWhere(where)
}
