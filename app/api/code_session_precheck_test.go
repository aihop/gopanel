package api

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aihop/gopanel/app/model"
)

// 源仓有未提交变更且不允许快照时，异步初始化必然失败，
// 创建阶段就应该直接报错，而不是留下一条失败会话。
func TestValidateCodeSessionPrerequisitesRejectsDirtySource(t *testing.T) {
	repositoryDir := createCodeGitRepository(t)
	if err := os.WriteFile(filepath.Join(repositoryDir, "README.md"), []byte("dirty\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	project := &model.AIProject{ID: 1, Name: "p", SourceDirs: []string{repositoryDir}}

	if err := validateCodeSessionPrerequisites(project, false); err == nil {
		t.Fatal("dirty source repository should be rejected before the session is created")
	}
	// 允许包含未提交内容时，脏工作区不该阻断创建。
	if err := validateCodeSessionPrerequisites(project, true); err != nil {
		t.Fatalf("snapshot mode should tolerate a dirty source: %v", err)
	}
}

func TestDiscoverCodeRepositoryCandidatesCanSkipStatusScan(t *testing.T) {
	repositoryDir := createCodeGitRepository(t)
	if err := os.WriteFile(filepath.Join(repositoryDir, "README.md"), []byte("dirty\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	withoutStatus, err := discoverCodeRepositoryCandidatesWithStatus([]string{repositoryDir}, false)
	if err != nil || len(withoutStatus) != 1 {
		t.Fatalf("discover without status = %#v, %v", withoutStatus, err)
	}
	if withoutStatus[0].Dirty {
		t.Fatal("status-free discovery should not inspect dirty files")
	}
	withStatus, err := discoverCodeRepositoryCandidatesWithStatus([]string{repositoryDir}, true)
	if err != nil || len(withStatus) != 1 || !withStatus[0].Dirty {
		t.Fatalf("discover with status = %#v, %v", withStatus, err)
	}
}

func TestCodeSessionRemoteProbeCacheExpires(t *testing.T) {
	resetCodeSessionRemoteProbeCache()
	t.Cleanup(resetCodeSessionRemoteProbeCache)
	cacheKey := strings.Join([]string{"repo", "origin", "remote", "0"}, "\x00")
	now := time.Now()
	codeSessionRemoteProbes.Lock()
	codeSessionRemoteProbes.succeeded[cacheKey] = now.Add(time.Second)
	codeSessionRemoteProbes.Unlock()

	if !codeSessionRemoteProbeCached(cacheKey, now) {
		t.Fatal("successful remote probe should be reused before expiry")
	}
	if codeSessionRemoteProbeCached(cacheKey, now.Add(2*time.Second)) {
		t.Fatal("expired remote probe should not be reused")
	}
}

func TestCodeSessionWorktreeUsesParallelCheckout(t *testing.T) {
	args := codeSessionWorktreeAddArgs("code/task", "/tmp/worktree", "base")
	if got := strings.Join(args, " "); !strings.Contains(got, "-c checkout.workers=0 worktree add") {
		t.Fatalf("worktree add should enable parallel checkout, got %q", got)
	}
}

// 纯本地仓库没有远端，不该因为探测不到远端而被拦下。
func TestValidateCodeSessionPrerequisitesAllowsLocalOnlyRepository(t *testing.T) {
	repositoryDir := createCodeGitRepository(t)
	project := &model.AIProject{ID: 2, Name: "local", SourceDirs: []string{repositoryDir}}

	if err := validateCodeSessionPrerequisites(project, true); err != nil {
		t.Fatalf("local-only repository should pass the precheck: %v", err)
	}
}

// 远端地址不可达时必须在创建阶段就报错。
func TestValidateCodeSessionPrerequisitesRejectsUnreachableRemote(t *testing.T) {
	repositoryDir := createCodeGitRepository(t)
	missing := filepath.Join(t.TempDir(), "not-a-repo.git")
	if _, err := runCodeGit(repositoryDir, "remote", "add", "origin", missing); err != nil {
		t.Fatal(err)
	}
	project := &model.AIProject{ID: 3, Name: "broken", SourceDirs: []string{repositoryDir}}

	err := validateCodeSessionPrerequisites(project, true)
	if err == nil {
		t.Fatal("unreachable remote should be rejected before the session is created")
	}
	if got := err.Error(); got == "" {
		t.Fatal("precheck error should explain which repository failed")
	}
}

func TestCodeRepositoryProbeRemotePrefersOrigin(t *testing.T) {
	repositoryDir := createCodeGitRepository(t)
	if got := codeRepositoryProbeRemote(repositoryDir); got != "" {
		t.Fatalf("repository without remotes should yield no probe target, got %q", got)
	}
	if _, err := runCodeGit(repositoryDir, "remote", "add", "upstream", "https://example.invalid/a.git"); err != nil {
		t.Fatal(err)
	}
	if _, err := runCodeGit(repositoryDir, "remote", "add", "origin", "https://example.invalid/b.git"); err != nil {
		t.Fatal(err)
	}
	if got := codeRepositoryProbeRemote(repositoryDir); got != "origin" {
		t.Fatalf("origin should win, got %q", got)
	}
}
