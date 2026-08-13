package api

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/aiprovider"
	"github.com/aihop/gopanel/utils/encrypt"
	"github.com/aihop/gopanel/utils/token"
	"github.com/gofiber/fiber/v3"
)

func TestCreateAISessionUsesProjectDirectoryWhenIsolationDisabled(t *testing.T) {
	database := withCodeGovernanceDB(t)
	repositoryDir := createCodeGitRepository(t)
	project := &model.AIProject{ID: 301, Name: "direct", CreatorID: 7, SourceDirs: []string{repositoryDir}}
	if err := database.Create(project).Error; err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(map[string]any{
		"title": "direct task", "projectId": project.ID, "executorId": "terminal",
		"approvalPolicy": codeApprovalPolicyFullAuto, "isolated": false,
	})
	if err != nil {
		t.Fatal(err)
	}
	app := fiber.New()
	app.Post("/sessions", func(c fiber.Ctx) error {
		c.Locals(constant.AppAuthName, &token.CustomClaims{UserId: project.CreatorID, Role: constant.UserRoleSuper})
		return CreateAISession(c)
	})
	request := httptest.NewRequest("POST", "/sessions", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response, err := app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	var result struct {
		Code int                `json:"code"`
		Msg  string             `json:"msg"`
		Data model.AIDevSession `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if result.Code != 0 {
		t.Fatalf("create direct session failed: %s", result.Msg)
	}
	if result.Data.WorkDir != repositoryDir || result.Data.IsolationMode != codeIsolationDirect ||
		result.Data.WorktreeBranch != "" || result.Data.Status != codeSessionStatusActive {
		t.Fatalf("unexpected direct session: %#v", result.Data)
	}
}

func TestCreateAISessionUsesSavedProviderAccount(t *testing.T) {
	database := withCodeGovernanceDB(t)
	if err := database.AutoMigrate(&model.AIProviderAccount{}); err != nil {
		t.Fatal(err)
	}
	oldKey := global.CONF.System.EncryptKey
	global.CONF.System.EncryptKey = "0123456789abcdef0123456789abcdef"
	t.Cleanup(func() { global.CONF.System.EncryptKey = oldKey })
	ciphertext, err := encrypt.StringEncrypt("saved-secret")
	if err != nil {
		t.Fatal(err)
	}
	account := &model.AIProviderAccount{
		UserID: 7, Name: "开发账号", BaseURL: "https://gateway.example.com/v1",
		APIKey: ciphertext, Model: "gpt-5", Protocol: aiprovider.ProtocolOpenAIResponses, Enabled: true,
	}
	if err := database.Create(account).Error; err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(map[string]any{
		"title": "saved account task", "executorId": "codex",
		"approvalPolicy": codeApprovalPolicyFullAuto, "providerAccountId": account.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	app := fiber.New()
	app.Post("/sessions", func(c fiber.Ctx) error {
		c.Locals(constant.AppAuthName, &token.CustomClaims{UserId: account.UserID, Role: constant.UserRoleSuper})
		return CreateAISession(c)
	})
	request := httptest.NewRequest("POST", "/sessions", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response, err := app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	var result struct {
		Code int                `json:"code"`
		Msg  string             `json:"msg"`
		Data model.AIDevSession `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if result.Code != 0 {
		t.Fatalf("create session with saved account failed: %s", result.Msg)
	}
	var stored model.AIDevSession
	if err := database.First(&stored, result.Data.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.ProviderBaseURL != account.BaseURL || stored.ProviderModel != account.Model ||
		stored.ProviderAPIKey == "saved-secret" || stored.ProviderAPIKey == account.APIKey {
		t.Fatalf("saved account should be copied into encrypted session config: %#v", stored)
	}
	decrypted, err := encrypt.StringDecrypt(stored.ProviderAPIKey)
	if err != nil || decrypted != "saved-secret" {
		t.Fatalf("unexpected session provider key: %q, %v", decrypted, err)
	}
}
