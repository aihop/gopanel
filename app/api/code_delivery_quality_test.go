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

func createQualityDeliverySession(t *testing.T, sessionID uint, command string) *model.AIDevSession {
	t.Helper()
	database := withCodeGovernanceDB(t)
	session, _ := createDeliveryWorktree(t, sessionID)
	session.ProjectID, session.Status = sessionID, codeSessionStatusActive
	project := &model.AIProject{ID: session.ProjectID, Name: "quality-delivery", CreatorID: session.UserID, RequireQualityGate: true}
	if err := database.Create(project).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(session).Error; err != nil {
		t.Fatal(err)
	}
	packageJSON := `{"scripts":{"test":"` + command + `"}}`
	if err := os.WriteFile(filepath.Join(session.WorkDir, "package.json"), []byte(packageJSON), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := saveCodeSessionWorktree(session, "test: configure quality gate"); err != nil {
		t.Fatal(err)
	}
	if _, err := persistCodeDeliveryJob(session, session.UserID, "127.0.0.1"); err != nil {
		t.Fatal(err)
	}
	return session
}

func TestRunCodeDeliveryQualityGateRunsMissingChecks(t *testing.T) {
	session := createQualityDeliverySession(t, 147, "true")
	if err := runCodeDeliveryQualityGate(session, session.UserID, nil, nil); err != nil {
		t.Fatal(err)
	}
	if err := validateCodeQualityGate(session); err != nil {
		t.Fatalf("automatic quality result did not satisfy gate: %v", err)
	}
}

func TestDetectCodeDeliveryQualityChecksUsesSessionFlutterToolchain(t *testing.T) {
	deliveryDir := t.TempDir()
	identityDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(deliveryDir, "pubspec.yaml"), []byte("name: example\ndependencies:\n  flutter:\n    sdk: flutter\n"), 0600); err != nil {
		t.Fatal(err)
	}
	toolchain := filepath.Join(identityDir, ".toolchains", "flutter", "bin")
	if err := os.MkdirAll(toolchain, 0700); err != nil {
		t.Fatal(err)
	}
	flutterPath := filepath.Join(toolchain, "flutter")
	if err := os.WriteFile(flutterPath, []byte("#!/bin/sh\nexit 0\n"), 0700); err != nil {
		t.Fatal(err)
	}
	checks := detectCodeDeliveryQualityChecks(0, []codeDeliveryQualityRoot{{
		WorkDir: deliveryDir, IdentityDir: identityDir, Commit: "commit", Label: "Flutter",
	}})
	if len(checks) != 1 || checks[0].Command != "flutter analyze --no-pub" || checks[0].Executable != flutterPath ||
		len(checks[0].Args) != 2 || checks[0].Args[1] != "--no-pub" {
		t.Fatalf("unexpected Flutter delivery check: %#v", checks)
	}
}

func TestAutomaticCodeDeliveryQualityChecksSkipsBuild(t *testing.T) {
	detected := []codeQualityCheck{
		{Kind: "test", Command: "yarn test"},
		{Kind: "build", Command: "yarn build"},
	}
	checks := automaticCodeDeliveryQualityChecks(detected)
	if len(checks) != 1 || checks[0].Kind != "test" {
		t.Fatalf("automatic delivery checks should exclude build: %#v", checks)
	}
}

func TestDetectCodeDeliveryQualityChecksDoesNotAutoRunNodeBuild(t *testing.T) {
	repository := createCodeGitRepository(t)
	if err := os.WriteFile(filepath.Join(repository, "package.json"), []byte(`{"scripts":{"build":"nuxt build"}}`), 0600); err != nil {
		t.Fatal(err)
	}
	checks := detectCodeDeliveryQualityChecks(0, []codeDeliveryQualityRoot{{
		WorkDir: repository, IdentityDir: repository, RuntimeDir: repository, Commit: "commit",
	}})
	if len(checks) != 0 {
		t.Fatalf("detected Node build should not block Git delivery: %#v", checks)
	}
}

func TestDetectCodeDeliveryQualityChecksKeepsConfiguredBuild(t *testing.T) {
	database := withCodeGovernanceDB(t)
	repository := createCodeGitRepository(t)
	if err := os.WriteFile(filepath.Join(repository, "package.json"), []byte(`{"scripts":{"build":"true"}}`), 0600); err != nil {
		t.Fatal(err)
	}
	project := &model.AIProject{
		ID: 151, Name: "explicit-build", CreatorID: 1,
		QualityChecks: []model.AIProjectQualityCheck{{
			Name: "Production build", Kind: "build", Repository: repository, WorkDir: ".", Command: "yarn build",
		}},
	}
	if err := database.Create(project).Error; err != nil {
		t.Fatal(err)
	}
	var stored model.AIProject
	if err := database.First(&stored, project.ID).Error; err != nil || len(stored.QualityChecks) != 1 {
		t.Fatalf("configured build was not stored: %#v, %v", stored, err)
	}
	direct := configuredCodeQualityChecks(stored.QualityChecks, []codeDeliveryQualityRoot{{
		WorkDir: repository, IdentityDir: repository, RuntimeDir: repository, Commit: "commit",
	}})
	if len(direct) != 1 {
		resolved, _ := filepath.EvalSymlinks(repository)
		t.Fatalf("configured build did not map to root: repository=%q resolved=%q stored=%#v", repository, resolved, stored.QualityChecks)
	}
	checks := detectCodeDeliveryQualityChecks(project.ID, []codeDeliveryQualityRoot{{
		WorkDir: repository, IdentityDir: repository, RuntimeDir: repository, Commit: "commit",
	}})
	if len(checks) != 1 || checks[0].Kind != "build" || checks[0].Label != "Production build" {
		t.Fatalf("configured build was not kept: %#v", checks)
	}
}

func TestPrepareCodeDeliveryQualityEnvironmentRestoresNodeModules(t *testing.T) {
	sourceDir := createCodeGitRepository(t)
	deliveryDir := createCodeGitRepository(t)
	packagePath := "admin"
	for _, root := range []string{sourceDir, deliveryDir} {
		if err := os.Mkdir(filepath.Join(root, packagePath), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, packagePath, "package.json"), []byte("{}"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(deliveryDir, ".gitignore"), []byte("node_modules\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := runCodeGit(deliveryDir, "add", ".gitignore"); err != nil {
		t.Fatal(err)
	}
	if _, err := runCodeGit(deliveryDir, "-c", "user.name=GoPanel Test", "-c", "user.email=test@gopanel.local", "commit", "-m", "ignore dependencies"); err != nil {
		t.Fatal(err)
	}
	sourceModules := filepath.Join(sourceDir, packagePath, "node_modules")
	if err := os.Mkdir(sourceModules, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourceModules, "ready"), []byte("yes"), 0600); err != nil {
		t.Fatal(err)
	}
	deliveryModules := filepath.Join(deliveryDir, packagePath, "node_modules")
	if err := os.Mkdir(deliveryModules, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(deliveryModules, "cache"), []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	cleanup, err := prepareCodeDeliveryQualityEnvironment([]codeDeliveryQualityRoot{{
		WorkDir: deliveryDir, RuntimeDir: sourceDir,
	}})
	if err != nil {
		t.Fatal(err)
	}
	if target, err := filepath.EvalSymlinks(deliveryModules); err != nil || target != sourceModules {
		t.Fatalf("node_modules target=%q want=%q err=%v", target, sourceModules, err)
	}
	if err := cleanup(); err != nil {
		t.Fatal(err)
	}
	if content, err := os.ReadFile(filepath.Join(deliveryModules, "cache")); err != nil || string(content) != "keep" {
		t.Fatalf("delivery cache was not restored: %q, %v", content, err)
	}
}

func TestPrepareCodeDeliveryQualityEnvironmentRestoresNestedDartTool(t *testing.T) {
	sourceDir := createCodeGitRepository(t)
	mainRepositoryDir := createCodeGitRepository(t)
	deliveryDir := createCodeGitRepository(t)
	packagePath := filepath.Join("packages", "plugin", "example")
	for _, root := range []string{sourceDir, deliveryDir} {
		if err := os.MkdirAll(filepath.Join(root, packagePath), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, packagePath, "pubspec.yaml"), []byte("name: example\n"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(deliveryDir, ".gitignore"), []byte(".dart_tool/\n"), 0600); err != nil {
		t.Fatal(err)
	}
	sourceDartTool := filepath.Join(sourceDir, packagePath, ".dart_tool")
	if err := os.Mkdir(sourceDartTool, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourceDartTool, "package_config.json"), []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	deliveryDartTool := filepath.Join(deliveryDir, packagePath, ".dart_tool")
	if err := os.Mkdir(deliveryDartTool, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(deliveryDartTool, "keep"), []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(sourceDir, ".toolchains", "ignored", ".dart_tool"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(deliveryDir, ".toolchains", "ignored"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(deliveryDir, ".toolchains", "ignored", "pubspec.yaml"), []byte("name: ignored\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, ".toolchains", "ignored", "pubspec.yaml"), []byte("name: ignored\n"), 0600); err != nil {
		t.Fatal(err)
	}

	cleanup, err := prepareCodeDeliveryQualityEnvironment([]codeDeliveryQualityRoot{{
		WorkDir: deliveryDir, IdentityDir: sourceDir, RuntimeDir: mainRepositoryDir,
	}})
	if err != nil {
		t.Fatal(err)
	}
	if target, err := filepath.EvalSymlinks(deliveryDartTool); err != nil || target != sourceDartTool {
		t.Fatalf(".dart_tool target=%q want=%q err=%v", target, sourceDartTool, err)
	}
	if _, err := os.Lstat(filepath.Join(deliveryDir, ".toolchains", "ignored", ".dart_tool")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("ignored toolchain was traversed: %v", err)
	}
	if err := cleanup(); err != nil {
		t.Fatal(err)
	}
	if content, err := os.ReadFile(filepath.Join(deliveryDartTool, "keep")); err != nil || string(content) != "original" {
		t.Fatalf("delivery .dart_tool was not restored: %q, %v", content, err)
	}
}

func TestPrepareCodeDeliveryQualityEnvironmentSkipsTrackedDartTool(t *testing.T) {
	sourceDir := createCodeGitRepository(t)
	deliveryDir := createCodeGitRepository(t)
	for _, root := range []string{sourceDir, deliveryDir} {
		if err := os.WriteFile(filepath.Join(root, "pubspec.yaml"), []byte("name: example\n"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Mkdir(filepath.Join(sourceDir, ".dart_tool"), 0700); err != nil {
		t.Fatal(err)
	}
	cleanup, err := prepareCodeDeliveryQualityEnvironment([]codeDeliveryQualityRoot{{
		WorkDir: deliveryDir, RuntimeDir: sourceDir,
	}})
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	if _, err := os.Lstat(filepath.Join(deliveryDir, ".dart_tool")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("non-ignored .dart_tool should not be linked: %v", err)
	}
}

func TestChangedCodeDeliveryQualityRootRestoresTrackedChanges(t *testing.T) {
	workDir := createCodeGitRepository(t)
	commit, err := runCodeGit(workDir, "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	readme := filepath.Join(workDir, "README.md")
	if err := os.WriteFile(readme, []byte("changed\n"), 0600); err != nil {
		t.Fatal(err)
	}
	root := codeDeliveryQualityRoot{WorkDir: workDir, Commit: commit}
	check := newCodeQualityCheck("lint", "Lint", workDir, workDir, "true")
	changed := changedCodeDeliveryQualityRoot(check, []codeDeliveryQualityRoot{root})
	if changed == nil {
		t.Fatal("tracked quality mutation was not detected")
	}
	if err := restoreCodeDeliveryQualityRoot(*changed); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(readme)
	if err != nil || string(content) != "test\n" {
		t.Fatalf("quality snapshot was not restored: %q, %v", content, err)
	}
}

func TestRunCodeDeliveryQualityGateStopsFailedDelivery(t *testing.T) {
	session := createQualityDeliverySession(t, 148, "false")
	err := runCodeDeliveryQualityGate(session, session.UserID, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "质量门禁未通过") || !strings.Contains(err.Error(), "> false") {
		t.Fatalf("failed quality check should stop delivery: %v", err)
	}
	checks, detectErr := detectCodeQualityChecks(session)
	if detectErr != nil || len(checks) != 1 {
		t.Fatalf("detect checks: %#v, %v", checks, detectErr)
	}
	loadCodeQualityResults(session.ID, checks)
	if checks[0].LastResult == nil || checks[0].LastResult.Status != "failed" {
		t.Fatalf("failed result was not persisted: %#v", checks[0].LastResult)
	}
}

func TestRunCodeDeliveryQualityGateSkipsDisabledProject(t *testing.T) {
	database := withCodeGovernanceDB(t)
	session, _ := createDeliveryWorktree(t, 150)
	session.ProjectID = 150
	if err := database.Create(&model.AIProject{
		ID: session.ProjectID, Name: "quality-disabled", CreatorID: session.UserID,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(session).Error; err != nil {
		t.Fatal(err)
	}
	if err := runCodeDeliveryQualityGate(session, session.UserID, nil, nil); err != nil {
		t.Fatalf("disabled quality gate should not block delivery: %v", err)
	}
}

func TestCodeDeliveryDoesNotRunQualityChecks(t *testing.T) {
	session := createQualityDeliverySession(t, 149, "false")
	var job model.AICodeDeliveryJob
	if err := global.DB.Where("session_id = ?", session.ID).First(&job).Error; err != nil {
		t.Fatal(err)
	}
	runner := &codeDeliveryRunner{
		queued: make(map[uint]struct{}), cancelled: make(map[uint]struct{}),
		owner: newCodeRepositoryLeaseOwner("quality-test"),
	}
	runner.run(job.ID)
	if err := global.DB.First(&job, job.ID).Error; err != nil {
		t.Fatal(err)
	}
	if job.Status != codeDeliveryJobCompleted || job.Stage != codeDeliveryStageCompleted || job.FailureCode != "" {
		t.Fatalf("quality check blocked Git delivery: %#v", job)
	}
	var qualityEvents int64
	if err := global.DB.Model(&model.AITimelineEvent{}).
		Where("session_id = ? AND event_type = ?", session.ID, "quality_check").Count(&qualityEvents).Error; err != nil {
		t.Fatal(err)
	}
	if qualityEvents != 0 {
		t.Fatalf("Git delivery ran %d quality checks", qualityEvents)
	}
}
