package api

import (
	"os"
	"path/filepath"
	"reflect"
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

func TestWorkspaceGitMetadataMountsForMultiRepositorySession(t *testing.T) {
	session, _, _ := createMultiRepositorySession(t, 84)
	repositories, err := loadCodeSessionRepositories(session.ID)
	if err != nil {
		t.Fatal(err)
	}
	want := make([]string, 0, len(repositories))
	for _, repository := range repositories {
		commonDir, err := resolveCodeGitPath(repository.WorktreeDir, "--git-common-dir")
		if err != nil {
			t.Fatal(err)
		}
		want = append(want, commonDir)
	}
	got, err := workspaceGitMetadataMounts(session, session.WorkDir)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("metadata mounts = %#v, want %#v", got, want)
	}
	for _, metadataDir := range got {
		if _, err := os.Stat(metadataDir); err != nil {
			t.Fatalf("metadata mount unavailable: %s", metadataDir)
		}
	}
}

func TestWorkspaceContainerKeepsWorktreeAbsolutePathAvailable(t *testing.T) {
	workDir := "/srv/gopanel/worktrees/session_84"
	metadataDirs := []string{"/srv/project-one/.git", "/srv/project-two/.git"}
	args := buildAIAgentWorkspaceRunArgs("workspace", workDir)
	for _, metadataDir := range metadataDirs {
		args = append(args, "-v", metadataDir+":"+metadataDir)
	}
	args = append(args, "-v", workDir+":"+workDir)
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, workDir+":"+workDir) || !strings.Contains(joined, workDir+":/workspace") {
		t.Fatalf("worktree paths are not both available: %s", joined)
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
