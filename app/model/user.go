package model

import (
	"time"
)

type User struct {
	ID              uint      `gorm:"column:id;primaryKey;type:uint;size:30;comment:主键" json:"id"`
	NickName        string    `gorm:"column:nick_name;type:varchar(64);index;comment:昵称;" json:"nickName"`
	Email           string    `gorm:"column:email;type:varchar(50);uniqueIndex;not null;comment:邮箱地址;" json:"email"`
	Mobile          string    `gorm:"column:mobile;type:varchar(30);index;comment:手机号码;" json:"mobile"`
	Salt            string    `gorm:"column:salt;type:varchar(6);comment:安全符;" json:"-"`
	Role            string    `gorm:"column:role;type:varchar(20);comment:角色;default:USER" json:"role"`
	FileBaseDir     string    `gorm:"column:file_base_dir;type:varchar(255);comment:限制的根目录;default:''" json:"fileBaseDir"`
	Password        string    `gorm:"column:password;type:varchar(128);comment:账号密码;" json:"-"`
	IsEmailVerified int       `gorm:"column:is_email_verified;type:int;size:4;default:10;comment:Email认证;" json:"isEmailVerified"`
	Token           string    `gorm:"column:token;type:varchar(64);comment:token;" json:"-"`
	Status          int       `gorm:"column:status;type:int;size:4;default:10;comment:状态;" json:"status"`
	LoginAt         time.Time `gorm:"column:login_at;default:NULL;comment:登录时间" json:"loginAt"`
	Menus           string    `gorm:"column:menus;type:text;comment:允许访问的菜单路由列表，用逗号分隔;" json:"menus"`
}
