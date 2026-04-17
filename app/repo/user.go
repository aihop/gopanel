package repo

import (
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/init/conf"
	"github.com/aihop/gopanel/utils/cryptx"
	"github.com/duke-git/lancet/v2/random"

	"github.com/aihop/gopanel/pkg/gormx"

	"gorm.io/gorm"
)

func NewUser(db *gorm.DB) *UserRepo {
	return &UserRepo{DB: db}
}

type UserRepo struct {
	DB *gorm.DB
}

func (r *UserRepo) WithTx(tx *gorm.DB) *UserRepo {
	if tx == nil {
		tx = global.DB
	}
	r.DB = tx
	return r
}

func (r *UserRepo) MigrateTable() error {
	if !r.DB.Migrator().HasTable(&model.User{}) {
		r.DB.AutoMigrate(&model.User{})
		if conf.InitInstall.User != "" && conf.InitInstall.Password != "" {
			return r.Create(&model.User{Email: conf.InitInstall.User, Salt: random.RandString(constant.HashDefaultLen), Password: cryptx.EncodePassword(conf.InitInstall.Password), Status: constant.UserStatusNormal, Role: constant.UserRoleSuper, NickName: conf.InitInstall.User, Token: random.RandString(constant.StrLen32)})
		}
		return r.Create(&model.User{Email: "admin", Salt: random.RandString(constant.HashDefaultLen), Password: cryptx.EncodePassword("123456"), Status: constant.UserStatusNormal, Role: constant.UserRoleSuper, NickName: "admin", Token: random.RandString(constant.StrLen32)})
	} else {
		return r.DB.AutoMigrate(&model.User{})
	}
}

func (r *UserRepo) Create(item *model.User) error {
	return r.DB.Create(item).Error
}

func (r *UserRepo) CreateInBatches(items []*model.User, batchSize int) error {
	return r.DB.CreateInBatches(items, batchSize).Error
}

func (r *UserRepo) Update(item *model.User) error {
	return r.DB.Omit("id").Where("id = ?", item.ID).Updates(item).Error
}

func (r *UserRepo) Delete(id uint) error {
	return r.DB.Delete(&model.User{ID: id}).Error
}

func (r *UserRepo) Get(id uint) (res *model.User, err error) {
	err = r.DB.Where("id = ?", id).First(&res).Error
	return
}

func (r *UserRepo) GetByToken(token string) (res *model.User, err error) {
	err = r.DB.Where("token = ?", token).Find(&res).Error
	return
}

func (r *UserRepo) GetEmail() (res *model.User, err error) {
	err = r.DB.Where("email like ?", "%@%").First(&res).Error
	return
}

func (r *UserRepo) GetByEmail(email string, preloads ...string) (user *model.User, err error) {
	db := r.DB.Model(&model.User{})
	if len(preloads) > 0 {
		for _, v := range preloads {
			db = db.Preload(v)
		}
	}
	err = db.Where("email = ?", email).First(&user).Error
	return
}

func (r *UserRepo) GetByMobile(mobile string) (user *model.User, err error) {
	err = r.DB.Model(model.User{}).Where("mobile = ?", mobile).First(&user).Error
	return
}

func (r *UserRepo) GetIdByMobile(mobile string) (id int, err error) {
	err = r.DB.Model(model.User{}).Where("mobile = ?", mobile).Limit(1).Pluck("id", &id).Error
	return
}

func (r *UserRepo) GetIdByEmail(email string) (id int, err error) {
	err = r.DB.Model(model.User{}).Where("email = ?", email).Limit(1).Pluck("id", &id).Error
	return
}

func (r *UserRepo) CountByWhere(where *gormx.Wherex) (res int64, err error) {
	err = r.DB.Model(model.User{}).Scopes(gormx.Wheres(where)).Count(&res).Error
	return
}

func (r *UserRepo) CountTotal() (res int64, err error) {
	err = r.DB.Model(model.User{}).Count(&res).Error
	return
}

func (r *UserRepo) List(ctx *gormx.Contextx) (res []*model.User, err error) {
	err = r.DB.Model(model.User{}).Scopes(gormx.Context(ctx)).Find(&res).Error
	return
}

func (r *UserRepo) ListByIds(ctx *gormx.Contextx, ids []uint) (res []*model.User, err error) {
	err = r.DB.Model(model.User{}).Scopes(gormx.Context(ctx), gormx.WhereIds(ids)).Find(&res).Error
	return
}

func (r *UserRepo) UpdateDecCreditsBalanceById(id, credits uint) error {
	return r.DB.Model(&model.User{}).Where("id = ?", id).UpdateColumn("credits_balance", gorm.Expr("credits_balance - ?", credits)).Error
}
func (r *UserRepo) UpdateIncCreditsBalanceById(id, credits uint) error {
	return r.DB.Model(&model.User{}).Where("id = ?", id).UpdateColumn("credits_balance", gorm.Expr("credits_balance + ?", credits)).Error
}

func (r *UserRepo) UpdateIncPayMoneyById(id uint, payMoney float64) error {
	return r.DB.Model(&model.User{}).Where("id = ?", id).UpdateColumn("pay_money", gorm.Expr("pay_money + ?", payMoney)).Error
}

// 统计开始和结束时间数量
func (r *UserRepo) CountStartEndTime(where *gormx.Wherex, startTime, endTime time.Time) (res int64, err error) {
	err = r.DB.Model(&model.User{}).Scopes(gormx.Wheres(where)).Where("created_at >= ? and created_at < ?", startTime, endTime).Count(&res).Error
	return
}
