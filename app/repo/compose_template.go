package repo

import (
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
	"gorm.io/gorm"
)

type ComposeTemplateRepo struct {
	DB *gorm.DB
}

type IComposeTemplateRepo interface {
	Get(opts ...DBOption) (model.ComposeTemplate, error)
	List(opts ...DBOption) ([]model.ComposeTemplate, error)
	Page(limit, offset int, opts ...DBOption) (int64, []model.ComposeTemplate, error)
	Create(compose *model.ComposeTemplate) error
	Update(id uint, vars map[string]interface{}) error
	Delete(opts ...DBOption) error

	GetRecord(opts ...DBOption) (model.Compose, error)
	CreateRecord(compose *model.Compose) error
	DeleteRecord(opts ...DBOption) error
	ListRecord() ([]model.Compose, error)
	UpdateRecord(name string, vars map[string]interface{}) error

	WithName(name string) DBOption
}

func NewIComposeTemplateRepo() IComposeTemplateRepo {
	return &ComposeTemplateRepo{}
}

func NewComposeTemplate(db *gorm.DB) *ComposeTemplateRepo {
	return &ComposeTemplateRepo{DB: db}
}

func NewCompose(db *gorm.DB) *ComposeTemplateRepo {
	return &ComposeTemplateRepo{DB: db}
}

func (r *ComposeTemplateRepo) WithTx(tx *gorm.DB) *ComposeTemplateRepo {
	if tx == nil {
		tx = global.DB
	}
	r.DB = tx
	return r
}

func (r *ComposeTemplateRepo) MigrateTable() error {
	return r.DB.AutoMigrate(&model.ComposeTemplate{})
}

func (r *ComposeTemplateRepo) MigrateTableCompose() error {
	return r.DB.AutoMigrate(&model.Compose{})
}

func (u *ComposeTemplateRepo) Get(opts ...DBOption) (model.ComposeTemplate, error) {
	var compose model.ComposeTemplate
	db := global.DB
	for _, opt := range opts {
		db = opt(db)
	}
	err := db.First(&compose).Error
	return compose, err
}

func (u *ComposeTemplateRepo) Page(page, size int, opts ...DBOption) (int64, []model.ComposeTemplate, error) {
	var users []model.ComposeTemplate
	db := global.DB.Model(&model.ComposeTemplate{})
	for _, opt := range opts {
		db = opt(db)
	}
	count := int64(0)
	db = db.Count(&count)
	err := db.Limit(size).Offset(size * (page - 1)).Find(&users).Error
	return count, users, err
}

func (u *ComposeTemplateRepo) List(opts ...DBOption) ([]model.ComposeTemplate, error) {
	var composes []model.ComposeTemplate
	db := global.DB.Model(&model.ComposeTemplate{})
	for _, opt := range opts {
		db = opt(db)
	}
	err := db.Find(&composes).Error
	return composes, err
}

func (u *ComposeTemplateRepo) Create(compose *model.ComposeTemplate) error {
	return global.DB.Create(compose).Error
}

func (u *ComposeTemplateRepo) Update(id uint, vars map[string]interface{}) error {
	return global.DB.Model(&model.ComposeTemplate{}).Where("id = ?", id).Updates(vars).Error
}

func (u *ComposeTemplateRepo) Delete(opts ...DBOption) error {
	db := global.DB
	for _, opt := range opts {
		db = opt(db)
	}
	return db.Delete(&model.ComposeTemplate{}).Error
}

func (u *ComposeTemplateRepo) GetRecord(opts ...DBOption) (model.Compose, error) {
	var compose model.Compose
	db := global.DB
	for _, opt := range opts {
		db = opt(db)
	}
	err := db.First(&compose).Error
	return compose, err
}

func (u *ComposeTemplateRepo) ListRecord() ([]model.Compose, error) {
	var composes []model.Compose
	if err := global.DB.Find(&composes).Error; err != nil {
		return nil, err
	}
	return composes, nil
}

func (u *ComposeTemplateRepo) CreateRecord(compose *model.Compose) error {
	return global.DB.Create(compose).Error
}

func (u *ComposeTemplateRepo) DeleteRecord(opts ...DBOption) error {
	db := global.DB
	for _, opt := range opts {
		db = opt(db)
	}
	return db.Delete(&model.Compose{}).Error
}

func (u *ComposeTemplateRepo) UpdateRecord(name string, vars map[string]interface{}) error {
	return global.DB.Model(&model.Compose{}).Where("name = ?", name).Updates(vars).Error
}

func (a *ComposeTemplateRepo) WithName(name string) DBOption {
	return func(g *gorm.DB) *gorm.DB {
		return g.Where("name = ?", name)
	}
}
