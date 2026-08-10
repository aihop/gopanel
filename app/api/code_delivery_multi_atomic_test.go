package api

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
)

func TestMultiRepositoryDeliveryDoesNotRunQualityChecks(t *testing.T) {
	session, project, _ := createMultiRepositorySession(t, 940)
	if err := global.DB.Model(project).Update("require_quality_gate", true).Error; err != nil {
		t.Fatal(err)
	}
	repositories, err := loadCodeSessionRepositories(session.ID)
	if err != nil || len(repositories) == 0 {
		t.Fatalf("load repositories: %#v, %v", repositories, err)
	}
	repository := &repositories[0]
	observation := filepath.Join(t.TempDir(), "quality-heads.txt")
	packageJSON := `{"scripts":{"test":"node verify.js"}}`
	verifyJS := "const fs=require('fs');const {execFileSync}=require('child_process');" +
		"fs.appendFileSync(" + quotedCodeTestJS(observation) + ",execFileSync('git',['rev-parse','HEAD'],{encoding:'utf8'}));process.exit(1);\n"
	writeAndCommitCodeTestFiles(t, repository.WorktreeDir, map[string]string{
		"package.json": packageJSON, "verify.js": verifyJS,
	})
	job, err := persistCodeDeliveryJob(session, session.UserID, "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	commitCodeTestFile(t, repository.SourceDir, "source-only.txt", "source\n")
	previousCoordinator := codeExecutions
	codeExecutions = newCodeExecutionCoordinator(2)
	t.Cleanup(func() { codeExecutions = previousCoordinator })
	runner := &codeDeliveryRunner{
		queued: make(map[uint]struct{}), cancelled: make(map[uint]struct{}), owner: newCodeRepositoryLeaseOwner("multi-final-quality"),
	}
	runner.run(job.ID)
	if err := global.DB.First(job, job.ID).Error; err != nil {
		t.Fatal(err)
	}
	if job.Status != codeDeliveryJobCompleted || job.Stage != codeDeliveryStageCompleted || job.FailureCode != "" {
		t.Fatalf("quality check blocked multi-repository delivery: %#v", job)
	}
	if _, err := os.Stat(observation); !os.IsNotExist(err) {
		t.Fatalf("Git delivery executed quality script: %v", err)
	}
}

func TestGitlinkMultiRepositoryDeliveryUsesLocalChildTree(t *testing.T) {
	session, parentSource, childSource := createCodeGitlinkDeliverySession(t, 941, true)
	repositories, err := loadCodeSessionRepositories(session.ID)
	if err != nil {
		t.Fatal(err)
	}
	parent := codeTestRepositoryBySource(t, repositories, parentSource)
	child := codeTestRepositoryBySource(t, repositories, childSource)
	parentBefore, _ := runCodeGit(parent.SourceDir, "rev-parse", "HEAD")
	childBefore, _ := runCodeGit(child.SourceDir, "rev-parse", "HEAD")
	updater := cloneCodeRepository(t, codeTestRemoteURL(t, child.SourceDir))
	remoteCommit := commitCodeTestFile(t, updater, "remote-only.txt", "remote\n")
	if _, err := runCodeGit(updater, "push", "origin", "HEAD"); err != nil {
		t.Fatal(err)
	}
	commitCodeTestFile(t, child.WorktreeDir, "delivery.txt", "child-final\n")
	writeAndCommitCodeTestFiles(t, parent.WorktreeDir, map[string]string{
		"package.json": `{"scripts":{"test":"node verify.js"}}`,
		"verify.js":    "const fs=require('fs');if(fs.readFileSync('themes/custom/delivery.txt','utf8')!=='child-final\\n')process.exit(1);\n",
	})
	if err := captureCodeMultiRepositoryDeliverySnapshot(session); err != nil {
		t.Fatal(err)
	}
	prepared, err := prepareCodeMultiRepositoryDeliveryWithProgress(session, nil)
	if err != nil || prepared.Status != codeDeliveryMerged {
		t.Fatalf("prepare gitlink delivery: %#v, %v", prepared, err)
	}
	stored, err := loadCodeSessionRepositories(session.ID)
	if err != nil {
		t.Fatal(err)
	}
	parent = codeTestRepositoryBySource(t, stored, parentSource)
	child = codeTestRepositoryBySource(t, stored, childSource)
	if child.SourceCommit != childBefore || child.RemoteCommit == remoteCommit ||
		child.IntegrationWorkDir != filepath.Join(parent.IntegrationWorkDir, "themes", "custom") {
		t.Fatalf("unexpected nested integration metadata: parent=%#v child=%#v", parent, child)
	}
	for path, expected := range map[string]string{"delivery.txt": "child-final\n"} {
		content, readErr := os.ReadFile(filepath.Join(parent.IntegrationWorkDir, "themes", "custom", path))
		if readErr != nil || string(content) != expected {
			t.Fatalf("final child tree %s = %q, %v", path, content, readErr)
		}
	}
	if _, err := os.Stat(filepath.Join(parent.IntegrationWorkDir, "themes", "custom", "remote-only.txt")); !os.IsNotExist(err) {
		t.Fatalf("local delivery unexpectedly included remote-only change: %v", err)
	}
	entry, err := runCodeGit(parent.IntegrationWorkDir, "ls-tree", parent.MergeCommit, "--", "themes/custom")
	if err != nil || !strings.Contains(entry, child.MergeCommit) {
		t.Fatalf("parent merge gitlink = %q, want %s: %v", entry, child.MergeCommit, err)
	}
	if err := runCodeDeliveryQualityGate(session, session.UserID, nil, nil); err != nil {
		t.Fatal(err)
	}
	assertCodeTestSourceState(t, parent, parentBefore)
	assertCodeTestSourceState(t, child, childBefore)
	if _, err := publishCodeMultiRepositoryDeliveryWithProgress(session, nil); err != nil {
		t.Fatal(err)
	}
	for _, repository := range []*model.AIDevSessionRepository{parent, child} {
		status, statusErr := runCodeGit(repository.SourceDir, "status", "--porcelain")
		if statusErr != nil || strings.TrimSpace(status) != "" {
			t.Fatalf("source repository %s is dirty: %q, %v", repository.LinkName, status, statusErr)
		}
	}
}

func TestMultiRepositoryManualPushRejectsAdvancedRemoteBeforeAnyPush(t *testing.T) {
	session, _ := createRemoteCodeMultiRepositorySession(t, 942)
	repositories, err := loadCodeSessionRepositories(session.ID)
	if err != nil {
		t.Fatal(err)
	}
	for index := range repositories {
		commitCodeTestFile(t, repositories[index].WorktreeDir, "delivery.txt", repositories[index].LinkName+"\n")
	}
	if err := captureCodeMultiRepositoryDeliverySnapshot(session); err != nil {
		t.Fatal(err)
	}
	if _, err := prepareCodeMultiRepositoryDeliveryWithProgress(session, nil); err != nil {
		t.Fatal(err)
	}
	prepared, err := loadCodeSessionRepositories(session.ID)
	if err != nil {
		t.Fatal(err)
	}
	prepared, err = codeDeliveryRepositoriesInOrder(prepared, false)
	if err != nil {
		t.Fatal(err)
	}
	remoteHeads := make(map[uint]string, len(prepared))
	for index := range prepared {
		remoteHeads[prepared[index].ID] = prepared[index].RemoteCommit
	}
	advanced := &prepared[len(prepared)-1]
	updater := cloneCodeRepository(t, codeTestRemoteURL(t, advanced.SourceDir))
	commitCodeTestFile(t, updater, "advanced.txt", "advanced\n")
	if _, err := runCodeGit(updater, "push", "origin", "HEAD"); err != nil {
		t.Fatal(err)
	}
	if _, err = publishCodeMultiRepositoryDeliveryWithProgress(session, nil); err != nil {
		t.Fatalf("local delivery was blocked by remote advance: %v", err)
	}
	_, err = pushCodeSessionDelivery(session)
	if !errors.Is(err, errCodePushRemoteAdvanced) {
		t.Fatalf("remote advance should block manual push: %v", err)
	}
	stored, err := loadCodeSessionRepositories(session.ID)
	if err != nil {
		t.Fatal(err)
	}
	for index := range stored {
		assertCodeTestSourceState(t, &stored[index], stored[index].MergeCommit)
		if stored[index].ID != advanced.ID {
			remoteHead, remoteErr := codeTestRemoteHead(t, stored[index].SourceDir)
			if remoteErr != nil || remoteHead != remoteHeads[stored[index].ID] {
				t.Fatalf("repository %s was pushed before full preflight: got=%q want=%q err=%v", stored[index].LinkName, remoteHead, remoteHeads[stored[index].ID], remoteErr)
			}
		}
		if stored[index].Status != codeDeliveryCompleted || stored[index].PushedCommit != "" {
			t.Fatalf("local delivery state changed after failed push preflight: %#v", stored[index])
		}
	}
}

func createCodeGitlinkDeliverySession(
	t *testing.T,
	sessionID uint,
	withRemote bool,
) (*model.AIDevSession, string, string) {
	t.Helper()
	database := withCodeGovernanceDB(t)
	withAIProjectBaseDir(t)
	parent, child := createGitlinkRepositoryTree(t)
	if withRemote {
		attachCodeTestRemote(t, parent)
		attachCodeTestRemote(t, child)
	}
	project := &model.AIProject{
		ID: sessionID, Name: "gitlink-atomic", CreatorID: 7, SourceDirs: []string{parent}, WorkDir: parent,
		RequireQualityGate: true,
	}
	if err := database.Create(project).Error; err != nil {
		t.Fatal(err)
	}
	session := &model.AIDevSession{ID: sessionID, UserID: 7, ProjectID: project.ID, WorkDir: parent}
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
	return session, parent, child
}

func createRemoteCodeMultiRepositorySession(t *testing.T, sessionID uint) (*model.AIDevSession, []string) {
	t.Helper()
	database := withCodeGovernanceDB(t)
	withAIProjectBaseDir(t)
	first, _ := createCodeRemoteRepository(t)
	second, _ := createCodeRemoteRepository(t)
	sourceDirs := []string{first, second}
	project := &model.AIProject{ID: sessionID, Name: "remote-multi", CreatorID: 7, SourceDirs: sourceDirs}
	workDir, err := syncAIProjectWorkspace(project, sourceDirs)
	if err != nil {
		t.Fatal(err)
	}
	project.WorkDir = workDir
	if err := database.Create(project).Error; err != nil {
		t.Fatal(err)
	}
	session := &model.AIDevSession{ID: sessionID, UserID: 7, ProjectID: project.ID, WorkDir: workDir}
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
	return session, sourceDirs
}

func attachCodeTestRemote(t *testing.T, repository string) {
	t.Helper()
	remoteDir := filepath.Join(t.TempDir(), "remote.git")
	if _, err := runCodeGit(filepath.Dir(remoteDir), "init", "--bare", remoteDir); err != nil {
		t.Fatal(err)
	}
	if _, err := runCodeGit(repository, "remote", "add", "origin", remoteDir); err != nil {
		t.Fatal(err)
	}
	branch, _ := runCodeGit(repository, "branch", "--show-current")
	if _, err := runCodeGit(repository, "push", "-u", "origin", branch); err != nil {
		t.Fatal(err)
	}
}

func writeAndCommitCodeTestFiles(t *testing.T, repository string, files map[string]string) string {
	t.Helper()
	paths := make([]string, 0, len(files))
	for path, content := range files {
		if err := os.WriteFile(filepath.Join(repository, path), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
		paths = append(paths, path)
	}
	if _, err := runCodeGit(repository, append([]string{"add", "--"}, paths...)...); err != nil {
		t.Fatal(err)
	}
	if _, err := runCodeGit(repository, "-c", "user.name=GoPanel Test", "-c", "user.email=test@gopanel.local", "commit", "-m", "test quality"); err != nil {
		t.Fatal(err)
	}
	commit, err := runCodeGit(repository, "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	return commit
}

func quotedCodeTestJS(value string) string {
	return "`" + strings.ReplaceAll(value, "`", "") + "`"
}

func codeTestRepositoryByID(t *testing.T, repositories []model.AIDevSessionRepository, id uint) *model.AIDevSessionRepository {
	t.Helper()
	for index := range repositories {
		if repositories[index].ID == id {
			return &repositories[index]
		}
	}
	t.Fatalf("repository %d not found", id)
	return nil
}

func codeTestRepositoryBySource(t *testing.T, repositories []model.AIDevSessionRepository, source string) *model.AIDevSessionRepository {
	t.Helper()
	for index := range repositories {
		if repositories[index].SourceDir == source {
			return &repositories[index]
		}
	}
	t.Fatalf("repository %s not found", source)
	return nil
}

func codeTestRemoteURL(t *testing.T, repository string) string {
	t.Helper()
	remote, err := runCodeGit(repository, "remote", "get-url", "origin")
	if err != nil {
		t.Fatal(err)
	}
	return remote
}

func codeTestRemoteHead(t *testing.T, repository string) (string, error) {
	t.Helper()
	branch, err := runCodeGit(repository, "branch", "--show-current")
	if err != nil {
		return "", err
	}
	remote := codeTestRemoteURL(t, repository)
	output, err := runCodeGit(repository, "ls-remote", remote, "refs/heads/"+branch)
	if err != nil {
		return "", err
	}
	return strings.Fields(output)[0], nil
}

func assertCodeTestSourceState(t *testing.T, repository *model.AIDevSessionRepository, expectedHead string) {
	t.Helper()
	head, err := runCodeGit(repository.SourceDir, "rev-parse", "HEAD")
	if err != nil || head != expectedHead {
		t.Fatalf("source repository %s head=%q want=%q err=%v", repository.LinkName, head, expectedHead, err)
	}
	status, err := runCodeGit(repository.SourceDir, "status", "--porcelain")
	if err != nil || strings.TrimSpace(status) != "" {
		t.Fatalf("source repository %s dirty: %q, %v", repository.LinkName, status, err)
	}
}
