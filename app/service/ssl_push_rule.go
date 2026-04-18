package service

import (
	"context"
	"errors"

	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/pkg/gormx"
)

func NewSSLPushRuleService() *SSLPushRuleService {
	return &SSLPushRuleService{
		repo: repo.NewSSLPushRule(),
	}
}

type SSLPushRuleService struct {
	repo *repo.SSLPushRuleRepo
}

func (s *SSLPushRuleService) Search(ctx *gormx.Contextx) (res []*model.SSLPushRule, err error) {
	return s.repo.Search(ctx)
}

func (s *SSLPushRuleService) Count(ctx *gormx.Wherex) (res int64, err error) {
	return s.repo.CountByWhere(ctx)
}

func (s *SSLPushRuleService) Create(req *request.SSLPushRuleCreate) error {
	item := &model.SSLPushRule{
		SSLID:          req.SSLID,
		CloudAccountID: req.CloudAccountID,
		TargetDomain:   req.TargetDomain,
		Status:         "pending",
	}
	return s.repo.Create(context.Background(), item)
}

func (s *SSLPushRuleService) Update(req *request.SSLPushRuleUpdate) error {
	item, err := s.repo.GetFirst(s.repo.WithID(req.ID))
	if err != nil {
		return errors.New("推送规则不存在")
	}

	item.CloudAccountID = req.CloudAccountID
	item.TargetDomain = req.TargetDomain

	return s.repo.Save(context.Background(), &item)
}

func (s *SSLPushRuleService) Delete(id uint) error {
	return s.repo.DeleteBy(context.Background(), s.repo.WithID(id))
}
