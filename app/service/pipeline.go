package service

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/global"
	udocker "github.com/aihop/gopanel/utils/docker"
	"gorm.io/gorm"
)

var (
	pipelineCancels sync.Map
)

func StopPipeline(recordID uint) {
	if cancel, ok := pipelineCancels.Load(recordID); ok {
		if cancelFunc, isFunc := cancel.(context.CancelFunc); isFunc {
			cancelFunc()
		}
	}
	// 不管怎样，强制把状态改成 failed
	repo.NewPipelineRecord(global.DB).UpdateStatus(recordID, "failed", "用户手动强制终止")
}

type PipelineService struct {
	repo       *repo.PipelineRepo
	recordRepo *repo.PipelineRecordRepo
}

type RunnerPresetDetectResult struct {
	Preset string   `json:"preset"`
	Hits   []string `json:"hits"`
}

type pipelinePackageJSON struct {
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
}

func NewPipelineService(db *gorm.DB) *PipelineService {
	return &PipelineService{
		repo:       repo.NewPipeline(db),
		recordRepo: repo.NewPipelineRecord(db),
	}
}

func (s *PipelineService) DetectRunnerPreset(ctx context.Context, req request.PipelineDetect) (*RunnerPresetDetectResult, error) {
	repoURL := strings.TrimSpace(req.RepoUrl)
	branch := strings.TrimSpace(req.Branch)
	if repoURL == "" {
		return nil, fmt.Errorf("仓库地址不能为空")
	}
	if branch == "" {
		branch = "main"
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 20*time.Second)
		defer cancel()
	}

	tmpDir, err := os.MkdirTemp("", "pipeline_detect_*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)

	authURL := buildPipelineRepoURL(repoURL, req.AuthType, req.AuthData)
	cloneCmd := exec.CommandContext(ctx, "git", "clone", "-b", branch, "--single-branch", "--depth", "1", authURL, tmpDir)
	cloneCmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "GIT_SSH_COMMAND=ssh -o StrictHostKeyChecking=accept-new")
	var cloneErr bytes.Buffer
	cloneCmd.Stdout = io.Discard
	cloneCmd.Stderr = &cloneErr
	if err := cloneCmd.Run(); err != nil {
		msg := strings.TrimSpace(cloneErr.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("仓库探测失败: %s", msg)
	}

	result := detectRunnerPresetFromDir(tmpDir)
	if result.Preset == "" {
		result.Preset = "custom"
	}
	if result.Hits == nil {
		result.Hits = []string{}
	}
	return &result, nil
}

func detectRunnerPresetFromDir(dir string) RunnerPresetDetectResult {
	hits := make([]string, 0, 6)
	hasFile := func(rel string) bool {
		_, err := os.Stat(filepath.Join(dir, rel))
		return err == nil
	}
	appendHit := func(hit string) {
		hits = append(hits, hit)
	}

	if hasFile("go.mod") {
		appendHit("go.mod")
	}
	if hasFile("main.go") {
		appendHit("main.go")
	}
	if hasFile("requirements.txt") {
		appendHit("requirements.txt")
	}
	if hasFile("pyproject.toml") {
		appendHit("pyproject.toml")
	}
	if hasFile("app.py") {
		appendHit("app.py")
	}
	if hasFile("manage.py") {
		appendHit("manage.py")
	}
	if hasFile("composer.json") {
		appendHit("composer.json")
	}
	if hasFile("artisan") {
		appendHit("artisan")
	}
	if hasFile("public/index.php") {
		appendHit("public/index.php")
	}
	if hasFile("package.json") {
		appendHit("package.json")
	}

	if hasFile("go.mod") || hasFile("main.go") || dirHasEntry(filepath.Join(dir, "cmd")) {
		return RunnerPresetDetectResult{Preset: "go", Hits: hits}
	}
	if hasFile("requirements.txt") || hasFile("pyproject.toml") || hasFile("app.py") || hasFile("manage.py") {
		return RunnerPresetDetectResult{Preset: "python", Hits: hits}
	}
	if hasFile("composer.json") || hasFile("artisan") || hasFile("public/index.php") {
		return RunnerPresetDetectResult{Preset: "php", Hits: hits}
	}

	packageJSONPath := filepath.Join(dir, "package.json")
	if data, err := os.ReadFile(packageJSONPath); err == nil {
		var pkg pipelinePackageJSON
		if json.Unmarshal(data, &pkg) == nil {
			if hasPackageDep(pkg, "nuxt") {
				appendHit("package.json:nuxt")
				return RunnerPresetDetectResult{Preset: "nuxt", Hits: hits}
			}
			if hasPackageDep(pkg, "next") {
				appendHit("package.json:next")
				return RunnerPresetDetectResult{Preset: "next", Hits: hits}
			}
			return RunnerPresetDetectResult{Preset: "node", Hits: hits}
		}
	}
	return RunnerPresetDetectResult{Preset: "custom", Hits: hits}
}

func hasPackageDep(pkg pipelinePackageJSON, name string) bool {
	if _, ok := pkg.Dependencies[name]; ok {
		return true
	}
	_, ok := pkg.DevDependencies[name]
	return ok
}

func dirHasEntry(dir string) bool {
	entries, err := os.ReadDir(dir)
	return err == nil && len(entries) > 0
}

func buildPipelineRepoURL(repoURL, authType, authData string) string {
	repoURL = strings.TrimSpace(repoURL)
	if authType == "token" && authData != "" {
		tokenEncoded := url.QueryEscape(authData)
		if strings.HasPrefix(repoURL, "https://") {
			repoURL = strings.Replace(repoURL, "https://", fmt.Sprintf("https://%s@", tokenEncoded), 1)
		} else if strings.HasPrefix(repoURL, "http://") {
			repoURL = strings.Replace(repoURL, "http://", fmt.Sprintf("http://%s@", tokenEncoded), 1)
		}
	} else if authType == "password" && authData != "" {
		parts := strings.SplitN(authData, ":", 2)
		if len(parts) == 2 {
			username := url.QueryEscape(parts[0])
			password := url.QueryEscape(parts[1])
			authString := fmt.Sprintf("%s:%s", username, password)
			if strings.HasPrefix(repoURL, "https://") {
				repoURL = strings.Replace(repoURL, "https://", fmt.Sprintf("https://%s@", authString), 1)
			} else if strings.HasPrefix(repoURL, "http://") {
				repoURL = strings.Replace(repoURL, "http://", fmt.Sprintf("http://%s@", authString), 1)
			}
		} else {
			if strings.HasPrefix(repoURL, "https://") {
				repoURL = strings.Replace(repoURL, "https://", fmt.Sprintf("https://%s@", authData), 1)
			} else if strings.HasPrefix(repoURL, "http://") {
				repoURL = strings.Replace(repoURL, "http://", fmt.Sprintf("http://%s@", authData), 1)
			}
		}
	}
	return repoURL
}

func (s *PipelineService) RunPipeline(pipelineID uint, version string) (uint, error) {
	pipeline, err := s.repo.Get(pipelineID)
	if err != nil {
		return 0, err
	}

	record := &model.PipelineRecord{
		PipelineID: pipeline.ID,
		Status:     "pending",
		Version:    version,
	}
	err = s.recordRepo.Create(record)
	if err != nil {
		return 0, err
	}

	// 更新主表的当前版本号
	if pipeline.Version != version {
		pipeline.Version = version
		_ = s.repo.Update(pipeline)
	}

	// 异步执行流水线引擎
	go s.executePipeline(pipeline, record)

	return record.ID, nil
}

func (s *PipelineService) executePipeline(p *model.Pipeline, record *model.PipelineRecord) {
	recordID := record.ID
	logger := GetPipelineLogger(recordID)

	ctx, cancel := context.WithCancel(context.Background())
	pipelineCancels.Store(recordID, cancel)

	defer func() {
		pipelineCancels.Delete(recordID)
		logger.Info("EOF")
		// 立即移除 logger 而不是等待 10 秒
		RemovePipelineLogger(recordID)
	}()

	logger.Info("====== Pipeline #%d 执行开始 ======", recordID)
	logger.Info("应用: %s | 分支: %s", p.Name, p.Branch)

	workspaceDir := pipelineWorkspaceDir(p)
	runtimeDir := pipelineRuntimeDir(p)
	_ = os.MkdirAll(runtimeDir, 0755)
	logger.Info("工作区目录: %s", workspaceDir)
	logger.Info("运行时目录: %s", runtimeDir)

	// 1. Clone
	if p.RepoUrl != "" {
		s.recordRepo.UpdateStatus(recordID, "cloning", "")
		commitHash, err := s.stepClone(ctx, logger, p, workspaceDir)
		if err != nil {
			if ctx.Err() != nil {
				s.recordRepo.UpdateStatus(recordID, "failed", "用户手动终止")
				logger.Error("流水线已手动取消")
			} else {
				s.recordRepo.UpdateStatus(recordID, "failed", fmt.Sprintf("Clone failed: %v", err))
			}
			return
		}
		if strings.TrimSpace(commitHash) != "" {
			_ = s.recordRepo.UpdateCommitHash(recordID, commitHash)
		}
	} else {
		logger.Info("未配置 RepoUrl，采用纯脚本模式，跳过自动拉取...")
		_ = os.MkdirAll(workspaceDir, 0755)
	}

	// === 新增检查 ===
	// 如果是本地构建，打印一下当前拉取目录的文件列表，确保代码拉取正确
	if p.BuildImage == "host" || p.BuildImage == "" {
		files, _ := os.ReadDir(workspaceDir)
		var fileNames []string
		for _, f := range files {
			fileNames = append(fileNames, f.Name())
		}
		logger.Info("工作区目录检查 (%s): [%s]", workspaceDir, strings.Join(fileNames, ", "))
	}

	// 2. Build
	s.recordRepo.UpdateStatus(recordID, "building", "")
	// 开始构建版本
	logger.Info("开始构建版本...，版本号: %s", record.Version)
	err := s.stepBuild(ctx, logger, p, workspaceDir, runtimeDir, record.Version)
	if err != nil {
		if ctx.Err() != nil {
			s.recordRepo.UpdateStatus(recordID, "failed", "用户手动终止")
			logger.Error("流水线已手动取消")
		} else {
			s.recordRepo.UpdateStatus(recordID, "failed", fmt.Sprintf("Build failed: %v", err))
		}
		return
	}

	// 3. Archive (留档)
	archivePath, err := s.stepArchive(ctx, logger, p, workspaceDir, recordID)
	if err != nil {
		if ctx.Err() != nil {
			s.recordRepo.UpdateStatus(recordID, "failed", "用户手动终止")
			logger.Error("流水线已手动取消")
			return
		}
		logger.Error("归档失败，但不影响发布: %v", err)
	} else {
		s.recordRepo.UpdateArchive(recordID, archivePath)
	}

	// 4. Runner Step (可选)
	s.recordRepo.UpdateStatus(recordID, "deploying", "准备执行 Runner 步骤...")
	if strings.EqualFold(strings.TrimSpace(p.RunnerMode), "runner") {
		hostPort, containerID, releaseDir, err := s.stepRunner(ctx, logger, p, workspaceDir)
		if err != nil {
			if ctx.Err() != nil {
				s.recordRepo.UpdateStatus(recordID, "failed", "用户手动终止")
				logger.Error("流水线已手动取消")
			} else {
				s.recordRepo.UpdateStatus(recordID, "failed", fmt.Sprintf("Runner failed: %v", err))
				logger.Error("Runner 步骤失败: %v", err)
			}
			return
		}
		if containerID != "" {
			_ = s.recordRepo.UpdateRunnerResult(recordID, releaseDir, containerID, hostPort)
			logger.Info("Runner 容器已启动：containerId=%s, hostPort=%d", containerID, hostPort)
		}
	} else {
		logger.Info("未启用 Runner 步骤，跳过...")
	}

	// 5. Trigger Website Deployment
	s.recordRepo.UpdateStatus(recordID, "deploying", "通知关联网站进行部署...")
	logger.Info("正在通知所有关联此流水线的网站进行部署...")

	// 优先从构建日志中探测真实产出的镜像 tag，避免脚本内自定义 tag 与 record.Version 不一致。
	finalImage := detectBuiltImageRef(p, record.Version, logger.GetLogs())
	if finalImage != "" {
		logger.Info("检测到本次真实构建镜像: %s", finalImage)
	}

	summary, err := NewWebsite().DeployFromPipeline(ctx, p.ID, recordID, record.Version, archivePath, finalImage)
	if err != nil {
		s.recordRepo.UpdateStatus(recordID, "failed", err.Error())
		logger.Error("触发网站部署失败: %v", err)
		return
	}

	if summary != nil && summary.Matched == 0 {
		s.recordRepo.UpdateStatus(recordID, "success", "构建成功")
		logger.Info("流水线构建成功")
		return
	}

	msg := ""
	if summary != nil {
		msg = fmt.Sprintf("已完成 %d/%d 个网站发布", summary.Success, summary.Matched)
		logger.Info("%s", msg)
	}
	s.recordRepo.UpdateStatus(recordID, "success", msg)
	logger.Info("====== Pipeline #%d 执行成功！======", recordID)
}

func (s *PipelineService) stepRunner(ctx context.Context, logger *PipelineLogger, p *model.Pipeline, workspaceDir string) (int, string, string, error) {
	if ctx.Err() != nil {
		return 0, "", "", ctx.Err()
	}
	codeRoot, err := resolveRunnerCodeRoot(logger, p, workspaceDir)
	if err != nil {
		return 0, "", "", err
	}

	var runnerCfg map[string]interface{}
	if strings.TrimSpace(p.RunnerConfig) != "" {
		_ = json.Unmarshal([]byte(p.RunnerConfig), &runnerCfg)
	}
	if runnerCfg == nil {
		runnerCfg = map[string]interface{}{}
	}
	s.logRunnerProjectProfile(logger, codeRoot, runnerCfg)
	if err := validateRunnerModeSource(codeRoot, runnerCfg); err != nil {
		return 0, "", "", err
	}

	previousContainerID := ""
	if prev, err := s.recordRepo.LatestRunnerContainerID(p.ID); err == nil {
		previousContainerID = strings.TrimSpace(prev)
	}

	req := &request.WebsiteCreate{
		CodeSource:          "pipeline",
		GitRepo:             "",
		CodeDir:             "",
		CodeDirFallback:     codeRoot,
		PreviousContainerID: previousContainerID,
		PipelineKey:         strings.TrimSpace(p.PipelineKey),
		RunnerConfig:        runnerCfg,
	}

	progress := func(format string, a ...interface{}) {
		logger.Info("[Runner] "+format, a...)
	}
	alias := fmt.Sprintf("pipeline-%s", p.PipelineKey)
	hostPort, containerID, _, err := DeployWebsiteEngine(ctx, alias, req, progress)
	if err != nil {
		return 0, "", "", err
	}
	return hostPort, containerID, codeRoot, nil
}

func resolveRunnerCodeRoot(logger *PipelineLogger, p *model.Pipeline, workspaceDir string) (string, error) {
	sourceDir := strings.TrimSpace(workspaceDir)
	sourceLabel := "工作区目录"

	if sourceDir == "" {
		return "", fmt.Errorf("Runner 工作区目录为空")
	}

	artifactPath := strings.TrimSpace(p.ArtifactPath)
	if artifactPath != "" {
		artifactSrc := filepath.Join(sourceDir, artifactPath)
		info, err := os.Stat(artifactSrc)
		if err == nil {
			if info.IsDir() {
				sourceDir = artifactSrc
				sourceLabel = fmt.Sprintf("产物目录(%s)", artifactPath)
			} else {
				sourceDir = filepath.Dir(artifactSrc)
				sourceLabel = fmt.Sprintf("产物所在目录(%s)", artifactPath)
			}
		} else if !os.IsNotExist(err) {
			return "", fmt.Errorf("检查 Runner 产物目录失败: %w", err)
		}
	}

	codeRoot := detectRunnerCodeRoot(sourceDir)
	logger.Info("Runner: 直接使用%s作为运行目录: %s", sourceLabel, sourceDir)
	if codeRoot != sourceDir {
		logger.Info("Runner: 检测到单一子目录，自动切换代码根目录到 %s", codeRoot)
	}
	return codeRoot, nil
}

func validateRunnerModeSource(codeRoot string, runnerCfg map[string]interface{}) error {
	if err := ValidateRunnerPersistentPaths(runnerCfg); err != nil {
		return err
	}
	mode := strings.ToLower(strings.TrimSpace(asString(runnerCfg["mode"])))
	if mode == "" {
		mode = "build_run"
	}
	if mode != "build_run" {
		return nil
	}
	if strings.TrimSpace(asString(runnerCfg["buildCommand"])) != "" {
		return nil
	}
	if _, err := os.Stat(filepath.Join(codeRoot, ".output/server/index.mjs")); err == nil {
		return nil
	}
	if _, err := os.Stat(filepath.Join(codeRoot, "package.json")); err == nil {
		return nil
	}
	return fmt.Errorf("Runner 当前为 build_run 模式，但运行目录 %s 中既没有 package.json，也没有 .output/server/index.mjs，且未提供自定义 buildCommand；请改为 run 模式，或填写构建命令", codeRoot)
}

func ValidateRunnerPersistentPaths(runnerCfg map[string]interface{}) error {
	if runnerCfg == nil {
		return nil
	}
	paths := normalizeRunnerPersistentPaths(runnerCfg["persistentPaths"])
	for _, item := range paths {
		if isForbiddenRunnerPersistentPath(item) {
			return fmt.Errorf("Runner 持久化目录不支持 `%s`；`node_modules` 属于依赖目录，会导致 npm/pnpm 权限和脏缓存问题，请删除后重试", item)
		}
	}
	return nil
}

func normalizeRunnerPersistentPaths(raw interface{}) []string {
	switch v := raw.(type) {
	case []string:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if trimmed := strings.TrimSpace(item); trimmed != "" {
				out = append(out, trimmed)
			}
		}
		return out
	case []interface{}:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if trimmed := strings.TrimSpace(asString(item)); trimmed != "" {
				out = append(out, trimmed)
			}
		}
		return out
	default:
		return nil
	}
}

func isForbiddenRunnerPersistentPath(raw string) bool {
	candidate := strings.TrimSpace(strings.ReplaceAll(raw, "\\", "/"))
	if candidate == "" {
		return false
	}
	candidate = strings.TrimPrefix(path.Clean("/"+candidate), "/")
	return candidate == "node_modules" || strings.HasPrefix(candidate, "node_modules/")
}

func detectRunnerCodeRoot(releaseDir string) string {
	current := releaseDir
	for i := 0; i < 4; i++ {
		if runnerDirLooksLikeAppRoot(current) {
			return current
		}
		entries, err := os.ReadDir(current)
		if err != nil {
			return current
		}
		dirs := make([]string, 0, 1)
		for _, entry := range entries {
			name := entry.Name()
			if name == ".DS_Store" || name == "__MACOSX" {
				continue
			}
			if !entry.IsDir() {
				return current
			}
			dirs = append(dirs, filepath.Join(current, name))
		}
		if len(dirs) != 1 {
			return current
		}
		current = dirs[0]
	}
	return current
}

func runnerDirLooksLikeAppRoot(dir string) bool {
	markers := []string{
		"package.json",
		".output",
		".next",
		"dist",
		"Dockerfile",
		"index.html",
		"server.js",
		"docker-compose.yml",
	}
	for _, marker := range markers {
		if _, err := os.Stat(filepath.Join(dir, marker)); err == nil {
			return true
		}
	}
	return false
}

func (s *PipelineService) logRunnerProjectProfile(logger *PipelineLogger, releaseDir string, runnerCfg map[string]interface{}) {
	type marker struct {
		path  string
		label string
	}
	markers := []marker{
		{path: ".output/server/index.mjs", label: "Nuxt SSR"},
		{path: ".next/standalone/server.js", label: "Next.js standalone"},
		{path: ".next", label: "Next.js"},
		{path: "server.js", label: "Node server.js"},
		{path: "go.mod", label: "Go module"},
		{path: "main.go", label: "Go main"},
		{path: "requirements.txt", label: "Python requirements"},
		{path: "pyproject.toml", label: "Python pyproject"},
		{path: "app.py", label: "Python app.py"},
		{path: "composer.json", label: "PHP Composer"},
		{path: "artisan", label: "Laravel artisan"},
		{path: "public/index.php", label: "PHP public entry"},
		{path: "dist/index.html", label: "静态站 dist"},
		{path: "index.html", label: "静态站 root index"},
		{path: "package.json", label: "Node package"},
	}

	hits := make([]string, 0, len(markers))
	for _, m := range markers {
		if _, err := os.Stat(filepath.Join(releaseDir, m.path)); err == nil {
			hits = append(hits, m.label)
		}
	}
	if len(hits) > 0 {
		logger.Info("Runner: 项目特征识别 => %s", strings.Join(hits, ", "))
	}
	logger.Info("Runner: 项目类型 => %s", detectRunnerProjectKind(releaseDir, parseRunnerConfig(runnerCfg)))

	startCmd := strings.TrimSpace(asString(runnerCfg["startCommand"]))
	mode := strings.TrimSpace(asString(runnerCfg["mode"]))
	if mode == "" {
		mode = "build_run"
	}
	if startCmd == "" {
		startCmd = "node .output/server/index.mjs"
	}
	logger.Info("Runner: 当前策略 => mode=%s, startCommand=%s", mode, startCmd)

	staticOnly := false
	if _, err := os.Stat(filepath.Join(releaseDir, "dist/index.html")); err == nil {
		if _, err2 := os.Stat(filepath.Join(releaseDir, ".output/server/index.mjs")); err2 != nil {
			if _, err3 := os.Stat(filepath.Join(releaseDir, "server.js")); err3 != nil {
				staticOnly = true
			}
		}
	}
	if staticOnly {
		logger.Error("Runner 警告: 当前产物更像静态站点（存在 dist/index.html，缺少 .output/server/index.mjs / server.js），请确认预设和启动命令是否匹配")
	}
}

func detectBuiltImageRef(p *model.Pipeline, version string, logs []string) string {
	outputImage := strings.TrimSpace(p.OutputImage)
	candidates := extractBuiltImageCandidates(logs)

	if outputImage != "" {
		for _, candidate := range candidates {
			if sameImageRepo(candidate, outputImage) && !strings.HasSuffix(candidate, ":latest") {
				return candidate
			}
		}
		for _, candidate := range candidates {
			if sameImageRepo(candidate, outputImage) {
				return candidate
			}
		}
		return fmt.Sprintf("%s:%s", outputImage, version)
	}

	for _, candidate := range candidates {
		if !strings.HasSuffix(candidate, ":latest") {
			return candidate
		}
	}
	if len(candidates) > 0 {
		return candidates[0]
	}

	if p.BuildImage != "host" && p.BuildImage != "" {
		return fmt.Sprintf("%s:%s", p.BuildImage, version)
	}
	return ""
}

func extractBuiltImageCandidates(logs []string) []string {
	candidates := make([]string, 0)
	seen := make(map[string]struct{})
	for i := len(logs) - 1; i >= 0; i-- {
		if imageRef := parseBuiltImageRef(logs[i]); imageRef != "" {
			if _, ok := seen[imageRef]; ok {
				continue
			}
			seen[imageRef] = struct{}{}
			candidates = append(candidates, imageRef)
		}
	}
	return candidates
}

func parseBuiltImageRef(line string) string {
	line = strings.TrimSpace(line)
	if idx := strings.Index(line, "naming to "); idx >= 0 {
		ref := strings.TrimSpace(line[idx+len("naming to "):])
		ref = strings.TrimSuffix(ref, " done")
		return normalizeBuiltImageRef(ref)
	}
	if idx := strings.Index(line, "Successfully tagged "); idx >= 0 {
		ref := strings.TrimSpace(line[idx+len("Successfully tagged "):])
		return normalizeBuiltImageRef(ref)
	}
	return ""
}

func normalizeBuiltImageRef(ref string) string {
	ref = strings.TrimSpace(ref)
	ref = strings.Trim(ref, "`\"'")
	ref = strings.TrimPrefix(ref, "docker.io/library/")
	return ref
}

func sameImageRepo(imageRef, outputImage string) bool {
	imageRef = normalizeBuiltImageRef(imageRef)
	outputImage = normalizeBuiltImageRef(outputImage)
	if imageRef == "" || outputImage == "" {
		return false
	}
	repo := imageRef
	if idx := strings.LastIndex(repo, ":"); idx > strings.LastIndex(repo, "/") {
		repo = repo[:idx]
	}
	return repo == outputImage || strings.HasSuffix(repo, "/"+outputImage)
}

func (s *PipelineService) stepClone(ctx context.Context, logger *PipelineLogger, p *model.Pipeline, workspace string) (string, error) {
	logger.Info("准备代码拉取目录...")
	_ = os.MkdirAll(workspace, 0755)

	repoUrl := buildPipelineRepoURL(p.RepoUrl, p.AuthType, p.AuthData)

	runGitCommand := func(cmd *exec.Cmd, action string) error {
		cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "GIT_SSH_COMMAND=ssh -o StrictHostKeyChecking=accept-new")

		var outBuf, errBuf bytes.Buffer
		cmd.Stdout = io.MultiWriter(&outBuf, newLogWriter(logger, false))
		cmd.Stderr = io.MultiWriter(&errBuf, newLogWriter(logger, true))

		if err := cmd.Run(); err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			logger.Error("%s 失败: %v", action, err)
			return err
		}
		return nil
	}

	// 由于带有 Auth 的 repoUrl 包含密码，我们要避免直接打印它。
	// 这里不使用命令打印参数。

	// 检查是否存在 .git 目录，如果存在则进行增量拉取（pull），否则全量拉取（clone --depth 1）
	gitDir := filepath.Join(workspace, ".git")
	if _, err := os.Stat(gitDir); !os.IsNotExist(err) {
		logger.Info("检测到本地缓存，正在执行 git pull (分支: %s)...", p.Branch)

		// 每次 pull 之前更新 remote origin url，确保密码变更能生效
		remoteCmd := exec.CommandContext(ctx, "git", "remote", "set-url", "origin", repoUrl)
		remoteCmd.Dir = workspace
		_ = runGitCommand(remoteCmd, "Git remote")

		checkoutCmd := exec.CommandContext(ctx, "git", "checkout", p.Branch)
		checkoutCmd.Dir = workspace
		if err := runGitCommand(checkoutCmd, "Git checkout"); err != nil {
			return "", err
		}

		// 现在 pull 时使用 origin
		pullCmd := exec.CommandContext(ctx, "git", "pull", "origin", p.Branch)
		pullCmd.Dir = workspace

		if err := runGitCommand(pullCmd, "Git pull"); err != nil {
			return "", err
		}
	} else {
		logger.Info("首次执行或缓存丢失，正在执行 git clone (分支: %s)...", p.Branch)
		cloneCmd := exec.CommandContext(ctx, "git", "clone", "-b", p.Branch, "--single-branch", "--depth", "1", repoUrl, workspace)
		if err := runGitCommand(cloneCmd, "Git clone"); err != nil {
			return "", err
		}
	}

	// 获取 Commit Hash
	hashCmd := exec.CommandContext(ctx, "git", "rev-parse", "HEAD")
	hashCmd.Dir = workspace
	if hashBytes, err := hashCmd.Output(); err == nil {
		commitHash := strings.TrimSpace(string(hashBytes))
		logger.Info("代码拉取成功, Commit Hash: %s", commitHash)
		return commitHash, nil
	} else {
		// 为了排错，把 git 的真实地址隐去敏感信息后输出
		safeUrl := repoUrl
		if idx := strings.Index(safeUrl, "@"); idx > 0 {
			if protocolIdx := strings.Index(safeUrl, "://"); protocolIdx > 0 {
				safeUrl = safeUrl[:protocolIdx+3] + "***@" + safeUrl[idx+1:]
			}
		}
		logger.Error("注意: 拉取可能失败，当前使用的远端地址: %s", safeUrl)
	}

	return "", nil
}

func (s *PipelineService) stepBuild(ctx context.Context, logger *PipelineLogger, p *model.Pipeline, workspace string, runtimeDir string, version string) error {
	if p.BuildScript == "" {
		logger.Info("未配置构建脚本，跳过容器构建阶段")
		return nil
	}

	// 如果构建镜像配置为 "host" 或者为空，直接在宿主机/当前运行环境执行脚本
	if p.BuildImage == "host" || p.BuildImage == "" {
		logger.Info("选择宿主机本地环境构建 (版本: v%s)", version)

		scriptPath := filepath.Join(workspace, ".gopanel_build.sh")
		runtimeCLI, rerr := udocker.RuntimeCLI(ctx)
		if rerr != nil {
			logger.Error("未找到可用容器运行时命令: %v", rerr)
			return rerr
		}
		resolvedRuntime := udocker.DefaultRuntimeAdapter().Resolve(ctx)
		compatHeader := fmt.Sprintf(`
# 1. 补全基础环境变量
export PATH=$PATH:/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin

# 2. 使用后端已解析好的运行时，避免脚本再次自探测导致与后端判定不一致
RUNTIME="%s"
CONTAINER_RUNTIME_KIND="%s"

CONTAINER_CLI="%s"
GOPANEL_SHIM_DIR="$PWD/.gopanel_shims"
export CONTAINER_CLI
export RUNTIME
export CONTAINER_RUNTIME_KIND
export GOPANEL_SHIM_DIR

cleanup_gopanel_shims() {
	rm -rf "$GOPANEL_SHIM_DIR"
}
trap cleanup_gopanel_shims EXIT

rm -rf "$GOPANEL_SHIM_DIR"
mkdir -p "$GOPANEL_SHIM_DIR"
export PATH="$GOPANEL_SHIM_DIR:$PATH"

# 兼容旧脚本：优先保留 docker 命令语义，只给 podman/podman-compose 做向后兼容别名
if [ "$CONTAINER_CLI" = "docker" ]; then
	podman() { docker "$@"; }
	cat > "$GOPANEL_SHIM_DIR/podman-compose" <<'EOF'
#!/bin/sh
exec docker compose "$@"
EOF
	chmod +x "$GOPANEL_SHIM_DIR/podman-compose"
elif [ "$CONTAINER_CLI" = "podman" ]; then
	if command -v docker > /dev/null 2>&1; then
		podman() { docker "$@"; }
		cat > "$GOPANEL_SHIM_DIR/podman-compose" <<'EOF'
#!/bin/sh
exec docker compose "$@"
EOF
		chmod +x "$GOPANEL_SHIM_DIR/podman-compose"
	fi
fi
echo "--- 使用运行时: $RUNTIME (类型: $CONTAINER_RUNTIME_KIND, 兼容别名: $CONTAINER_CLI) ---"
`, runtimeCLI, resolvedRuntime.Kind, runtimeCLI)
		// 注入 runtime 兼容层：历史脚本里写 docker ... 也可在 podman 环境运行
		fullScript := fmt.Sprintf("#!/bin/sh\nset -e\ncd \"%s\"\necho \"Current PWD: $(pwd)\"\n%s\n%s", workspace, compatHeader, p.BuildScript)
		_ = os.WriteFile(scriptPath, []byte(fullScript), 0755)
		defer os.Remove(scriptPath)

		cmd := exec.CommandContext(ctx, "sh", scriptPath)
		// 显式指定命令的工作目录
		cmd.Dir = workspace
		// 为了让 docker build 等命令能正确找到执行目录，并继承系统环境变量
		cmd.Env = os.Environ()
		// 覆盖或追加版本变量
		cmd.Env = append(cmd.Env,
			fmt.Sprintf("PIPELINE_VERSION=%s", version),
			fmt.Sprintf("VERSION=%s", version),
			fmt.Sprintf("CONTAINER_CLI=%s", runtimeCLI),
			fmt.Sprintf("PIPELINE_WORKSPACE_DIR=%s", workspace),
			fmt.Sprintf("GOPANEL_WORKSPACE_DIR=%s", workspace),
			fmt.Sprintf("PIPELINE_RUNTIME_DIR=%s", runtimeDir),
			fmt.Sprintf("GOPANEL_RUNTIME_DIR=%s", runtimeDir),
		)

		var outBuf, errBuf bytes.Buffer
		cmd.Stdout = io.MultiWriter(&outBuf, newLogWriter(logger, false))
		cmd.Stderr = io.MultiWriter(&errBuf, newLogWriter(logger, true))

		if err := cmd.Run(); err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			logger.Error("本地构建执行失败: %v", err)
			return err
		}
		return nil
	}

	runtimeCLI, rerr := udocker.RuntimeCLI(ctx)
	if rerr != nil {
		return rerr
	}
	// 检查容器引擎是否运行
	infoCmd, ierr := udocker.RuntimeCommand(ctx, "info")
	if ierr != nil {
		return ierr
	}
	if err := infoCmd.Run(); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		logger.Error("无法连接到容器引擎（%s）！请检查运行时是否已安装并正在运行。", runtimeCLI)
		logger.Error("错误详情: %v", err)
		return fmt.Errorf("%s daemon is not running", runtimeCLI)
	}

	logger.Info("启动构建容器: %s (版本: v%s)", p.BuildImage, version)

	// docker run -i --rm -v workspace:/workspace -e PIPELINE_VERSION=xxx -w /workspace node:18 sh
	cmdArgs := []string{
		"run", "-i", "--rm",
		"-v", fmt.Sprintf("%s:/workspace", workspace),
		"-v", fmt.Sprintf("%s:/runtime", runtimeDir),
	}
	resolved := udocker.ResolveRuntime(ctx)
	if strings.HasPrefix(resolved.Host, "unix://") {
		hostSock := strings.TrimPrefix(resolved.Host, "unix://")
		if hostSock != "" {
			cmdArgs = append(cmdArgs, "-v", fmt.Sprintf("%s:/var/run/docker.sock", hostSock))
		}
	}

	// 动态获取宿主机的 ~/.ssh 目录并挂载，解决 macOS 和不同用户的跨平台路径问题
	homeDir, err := os.UserHomeDir()
	if err == nil {
		sshDir := filepath.Join(homeDir, ".ssh")
		if _, err := os.Stat(sshDir); !os.IsNotExist(err) {
			cmdArgs = append(cmdArgs, "-v", fmt.Sprintf("%s:/root/.ssh:ro", sshDir))
		}
	}

	cmdArgs = append(cmdArgs,
		"-e", fmt.Sprintf("PIPELINE_VERSION=%s", version), // 兼容旧变量
		"-e", fmt.Sprintf("VERSION=%s", version), // 给脚本使用的通用版本号
		"-e", fmt.Sprintf("CONTAINER_CLI=%s", runtimeCLI),
		"-e", "PIPELINE_WORKSPACE_DIR=/workspace",
		"-e", "GOPANEL_WORKSPACE_DIR=/workspace",
		"-e", "PIPELINE_RUNTIME_DIR=/runtime",
		"-e", "GOPANEL_RUNTIME_DIR=/runtime",
		"-e", "DOCKER_HOST=unix:///var/run/docker.sock",
		"-w", "/workspace",
		p.BuildImage,
		"sh", // 不指定文件，直接运行 sh 并从 stdin 接收脚本
	)

	cmd, cerr := udocker.RuntimeCommand(ctx, cmdArgs...)
	if cerr != nil {
		return cerr
	}

	// 无痕注入脚本内容
	// 兼容旧的 docker/podman alias
	compatHeader := fmt.Sprintf(`
CONTAINER_CLI="%s"
GOPANEL_SHIM_DIR="$PWD/.gopanel_shims"
export CONTAINER_CLI
export GOPANEL_SHIM_DIR
cleanup_gopanel_shims() {
	rm -rf "$GOPANEL_SHIM_DIR"
}
trap cleanup_gopanel_shims EXIT
rm -rf "$GOPANEL_SHIM_DIR"
mkdir -p "$GOPANEL_SHIM_DIR"
export PATH="$GOPANEL_SHIM_DIR:$PATH"
if [ "$CONTAINER_CLI" = "docker" ]; then
	podman() { docker "$@"; }
	cat > "$GOPANEL_SHIM_DIR/podman-compose" <<'EOF'
#!/bin/sh
exec docker compose "$@"
EOF
	chmod +x "$GOPANEL_SHIM_DIR/podman-compose"
elif [ "$CONTAINER_CLI" = "podman" ]; then
	if command -v docker > /dev/null 2>&1; then
		podman() { docker "$@"; }
		cat > "$GOPANEL_SHIM_DIR/podman-compose" <<'EOF'
#!/bin/sh
exec docker compose "$@"
EOF
		chmod +x "$GOPANEL_SHIM_DIR/podman-compose"
	fi
fi
`, runtimeCLI)

	scriptContent := fmt.Sprintf("set -e\n%s\n%s\n", compatHeader, p.BuildScript)
	cmd.Stdin = strings.NewReader(scriptContent)

	// 捕获日志
	cmd.Stdout = newLogWriter(logger, false)
	cmd.Stderr = newLogWriter(logger, true)

	logger.Info("开始执行构建...")
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		logger.Error("容器构建失败: %v", err)
		return err
	}

	logger.Info("构建执行完毕")
	return nil
}

func (s *PipelineService) stepArchive(ctx context.Context, logger *PipelineLogger, p *model.Pipeline, workspace string, recordID uint) (string, error) {
	if ctx.Err() != nil {
		logger.Error("流水线已手动取消")
		return "", ctx.Err()
	}
	if p.ArtifactPath == "" {
		return "", nil
	}
	artifactSrc := filepath.Join(workspace, p.ArtifactPath)
	if _, err := os.Stat(artifactSrc); os.IsNotExist(err) {
		return "", nil
	}

	archiveDir := pipelineArchiveDir(p)
	_ = os.MkdirAll(archiveDir, 0755)

	archiveName := fmt.Sprintf("build_%d_%s.zip", recordID, time.Now().Format("20060102150405"))
	archivePath := filepath.Join(archiveDir, archiveName)

	logger.Info("正在对产物进行 Zip 归档留档...")
	err := createFilteredZipArchive(artifactSrc, archivePath)
	if err != nil {
		return "", err
	}
	logger.Info("产物归档成功: %s", archiveName)
	return archivePath, nil
}

var archiveExcludedNames = map[string]struct{}{
	".git":              {},
	".gopanel_artifact": {},
	"node_modules":      {},
	"__MACOSX":          {},
}

func pipelineDirName(p *model.Pipeline) string {
	if p == nil {
		return "project"
	}
	if key := strings.TrimSpace(p.PipelineKey); key != "" {
		return key
	}
	if p.ID > 0 {
		return fmt.Sprintf("project_%d", p.ID)
	}
	return "project"
}

func pipelineBaseDir(p *model.Pipeline) string {
	return filepath.Join(global.CONF.System.BaseDir, "pipelines", pipelineDirName(p))
}

func pipelineWorkspaceDir(p *model.Pipeline) string {
	return filepath.Join(pipelineBaseDir(p), "workspace")
}

func pipelineRuntimeDir(p *model.Pipeline) string {
	return filepath.Join(pipelineBaseDir(p), "runtime")
}

func pipelineArchiveDir(p *model.Pipeline) string {
	return filepath.Join(pipelineBaseDir(p), "archive")
}

func createFilteredZipArchive(srcPath, archivePath string) error {
	info, err := os.Stat(srcPath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(archivePath), 0755); err != nil {
		return err
	}
	out, err := os.OpenFile(archivePath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer out.Close()

	zw := zip.NewWriter(out)
	defer zw.Close()

	rootName := filepath.Base(filepath.Clean(srcPath))
	if !info.IsDir() {
		return addArchiveFileToZip(zw, srcPath, filepath.ToSlash(rootName), info)
	}

	rootHeader := &zip.FileHeader{
		Name:     filepath.ToSlash(rootName) + "/",
		Method:   zip.Deflate,
		Modified: info.ModTime(),
	}
	rootHeader.SetMode(info.Mode())
	if _, err := zw.CreateHeader(rootHeader); err != nil {
		return err
	}

	return filepath.Walk(srcPath, func(current string, currentInfo os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(srcPath, current)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if shouldSkipArchiveEntry(rel, currentInfo) {
			if currentInfo.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		nameInArchive := filepath.ToSlash(filepath.Join(rootName, rel))
		if currentInfo.IsDir() {
			header := &zip.FileHeader{
				Name:     nameInArchive + "/",
				Method:   zip.Deflate,
				Modified: currentInfo.ModTime(),
			}
			header.SetMode(currentInfo.Mode())
			_, err = zw.CreateHeader(header)
			return err
		}
		return addArchiveFileToZip(zw, current, nameInArchive, currentInfo)
	})
}

func shouldSkipArchiveEntry(rel string, info os.FileInfo) bool {
	name := info.Name()
	if name == ".DS_Store" || strings.HasPrefix(name, "._") {
		return true
	}
	if _, ok := archiveExcludedNames[name]; ok {
		return true
	}
	for _, part := range strings.Split(filepath.ToSlash(rel), "/") {
		if _, ok := archiveExcludedNames[part]; ok {
			return true
		}
	}
	return false
}

func addArchiveFileToZip(zw *zip.Writer, diskPath, nameInArchive string, info os.FileInfo) error {
	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	header.Name = nameInArchive
	header.Method = zip.Deflate
	writer, err := zw.CreateHeader(header)
	if err != nil {
		return err
	}
	file, err := os.Open(diskPath)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = io.Copy(writer, file)
	return err
}

// 辅助日志写入器，将 os/exec 的输出桥接到 PipelineLogger
type logWriter struct {
	logger *PipelineLogger
	isErr  bool
}

func newLogWriter(logger *PipelineLogger, isErr bool) *logWriter {
	return &logWriter{logger: logger, isErr: isErr}
}

func (w *logWriter) Write(p []byte) (n int, err error) {
	lines := strings.Split(string(p), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if w.isErr {
			w.logger.Error("%s", line)
		} else {
			w.logger.Info("%s", line)
		}
	}
	return len(p), nil
}
