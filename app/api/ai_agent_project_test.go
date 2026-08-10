package api

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/utils/token"
	"github.com/gofiber/fiber/v3"
	"net/http/httptest"
)

func TestNormalizeAIProjectWorkDir(t *testing.T) {
	baseDir := t.TempDir()
	projectDir := filepath.Join(baseDir, "project")
	if err := os.Mkdir(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}
	claims := &token.CustomClaims{Role: constant.UserRoleSubAdmin, FileBaseDir: baseDir}

	got, err := normalizeAIProjectWorkDir(projectDir, claims)
	if err != nil {
		t.Fatal(err)
	}
	want, err := filepath.EvalSymlinks(projectDir)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("workDir = %q, want %q", got, want)
	}
	if _, err := normalizeAIProjectWorkDir(filepath.Dir(baseDir), claims); err == nil {
		t.Fatal("expected a directory outside the sub-admin workspace to be rejected")
	}
}

func TestNormalizeAIProjectWorkDirRejectsInvalidPaths(t *testing.T) {
	claims := &token.CustomClaims{Role: constant.UserRoleSuper}
	if _, err := normalizeAIProjectWorkDir("relative/path", claims); err == nil {
		t.Fatal("expected a relative path to be rejected")
	}
	filePath := filepath.Join(t.TempDir(), "file.txt")
	if err := os.WriteFile(filePath, []byte("test"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := normalizeAIProjectWorkDir(filePath, claims); err == nil {
		t.Fatal("expected a file path to be rejected")
	}
}

func TestCanManageAIProject(t *testing.T) {
	project := &model.AIProject{CreatorID: 7}
	if !canManageAIProject(project, &token.CustomClaims{UserId: 7, Role: constant.UserRoleSubAdmin}) {
		t.Fatal("expected the project creator to manage the project")
	}
	if !canManageAIProject(project, &token.CustomClaims{UserId: 9, Role: constant.UserRoleSuper}) {
		t.Fatal("expected a super admin to manage the project")
	}
	if canManageAIProject(project, &token.CustomClaims{UserId: 9, Role: constant.UserRoleAdmin}) {
		t.Fatal("expected another admin to be denied")
	}
}

func TestAIProjectDirectoryDefaults(t *testing.T) {
	baseDir := t.TempDir()
	defaultDir, rootDir, err := aiProjectDirectoryDefaults(&token.CustomClaims{
		Role:        constant.UserRoleSubAdmin,
		FileBaseDir: baseDir,
	}, "/ignored")
	if err != nil {
		t.Fatal(err)
	}
	want, err := filepath.EvalSymlinks(baseDir)
	if err != nil {
		t.Fatal(err)
	}
	if defaultDir != want || rootDir != want {
		t.Fatalf("sub-admin defaults = (%q, %q), want %q", defaultDir, rootDir, want)
	}

	homeDir := t.TempDir()
	defaultDir, rootDir, err = aiProjectDirectoryDefaults(&token.CustomClaims{Role: constant.UserRoleAdmin}, homeDir)
	if err != nil {
		t.Fatal(err)
	}
	wantHome, err := filepath.EvalSymlinks(homeDir)
	if err != nil {
		t.Fatal(err)
	}
	wantRoot := string(filepath.Separator)
	if volume := filepath.VolumeName(homeDir); volume != "" {
		wantRoot = volume + string(filepath.Separator)
	}
	if defaultDir != wantHome || rootDir != wantRoot {
		t.Fatalf("admin defaults = (%q, %q), want (%q, %q)", defaultDir, rootDir, wantHome, wantRoot)
	}
}

func TestAIProjectDirectoryDefaultsRejectsInvalidSubAdminBase(t *testing.T) {
	_, _, err := aiProjectDirectoryDefaults(&token.CustomClaims{Role: constant.UserRoleSubAdmin}, t.TempDir())
	if err == nil {
		t.Fatal("expected an invalid sub-admin base directory to be rejected")
	}
}

func TestCreateAIProjectDoesNotRequireRemoteAccess(t *testing.T) {
	database := withCodeGovernanceDB(t)
	repository := createCodeGitRepository(t)
	deliveryBranch, err := runCodeGit(repository, "branch", "--show-current")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runCodeGit(repository, "remote", "add", "origin", filepath.Join(t.TempDir(), "missing.git")); err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(map[string]any{
		"name": "offline project", "sourceDirs": []string{repository}, "deliveryBranch": deliveryBranch,
	})
	if err != nil {
		t.Fatal(err)
	}
	app := fiber.New()
	app.Post("/projects", func(c fiber.Ctx) error {
		c.Locals(constant.AppAuthName, &token.CustomClaims{UserId: 7, Role: constant.UserRoleSuper})
		return CreateAIProject(c)
	})
	request := httptest.NewRequest("POST", "/projects", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response, err := app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if result.Code != 0 {
		t.Fatalf("project creation was blocked by remote access: %s", result.Msg)
	}
	var count int64
	if err := database.Model(&model.AIProject{}).Where("creator_id = ?", 7).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("stored projects = %d, err = %v", count, err)
	}
}
