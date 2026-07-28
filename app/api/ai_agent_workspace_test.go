package api

import (
	"path/filepath"
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
