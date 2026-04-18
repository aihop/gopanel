package repo

import (
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
	return r.db.AutoMigrate(&model.DatabaseServer{})
}

func (r *DatabaseServerRepo) Create(item *model.DatabaseServer) (err error) {
	return r.db.Model(&model.DatabaseServer{}).Create(item).Error
}

func (r *DatabaseServerRepo) Update(item *model.DatabaseServer) (err error) {
	if item.ID == 0 {
		return gorm.ErrMissingWhereClause
	}
	return r.db.Model(&model.DatabaseServer{}).Where("id = ?", item.ID).Updates(item).Error
}

func (r *DatabaseServerRepo) Get(id uint) (res *model.DatabaseServer, err error) {
	err = r.db.Model(&model.DatabaseServer{}).Where("id = ?", id).First(&res).Error
	return
}

func (r *DatabaseServerRepo) GetByNameType(name string, types model.DatabaseType) (res model.DatabaseServer, err error) {
	err = r.db.Model(&model.DatabaseServer{}).Where("name = ? AND type = ?", name, types).First(&res).Error
	return
}

func (r *DatabaseServerRepo) Delete(id uint) (err error) {
	err = r.db.Delete(&model.DatabaseServer{}, id).Error
	return
}

func (r *DatabaseServerRepo) List(ctx *gormx.Contextx) (res []*model.DatabaseServer, err error) {
	err = r.db.Model(&model.DatabaseServer{}).Scopes(gormx.Context(ctx)).Find(&res).Error
	return
}

func (r *DatabaseServerRepo) CountByWhere(where *gormx.Wherex) (res int64, err error) {
	err = r.db.Model(&model.DatabaseServer{}).Scopes(gormx.Wheres(where)).Count(&res).Error
	return
}
