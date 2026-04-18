package request

import "github.com/aihop/gopanel/app/dto"

type UserCreate struct {
	NickName    string `json:"nickName" validate:"required"`
	Email       string `json:"email" validate:"required,email"`
	Password    string `json:"password" validate:"required,min=6"`
	Role        string `json:"role" validate:"required"`
	FileBaseDir string `json:"fileBaseDir"`
	Menus       string `json:"menus"`
}

type UserUpdate struct {
	ID          uint   `json:"id" validate:"required"`
	NickName    string `json:"nickName"`
	Email       string `json:"email"`
	Password    string `json:"password"`
	Role        string `json:"role"`
	FileBaseDir string `json:"fileBaseDir"`
	Menus       string `json:"menus"`
}

type UserSearch struct {
	dto.PageInfo
}
