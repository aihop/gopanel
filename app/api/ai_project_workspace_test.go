package api

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/token"
)

func withAIProjectBaseDir(t *testing.T) string {
	t.Helper()
	previousBaseDir := global.CONF.System.BaseDir
	baseDir := t.TempDir()
	global.CONF.System.BaseDir = baseDir
	t.Cleanup(func() { global.CONF.System.BaseDir = previousBaseDir })
	return baseDir
}

func createAIProjectSourceDir(t *testing.T, parent, name string) string {
	t.Helper()
	directory := filepath.Join(parent, name)
	if err := os.MkdirAll(directory, 0755); err != nil {
		t.Fatal(err)
	}
	resolved, err := filepath.EvalSymlinks(directory)
	if err != nil {
		t.Fatal(err)
	}
	return resolved
}

func TestSyncAIProjectWorkspaceCreatesDeduplicatedLinks(t *testing.T) {
	baseDir := withAIProjectBaseDir(t)
	sourcesRoot := t.TempDir()
	first := createAIProjectSourceDir(t, filepath.Join(sourcesRoot, "first"), "app")
	second := createAIProjectSourceDir(t, filepath.Join(sourcesRoot, "second"), "app")
	project := &model.AIProject{ID: 12, CreatorID: 7}

	workspaceDir, err := syncAIProjectWorkspace(project, []string{first, second})
	if err != nil {
		t.Fatal(err)
	}
	resolvedBaseDir, err := filepath.EvalSymlinks(baseDir)
	if err != nil {
		t.Fatal(err)
	}
	wantWorkspace := filepath.Join(resolvedBaseDir, "code", "user_7", "project_12")
	if workspaceDir != wantWorkspace {
		t.Fatalf("workspaceDir = %q, want %q", workspaceDir, wantWorkspace)
	}
	manifest, err := readAIProjectWorkspaceManifest(workspaceDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(manifest.Sources) != 2 || manifest.Sources[0].LinkName != "app" || manifest.Sources[1].LinkName != "app-2" {
		t.Fatalf("unexpected workspace sources: %#v", manifest.Sources)
	}
	for _, source := range manifest.Sources {
		target, err := filepath.EvalSymlinks(filepath.Join(workspaceDir, source.LinkName))
		if err != nil || target != source.Path {
			t.Fatalf("link %q target = %q, err = %v", source.LinkName, target, err)
		}
	}
}

func TestSyncAIProjectWorkspacePreservesUnmanagedFiles(t *testing.T) {
	withAIProjectBaseDir(t)
	sourcesRoot := t.TempDir()
	source := createAIProjectSourceDir(t, filepath.Join(sourcesRoot, "first"), "api")
	second := createAIProjectSourceDir(t, filepath.Join(sourcesRoot, "second"), "web")
	project := &model.AIProject{ID: 3, CreatorID: 2}
	workspaceDir := aiProjectWorkspaceDir(project.CreatorID, project.ID)
	if err := os.MkdirAll(workspaceDir, 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspaceDir, "api"), []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := syncAIProjectWorkspace(project, []string{source, second}); err != nil {
		t.Fatal(err)
	}
	manifest, err := readAIProjectWorkspaceManifest(workspaceDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(manifest.Sources) != 2 || manifest.Sources[0].LinkName != "api-2" || manifest.Sources[1].LinkName != "web" {
		t.Fatalf("unexpected workspace sources: %#v", manifest.Sources)
	}
	content, err := os.ReadFile(filepath.Join(workspaceDir, "api"))
	if err != nil || string(content) != "keep" {
		t.Fatalf("unmanaged file changed: %q, %v", content, err)
	}
}

func TestSyncAIProjectWorkspaceUsesSourceDirectlyForSingleDirectory(t *testing.T) {
	withAIProjectBaseDir(t)
	source := createCodeGitRepository(t)
	project := &model.AIProject{ID: 8, CreatorID: 4}

	workspaceDir, err := syncAIProjectWorkspace(project, []string{source})
	if err != nil {
		t.Fatal(err)
	}
	if workspaceDir != source {
		t.Fatalf("workspaceDir = %q, want source %q", workspaceDir, source)
	}
	if err := os.WriteFile(filepath.Join(workspaceDir, "result.txt"), []byte("done\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := runCodeGit(workspaceDir, "add", "result.txt"); err != nil {
		t.Fatalf("single-directory workspace must support Git staging: %v", err)
	}
	if _, err := runCodeGit(workspaceDir, "-c", "user.name=GoPanel Test", "-c", "user.email=test@gopanel.local", "commit", "-m", "result"); err != nil {
		t.Fatalf("single-directory workspace must support Git commits: %v", err)
	}
	if _, err := os.Lstat(aiProjectWorkspaceDir(project.CreatorID, project.ID)); !os.IsNotExist(err) {
		t.Fatalf("single-directory project created a managed link workspace: %v", err)
	}
}

func TestAIProjectSessionWorkDirMigratesLegacySingleDirectoryProject(t *testing.T) {
	withAIProjectBaseDir(t)
	source := createAIProjectSourceDir(t, t.TempDir(), "project")
	project := &model.AIProject{
		ID: 9, CreatorID: 5, SourceDirs: []string{source},
		WorkDir: aiProjectWorkspaceDir(5, 9),
	}

	workDir, err := aiProjectSessionWorkDir(project, &token.CustomClaims{Role: constant.UserRoleSuper})
	if err != nil {
		t.Fatal(err)
	}
	if workDir != source {
		t.Fatalf("workDir = %q, want source %q", workDir, source)
	}
}

func TestNormalizeAIProjectSourceDirsEnforcesUserBoundary(t *testing.T) {
	baseDir := withAIProjectBaseDir(t)
	allowedRoot := t.TempDir()
	allowed := createAIProjectSourceDir(t, allowedRoot, "project")
	claims := &token.CustomClaims{UserId: 5, Role: constant.UserRoleSubAdmin, FileBaseDir: allowedRoot}

	normalized, err := normalizeAIProjectSourceDirs([]string{allowed, allowed}, claims)
	if err != nil {
		t.Fatal(err)
	}
	if len(normalized) != 1 || normalized[0] != allowed {
		t.Fatalf("normalized = %#v", normalized)
	}
	if _, err := normalizeAIProjectSourceDirs([]string{t.TempDir()}, claims); err == nil {
		t.Fatal("expected a source outside the sub-admin workspace to be rejected")
	}
	if _, err := normalizeAIProjectSourceDirs([]string{baseDir}, &token.CustomClaims{Role: constant.UserRoleSuper}); err == nil {
		t.Fatal("expected the GoPanel base directory to be rejected")
	}
	managedDir := aiProjectWorkspaceDir(claims.UserId, 9)
	if err := os.MkdirAll(managedDir, 0750); err != nil {
		t.Fatal(err)
	}
	if err := writeAIProjectWorkspaceManifest(managedDir, aiProjectWorkspaceManifest{Version: 1}); err != nil {
		t.Fatal(err)
	}
	if err := validateAIProjectWorkDirForClaims(managedDir, claims); err != nil {
		t.Fatalf("managed workspace rejected: %v", err)
	}
	if !isAnyManagedAIProjectWorkDir(managedDir) {
		t.Fatal("expected the managed workspace to be recognized")
	}
	otherUserDir := aiProjectWorkspaceDir(6, 9)
	if err := os.MkdirAll(otherUserDir, 0750); err != nil {
		t.Fatal(err)
	}
	if err := writeAIProjectWorkspaceManifest(otherUserDir, aiProjectWorkspaceManifest{Version: 1}); err != nil {
		t.Fatal(err)
	}
	if err := validateAIProjectWorkDirForClaims(otherUserDir, claims); err == nil {
		t.Fatal("expected another user's managed workspace to be rejected")
	}
}
