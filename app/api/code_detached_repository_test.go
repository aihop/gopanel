package api

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aihop/gopanel/app/model"
)

func createDetachedMultiRepositorySession(t *testing.T, sessionID uint) (*model.AIDevSession, string) {
	t.Helper()
	database := withCodeGovernanceDB(t)
	withAIProjectBaseDir(t)
	primary := createCodeGitRepository(t)
	detached := createCodeGitRepository(t)
	primaryBranch, err := runCodeGit(primary, "branch", "--show-current")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runCodeGit(detached, "checkout", "--detach", "HEAD"); err != nil {
		t.Fatal(err)
	}
	project := &model.AIProject{
		ID: sessionID, Name: "detached", CreatorID: 7, SourceDirs: []string{primary, detached},
		PrimaryRepository: primary, DeliveryBranch: primaryBranch,
	}
	session := &model.AIDevSession{ID: sessionID, UserID: 7, ProjectID: project.ID, Title: "session"}
	if err := database.Create(project).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(session).Error; err != nil {
		t.Fatal(err)
	}
	if err := createCodeSessionWorktree(session, project); err != nil {
		t.Fatal(err)
	}
	if err := database.Save(session).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { rollbackCodeSessionWorktree(session) })
	return session, detached
}

func detachedSessionRepository(t *testing.T, sessionID uint, sourceDir string) model.AIDevSessionRepository {
	t.Helper()
	repositories, err := loadCodeSessionRepositories(sessionID)
	if err != nil {
		t.Fatal(err)
	}
	for _, repository := range repositories {
		if repository.SourceDir == sourceDir {
			return repository
		}
	}
	t.Fatalf("detached repository %s not found", sourceDir)
	return model.AIDevSessionRepository{}
}

func TestCodeProjectRepositorySpecsAllowDetachedSecondaryRepository(t *testing.T) {
	primary := createCodeGitRepository(t)
	detached := createCodeGitRepository(t)
	primaryBranch, _ := runCodeGit(primary, "branch", "--show-current")
	if _, err := runCodeGit(detached, "checkout", "--detach", "HEAD"); err != nil {
		t.Fatal(err)
	}
	project := &model.AIProject{
		SourceDirs: []string{primary, detached}, PrimaryRepository: primary, DeliveryBranch: primaryBranch,
	}
	specs, err := codeProjectRepositorySpecs(project)
	if err != nil {
		t.Fatal(err)
	}
	for _, spec := range specs {
		if spec.Path == detached {
			if spec.Branch != "" || spec.Remote != "" || inspectCodeProjectRepositorySync(spec).Status != "local" {
				t.Fatalf("unexpected detached repository spec: %#v", spec)
			}
			return
		}
	}
	t.Fatal("detached secondary repository was not discovered")
}

func TestValidateCodeProjectGitlinkTargetsAcceptsDetachedChildHEAD(t *testing.T) {
	parent, child := createGitlinkRepositoryTree(t)
	parentBranch, _ := runCodeGit(parent, "branch", "--show-current")
	if _, err := runCodeGit(child, "checkout", "--detach", "HEAD"); err != nil {
		t.Fatal(err)
	}
	specs := []codeProjectRepositorySpec{
		{Path: parent, Name: "parent", Branch: parentBranch},
		{Path: child, Name: "child", ParentPath: parent, GitlinkPath: "themes/custom"},
	}
	if err := validateCodeProjectGitlinkTargets(specs); err != nil {
		t.Fatalf("detached child HEAD was rejected: %v", err)
	}
}

func TestDetachedSecondaryRepositoryCreatesWorktreeAndSkipsUnchangedDelivery(t *testing.T) {
	session, detached := createDetachedMultiRepositorySession(t, 181)
	repository := detachedSessionRepository(t, session.ID, detached)
	head, _ := runCodeGit(detached, "rev-parse", "HEAD")
	if repository.TargetBranch != "" || repository.BaseCommit != head || repository.SyncStatus != "local" {
		t.Fatalf("unexpected detached repository metadata: %#v", repository)
	}
	if _, err := os.Stat(filepath.Join(repository.WorktreeDir, "README.md")); err != nil {
		t.Fatalf("detached repository Worktree unavailable: %v", err)
	}
	if err := captureCodeMultiRepositoryDeliverySnapshot(session); err != nil {
		t.Fatal(err)
	}
	repository = detachedSessionRepository(t, session.ID, detached)
	if repository.Status != codeDeliveryCompleted || repository.WorktreeCommit != repository.BaseCommit {
		t.Fatalf("unchanged detached repository was not skipped: %#v", repository)
	}
	result, err := resumeCodeMultiRepositoryDelivery(session, session.UserID)
	if err != nil || result.Status != "merged" {
		t.Fatalf("multi-repository delivery did not complete: %#v, %v", result, err)
	}
}

func TestDetachedSecondaryRepositoryRequiresBranchOnlyAfterNewCommit(t *testing.T) {
	session, detached := createDetachedMultiRepositorySession(t, 182)
	repository := detachedSessionRepository(t, session.ID, detached)
	commitCodeTestFile(t, repository.WorktreeDir, "detached.txt", "changed\n")
	err := captureCodeMultiRepositoryDeliverySnapshot(session)
	if err == nil || !strings.Contains(err.Error(), "detached HEAD") || !strings.Contains(err.Error(), "配置交付分支") {
		t.Fatalf("unexpected delivery error: %v", err)
	}
}
