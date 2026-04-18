package service

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/pkg/gormx"
)

type CloudAccountService struct {
	repo *repo.CloudAccountRepo
}

func NewCloudAccount() *CloudAccountService {
	return &CloudAccountService{
		repo: repo.NewCloudAccount(),
	}
}

func (s *CloudAccountService) Page(ctx gormx.Contextx) (dto.PageResult, error) {
	var res dto.PageResult
	total, err := s.repo.Count()
	if err != nil {
		return res, err
	}
	res.Total = total
	list, err := s.repo.List(ctx.Page, ctx.Limit)
	if err != nil {
		return res, err
	}
	res.Items = list
	return res, nil
}

func (s *CloudAccountService) Create(req *request.CloudAccountCreate) error {
	authBytes, _ := json.Marshal(req.Authorization)
	account := &model.CloudAccount{
		Name:          strings.TrimSpace(req.Name),
		Type:          req.Type,
		Authorization: string(authBytes),
	}
	return s.repo.Create(context.Background(), account)
}

func (s *CloudAccountService) Update(req *request.CloudAccountUpdate) error {
	account, err := s.repo.GetByID(req.ID)
	if err != nil {
		return err
	}
	if req.Name != "" {
		account.Name = strings.TrimSpace(req.Name)
	}
	if req.Type != "" {
		account.Type = req.Type
	}
	if req.Authorization != nil {
		authBytes, _ := json.Marshal(req.Authorization)
		account.Authorization = string(authBytes)
	}

	return s.repo.Update(context.Background(), &account)
}

func (s *CloudAccountService) Delete(id uint) error {
	return s.repo.DeleteByID(context.Background(), id)
}
