package repo

import (
	"strings"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/pkg/gormx"
	"gorm.io/gorm"
)

type DatabaseServerRepo struct {
	db *gorm.DB
}

func NewDatabaseServer() *DatabaseServerRepo {
	return &DatabaseServerRepo{
		db: global.DB,
	}
}

func (r *DatabaseServerRepo) MigrateTable() error {
	if err := r.db.AutoMigrate(&model.DatabaseServer{}); err != nil {
		return err
	}
	var servers []model.DatabaseServer
	if err := r.db.Find(&servers).Error; err != nil {
		return err
	}
	for _, server := range servers {
		password, err := encryptDatabaseSecret(server.Password)
		if err != nil {
			return err
		}
		if password != server.Password {
			if err := r.db.Model(&model.DatabaseServer{}).Where("id = ?", server.ID).Update("password", password).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *DatabaseServerRepo) Create(item *model.DatabaseServer) (err error) {
	stored := *item
	stored.Password, err = encryptDatabaseSecret(item.Password)
	if err != nil {
		return err
	}
	if err = r.db.Model(&model.DatabaseServer{}).Create(&stored).Error; err == nil {
		item.ID = stored.ID
	}
	return err
}

func (r *DatabaseServerRepo) Update(item *model.DatabaseServer) (err error) {
	if item.ID == 0 {
		return gorm.ErrMissingWhereClause
	}
	stored := *item
	stored.Password, err = encryptDatabaseSecret(item.Password)
	if err != nil {
		return err
	}
	return r.db.Model(&model.DatabaseServer{}).Where("id = ?", item.ID).Updates(&stored).Error
}

func (r *DatabaseServerRepo) Get(id uint) (res *model.DatabaseServer, err error) {
	err = r.db.Model(&model.DatabaseServer{}).Where("id = ?", id).First(&res).Error
	if err == nil {
		res.Password, err = decryptDatabaseSecret(res.Password)
	}
	return
}

func (r *DatabaseServerRepo) GetByNameType(name string, types model.DatabaseType) (res model.DatabaseServer, err error) {
	err = r.db.Model(&model.DatabaseServer{}).Where("name = ? AND type = ?", name, types).First(&res).Error
	if err == nil {
		res.Password, err = decryptDatabaseSecret(res.Password)
	}
	return
}

func (r *DatabaseServerRepo) GetByName(name string) (res model.DatabaseServer, err error) {
	err = r.db.Model(&model.DatabaseServer{}).Where("name = ?", name).First(&res).Error
	if err == nil {
		res.Password, err = decryptDatabaseSecret(res.Password)
	}
	return
}

func (r *DatabaseServerRepo) GetByEndpoint(types model.DatabaseType, host string, port uint) (res model.DatabaseServer, err error) {
	databaseTypes := []model.DatabaseType{types}
	if types == model.DatabaseTypeMysql || types == model.DatabaseTypeMariaDB {
		databaseTypes = []model.DatabaseType{model.DatabaseTypeMysql, model.DatabaseTypeMariaDB}
	}
	hosts := []string{host}
	if isLocalDatabaseEndpointHost(host) {
		hosts = []string{"127.0.0.1", "localhost", "0.0.0.0", "::", "::1"}
	}
	err = r.db.Model(&model.DatabaseServer{}).Where("type IN ? AND host IN ? AND port = ?", databaseTypes, hosts, port).First(&res).Error
	if err == nil {
		res.Password, err = decryptDatabaseSecret(res.Password)
	}
	return
}

func isLocalDatabaseEndpointHost(host string) bool {
	switch strings.ToLower(strings.TrimSpace(host)) {
	case "127.0.0.1", "localhost", "0.0.0.0", "::", "::1":
		return true
	default:
		return false
	}
}

func (r *DatabaseServerRepo) Delete(id uint) (err error) {
	err = r.db.Delete(&model.DatabaseServer{}, id).Error
	return
}

func (r *DatabaseServerRepo) List(ctx *gormx.Contextx) (res []*model.DatabaseServer, err error) {
	err = r.db.Model(&model.DatabaseServer{}).Scopes(gormx.Context(ctx)).Find(&res).Error
	if err != nil {
		return nil, err
	}
	for _, server := range res {
		server.Password, err = decryptDatabaseSecret(server.Password)
		if err != nil {
			return nil, err
		}
	}
	return
}

func (r *DatabaseServerRepo) CountByWhere(where *gormx.Wherex) (res int64, err error) {
	err = r.db.Model(&model.DatabaseServer{}).Scopes(gormx.Wheres(where)).Count(&res).Error
	return
}
