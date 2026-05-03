package service

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
)

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
	releaseDir := pipelineReleaseDir(p)
	logger.Info("工作区目录: %s", workspaceDir)
	logger.Info("发布目录: %s", releaseDir)

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
			record.CommitHash = commitHash
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

	if err := preparePipelineReleaseDir(logger, workspaceDir, releaseDir); err != nil {
		s.recordRepo.UpdateStatus(recordID, "failed", fmt.Sprintf("Prepare release failed: %v", err))
		logger.Error("准备发布目录失败: %v", err)
		return
	}

	// 2. Build
	s.recordRepo.UpdateStatus(recordID, "building", "")
	// 开始构建版本
	logger.Info("开始构建版本...，版本号: %s", record.Version)
	err := s.stepBuild(ctx, logger, p, workspaceDir, releaseDir, record.Version)
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
	archivePath, err := s.stepArchive(ctx, logger, p, releaseDir, recordID)
	if err != nil {
		if ctx.Err() != nil {
			s.recordRepo.UpdateStatus(recordID, "failed", "用户手动终止")
			logger.Error("流水线已手动取消")
			return
		}
		logger.Error("归档失败，但不影响发布: %v", err)
	} else {
		s.recordRepo.UpdateArchive(recordID, archivePath)
		record.ArchiveFile = archivePath
	}

	// 4. Runner Step (可选)
	s.recordRepo.UpdateStatus(recordID, "deploying", "准备执行 Runner 步骤...")
	if strings.EqualFold(strings.TrimSpace(p.RunnerMode), "runner") {
		runnerHostPort, runnerContainerID, runnerReleaseDir, err := s.stepRunner(ctx, logger, p, releaseDir)
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
		if runnerContainerID != "" {
			_ = s.recordRepo.UpdateRunnerResult(recordID, runnerReleaseDir, runnerContainerID, runnerHostPort)
			record.RunnerReleaseDir = runnerReleaseDir
			record.RunnerContainerID = runnerContainerID
			record.RunnerHostPort = runnerHostPort
			logger.Info("Runner 容器已启动：containerId=%s, hostPort=%d", runnerContainerID, runnerHostPort)
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
		_ = s.recordRepo.UpdateImageTag(recordID, finalImage)
		record.ImageTag = finalImage
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

var archiveExcludedNames = map[string]struct{}{
	".git":              {},
	".gopanel_artifact": {},
	"node_modules":      {},
	"__MACOSX":          {},
}

var releaseExcludedNames = map[string]struct{}{
	".git":              {},
	".gopanel_artifact": {},
	".gopanel_shims":    {},
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

func pipelineReleaseDir(p *model.Pipeline) string {
	return filepath.Join(pipelineBaseDir(p), "release")
}

func pipelineArchiveDir(p *model.Pipeline) string {
	return filepath.Join(pipelineBaseDir(p), "archive")
}

func preparePipelineReleaseDir(logger *PipelineLogger, workspaceDir, releaseDir string) error {
	workspaceDir = strings.TrimSpace(workspaceDir)
	releaseDir = strings.TrimSpace(releaseDir)
	if workspaceDir == "" || releaseDir == "" {
		return fmt.Errorf("工作区目录或发布目录为空")
	}
	if err := os.RemoveAll(releaseDir); err != nil {
		return err
	}
	if err := os.MkdirAll(releaseDir, 0755); err != nil {
		return err
	}
	logger.Info("正在同步工作区到发布目录...")
	if err := copyPipelineTree(workspaceDir, releaseDir, releaseExcludedNames); err != nil {
		return err
	}
	logger.Info("发布目录同步完成: %s", releaseDir)
	return nil
}

func copyPipelineTree(srcDir, dstDir string, excluded map[string]struct{}) error {
	return filepath.Walk(srcDir, func(current string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(srcDir, current)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if shouldSkipPipelineReleaseEntry(rel, info, excluded) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		target := filepath.Join(dstDir, rel)
		if info.Mode()&os.ModeSymlink != 0 {
			linkTarget, err := os.Readlink(current)
			if err != nil {
				return err
			}
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			return os.Symlink(linkTarget, target)
		}
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		return copyPipelineFile(current, target, info.Mode())
	})
}

func shouldSkipPipelineReleaseEntry(rel string, info os.FileInfo, excluded map[string]struct{}) bool {
	name := info.Name()
	if name == ".DS_Store" || strings.HasPrefix(name, "._") {
		return true
	}
	if _, ok := excluded[name]; ok {
		return true
	}
	for _, part := range strings.Split(filepath.ToSlash(rel), "/") {
		if _, ok := excluded[part]; ok {
			return true
		}
	}
	return false
}

func copyPipelineFile(srcPath, dstPath string, mode os.FileMode) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.OpenFile(dstPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	return err
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
