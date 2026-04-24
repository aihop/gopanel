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
	Email        string `json:"email"`
	Mobile       string `json:"mobile"`
	Code         string `json:"code"`
	Password     string `json:"password"`
	CaptchaToken string `json:"captchaToken"`
}

type VerifyCaptchaGetReq struct {
	CaptchaType string `json:"captchaType"`
}

type VerifyCaptchaCheckReq struct {
	CaptchaType string `json:"captchaType"`
	Point       string `json:"point" validate:"required"`
	Token       string `json:"token" validate:"required"`
}

type VerifyCaptchaGetResp struct {
	OriginalImg string `json:"originalImg"`
	BlockImg    string `json:"blockImg"`
	Token       string `json:"token"`
	SecretKey   string `json:"secretKey"`
	IsOK        bool   `json:"isOk"`
	PieceTop    int    `json:"pieceTop"`
	PieceWidth  int    `json:"pieceWidth"`
	PieceHeight int    `json:"pieceHeight"`
}
