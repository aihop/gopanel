package api

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aihop/gopanel/app/model"
)

func TestParseCodeGitStatus(t *testing.T) {
	files := parseCodeGitStatus(" M main.go\x00A  staged.go\x00?? notes.txt\x00R  renamed.go\x00old.go\x00", "source")
	if len(files) != 4 {
		t.Fatalf("file count = %d, want 4: %#v", len(files), files)
	}
	if !files[0].Changed || files[0].Staged || files[0].WorkspacePath != "source/main.go" {
		t.Fatalf("unexpected changed file: %#v", files[0])
	}
	if !files[1].Staged || files[1].Changed || files[1].Untracked {
		t.Fatalf("unexpected staged file: %#v", files[1])
	}
	if !files[2].Untracked || files[2].Staged || files[2].Changed {
		t.Fatalf("unexpected untracked file: %#v", files[2])
	}
	if files[3].OldPath != "old.go" || files[3].Path != "renamed.go" {
		t.Fatalf("unexpected renamed file: %#v", files[3])
	}
}

func TestParseCodeGitNumstatSkipsBinaryFiles(t *testing.T) {
	additions, deletions := parseCodeGitNumstat("12\t3\tsrc/main.go\n-\t-\tlogo.png\n2\t0\tREADME.md\n")
	if additions != 14 || deletions != 3 {
		t.Fatalf("numstat = +%d/-%d, want +14/-3", additions, deletions)
	}
}

func TestLoadCodeGitStatusAndDiff(t *testing.T) {
	repositoryDir := createCodeGitRepository(t)
	if err := os.WriteFile(filepath.Join(repositoryDir, "README.md"), []byte("test\nchanged\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repositoryDir, "new.txt"), []byte("new\n"), 0600); err != nil {
		t.Fatal(err)
	}
	session := &model.AIDevSession{WorkDir: repositoryDir}
	status, err := loadCodeGitStatus(session, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !status.Available || status.Files != 2 || status.Changed != 1 || status.Untracked != 1 {
		t.Fatalf("unexpected status: %#v", status)
	}
	repository := status.Repositories[0]
	file, err := findCodeGitFile(repository, "README.md", "working")
	if err != nil {
		t.Fatal(err)
	}
	diff, truncated, err := loadCodeGitDiff(repository, *file, "working")
	if err != nil || truncated || !strings.Contains(diff, "+changed") {
		t.Fatalf("unexpected diff: truncated=%v err=%v output=%q", truncated, err, diff)
	}
	untracked, err := findCodeGitFile(repository, "new.txt", "working")
	if err != nil {
		t.Fatal(err)
	}
	diff, truncated, err = loadCodeGitDiff(repository, *untracked, "working")
	if err != nil || truncated || !strings.Contains(diff, "+new") {
		t.Fatalf("unexpected untracked diff: truncated=%v err=%v output=%q", truncated, err, diff)
	}
}

func TestLoadCodeGitStatusShowsSavedCommits(t *testing.T) {
	session, _ := createDeliveryWorktree(t, 146)
	if err := os.WriteFile(filepath.Join(session.WorkDir, "saved.txt"), []byte("saved\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := saveCodeSessionWorktree(session, "test: saved state"); err != nil {
		t.Fatal(err)
	}
	status, err := loadCodeGitStatus(session, nil, nil)
	if err != nil || len(status.Repositories) != 1 {
		t.Fatalf("load saved status: %#v, %v", status, err)
	}
	repository := status.Repositories[0]
	if repository.SavedCommits != 1 || len(repository.HeadCommit) != 8 || len(repository.Files) != 0 {
		t.Fatalf("unexpected saved state: %#v", repository)
	}
}

func TestLoadCodeGitStatusHidesRepositoryExcludedAfterSessionCreation(t *testing.T) {
	session, _, sourceDirs := createMultiRepositorySession(t, 147)
	repositories, err := loadCodeSessionRepositories(session.ID)
	if err != nil || len(repositories) != 2 {
		t.Fatalf("load repositories: %#v, %v", repositories, err)
	}
	excluded := sourceDirs[0]
	status, err := loadCodeGitStatus(session, sourceDirs, []string{excluded})
	if err != nil {
		t.Fatal(err)
	}
	if len(status.Repositories) != 1 {
		t.Fatalf("excluded historical repository remained visible: %#v", status.Repositories)
	}
	for _, repository := range repositories {
		if repository.SourceDir != excluded {
			continue
		}
		if _, err := findCodeGitRepository(
			discoverCodeGitRepositories(session, sourceDirs, []string{excluded}),
			codeSessionRepositoryID(repository.ID),
		); err == nil {
			t.Fatal("excluded repository remained addressable by repository ID")
		}
	}
}

func TestCodeGitStageAndUnstage(t *testing.T) {
	repositoryDir := createCodeGitRepository(t)
	if err := os.WriteFile(filepath.Join(repositoryDir, "README.md"), []byte("updated\n"), 0600); err != nil {
		t.Fatal(err)
	}
	repository, ok := inspectCodeGitRepository("session", "test", repositoryDir, "")
	if !ok {
		t.Fatal("repository not detected")
	}
	if _, err := runCodeGit(repository.root, "add", "--", "README.md"); err != nil {
		t.Fatal(err)
	}
	status, err := loadCodeGitRepositoryStatus(repository)
	if err != nil || status.StagedCount != 1 || status.ChangedCount != 0 {
		t.Fatalf("unexpected staged status: %#v, %v", status, err)
	}
	if _, err := runCodeGit(repository.root, "reset", "--quiet", "HEAD", "--", "README.md"); err != nil {
		t.Fatal(err)
	}
	status, err = loadCodeGitRepositoryStatus(repository)
	if err != nil || status.StagedCount != 0 || status.ChangedCount != 1 {
		t.Fatalf("unexpected unstaged status: %#v, %v", status, err)
	}
}

func TestCodeGitHandlesLiteralPathsAndSplitChanges(t *testing.T) {
	repositoryDir := createCodeGitRepository(t)
	specialPath := ":notes with space.txt"
	if err := os.WriteFile(filepath.Join(repositoryDir, specialPath), []byte("first\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := runCodeGit(repositoryDir, "--literal-pathspecs", "add", "--", specialPath); err != nil {
		t.Fatal(err)
	}
	if _, err := runCodeGit(repositoryDir, "-c", "user.name=GoPanel Test", "-c", "user.email=test@gopanel.local", "commit", "-m", "special path"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repositoryDir, specialPath), []byte("first\nstaged\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := runCodeGit(repositoryDir, "--literal-pathspecs", "add", "--", specialPath); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repositoryDir, specialPath), []byte("first\nstaged\nworking\n"), 0600); err != nil {
		t.Fatal(err)
	}
	repository, ok := inspectCodeGitRepository("session", "test", repositoryDir, "")
	if !ok {
		t.Fatal("repository not detected")
	}
	status, err := loadCodeGitRepositoryStatus(repository)
	if err != nil || len(status.Files) != 1 || !status.Files[0].Staged || !status.Files[0].Changed {
		t.Fatalf("unexpected split status: %#v, %v", status, err)
	}
	stagedDiff, _, err := loadCodeGitDiff(repository, status.Files[0], "staged")
	if err != nil || !strings.Contains(stagedDiff, "+staged") || strings.Contains(stagedDiff, "+working") {
		t.Fatalf("unexpected staged diff: %v, %q", err, stagedDiff)
	}
	workingDiff, _, err := loadCodeGitDiff(repository, status.Files[0], "working")
	if err != nil || !strings.Contains(workingDiff, "+working") || strings.Contains(workingDiff, "+staged") {
		t.Fatalf("unexpected working diff: %v, %q", err, workingDiff)
	}
}

func TestCodeGitUnstagesUnbornRepository(t *testing.T) {
	repositoryDir := t.TempDir()
	if _, err := runCodeGit(repositoryDir, "init"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repositoryDir, "first.txt"), []byte("first\n"), 0600); err != nil {
		t.Fatal(err)
	}
	repository, ok := inspectCodeGitRepository("session", "test", repositoryDir, "")
	if !ok {
		t.Fatal("unborn repository not detected")
	}
	if err := updateCodeGitPathsStage(repository, []string{"first.txt"}, true); err != nil {
		t.Fatal(err)
	}
	status, err := loadCodeGitRepositoryStatus(repository)
	if err != nil || status.StagedCount != 1 {
		t.Fatalf("unexpected staged status: %#v, %v", status, err)
	}
	if err := updateCodeGitPathsStage(repository, []string{"first.txt"}, false); err != nil {
		t.Fatal(err)
	}
	status, err = loadCodeGitRepositoryStatus(repository)
	if err != nil || status.StagedCount != 0 || status.UntrackedCount != 1 {
		t.Fatalf("unexpected unstaged status: %#v, %v", status, err)
	}
}

func TestCodeGitStatusCountsUntrackedFileAdditions(t *testing.T) {
	repositoryDir := t.TempDir()
	if _, err := runCodeGit(repositoryDir, "init"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repositoryDir, "new.txt"), []byte("one\ntwo\nthree\n"), 0600); err != nil {
		t.Fatal(err)
	}
	repository, ok := inspectCodeGitRepository("session", "test", repositoryDir, "")
	if !ok {
		t.Fatal("repository not detected")
	}
	status, err := loadCodeGitRepositoryStatus(repository)
	if err != nil {
		t.Fatal(err)
	}
	// git diff --numstat 看不见未跟踪文件，新建文件必须单独计入，否则新写的文件在 +/- 上是 0。
	if status.Additions != 3 {
		t.Fatalf("unexpected untracked additions: %#v", status)
	}
}
