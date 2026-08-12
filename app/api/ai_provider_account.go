package api

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/encrypt"
	"github.com/aihop/gopanel/utils/token"
	"github.com/gofiber/fiber/v3"
)

type aiProviderAccountView struct {
	model.AIProviderAccount
	HasAPIKey bool `json:"hasApiKey"`
}

func aiProviderAccountViews(accounts []model.AIProviderAccount) []aiProviderAccountView {
	views := make([]aiProviderAccountView, 0, len(accounts))
	for _, account := range accounts {
		views = append(views, aiProviderAccountView{
			AIProviderAccount: account,
			HasAPIKey:         strings.TrimSpace(account.APIKey) != "",
		})
	}
	return views
}

func GetAIProviderAccounts(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	var accounts []model.AIProviderAccount
	if err := global.DB.Where("user_id = ?", claims.UserId).
		Order("priority asc, name asc, id asc").Find(&accounts).Error; err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(aiProviderAccountViews(accounts)))
}

// SaveAIProviderAccount 保存账号，并在保存前探测一次。
//
// 探测不是可选步骤：这份凭据的主要消费者是后台的记忆抽取，密钥填错的话
// 它会一直静默失败，用户只会看到记忆永远不出现，完全不知道为什么。
// 保存时挡下来，是唯一能让用户当场知道的时机。
func SaveAIProviderAccount(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	accountID, err := strconv.ParseUint(c.Params("id", "0"), 10, 64)
	if err != nil {
		return c.JSON(e.Fail(errors.New("AI 账号参数无效")))
	}
	var req struct {
		Name                   string `json:"name"`
		BaseURL                string `json:"baseUrl"`
		APIKey                 string `json:"apiKey"`
		Model                  string `json:"model"`
		Enabled                bool   `json:"enabled"`
		UseForMemoryExtraction bool   `json:"useForMemoryExtraction"`
		Priority               int    `json:"priority"`
		DefaultReasoningEffort string `json:"defaultReasoningEffort"`
	}
	if err := c.Bind().JSON(&req); err != nil {
		return c.JSON(e.Fail(err))
	}
	account := model.AIProviderAccount{ID: uint(accountID), UserID: claims.UserId}
	if accountID > 0 {
		if err := global.DB.Where("id = ? AND user_id = ?", accountID, claims.UserId).
			First(&account).Error; err != nil {
			return c.JSON(e.Fail(errors.New("AI 账号不存在")))
		}
	}
	account.Name = strings.TrimSpace(req.Name)
	account.BaseURL = strings.TrimRight(strings.TrimSpace(req.BaseURL), "/")
	account.Model = strings.TrimSpace(req.Model)
	account.Enabled = req.Enabled
	account.UseForMemoryExtraction = req.UseForMemoryExtraction
	account.Priority = normalizeAIProviderPriority(req.Priority)
	account.DefaultReasoningEffort = normalizeCodeReasoningEffort(req.DefaultReasoningEffort)
	if account.Name == "" || account.BaseURL == "" || account.Model == "" {
		return c.JSON(e.Fail(errors.New("请填写账号名称、服务地址和模型名称")))
	}
	// 留空表示沿用已保存的密钥，和 Git 凭据一致。
	if strings.TrimSpace(req.APIKey) != "" {
		ciphertext, encryptErr := encrypt.StringEncrypt(req.APIKey)
		if encryptErr != nil {
			return c.JSON(e.Fail(errors.New("AI 账号密钥加密失败")))
		}
		account.APIKey = ciphertext
	}
	if strings.TrimSpace(account.APIKey) == "" {
		return c.JSON(e.Fail(errors.New("请填写模型密钥")))
	}
	apiKey, err := encrypt.StringDecrypt(account.APIKey)
	if err != nil {
		return c.JSON(e.Fail(errors.New("已保存的密钥无法解密，请重新填写")))
	}
	probe, probeErr := probeAIProviderAccount(context.Background(), codeMemoryLLMConfig{
		BaseURL: account.BaseURL, APIKey: apiKey, Model: account.Model,
	}, account.DefaultReasoningEffort)
	if probeErr != nil {
		return c.JSON(e.Fail(probeErr))
	}
	now := time.Now()
	account.SupportsTemperature = probe.SupportsTemperature
	account.SupportsJSONSchema = probe.SupportsJSONSchema
	account.SupportsReasoningEffort = probe.SupportsReasoningEffort
	account.ProbedAt, account.ProbeError = &now, ""
	if err := global.DB.Save(&account).Error; err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(aiProviderAccountViews([]model.AIProviderAccount{account})[0]))
}

func DeleteAIProviderAccount(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	accountID, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil || accountID == 0 {
		return c.JSON(e.Fail(errors.New("AI 账号参数无效")))
	}
	result := global.DB.Where("id = ? AND user_id = ?", accountID, claims.UserId).
		Delete(&model.AIProviderAccount{})
	if result.Error != nil {
		return c.JSON(e.Fail(result.Error))
	}
	if result.RowsAffected == 0 {
		return c.JSON(e.Fail(errors.New("AI 账号不存在")))
	}
	return c.JSON(e.Succ(nil))
}

// normalizeAIProviderPriority 收敛优先级。
// 允许 0：用户想把某个账号钉在最前时不该被迫从 1 开始数。
func normalizeAIProviderPriority(value int) int {
	if value < 0 {
		return 0
	}
	if value > 999 {
		return 999
	}
	return value
}

// selectAIProviderAccountForMemory 挑一个用于记忆抽取的账号。
//
// 指定了就用指定的（前提是它仍然启用且仍被授权抽取——授权可能在配置后被撤回）；
// 没指定就按优先级自动挑，与 codux 的 automatic 一致。
func selectAIProviderAccountForMemory(userID, preferredID uint) (*model.AIProviderAccount, error) {
	if global.DB == nil || userID == 0 {
		return nil, errors.New("AI 账号不可用")
	}
	if preferredID > 0 {
		var account model.AIProviderAccount
		err := global.DB.Where(
			"id = ? AND user_id = ? AND enabled = ? AND use_for_memory_extraction = ?",
			preferredID, userID, true, true,
		).First(&account).Error
		if err == nil {
			return &account, nil
		}
		// 指定的账号已被停用或撤回授权时不静默回落到别的账号：
		// 那等于把对话记录发去了用户没点头的地方。
		return nil, errors.New("指定的 AI 账号已停用或未授权用于记忆抽取")
	}
	var account model.AIProviderAccount
	err := global.DB.Where(
		"user_id = ? AND enabled = ? AND use_for_memory_extraction = ?", userID, true, true,
	).Order("priority asc, id asc").First(&account).Error
	if err != nil {
		return nil, errors.New("没有可用于记忆抽取的 AI 账号，请在系统设置里添加并授权")
	}
	return &account, nil
}

func aiProviderAccountLLMConfig(account *model.AIProviderAccount) (codeMemoryLLMConfig, error) {
	if account == nil {
		return codeMemoryLLMConfig{}, errors.New("AI 账号不可用")
	}
	apiKey := ""
	if strings.TrimSpace(account.APIKey) != "" {
		decrypted, err := encrypt.StringDecrypt(account.APIKey)
		if err != nil {
			return codeMemoryLLMConfig{}, errors.New("AI 账号密钥无法解密，请重新保存")
		}
		apiKey = decrypted
	}
	return codeMemoryLLMConfig{BaseURL: account.BaseURL, APIKey: apiKey, Model: account.Model}, nil
}
