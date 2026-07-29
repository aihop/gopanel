package repo

import (
	"errors"

	"github.com/aihop/gopanel/app/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UserNoteRepo struct {
	db *gorm.DB
}

func NewUserNote(db *gorm.DB) *UserNoteRepo {
	return &UserNoteRepo{db: db}
}

func (r *UserNoteRepo) MigrateTable() error {
	return r.db.AutoMigrate(&model.UserNote{})
}

func (r *UserNoteRepo) GetByUserID(userID uint) (*model.UserNote, error) {
	note := &model.UserNote{UserID: userID}
	err := r.db.Where("user_id = ?", userID).First(note).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return note, nil
	}
	return note, err
}

func (r *UserNoteRepo) Save(userID uint, content string) (*model.UserNote, error) {
	note := &model.UserNote{UserID: userID, Content: content}
	err := r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"content", "updated_at"}),
	}).Create(note).Error
	if err != nil {
		return nil, err
	}
	return r.GetByUserID(userID)
}
