package api

import (
	"errors"

	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/token"
	"github.com/gofiber/fiber/v3"
)

const codeMemoryDefaultGrowthThreshold = 8

type codeMemorySettingView struct {
	Enabled         bool   `json:"enabled"`
	AccountID       uint   `json:"accountId"`
	AccountName     string `json:"accountName,omitempty"`
	GrowthThreshold int    `json:"growthThreshold"`
	// Ready 表示现在真的能抽。开了开关但没有可用账号时它是 false——
	// 界面据此提示「去添加 AI 账号」，而不是让用户对着空列表猜。
	Ready       bool   `json:"ready"`
	ReadyReason string `json:"readyReason,omitempty"`
}

func buildCodeMemorySettingView(userID uint, setting *model.AICodeMemorySetting) codeMemorySettingView {
	view := codeMemorySettingView{GrowthThreshold: codeMemoryDefaultGrowthThreshold}
	if setting != nil {
		view.Enabled = setting.Enabled
		view.AccountID = setting.AccountID
		view.GrowthThreshold = setting.GrowthThreshold
	}
	if !view.Enabled {
		view.ReadyReason = "尚未启用记忆抽取"
		return view
	}
	account, err := selectAIProviderAccountForMemory(userID, view.AccountID)
	if err != nil {
		view.ReadyReason = err.Error()
		return view
	}
	view.Ready, view.AccountName = true, account.Name
	return view
}

func GetCodeMemorySetting(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	setting, err := loadCodeMemorySetting(claims.UserId)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(buildCodeMemorySettingView(claims.UserId, setting)))
}

// SaveCodeMemorySetting 保存记忆抽取的开关与调度参数。
//
// 启用时必须当场解析出一个可用账号：开了开关却没有可用账号，
// 抽取会在后台一直静默失败，而用户以为自己已经打开了。
func SaveCodeMemorySetting(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	var req struct {
		Enabled         bool `json:"enabled"`
		AccountID       uint `json:"accountId"`
		GrowthThreshold int  `json:"growthThreshold"`
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
	setting.Enabled = req.Enabled
	setting.AccountID = req.AccountID
	setting.GrowthThreshold = normalizeCodeMemoryGrowthThreshold(req.GrowthThreshold)
	if setting.Enabled {
		if _, err := selectAIProviderAccountForMemory(claims.UserId, setting.AccountID); err != nil {
			return c.JSON(e.Fail(err))
		}
	}
	if err := global.DB.Save(setting).Error; err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(buildCodeMemorySettingView(claims.UserId, setting)))
}

// normalizeCodeMemoryGrowthThreshold 收敛阈值。
// 上限压在 100：再大就等于关掉自动抽取，那该用 enabled 开关表达，
// 而不是把阈值调到一个「实际上永远达不到」的值。
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

// resolveCodeMemoryExtraction 解出这次抽取要用的账号和调度参数。
func resolveCodeMemoryExtraction(userID uint) (*model.AIProviderAccount, codeMemoryLLMConfig, int, error) {
	setting, err := loadCodeMemorySetting(userID)
	if err != nil {
		return nil, codeMemoryLLMConfig{}, 0, err
	}
	if setting == nil || !setting.Enabled {
		return nil, codeMemoryLLMConfig{}, 0, errors.New("尚未启用记忆抽取")
	}
	account, err := selectAIProviderAccountForMemory(userID, setting.AccountID)
	if err != nil {
		return nil, codeMemoryLLMConfig{}, 0, err
	}
	config, err := aiProviderAccountLLMConfig(account)
	if err != nil {
		return nil, codeMemoryLLMConfig{}, 0, err
	}
	return account, config, setting.GrowthThreshold, nil
}
