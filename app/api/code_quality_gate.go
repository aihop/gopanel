package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/token"
	"github.com/gofiber/fiber/v3"
)

const (
	codeQualityTimeout       = 10 * time.Minute
	codeQualityOutputLimit   = 512 * 1024
	codeQualityPersistLimit  = 64 * 1024
	codeQualityMaxRootChecks = 80
)

type codeQualityCheck struct {
	ID          string   `json:"id"`
	Kind        string   `json:"kind"`
	Label       string   `json:"label"`
	Command     string   `json:"command"`
	WorkDir     string   `json:"workDir"`
	Args        []string `json:"-"`
	Executable  string   `json:"-"`
	workDirPath string
	LastResult  *codeQualityCheckResult `json:"lastResult,omitempty"`
}

type codeQualityCheckResult struct {
	CheckID         string    `json:"checkId"`
	Status          string    `json:"status"`
	ExitCode        int       `json:"exitCode"`
	DurationMS      int64     `json:"durationMs"`
	Output          string    `json:"output"`
	OutputTruncated bool      `json:"outputTruncated"`
	StartedAt       time.Time `json:"startedAt"`
	CompletedAt     time.Time `json:"completedAt"`
	Revision        string    `json:"revision,omitempty"`
	Current         bool      `json:"current"`
}

type codeQualityPackage struct {
	Scripts map[string]string `json:"scripts"`
}

type codeQualityTimelineMeta struct {
	UserID  uint                   `json:"userId"`
	CheckID string                 `json:"checkId"`
	Kind    string                 `json:"kind"`
	Command string                 `json:"command"`
	WorkDir string                 `json:"workDir"`
	Result  codeQualityCheckResult `json:"result"`
}

func GetCodeQualityChecks(c fiber.Ctx) error {
	session, _, err := getCodeQualitySession(c)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	checks, err := detectCodeQualityChecks(session)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	loadCodeQualityResults(session.ID, checks)
	return c.JSON(e.Succ(fiber.Map{"items": checks}))
}

func RunCodeQualityCheck(c fiber.Ctx) error {
	session, claims, err := getCodeQualitySession(c)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	var request struct {
		CheckID string `json:"checkId"`
	}
	if bindErr := c.Bind().JSON(&request); bindErr != nil {
		return c.JSON(e.Fail(bindErr))
	}
	checks, err := detectCodeQualityChecks(session)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	check := findCodeQualityCheck(checks, request.CheckID)
	if check == nil {
		return c.JSON(e.Fail(errors.New("质量检查项不存在或已发生变化，请刷新后重试")))
	}
	lease, err := codeExecutions.acquireSession(context.Background(), session, codeExecutionQuality, false)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	defer lease.Release()
	revision, err := codeQualityRevision(check.workDirPath)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	result := executeCodeQualityCheck(lease, *check)
	currentRevision, currentErr := codeQualityRevision(check.workDirPath)
	result.Revision = revision
	result.Current = currentErr == nil && currentRevision == revision
	if err := persistCodeQualityResult(session, claims.UserId, *check, result); err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(result))
}

func getCodeQualitySession(c fiber.Ctx) (*model.AIDevSession, *token.CustomClaims, error) {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	sessionID, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil || sessionID == 0 {
		return nil, claims, errors.New("会话 ID 无效")
	}
	session, err := getAISessionWithPermission(uint(sessionID), claims)
	if err != nil {
		return nil, claims, err
	}
	if err := validateAIProjectWorkDirForClaims(session.WorkDir, claims); err != nil {
		return nil, claims, err
	}
	return session, claims, nil
}

func detectCodeQualityChecks(session *model.AIDevSession) ([]codeQualityCheck, error) {
	roots, err := codeQualityRoots(session)
	if err != nil {
		return nil, err
	}
	checks := make([]codeQualityCheck, 0)
	for _, root := range roots {
		checks = append(checks, detectCodeQualityChecksAt(root, root)...)
		entries, readErr := os.ReadDir(root)
		if readErr != nil {
			continue
		}
		for _, entry := range entries {
			if len(checks) >= codeQualityMaxRootChecks || !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
				continue
			}
			if _, ignored := ignoredAIStructureNames[entry.Name()]; ignored {
				continue
			}
			child := filepath.Join(root, entry.Name())
			checks = append(checks, detectCodeQualityChecksAt(child, root)...)
		}
	}
	if len(checks) > codeQualityMaxRootChecks {
		checks = checks[:codeQualityMaxRootChecks]
	}
	sort.SliceStable(checks, func(i, j int) bool {
		if checks[i].WorkDir == checks[j].WorkDir {
			return codeQualityKindOrder(checks[i].Kind) < codeQualityKindOrder(checks[j].Kind)
		}
		return checks[i].WorkDir < checks[j].WorkDir
	})
	return checks, nil
}

func codeQualityRoots(session *model.AIDevSession) ([]string, error) {
	if session.IsolationMode == codeIsolationMultiWorktree {
		repositories, err := loadCodeSessionRepositories(session.ID)
		if err != nil {
			return nil, err
		}
		roots := make([]string, 0, len(repositories))
		for _, repository := range repositories {
			if repository.Status == codeDeliveryCompleted {
				continue
			}
			resolved, resolveErr := filepath.EvalSymlinks(repository.WorktreeDir)
			if resolveErr != nil {
				return nil, errors.New("会话仓库目录不存在或无法访问")
			}
			roots = append(roots, resolved)
		}
		return roots, nil
	}
	workDir, err := filepath.EvalSymlinks(filepath.Clean(session.WorkDir))
	if err != nil {
		return nil, errors.New("会话工作目录不存在或无法访问")
	}
	if !isAIProjectWorkspaceDirectory(session.WorkDir) {
		return []string{workDir}, nil
	}
	manifest, err := readAIProjectWorkspaceManifest(session.WorkDir)
	if err != nil {
		return nil, err
	}
	roots := make([]string, 0, len(manifest.Sources))
	for _, source := range manifest.Sources {
		resolved, resolveErr := filepath.EvalSymlinks(source.Path)
		if resolveErr == nil {
			roots = append(roots, resolved)
		}
	}
	return roots, nil
}

func detectCodeQualityChecksAt(workDir, displayRoot string) []codeQualityCheck {
	checks := detectNodeQualityChecks(workDir, displayRoot)
	if fileExists(filepath.Join(workDir, "go.mod")) {
		checks = append(checks,
			newCodeQualityCheck("test", "Go test", workDir, displayRoot, "go", "test", "./..."),
			newCodeQualityCheck("build", "Go build", workDir, displayRoot, "go", "build", "./..."),
		)
	}
	if fileExists(filepath.Join(workDir, "Cargo.toml")) {
		checks = append(checks,
			newCodeQualityCheck("test", "Cargo test", workDir, displayRoot, "cargo", "test"),
			newCodeQualityCheck("build", "Cargo check", workDir, displayRoot, "cargo", "check"),
		)
	}
	return checks
}

func detectNodeQualityChecks(workDir, displayRoot string) []codeQualityCheck {
	content, err := os.ReadFile(filepath.Join(workDir, "package.json"))
	if err != nil {
		return nil
	}
	var packageFile codeQualityPackage
	if json.Unmarshal(content, &packageFile) != nil {
		return nil
	}
	manager := "npm"
	prefix := []string{"run"}
	if fileExists(filepath.Join(workDir, "pnpm-lock.yaml")) {
		manager = "pnpm"
	} else if fileExists(filepath.Join(workDir, "yarn.lock")) {
		manager, prefix = "yarn", nil
	} else if fileExists(filepath.Join(workDir, "bun.lock")) || fileExists(filepath.Join(workDir, "bun.lockb")) {
		manager = "bun"
	}
	scriptCandidates := []struct {
		kind, label string
		names       []string
	}{
		{kind: "test", label: "Test", names: []string{"test", "test:unit"}},
		{kind: "lint", label: "Lint", names: []string{"lint"}},
		{kind: "typecheck", label: "Type check", names: []string{"type-check", "typecheck", "check:types"}},
		{kind: "build", label: "Build", names: []string{"build"}},
	}
	checks := make([]codeQualityCheck, 0, len(scriptCandidates))
	for _, candidate := range scriptCandidates {
		for _, name := range candidate.names {
			if _, exists := packageFile.Scripts[name]; !exists {
				continue
			}
			args := append(append([]string{}, prefix...), name)
			checks = append(checks, newCodeQualityCheck(candidate.kind, candidate.label, workDir, displayRoot, manager, args...))
			break
		}
	}
	return checks
}

func newCodeQualityCheck(kind, label, workDir, displayRoot, executable string, args ...string) codeQualityCheck {
	relative, err := filepath.Rel(displayRoot, workDir)
	if err != nil || relative == "." {
		relative = filepath.Base(displayRoot)
	} else {
		relative = filepath.Join(filepath.Base(displayRoot), relative)
	}
	command := strings.Join(append([]string{executable}, args...), " ")
	hash := sha256.Sum256([]byte(workDir + "\x00" + kind + "\x00" + command))
	return codeQualityCheck{
		ID: hex.EncodeToString(hash[:8]), Kind: kind, Label: label, Command: command,
		WorkDir: filepath.ToSlash(relative), Executable: executable, Args: args, workDirPath: workDir,
	}
}

func executeCodeQualityCheck(lease *codeExecutionLease, check codeQualityCheck) codeQualityCheckResult {
	startedAt := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), codeQualityTimeout)
	defer cancel()
	lease.SetCancel(cancel)
	output := &boundedCodeOutput{}
	command := exec.CommandContext(ctx, check.Executable, check.Args...)
	command.Dir = check.workDirPath
	command.Env = append(os.Environ(), "CI=1", "NO_COLOR=1")
	command.Stdout, command.Stderr = output, output
	err := command.Run()
	completedAt := time.Now()
	text, truncated := truncateCodeQualityOutput(string(output.Bytes()), codeQualityOutputLimit)
	status := "passed"
	if err != nil {
		status = "failed"
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			status = "timed_out"
			text = strings.TrimSpace(text + "\n[GoPanel: quality check timed out]")
		}
	}
	return codeQualityCheckResult{
		CheckID: check.ID, Status: status, ExitCode: executionExitCode(err),
		DurationMS: completedAt.Sub(startedAt).Milliseconds(), Output: text,
		OutputTruncated: truncated, StartedAt: startedAt, CompletedAt: completedAt,
	}
}

func findCodeQualityCheck(checks []codeQualityCheck, checkID string) *codeQualityCheck {
	for index := range checks {
		if checks[index].ID == strings.TrimSpace(checkID) {
			return &checks[index]
		}
	}
	return nil
}

func codeQualityKindOrder(kind string) int {
	for index, candidate := range []string{"test", "lint", "typecheck", "build"} {
		if kind == candidate {
			return index
		}
	}
	return 99
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

func truncateCodeQualityOutput(output string, limit int) (string, bool) {
	if len(output) <= limit {
		return strings.TrimSpace(output), false
	}
	headLimit := limit / 3
	tailLimit := limit - headLimit
	return strings.TrimSpace(output[:headLimit] + "\n[GoPanel: quality output truncated]\n" + output[len(output)-tailLimit:]), true
}

func loadCodeQualityResults(sessionID uint, checks []codeQualityCheck) {
	var events []*model.AITimelineEvent
	err := global.DB.Where("session_id = ? AND event_type = ?", sessionID, "quality_check").Order("created_at desc").Limit(1000).Find(&events).Error
	if err != nil {
		return
	}
	byID := make(map[string]codeQualityCheckResult)
	for _, event := range events {
		if event == nil || event.EventType != "quality_check" {
			continue
		}
		var meta codeQualityTimelineMeta
		if json.Unmarshal([]byte(event.Meta), &meta) == nil {
			if _, exists := byID[meta.CheckID]; !exists {
				byID[meta.CheckID] = meta.Result
			}
		}
	}
	for index := range checks {
		if result, exists := byID[checks[index].ID]; exists {
			revision, revisionErr := codeQualityRevision(checks[index].workDirPath)
			result.Current = revisionErr == nil && result.Revision != "" && result.Revision == revision
			resultCopy := result
			checks[index].LastResult = &resultCopy
		}
	}
}

func codeQualityRevision(workDir string) (string, error) {
	revision, err := runCodeGit(workDir, "rev-parse", "HEAD")
	if err != nil {
		return "", errors.New("质量检查需要有效的 Git 提交")
	}
	return strings.TrimSpace(revision), nil
}

func validateCodeQualityGate(session *model.AIDevSession) error {
	if session == nil || session.ProjectID == 0 {
		return nil
	}
	project, err := repo.NewAIProjectRepo().GetProjectByID(session.ProjectID)
	if err != nil || !project.RequireQualityGate {
		return err
	}
	checks, err := detectCodeQualityChecks(session)
	if err != nil {
		return err
	}
	if len(checks) == 0 {
		return errors.New("项目已启用质量门禁，但未识别到可执行检查")
	}
	loadCodeQualityResults(session.ID, checks)
	for _, check := range checks {
		if check.LastResult == nil {
			return fmt.Errorf("质量门禁未完成：%s 尚未运行", check.Label)
		}
		if !check.LastResult.Current {
			return fmt.Errorf("质量门禁已过期：%s 需要针对当前提交重新运行", check.Label)
		}
		if check.LastResult.Status != "passed" {
			return fmt.Errorf("质量门禁未通过：%s", check.Label)
		}
	}
	return nil
}

func persistCodeQualityResult(session *model.AIDevSession, userID uint, check codeQualityCheck, result codeQualityCheckResult) error {
	storedResult := result
	storedResult.Output, storedResult.OutputTruncated = truncateCodeQualityOutput(result.Output, codeQualityPersistLimit)
	meta, err := json.Marshal(codeQualityTimelineMeta{
		UserID: userID, CheckID: check.ID, Kind: check.Kind, Command: check.Command,
		WorkDir: check.WorkDir, Result: storedResult,
	})
	if err != nil {
		return err
	}
	status, title := "success", "质量检查通过"
	if result.Status != "passed" {
		status, title = "error", "质量检查失败"
	}
	content := fmt.Sprintf("%s · %s · %d ms", check.Command, check.WorkDir, result.DurationMS)
	return repo.NewAIDevSessionRepo().CreateTimelineEvent(&model.AITimelineEvent{
		SessionID: session.ID, TaskID: session.LastTaskID, EventType: "quality_check",
		Stage: "quality_check", Title: title, Content: content, Status: status, Meta: string(meta),
	})
}
