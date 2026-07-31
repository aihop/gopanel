package api

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/utils/token"
)

func TestNormalizeAIAgentAuthorizedWorkDirKeepsSubAdminInBase(t *testing.T) {
	base := t.TempDir()
	workDir := filepath.Join(base, "project")
	claims := &token.CustomClaims{UserId: 7, Role: constant.UserRoleSubAdmin, FileBaseDir: base}

	got, err := normalizeAIAgentAuthorizedWorkDir(workDir, claims.UserId, claims)
	if err != nil {
		t.Fatal(err)
	}
	if got != workDir {
		t.Fatalf("workDir = %q, want %q", got, workDir)
	}
	if _, err := normalizeAIAgentAuthorizedWorkDir(filepath.Join(base, "..", "outside"), claims.UserId, claims); err == nil {
		t.Fatal("expected a work directory outside the sub-admin base to be rejected")
	}
}

func TestWorkspaceRunArgsDoNotExposeHostCredentials(t *testing.T) {
	args := strings.Join(buildAIAgentWorkspaceRunArgs("workspace", "/srv/project"), " ")
	for _, sensitivePath := range []string{".ssh", ".aws", ".npmrc", ".gitconfig", ".trae"} {
		if strings.Contains(args, sensitivePath) {
			t.Fatalf("workspace args expose host credential path %s: %s", sensitivePath, args)
		}
	}
	if !strings.Contains(args, "/srv/project:/workspace") {
		t.Fatalf("workspace mount missing: %s", args)
	}
}

func TestDetectSensitiveWorkspaceMounts(t *testing.T) {
	if !hasSensitiveWorkspaceMount("/workspace\n/root/.ssh\n") {
		t.Fatal("legacy SSH mount should trigger sandbox migration")
	}
	if hasSensitiveWorkspaceMount("/workspace\n/usr/local/bin\n") {
		t.Fatal("normal workspace mounts should not trigger migration")
	}
}
