package service

import (
	"unicode/utf8"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/buserr"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
)

const UserNoteMaxLength = 20_000

type UserNoteService struct {
	repo *repo.UserNoteRepo
}

func NewUserNoteService() *UserNoteService {
	return &UserNoteService{repo: repo.NewUserNote(global.DB)}
}

func (s *UserNoteService) Get(userID uint) (*model.UserNote, error) {
	return s.repo.GetByUserID(userID)
}

func (s *UserNoteService) Save(userID uint, content string) (*model.UserNote, error) {
	if utf8.RuneCountInString(content) > UserNoteMaxLength {
		return nil, buserr.WithMap(constant.ErrUserNoteTooLong, map[string]interface{}{"max": UserNoteMaxLength})
	}
	return s.repo.Save(userID, content)
}
