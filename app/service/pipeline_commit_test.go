package service

import (
	"context"
	"os/exec"
	"strings"
	"testing"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
)

func TestNormalizePipelineExpectedCommit(t *testing.T) {
	valid := strings.Repeat("A", 40)
	got, err := normalizePipelineExpectedCommit("  " + valid + "  ")
	if err != nil {
		t.Fatal(err)
	}
	if got != strings.ToLower(valid) {
		t.Fatalf("got %q", got)
	}
	for _, value := range []string{"main", "HEAD~1", strings.Repeat("a", 39), strings.Repeat("g", 40)} {
		if _, err := normalizePipelineExpectedCommit(value); err == nil {
			t.Fatalf("expected %q to be rejected", value)
		}
	}
}

func TestStepCloneLocksExpectedCommit(t *testing.T) {
	sourceDir := t.TempDir()
	remoteDir := t.TempDir()
	workspace := t.TempDir() + "/workspace"
	gitInRepo(t, sourceDir, "init", "-q", "-b", "main")
	firstCommit := commitEmpty(t, sourceDir, "feat: first")
	secondCommit := commitEmpty(t, sourceDir, "feat: second")
	gitInRepo(t, remoteDir, "init", "--bare", "-q")
	gitInRepo(t, sourceDir, "remote", "add", "origin", remoteDir)
	gitInRepo(t, sourceDir, "push", "-q", "-u", "origin", "main")

	oldBaseDir := global.CONF.System.BaseDir
	global.CONF.System.BaseDir = t.TempDir()
	defer func() { global.CONF.System.BaseDir = oldBaseDir }()
	logger := GetPipelineLogger(900001)
	defer RemovePipelineLogger(900001)

	pipeline := &model.Pipeline{RepoUrl: remoteDir, Branch: "main", AuthType: "none"}
	service := &PipelineService{}
	actualCommit, _, err := service.stepClone(context.Background(), logger, pipeline, workspace, "", firstCommit)
	if err != nil {
		t.Fatal(err)
	}
	if actualCommit != firstCommit {
		t.Fatalf("built %s, want locked commit %s; branch head is %s", actualCommit, firstCommit, secondCommit)
	}
	if err := verifyPipelineExpectedCommit(context.Background(), workspace, firstCommit); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("git", "symbolic-ref", "-q", "HEAD")
	cmd.Dir = workspace
	if err := cmd.Run(); err == nil {
		t.Fatal("expected detached HEAD")
	}

	thirdCommit := commitEmpty(t, sourceDir, "feat: third")
	gitInRepo(t, sourceDir, "push", "-q", "origin", "main")
	actualCommit, _, err = service.stepClone(context.Background(), logger, pipeline, workspace, firstCommit, secondCommit)
	if err != nil {
		t.Fatal(err)
	}
	if actualCommit != secondCommit {
		t.Fatalf("built %s, want locked commit %s; branch head is %s", actualCommit, secondCommit, thirdCommit)
	}

	actualCommit, _, err = service.stepClone(context.Background(), logger, pipeline, workspace, secondCommit, "")
	if err != nil {
		t.Fatal(err)
	}
	if actualCommit != thirdCommit {
		t.Fatalf("manual run built %s, want branch head %s", actualCommit, thirdCommit)
	}
	branch := gitInRepo(t, workspace, "branch", "--show-current")
	if branch != "main" {
		t.Fatalf("manual run should restore configured branch, got %q", branch)
	}
}

func TestVerifyPipelineExpectedCommitDetectsDrift(t *testing.T) {
	dir := t.TempDir()
	gitInRepo(t, dir, "init", "-q", "-b", "main")
	expectedCommit := commitEmpty(t, dir, "feat: expected")
	commitEmpty(t, dir, "feat: drifted")
	if err := verifyPipelineExpectedCommit(context.Background(), dir, expectedCommit); err == nil {
		t.Fatal("expected commit drift to be rejected")
	}
}

func TestCheckoutPipelineCommitRejectsOtherBranch(t *testing.T) {
	dir := t.TempDir()
	gitInRepo(t, dir, "init", "-q", "-b", "main")
	commitEmpty(t, dir, "feat: main")
	gitInRepo(t, dir, "branch", "origin/main")
	gitInRepo(t, dir, "checkout", "-q", "-b", "other")
	otherCommit := commitEmpty(t, dir, "feat: other")
	runGit := func(cmd *exec.Cmd, _ string) error { return cmd.Run() }
	oldBaseDir := global.CONF.System.BaseDir
	global.CONF.System.BaseDir = t.TempDir()
	defer func() { global.CONF.System.BaseDir = oldBaseDir }()
	logger := GetPipelineLogger(900002)
	defer RemovePipelineLogger(900002)
	if err := checkoutPipelineCommitFromRef(context.Background(), logger, dir, otherCommit, "origin/main", runGit); err == nil {
		t.Fatal("expected commit outside configured branch to be rejected")
	}
}
