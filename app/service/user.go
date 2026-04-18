package service

import (
	"errors"
	"sync"

	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/pkg/gormx"
	"github.com/aihop/gopanel/utils/cryptx"
	"gorm.io/gorm"
)

func NewUser() *UserService {
	return &UserService{
		repo: repo.NewUser(global.DB),
		db:   global.DB,
	}
}

type UserService struct {
	tx    *gorm.DB
	cache bool
	mu    sync.Mutex
	repo  *repo.UserRepo
	db    *gorm.DB
}

func (s *UserService) WithTx(tx *gorm.DB) *UserService {
	s.tx = tx
	return s
}

func (s *UserService) Get(id uint) (res *model.User, err error) {
	if id == 0 {
		return nil, errors.New(constant.ErrIdRequired)
	}
	if res, err = s.repo.Get(id); err != nil {
		return
	}
	if res.ID == 0 {
		return nil, errors.New(constant.ErrRecordNotFound)
	}

	return
}

func (s *UserService) GetByAdminEmail() (email string) {
	if res, err := s.repo.GetEmail(); err == nil {
		email = res.Email
	}
	return
}

func (s *UserService) GetByToken(token string) (res *model.User, err error) {
	if token == "" {
		return res, errors.New("token is required")
	}
	if res, err = s.repo.GetByToken(token); err != nil {
		return
	}
	if res.ID == 0 {
		return res, errors.New(constant.ErrRecordNotFound)
	}
	return
}

func (s *UserService) GetByEmail(email string) (res *model.User, err error) {
	if res, err = s.repo.GetByEmail(email); err != nil {
		return
	}
	if res.ID == 0 {
		return res, errors.New(constant.ErrRecordNotFound)
	}
	return
}

func (s *UserService) GetByMobile(mobile string) (res *model.User, err error) {
	if res, err = s.repo.GetByMobile(mobile); err != nil {
		return
	}
	if res.ID == 0 {
		return res, errors.New(constant.ErrRecordNotFound)
	}
	return
}

// 封装事务处理逻辑
func (s *UserService) withTransaction(fn func(tx *gorm.DB) error) error {
	txLocal := false
	var tx *gorm.DB
	// 如果当前没事务，则创建局部tx变量，不改变 s.tx
	if s.tx == nil {
		txLocal = true
		tx = global.DB.Begin()
	} else {
		// 已有事务则直接使用
		tx = s.tx
	}
	err := fn(tx)
	if err != nil {
		if txLocal {
			tx.Rollback()
		}
		// 若是外部事务，则不清空局部的 tx
		if txLocal {
			s.tx = nil
		}
		return err
	}
	if txLocal {
		if err = tx.Commit().Error; err != nil {
			s.tx = nil
			return err
		}
		s.tx = nil
	}
	return nil
}

func (s *UserService) Create(req request.UserCreate) error {
	user := &model.User{
		NickName:    req.NickName,
		Email:       req.Email,
		Password:    cryptx.EncodePassword(req.Password),
		Role:        req.Role,
		FileBaseDir: req.FileBaseDir,
		Menus:       req.Menus,
		Status:      constant.UserStatusNormal,
	}
	return s.repo.Create(user)
}

func (s *UserService) UpdateUser(req request.UserUpdate) error {
	user, err := s.repo.Get(req.ID)
	if err != nil {
		return err
	}
	if req.NickName != "" {
		user.NickName = req.NickName
	}
	if req.Email != "" {
		user.Email = req.Email
	}
	if req.Password != "" {
		user.Password = cryptx.EncodePassword(req.Password)
	}
	if req.Role != "" {
		user.Role = req.Role
	}
	// FileBaseDir 可以为空字符串，表示清除限制
	user.FileBaseDir = req.FileBaseDir
	user.Menus = req.Menus

	return s.repo.Update(user)
}

func (s *UserService) Page(req request.UserSearch) (int64, interface{}, error) {
	// Usually admin manages SUB_ADMIN, let's list all users
	count, err := s.repo.CountTotal()
	if err != nil {
		return 0, nil, err
	}

	ctx := gormx.NewContext(req.PageSize, "id desc")
	ctx.Page = req.Page
	list, err := s.repo.List(ctx)
	if err != nil {
		return 0, nil, err
	}

	// 屏蔽密码
	for _, user := range list {
		user.Password = ""
		user.Salt = ""
		user.Token = ""
	}

	return count, list, nil
}

func (s *UserService) CreateInBatches(items []*model.User) (err error) {
	if err = s.repo.WithTx(s.tx).CreateInBatches(items, len(items)); err != nil {
		return err
	}
	return
}

func (s *UserService) Delete(id uint) (err error) {
	if id == 0 {
		return errors.New(constant.ErrIdRequired)
	}

	return s.withTransaction(func(tx *gorm.DB) error {
		return s.repo.WithTx(tx).Delete(id)
	})
}

func (s *UserService) List(ctx *gormx.Contextx) (res []*model.User, err error) {
	res, err = s.repo.List(ctx)
	return
}

func (s *UserService) CountByWhere(where *gormx.Wherex) (res int64, err error) {
	return s.repo.CountByWhere(where)
}

func (s *UserService) ListByIds(ctx *gormx.Contextx, ids []uint) (res []*model.User, err error) {
	if len(ids) == 0 {
		return
	}
	res, err = s.repo.ListByIds(ctx, ids)
	return
}

// 重置账号
func (s *UserService) ResetAccount(id uint, email string, password string) (err error) {
	user := &model.User{
		ID:       id,
		Email:    email,
		NickName: email,
		Password: cryptx.EncodePassword(password),
	}
	if user.ID == 0 {
		return errors.New(constant.ErrIdRequired)
	}
	if user.Email == "" {
		return errors.New("email is required")
	}
	if user.Password == "" {
		return errors.New("password is required")
	}
	s.repo.Update(user)
	return
}

func (s *UserService) Update(user *model.User) error {
	return s.repo.Update(user)
}
