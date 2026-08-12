package api

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/constant"
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
