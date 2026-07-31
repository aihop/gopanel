package api

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestResolveCodexWritableDirsResolvesSymlinksAndDeduplicates(t *testing.T) {
	realDir := t.TempDir()
	linkDir := filepath.Join(t.TempDir(), "project")
	if err := os.Symlink(realDir, linkDir); err != nil {
		t.Fatal(err)
	}
	expectedDir, err := filepath.EvalSymlinks(realDir)
	if err != nil {
		t.Fatal(err)
	}

	resolvedDirs, err := resolveCodexWritableDirs([]string{linkDir, realDir})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(resolvedDirs, []string{expectedDir}) {
		t.Fatalf("resolved writable dirs = %#v, want %#v", resolvedDirs, []string{expectedDir})
	}
}

func TestAddCodexWritableDirArgsBeforeSubcommand(t *testing.T) {
	args := []string{"--ask-for-approval", "on-request", "--sandbox", "workspace-write", "exec", "--json", "prompt"}
	got := addCodexWritableDirArgs(args, []string{"/code/one", "/code/two"})
	want := []string{
		"--ask-for-approval", "on-request",
		"--sandbox", "workspace-write",
		"--add-dir", "/code/one",
		"--add-dir", "/code/two",
		"exec", "--json", "prompt",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Codex args = %#v, want %#v", got, want)
	}
}

func TestAddCodexWritableDirArgsToInteractiveCommand(t *testing.T) {
	args := []string{"--sandbox", "workspace-write", "--cd", "/workspace"}
	got := addCodexWritableDirArgs(args, []string{"/code/project"})
	want := []string{"--sandbox", "workspace-write", "--cd", "/workspace", "--add-dir", "/code/project"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("interactive Codex args = %#v, want %#v", got, want)
	}
}

func TestCodexWritableDirsForIsolatedWorktree(t *testing.T) {
	withAIProjectBaseDir(t)
	repositoryDir := createCodeGitRepository(t)
	session := &model.AIDevSession{ID: 31, UserID: 7, ProjectID: 9}
	if err := createCodeSessionWorktree(session, &model.AIGroup{SourceDirs: []string{repositoryDir}}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { rollbackCodeSessionWorktree(session) })

	writableDirs, err := codexWritableDirsForSession(session)
	if err != nil {
		t.Fatal(err)
	}
	gitDir, err := resolveCodeGitPath(session.WorkDir, "--git-dir")
	if err != nil {
		t.Fatal(err)
	}
	commonDir, err := resolveCodeGitPath(session.WorkDir, "--git-common-dir")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		gitDir,
		filepath.Join(commonDir, "objects"),
		filepath.Join(commonDir, "refs"),
		filepath.Join(commonDir, "logs"),
	}
	if !reflect.DeepEqual(writableDirs, want) {
		t.Fatalf("isolated worktree writable dirs = %#v, want %#v", writableDirs, want)
	}
}

func TestCodexWorktreeWritableDirsRejectTamperedSession(t *testing.T) {
	withAIProjectBaseDir(t)
	repositoryDir := createCodeGitRepository(t)
	session := &model.AIDevSession{ID: 32, UserID: 7, ProjectID: 9}
	if err := createCodeSessionWorktree(session, &model.AIGroup{SourceDirs: []string{repositoryDir}}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { rollbackCodeSessionWorktree(session) })

	tests := []struct {
		name   string
		mutate func(*model.AIDevSession)
	}{
		{name: "managed worktree", mutate: func(candidate *model.AIDevSession) { candidate.WorkDir = t.TempDir() }},
		{name: "session ID", mutate: func(candidate *model.AIDevSession) { candidate.ID++ }},
		{name: "source repository", mutate: func(candidate *model.AIDevSession) { candidate.SourceWorkDir = t.TempDir() }},
		{name: "worktree branch", mutate: func(candidate *model.AIDevSession) { candidate.WorktreeBranch += "-tampered" }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			candidate := *session
			test.mutate(&candidate)
			if _, err := codexWritableDirsForSession(&candidate); err == nil {
				t.Fatal("tampered Worktree session should be rejected")
			}
		})
	}
}

func TestCodexWorktreeWritableDirsAllowGitCommit(t *testing.T) {
	withAIProjectBaseDir(t)
	repositoryDir := createCodeGitRepository(t)
	session := &model.AIDevSession{ID: 33, UserID: 7, ProjectID: 9}
	if err := createCodeSessionWorktree(session, &model.AIGroup{SourceDirs: []string{repositoryDir}}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { rollbackCodeSessionWorktree(session) })
	writableDirs, err := codexWritableDirsForSession(session)
	if err != nil {
		t.Fatal(err)
	}
	commonDir, err := resolveCodeGitPath(session.WorkDir, "--git-common-dir")
	if err != nil {
		t.Fatal(err)
	}
	setGitTreePermissions(t, commonDir, 0555, 0444)
	t.Cleanup(func() { setGitTreePermissions(t, commonDir, 0755, 0644) })
	for _, writableDir := range writableDirs {
		setGitTreePermissions(t, writableDir, 0755, 0644)
	}
	if err := os.WriteFile(filepath.Join(session.WorkDir, "result.txt"), []byte("finished\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := runCodeGit(session.WorkDir, "add", "result.txt"); err != nil {
		t.Fatalf("git add with restricted metadata failed: %v", err)
	}
	if _, err := runCodeGit(session.WorkDir, "-c", "user.name=GoPanel Test", "-c", "user.email=test@gopanel.local", "commit", "-m", "result"); err != nil {
		t.Fatalf("git commit with restricted metadata failed: %v", err)
	}
}

func setGitTreePermissions(t *testing.T, root string, directoryMode, fileMode os.FileMode) {
	t.Helper()
	if err := filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		mode := fileMode
		if info.IsDir() {
			mode = directoryMode
		}
		return os.Chmod(path, mode)
	}); err != nil {
		t.Fatal(err)
	}
}

func assertCodexWritableDirArgs(t *testing.T, args, writableDirs []string) {
	t.Helper()
	for _, writableDir := range writableDirs {
		found := false
		for index := 0; index+1 < len(args); index++ {
			if args[index] == "--add-dir" && args[index+1] == writableDir {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("Codex command missing writable directory %q: %#v", writableDir, args)
		}
	}
}

func TestCodexWritableDirsForProjectSession(t *testing.T) {
	realDir := t.TempDir()
	linkDir := filepath.Join(t.TempDir(), "project")
	if err := os.Symlink(realDir, linkDir); err != nil {
		t.Fatal(err)
	}
	expectedDir, err := filepath.EvalSymlinks(realDir)
	if err != nil {
		t.Fatal(err)
	}
	database, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "project.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.AutoMigrate(&model.AIGroup{}); err != nil {
		t.Fatal(err)
	}
	oldDatabase := global.DB
	global.DB = database
	t.Cleanup(func() { global.DB = oldDatabase })
	project := &model.AIGroup{Name: "project", SourceDirs: []string{linkDir}, CreatorID: 1}
	if err := database.Create(project).Error; err != nil {
		t.Fatal(err)
	}

	writableDirs, err := codexWritableDirsForSession(&model.AIDevSession{ProjectID: project.ID})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(writableDirs, []string{expectedDir}) {
		t.Fatalf("project writable dirs = %#v, want %#v", writableDirs, []string{expectedDir})
	}
}
