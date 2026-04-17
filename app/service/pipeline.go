package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/files"
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

func NewPipelineService(db *gorm.DB) *PipelineService {
	return &PipelineService{
		repo:       repo.NewPipeline(db),
		recordRepo: repo.NewPipelineRecord(db),
	}
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

	workspaceDir := filepath.Join(global.CONF.System.BaseDir, "pipelines", fmt.Sprintf("project_%d", p.ID))

	// 1. Clone
	if p.RepoUrl != "" {
		s.recordRepo.UpdateStatus(recordID, "cloning", "")
		err := s.stepClone(ctx, logger, p, workspaceDir)
		if err != nil {
			if ctx.Err() != nil {
				s.recordRepo.UpdateStatus(recordID, "failed", "用户手动终止")
				logger.Error("流水线已手动取消")
			} else {
				s.recordRepo.UpdateStatus(recordID, "failed", fmt.Sprintf("Clone failed: %v", err))
			}
			return
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
	err := s.stepBuild(ctx, logger, p, workspaceDir, record.Version)
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

	// 4. Trigger Website Deployment
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
		s.recordRepo.UpdateStatus(recordID, "success", "构建成功，未关联任何网站")
		logger.Info("流水线构建成功，但当前没有绑定网站，跳过发布。")
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

func (s *PipelineService) stepClone(ctx context.Context, logger *PipelineLogger, p *model.Pipeline, workspace string) error {
	logger.Info("准备代码拉取目录...")
	_ = os.MkdirAll(workspace, 0755)

	repoUrl := p.RepoUrl
	// 处理认证
	if p.AuthType == "token" && p.AuthData != "" {
		// Token 方式通常 AuthData 就是 Token 字符串，有时可能包含特殊字符
		tokenEncoded := url.QueryEscape(p.AuthData)
		if strings.HasPrefix(repoUrl, "https://") {
			repoUrl = strings.Replace(repoUrl, "https://", fmt.Sprintf("https://%s@", tokenEncoded), 1)
		} else if strings.HasPrefix(repoUrl, "http://") {
			repoUrl = strings.Replace(repoUrl, "http://", fmt.Sprintf("http://%s@", tokenEncoded), 1)
		}
	} else if p.AuthType == "password" && p.AuthData != "" {
		// 支持账户密码方式
		// p.AuthData 应该是 username:password 格式
		parts := strings.SplitN(p.AuthData, ":", 2)
		if len(parts) == 2 {
			username := url.QueryEscape(parts[0])
			password := url.QueryEscape(parts[1])
			authString := fmt.Sprintf("%s:%s", username, password)

			if strings.HasPrefix(repoUrl, "https://") {
				repoUrl = strings.Replace(repoUrl, "https://", fmt.Sprintf("https://%s@", authString), 1)
			} else if strings.HasPrefix(repoUrl, "http://") {
				repoUrl = strings.Replace(repoUrl, "http://", fmt.Sprintf("http://%s@", authString), 1)
			}
		} else {
			// 回退到直接拼接
			if strings.HasPrefix(repoUrl, "https://") {
				repoUrl = strings.Replace(repoUrl, "https://", fmt.Sprintf("https://%s@", p.AuthData), 1)
			} else if strings.HasPrefix(repoUrl, "http://") {
				repoUrl = strings.Replace(repoUrl, "http://", fmt.Sprintf("http://%s@", p.AuthData), 1)
			}
		}
	}

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
			return err
		}

		// 现在 pull 时使用 origin
		pullCmd := exec.CommandContext(ctx, "git", "pull", "origin", p.Branch)
		pullCmd.Dir = workspace

		if err := runGitCommand(pullCmd, "Git pull"); err != nil {
			return err
		}
	} else {
		logger.Info("首次执行或缓存丢失，正在执行 git clone (分支: %s)...", p.Branch)
		cloneCmd := exec.CommandContext(ctx, "git", "clone", "-b", p.Branch, "--single-branch", "--depth", "1", repoUrl, workspace)
		if err := runGitCommand(cloneCmd, "Git clone"); err != nil {
			return err
		}
	}

	// 获取 Commit Hash
	hashCmd := exec.CommandContext(ctx, "git", "rev-parse", "HEAD")
	hashCmd.Dir = workspace
	if hashBytes, err := hashCmd.Output(); err == nil {
		logger.Info("代码拉取成功, Commit Hash: %s", strings.TrimSpace(string(hashBytes)))
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

	return nil
}

func (s *PipelineService) stepBuild(ctx context.Context, logger *PipelineLogger, p *model.Pipeline, workspace string, version string) error {
	if p.BuildScript == "" {
		logger.Info("未配置构建脚本，跳过容器构建阶段")
		return nil
	}

	// 如果构建镜像配置为 "host" 或者为空，直接在宿主机/当前运行环境执行脚本
	if p.BuildImage == "host" || p.BuildImage == "" {
		logger.Info("选择宿主机本地环境构建 (版本: v%s)", version)

		scriptPath := filepath.Join(workspace, ".gopanel_build.sh")
		// 在生成的脚本前面加上 cd 命令，确保脚本内部执行指令（如 docker build .）时是在正确的目录
		// 同时添加环境变量打印方便排错
		fullScript := fmt.Sprintf("#!/bin/sh\nset -e\ncd \"%s\"\necho \"Current PWD: $(pwd)\"\n%s", workspace, p.BuildScript)
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

	// 检查 Docker 引擎是否运行
	if err := exec.CommandContext(ctx, "docker", "info").Run(); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		logger.Error("无法连接到 Docker 引擎！请检查 Docker 是否已安装并正在运行。")
		logger.Error("错误详情: %v", err)
		return fmt.Errorf("docker daemon is not running")
	}

	logger.Info("启动构建容器: %s (版本: v%s)", p.BuildImage, version)

	// docker run -i --rm -v workspace:/workspace -e PIPELINE_VERSION=xxx -w /workspace node:18 sh
	cmdArgs := []string{
		"run", "-i", "--rm",
		"-v", fmt.Sprintf("%s:/workspace", workspace),
		"-v", "/var/run/docker.sock:/var/run/docker.sock", // 支持 DooD (Docker out of Docker)
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
		"-w", "/workspace",
		p.BuildImage,
		"sh", // 不指定文件，直接运行 sh 并从 stdin 接收脚本
	)

	cmd := exec.CommandContext(ctx, "docker", cmdArgs...)

	// 无痕注入脚本内容
	scriptContent := fmt.Sprintf("set -e\n%s\n", p.BuildScript)
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

	archiveDir := filepath.Join(global.CONF.System.BaseDir, "pipelines_archive", fmt.Sprintf("%d", p.ID))
	_ = os.MkdirAll(archiveDir, 0755)

	archiveName := fmt.Sprintf("build_%d_%s.zip", recordID, time.Now().Format("20060102150405"))
	archivePath := filepath.Join(archiveDir, archiveName)

	logger.Info("正在对产物进行 Zip 归档留档...")
	err := files.NewFileOp().Compress([]string{artifactSrc}, archiveDir, archiveName, files.Zip, "")
	if err != nil {
		return "", err
	}
	logger.Info("产物归档成功: %s", archiveName)
	return archivePath, nil
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
