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
	return r.db.AutoMigrate(&model.DatabaseUser{})
}

func (r *DatabaseUserRepo) Create(item *model.DatabaseUser) (err error) {
	return r.db.Model(&model.DatabaseUser{}).Create(item).Error
}

func (r *DatabaseUserRepo) Update(item *model.DatabaseUser) (err error) {
	if item.ID == 0 {
		return gorm.ErrMissingWhereClause
	}
	return r.db.Model(&model.DatabaseUser{}).Where("id = ?", item.ID).Updates(item).Error
}

func (r *DatabaseUserRepo) Get(id uint) (res *model.DatabaseUser, err error) {
	err = r.db.Model(&model.DatabaseUser{}).Preload("Server").Where("id = ?", id).First(&res).Error
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
	return
}

func (r DatabaseUserRepo) ClearUsers(serverID uint) error {
	return r.db.Where("server_id = ?", serverID).Delete(&model.DatabaseUser{}).Error
}

func (r *DatabaseUserRepo) FirstOrInit(ins, outs *model.DatabaseUser) (err error) {
	err = r.db.Model(&model.DatabaseUser{}).FirstOrInit(ins, outs).Error
	return
}

func (r *DatabaseUserRepo) Save(item *model.DatabaseUser) (err error) {
	err = r.db.Model(&model.DatabaseUser{}).Save(item).Error
	return
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
