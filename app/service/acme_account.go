package service

import (
	"context"
	"strings"

	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/pkg/gormx"
)

type AcmeAccountService struct {
	repo *repo.AcmeAccountRepo
}

func NewAcmeAccount() *AcmeAccountService {
	return &AcmeAccountService{repo: repo.NewAcmeAccount()}
}

func (s *AcmeAccountService) Create(req *request.AcmeAccountCreate) error {
	account := &model.AcmeAccount{
		Email: strings.TrimSpace(req.Email),
		URL:   strings.TrimSpace(req.URL),
		Type:  strings.TrimSpace(req.Type),
	}
	if account.Email == "" {
		return nil
	}
	if account.Type == "" {
		account.Type = "letsencrypt"
	}
	return s.repo.Create(context.Background(), account)
}

func (s *AcmeAccountService) Delete(id uint) error {
	return s.repo.DeleteByID(context.Background(), id)
}

func (s *AcmeAccountService) List(ctx *gormx.Contextx) (res []*model.AcmeAccount, err error) {
	items, err := s.repo.List(ctx.Page, ctx.Limit)
	if err != nil {
		return nil, err
	}
	res = make([]*model.AcmeAccount, 0, len(items))
	for i := range items {
		item := items[i]
		res = append(res, &item)
	}
	return
}
