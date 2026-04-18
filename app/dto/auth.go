package dto

type AuthResetPasswordReq struct {
	Password    string `json:"password" validate:"required"`    // 旧密码
	NewPassword string `json:"newPassword" validate:"required"` // 新密码
}

type AuthResetPassword struct {
	Token    string `json:"token" validate:"required"`
	Password string `json:"password" validate:"required"` // 密码
}

type AuthMFALogin struct {
	Name       string `json:"name" validate:"required"`
	Password   string `json:"password" validate:"required"`
	Code       string `json:"code" validate:"required"`
	AuthMethod string `json:"authMethod"`
}

type AuthSignin struct {
	Email    string `json:"email"`
	Mobile   string `json:"mobile"`
	Code     string `json:"code"`
	Password string `json:"password"`
}
