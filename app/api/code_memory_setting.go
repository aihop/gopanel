package api

import (
	"errors"
	"strings"

	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/encrypt"
	"github.com/aihop/gopanel/utils/token"
	"github.com/gofiber/fiber/v3"
)

const codeMemoryDefaultGrowthThreshold = 8

type codeMemorySettingView struct {
	Enabled         bool   `json:"enabled"`
	BaseURL         string `json:"baseUrl"`
	Model           string `json:"model"`
	HasAPIKey       bool   `json:"hasApiKey"`
	GrowthThreshold int    `json:"growthThreshold"`
}

func codeMemorySettingViewOf(setting *model.AICodeMemorySetting) codeMemorySettingView {
	if setting == nil {
		return codeMemorySettingView{GrowthThreshold: codeMemoryDefaultGrowthThreshold}
	}
	return codeMemorySettingView{
		Enabled: setting.Enabled, BaseURL: setting.BaseURL, Model: setting.Model,
		HasAPIKey: strings.TrimSpace(setting.APIKey) != "", GrowthThreshold: setting.GrowthThreshold,
	}
}

func GetCodeMemorySetting(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	setting, err := loadCodeMemorySetting(claims.UserId)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(codeMemorySettingViewOf(setting)))
}

// SaveCodeMemorySetting 保存抽取模型配置。
//
// 开启时必须给全 baseURL / model / 密钥：配了一半就开启，只会让抽取每次
// 静默失败——而抽取跑在后台，用户根本不会知道它一直在失败。
func SaveCodeMemorySetting(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	var req struct {
		Enabled         bool   `json:"enabled"`
		BaseURL         string `json:"baseUrl"`
		APIKey          string `json:"apiKey"`
		Model           string `json:"model"`
		GrowthThreshold int    `json:"growthThreshold"`
	}
	if err := c.Bind().JSON(&req); err != nil {
		return c.JSON(e.Fail(err))
	}
	setting, err := loadCodeMemorySetting(claims.UserId)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if setting == nil {
		setting = &model.AICodeMemorySetting{UserID: claims.UserId}
	}
	setting.BaseURL = strings.TrimRight(strings.TrimSpace(req.BaseURL), "/")
	setting.Model = strings.TrimSpace(req.Model)
	setting.Enabled = req.Enabled
	setting.GrowthThreshold = normalizeCodeMemoryGrowthThreshold(req.GrowthThreshold)
	// 留空表示沿用已保存的密钥，和 Git 凭据的做法一致。
	if strings.TrimSpace(req.APIKey) != "" {
		ciphertext, err := encrypt.StringEncrypt(req.APIKey)
		if err != nil {
			return c.JSON(e.Fail(errors.New("抽取模型密钥加密失败")))
		}
		setting.APIKey = ciphertext
	}
	if setting.Enabled {
		if setting.BaseURL == "" || setting.Model == "" {
			return c.JSON(e.Fail(errors.New("启用记忆抽取需要填写模型服务地址和模型名称")))
		}
		if strings.TrimSpace(setting.APIKey) == "" {
			return c.JSON(e.Fail(errors.New("启用记忆抽取需要填写模型密钥")))
		}
	}
	if err := global.DB.Save(setting).Error; err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(codeMemorySettingViewOf(setting)))
}

// normalizeCodeMemoryGrowthThreshold 收敛阈值。
// 上限压在 100：再大就等于关掉自动抽取，那该用 enabled 开关表达，
// 而不是把阈值调到一个"实际上永远达不到"的值。
func normalizeCodeMemoryGrowthThreshold(value int) int {
	if value < 0 {
		return codeMemoryDefaultGrowthThreshold
	}
	if value > 100 {
		return 100
	}
	return value
}

func loadCodeMemorySetting(userID uint) (*model.AICodeMemorySetting, error) {
	if global.DB == nil || userID == 0 {
		return nil, nil
	}
	var setting model.AICodeMemorySetting
	if err := global.DB.Where("user_id = ?", userID).First(&setting).Error; err != nil {
		return nil, nil
	}
	return &setting, nil
}

// resolveCodeMemoryLLMConfig 解出可用的抽取模型配置。
// 没配或没启用时返回明确的原因，让日志能说清楚为什么没抽。
func resolveCodeMemoryLLMConfig(userID uint) (codeMemoryLLMConfig, int, error) {
	setting, err := loadCodeMemorySetting(userID)
	if err != nil {
		return codeMemoryLLMConfig{}, 0, err
	}
	if setting == nil || !setting.Enabled {
		return codeMemoryLLMConfig{}, 0, errors.New("尚未启用记忆抽取")
	}
	apiKey := ""
	if strings.TrimSpace(setting.APIKey) != "" {
		decrypted, decryptErr := encrypt.StringDecrypt(setting.APIKey)
		if decryptErr != nil {
			return codeMemoryLLMConfig{}, 0, errors.New("抽取模型密钥无法解密，请重新保存")
		}
		apiKey = decrypted
	}
	return codeMemoryLLMConfig{
		BaseURL: setting.BaseURL, APIKey: apiKey, Model: setting.Model,
	}, setting.GrowthThreshold, nil
}
