package api

import (
	"errors"
	"strconv"
	"strings"

	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/encrypt"
	"github.com/aihop/gopanel/utils/token"
	"github.com/gofiber/fiber/v3"
)

type codeGitCredentialView struct {
	ID        uint   `json:"id"`
	Name      string `json:"name"`
	Username  string `json:"username"`
	HasSecret bool   `json:"hasSecret"`
}

func codeGitCredentialViews(credentials []model.AIGitCredential) []codeGitCredentialView {
	views := make([]codeGitCredentialView, 0, len(credentials))
	for _, credential := range credentials {
		views = append(views, codeGitCredentialView{
			ID: credential.ID, Name: credential.Name, Username: credential.Username,
			HasSecret: strings.TrimSpace(credential.Secret) != "",
		})
	}
	return views
}

func GetCodeGitCredentials(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	var credentials []model.AIGitCredential
	if err := global.DB.Where("creator_id = ?", claims.UserId).Order("name asc, id asc").Find(&credentials).Error; err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(codeGitCredentialViews(credentials)))
}

func SaveCodeGitCredential(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	credentialID, err := parseCodeGitCredentialID(c.Params("id", "0"))
	if err != nil {
		return c.JSON(e.Fail(errors.New("Git 凭据参数无效")))
	}
	var req struct {
		Name     string `json:"name"`
		Username string `json:"username"`
		Secret   string `json:"secret"`
		// 可选的校验仓库地址。填了就在保存前实连一次——一套连不上的凭据
		// 存进去不会当场报错，而是等到建会话或交付时才炸，
		// 实测这是全局最大的单一失败源。
		VerifyRemote string `json:"verifyRemote"`
	}
	if err := c.Bind().JSON(&req); err != nil {
		return c.JSON(e.Fail(err))
	}
	req.Name, req.Username = strings.TrimSpace(req.Name), strings.TrimSpace(req.Username)
	if req.Name == "" || req.Username == "" {
		return c.JSON(e.Fail(errors.New("凭据名称和用户名不能为空")))
	}
	credential := model.AIGitCredential{ID: credentialID, CreatorID: claims.UserId}
	if credentialID > 0 {
		if err := global.DB.Where("id = ? AND creator_id = ?", credentialID, claims.UserId).First(&credential).Error; err != nil {
			return c.JSON(e.Fail(errors.New("Git 凭据不存在")))
		}
	}
	if strings.TrimSpace(req.Secret) != "" {
		credential.Secret, err = encrypt.StringEncrypt(req.Secret)
		if err != nil {
			return c.JSON(e.Fail(errors.New("Git 凭据加密失败")))
		}
	}
	if credential.Secret == "" {
		return c.JSON(e.Fail(errors.New("请输入访问令牌或密码")))
	}
	credential.Name, credential.Username = req.Name, req.Username
	// 校验要用即将保存的这一份：编辑时没重填令牌就沿用库里的旧值。
	if strings.TrimSpace(req.VerifyRemote) != "" {
		secret := req.Secret
		if strings.TrimSpace(secret) == "" {
			if secret, err = encrypt.StringDecrypt(credential.Secret); err != nil {
				return c.JSON(e.Fail(errors.New("已保存的凭据无法解密，请重新填写访问令牌")))
			}
		}
		if err := probeCodeGitCredentialRemote(credential.Username, secret, req.VerifyRemote); err != nil {
			return c.JSON(e.Fail(err))
		}
	}
	if err := global.DB.Save(&credential).Error; err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(codeGitCredentialViews([]model.AIGitCredential{credential})[0]))
}

// VerifyCodeGitCredential 显式测试一套凭据能否访问指定仓库。
//
// 和保存时的校验分开，是为了让用户在填表过程中就能试，
// 而不是只能靠「保存被拒」来发现填错了。
func VerifyCodeGitCredential(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	var req struct {
		CredentialID uint   `json:"credentialId"`
		Username     string `json:"username"`
		Secret       string `json:"secret"`
		Remote       string `json:"remote"`
	}
	if err := c.Bind().JSON(&req); err != nil {
		return c.JSON(e.Fail(err))
	}
	username, secret := strings.TrimSpace(req.Username), req.Secret
	// 编辑已有凭据时前端通常不回传令牌，这时用库里那一份。
	if req.CredentialID > 0 && strings.TrimSpace(secret) == "" {
		if err := validateCodeGitCredentialAccess(req.CredentialID, claims.UserId); err != nil {
			return c.JSON(e.Fail(err))
		}
		var credential model.AIGitCredential
		if err := global.DB.First(&credential, req.CredentialID).Error; err != nil {
			return c.JSON(e.Fail(errors.New("Git 凭据不存在")))
		}
		stored, err := encrypt.StringDecrypt(credential.Secret)
		if err != nil {
			return c.JSON(e.Fail(errors.New("已保存的凭据无法解密，请重新填写访问令牌")))
		}
		secret = stored
		if username == "" {
			username = credential.Username
		}
	}
	if err := probeCodeGitCredentialRemote(username, secret, req.Remote); err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(nil))
}

func DeleteCodeGitCredential(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	credentialID, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil || credentialID == 0 {
		return c.JSON(e.Fail(errors.New("Git 凭据参数无效")))
	}
	var count int64
	if err := global.DB.Model(&model.AIProject{}).Where("git_credential_id = ?", credentialID).Count(&count).Error; err != nil {
		return c.JSON(e.Fail(err))
	}
	if count > 0 {
		return c.JSON(e.Fail(errors.New("该凭据仍被项目使用，请先解除项目绑定")))
	}
	result := global.DB.Where("id = ? AND creator_id = ?", credentialID, claims.UserId).Delete(&model.AIGitCredential{})
	if result.Error != nil {
		return c.JSON(e.Fail(result.Error))
	}
	if result.RowsAffected == 0 {
		return c.JSON(e.Fail(errors.New("Git 凭据不存在")))
	}
	return c.JSON(e.Succ(nil))
}

func validateCodeGitCredentialAccess(credentialID, userID uint) error {
	if credentialID == 0 {
		return nil
	}
	var count int64
	if err := global.DB.Model(&model.AIGitCredential{}).
		Where("id = ? AND creator_id = ?", credentialID, userID).Count(&count).Error; err != nil {
		return err
	}
	if count != 1 {
		return errors.New("选择的 Git 凭据不存在或无权使用")
	}
	return nil
}
