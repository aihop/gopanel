package service

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/buserr"
	"github.com/aihop/gopanel/constant"
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
	normalizedHost := normalizeDatabaseUserHost(server.Type, req.Host)

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
		if err = mysql.UserCreate(req.Username, req.Password, normalizedHost); err != nil {
			return err
		}
		for name := range slices.Values(req.Privileges) {
			if err = mysql.DatabaseCreate(name); err != nil {
				return err
			}
			if err = mysql.PrivilegesGrant(req.Username, name, normalizedHost); err != nil {
				return err
			}
		}
		user = &model.DatabaseUser{
			ServerID: req.ServerID,
			Username: req.Username,
			Host:     normalizedHost,
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
			Host:     normalizedHost,
		}
	}

	if err := s.repo.FirstOrInit(user, user); err != nil {
		return err
	}
	user.Password = req.Password
	user.Remark = req.Remark
	return s.repo.Save(user)
}

func normalizeDatabaseUserHost(serverType model.DatabaseType, host string) string {
	host = strings.TrimSpace(host)
	if serverType == model.DatabaseTypeMysql {
		if host == "" {
			// 容器化场景下应用多从其它容器经 TCP 连入，来源是容器网段 IP，
			// 绑定 localhost 会匹配不上导致 Access denied，故默认放行任意主机。
			// 安全性依赖不对公网 publish 数据库端口 + 强密码 + 库级授权。
			return "%"
		}
		return host
	}
	return ""
}

func databaseUserAccessScope(serverType model.DatabaseType, host string) model.DatabaseUserAccessScope {
	if serverType != model.DatabaseTypeMysql {
		return model.DatabaseUserAccessScopeUnknown
	}
	switch strings.ToLower(strings.TrimSpace(host)) {
	case "%":
		return model.DatabaseUserAccessScopeAll
	case "localhost", "127.0.0.1", "::1":
		return model.DatabaseUserAccessScopeLocal
	case "":
		return model.DatabaseUserAccessScopeUnknown
	default:
		return model.DatabaseUserAccessScopeSpecific
	}
}

func firstNonEmptyNonBlank(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func uniqueDatabaseNames(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
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
	user.Host = normalizeDatabaseUserHost(server.Type, firstNonEmptyNonBlank(req.Host, user.Host))
	targetPrivileges := uniqueDatabaseNames(req.Privileges)

	switch server.Type {
	case model.DatabaseTypeMysql:
		mysql, err := db.NewMySQL(server.Username, server.Password, fmt.Sprintf("%s:%d", server.Host, server.Port))
		if err != nil {
			return err
		}
		defer func(mysql *db.MySQL) {
			_ = mysql.Close()
		}(mysql)

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

func (s *DatabaseUserService) GetStoredPassword(req *request.DatabaseUserGet) (string, bool, error) {
	var user *model.DatabaseUser
	var err error
	if req.ID > 0 {
		user, err = s.repo.Get(req.ID)
	} else {
		if req.ServerID == 0 || strings.TrimSpace(req.Username) == "" {
			return "", false, buserr.New(constant.ErrDatabaseUserIdentityRequired)
		}
		user = &model.DatabaseUser{ServerID: req.ServerID, Username: req.Username, Host: req.Host}
		err = s.repo.FirstOrInit(user, user)
	}
	if err != nil {
		return "", false, err
	}
	password := user.Password
	return password, password != "", nil
}

func (r DatabaseUserService) fillUser(user *model.DatabaseUser) {
	server, err := NewDatabaseServer().Get(user.ServerID)
	if err == nil {
		user.AccessScope = databaseUserAccessScope(server.Type, user.Host)
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
			if user.Password == "" {
				user.Status = model.DatabaseUserStatusUnknown
			} else if mysql2, err := db.NewMySQL(user.Username, user.Password, fmt.Sprintf("%s:%d", server.Host, server.Port)); err == nil {
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
			if user.Password == "" {
				user.Status = model.DatabaseUserStatusUnknown
			} else if postgres2, err := db.NewPostgres(user.Username, user.Password, server.Host, server.Port); err == nil {
				_ = postgres2.Close()
				user.Status = model.DatabaseUserStatusValid
			} else {
				user.Status = model.DatabaseUserStatusInvalid
			}
		}
	}
	user.PasswordManaged = user.Password != ""
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
