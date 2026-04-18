package repo

import (
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"

	"gorm.io/gorm"
)

type ImageRepoRepo struct {
	DB *gorm.DB
}

type IImageRepoRepo interface {
	Get(opts ...DBOption) (model.ImageRepo, error)
	Page(limit, offset int, opts ...DBOption) (int64, []model.ImageRepo, error)
	List(opts ...DBOption) ([]model.ImageRepo, error)
	Create(imageRepo *model.ImageRepo) error
	Update(id uint, vars map[string]interface{}) error
	Delete(opts ...DBOption) error
}

func NewIImageRepoRepo() IImageRepoRepo {
	return &ImageRepoRepo{}
}

func NewImageRepo(db *gorm.DB) *ImageRepoRepo {
	if db == nil {
		db = global.DB
	}
	return &ImageRepoRepo{
		DB: db,
	}
}

func (r *ImageRepoRepo) WithTx(tx *gorm.DB) *ImageRepoRepo {
	if tx == nil {
		tx = global.DB
	}
	r.DB = tx
	return r
}

func (r *ImageRepoRepo) MigrateTable() error {
	if !r.DB.Migrator().HasTable(&model.ImageRepo{}) {
		r.DB.AutoMigrate(&model.ImageRepo{})
		return r.Create(&model.ImageRepo{
			Name:        "Docker Hub",
			DownloadUrl: "docker.io",
			Protocol:    "https",
			Auth:        false,
			Status:      "Success",
		})
	}
	return r.DB.AutoMigrate(&model.ImageRepo{})
}

func (u *ImageRepoRepo) Get(opts ...DBOption) (model.ImageRepo, error) {
	var imageRepo model.ImageRepo
	db := global.DB
	for _, opt := range opts {
		db = opt(db)
	}
	err := db.First(&imageRepo).Error
	return imageRepo, err
}

func (u *ImageRepoRepo) Page(page, size int, opts ...DBOption) (int64, []model.ImageRepo, error) {
	var ops []model.ImageRepo
	db := global.DB.Model(&model.ImageRepo{})
	for _, opt := range opts {
		db = opt(db)
	}
	count := int64(0)
	db = db.Count(&count)
	err := db.Limit(size).Offset(size * (page - 1)).Find(&ops).Error
	return count, ops, err
}

func (u *ImageRepoRepo) List(opts ...DBOption) ([]model.ImageRepo, error) {
	var ops []model.ImageRepo
	db := global.DB.Model(&model.ImageRepo{})
	for _, opt := range opts {
		db = opt(db)
	}
	count := int64(0)
	db = db.Count(&count)
	err := db.Find(&ops).Error
	return ops, err
}

func (u *ImageRepoRepo) Create(imageRepo *model.ImageRepo) error {
	return global.DB.Create(imageRepo).Error
}

func (u *ImageRepoRepo) Update(id uint, vars map[string]interface{}) error {
	return global.DB.Model(&model.ImageRepo{}).Where("id = ?", id).Updates(vars).Error
}

func (u *ImageRepoRepo) Delete(opts ...DBOption) error {
	db := global.DB
	for _, opt := range opts {
		db = opt(db)
	}
	return db.Delete(&model.ImageRepo{}).Error
}
