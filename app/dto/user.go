/*
 * @Author: coller
 * @Date: 2024-03-16 13:19:06
 * @LastEditors: coller
 * @LastEditTime: 2024-04-11 11:20:40
 * @Desc: 用户
 */
package dto

import (
	"time"
)

type UserCredits struct {
	UserId         uint `cache:"userId" json:"userId"`
	CreditsBalance uint `cache:"creditsBalance" json:"creditsBalance"`
	CreditRatio    uint `cache:"creditRatio" json:"creditRatio"`
}

type Token struct {
	Token string `json:"token" validate:"required"`
}

type UserInfo struct {
	ID               uint       `json:"id"`
	LastName         string     `json:"lastName"`
	FirstName        string     `json:"firstName"`
	IsdCode          string     `json:"isdCode"`
	Email            string     `json:"email"`
	Mobile           string     `json:"mobile"`
	IsEmailVerified  int        `json:"isEmailVerified"`
	IdentityVerified int        `json:"identityVerified"`
	CreditsBalance   uint       `json:"creditsBalance"`
	NickName         string     `json:"nickName"`
	Avatar           string     `json:"avatar"`
	PayMoney         float64    `json:"payMoney"`
	OrderNum         uint       `json:"orderNum"`
	Fans             uint       `json:"fans"`
	IsSubscribe      uint       `json:"isSubscribe"`
	Birthday         *time.Time `json:"birthday"`
	Bio              string     `json:"bio"`
	StoreId          uint       `json:"storeId"`
	Role             string     `json:"role"`
	Menus            string     `json:"menus"`
}

type UserXAuth struct {
	XAuth     string `json:"xAuth"`
	ExpiresIn int    `json:"expiresIn"`
}

type UserSigninInfo struct {
	UserXAuth
	UserInfo  *UserInfo `json:"userInfo"`
	VisitorId string    `json:"visitorId"`
	SendEmail bool      `json:"-"`
}

type UserEditEmail struct {
	Email string `json:"email" validate:"required"`
	Code  string `json:"code" validate:"required"`
}

type UserEditMobile struct {
	Mobile string `json:"mobile" validate:"required"`
	Code   string `json:"code" validate:"required"`
}

type UserToken struct {
	File      string `json:"file"`
	Timestamp int64  `json:"timestamp"`
}

type UserEditInfo struct {
	Email    string `json:"email"  validate:"required"`
	NickName string `json:"nickName"`
}
