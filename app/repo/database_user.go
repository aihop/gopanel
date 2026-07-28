package repo

import (
	"fmt"
	"slices"
	"strings"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/init/db"
	"github.com/aihop/gopanel/pkg/gormx"
	"gorm.io/gorm"
)

type DatabaseUserRepo struct {
	db *gorm.DB
}

func NewDatabaseUser() *DatabaseUserRepo {
	return &DatabaseUserRepo{
		db: global.DB,
	}
}

func (r *DatabaseUserRepo) MigrateTable() error {
	if err := r.db.AutoMigrate(&model.DatabaseUser{}); err != nil {
		return err
	}
	var users []model.DatabaseUser
	if err := r.db.Find(&users).Error; err != nil {
		return err
	}
	for _, user := range users {
		password, err := encryptDatabaseSecret(user.Password)
		if err != nil {
			return err
		}
		if password != user.Password {
			if err := r.db.Model(&model.DatabaseUser{}).Where("id = ?", user.ID).Update("password", password).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *DatabaseUserRepo) Create(item *model.DatabaseUser) (err error) {
	stored := *item
	stored.Password, err = encryptDatabaseSecret(item.Password)
	if err != nil {
		return err
	}
	if err = r.db.Model(&model.DatabaseUser{}).Create(&stored).Error; err == nil {
		item.ID = stored.ID
	}
	return err
}

func (r *DatabaseUserRepo) Update(item *model.DatabaseUser) (err error) {
	if item.ID == 0 {
		return gorm.ErrMissingWhereClause
	}
	stored := *item
	stored.Password, err = encryptDatabaseSecret(item.Password)
	if err != nil {
		return err
	}
	return r.db.Model(&model.DatabaseUser{}).Where("id = ?", item.ID).Updates(&stored).Error
}

func (r *DatabaseUserRepo) Get(id uint) (res *model.DatabaseUser, err error) {
	err = r.db.Model(&model.DatabaseUser{}).Preload("Server").Where("id = ?", id).First(&res).Error
	if err == nil {
		res.Password, err = decryptDatabaseSecret(res.Password)
	}
	if err == nil && res.Server != nil {
		res.Server.Password, err = decryptDatabaseSecret(res.Server.Password)
	}
	return
}

func (r *DatabaseUserRepo) Delete(id uint) (err error) {
	err = r.db.Delete(&model.DatabaseUser{}, id).Error
	return
}

func (r *DatabaseUserRepo) List(ctx *gormx.Contextx) (res []*model.DatabaseUser, err error) {
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
	for _, server := range databaseServer {
		server.Password, err = decryptDatabaseSecret(server.Password)
		if err != nil {
			return nil, err
		}
	}

	serverIDSet := make(map[uint]struct{})
	databaseUsers := make([]*model.DatabaseUser, 0)
	for _, server := range databaseServer {
		serverIDSet[server.ID] = struct{}{}
		switch server.Type {
		case model.DatabaseTypeMysql:
			mysql, err := db.NewMySQL(server.Username, server.Password, fmt.Sprintf("%s:%d", server.Host, server.Port))
			if err == nil {
				if users, err := mysql.Users(); err == nil {
					for item := range slices.Values(users) {
						databaseUsers = append(databaseUsers, &model.DatabaseUser{
							ServerID: server.ID,
							Username: item.User,
							Host:     item.Host,
							Server:   server,
						})
					}
				}
				_ = mysql.Close()
			}
		case model.DatabaseTypePostgresql:
			postgres, err := db.NewPostgres(server.Username, server.Password, server.Host, server.Port)
			if err == nil {
				if users, err := postgres.Users(); err == nil {
					for item := range slices.Values(users) {
						databaseUsers = append(databaseUsers, &model.DatabaseUser{
							ServerID: server.ID,
							Username: item.Role,
							Host:     "",
							Server:   server,
						})
					}
				}
				_ = postgres.Close()
			}
		}
	}

	serverIDs := make([]uint, 0, len(serverIDSet))
	for id := range serverIDSet {
		serverIDs = append(serverIDs, id)
	}
	var localUsers []model.DatabaseUser
	if len(serverIDs) > 0 {
		_ = r.db.Model(&model.DatabaseUser{}).Where("server_id IN ?", serverIDs).Find(&localUsers).Error
	}
	localMap := make(map[string]model.DatabaseUser, len(localUsers))
	for _, u := range localUsers {
		u.Password, err = decryptDatabaseSecret(u.Password)
		if err != nil {
			return nil, err
		}
		key := fmt.Sprintf("%d|%s|%s", u.ServerID, strings.ToLower(u.Username), strings.ToLower(u.Host))
		localMap[key] = u
	}
	for _, u := range databaseUsers {
		key := fmt.Sprintf("%d|%s|%s", u.ServerID, strings.ToLower(u.Username), strings.ToLower(u.Host))
		if local, ok := localMap[key]; ok {
			u.ID = local.ID
			u.Password = local.Password
			u.Remark = local.Remark
			u.CreatedAt = local.CreatedAt
			u.UpdatedAt = local.UpdatedAt
		}
	}

	// Pagination
	start := (ctx.Page - 1) * ctx.Limit
	if start >= len(databaseUsers) {
		return []*model.DatabaseUser{}, nil
	}
	end := start + ctx.Limit
	if end > len(databaseUsers) {
		end = len(databaseUsers)
	}

	return databaseUsers[start:end], nil
}

func (r *DatabaseUserRepo) ListByServerId(ctx *gormx.Contextx, serverId uint) (res []*model.DatabaseUser, err error) {
	err = r.db.Model(&model.DatabaseUser{}).Scopes(gormx.Context(ctx)).Where("server_id = ?", serverId).Find(&res).Error
	if err != nil {
		return nil, err
	}
	for _, user := range res {
		user.Password, err = decryptDatabaseSecret(user.Password)
		if err != nil {
			return nil, err
		}
	}
	return
}

func (r DatabaseUserRepo) ClearUsers(serverID uint) error {
	return r.db.Where("server_id = ?", serverID).Delete(&model.DatabaseUser{}).Error
}

func (r *DatabaseUserRepo) FirstOrInit(ins, outs *model.DatabaseUser) (err error) {
	err = r.db.Model(&model.DatabaseUser{}).FirstOrInit(ins, outs).Error
	if err == nil {
		outs.Password, err = decryptDatabaseSecret(outs.Password)
	}
	return
}

func (r *DatabaseUserRepo) Save(item *model.DatabaseUser) (err error) {
	if item == nil {
		return nil
	}
	if item.ID == 0 {
		return r.Create(item)
	}
	return r.Update(item)
}

func (r *DatabaseUserRepo) CountByWhere(where *gormx.Wherex) (res int64, err error) {
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
	for _, server := range databaseServer {
		server.Password, err = decryptDatabaseSecret(server.Password)
		if err != nil {
			return 0, err
		}
	}

	for _, server := range databaseServer {
		switch server.Type {
		case model.DatabaseTypeMysql:
			mysql, err := db.NewMySQL(server.Username, server.Password, fmt.Sprintf("%s:%d", server.Host, server.Port))
			if err == nil {
				if users, err := mysql.Users(); err == nil {
					res += int64(len(users))
				}
				_ = mysql.Close()
			}
		case model.DatabaseTypePostgresql:
			postgres, err := db.NewPostgres(server.Username, server.Password, server.Host, server.Port)
			if err == nil {
				if users, err := postgres.Users(); err == nil {
					res += int64(len(users))
				}
				_ = postgres.Close()
			}
		}
	}
	return res, nil
}
