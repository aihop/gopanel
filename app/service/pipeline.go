package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
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
	repo        *repo.PipelineRepo
	recordRepo  *repo.PipelineRecordRepo
	releaseRepo *repo.ReleaseRepo
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
		repo:        repo.NewPipeline(db),
		recordRepo:  repo.NewPipelineRecord(db),
		releaseRepo: repo.NewRelease(db),
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

func (s *PipelineService) stepBuild(ctx context.Context, logger *PipelineLogger, p *model.Pipeline, workspace string, releaseDir string, version string) error {
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
GOPANEL_RELEASE_SYNC_MARKER="$PIPELINE_RELEASE_DIR/.gopanel_release_synced"
REAL_PODMAN_BIN="$(command -v podman 2>/dev/null || true)"
REAL_DOCKER_BIN="$(command -v docker 2>/dev/null || true)"
REAL_PODMAN_COMPOSE_BIN="$(command -v podman-compose 2>/dev/null || true)"
export CONTAINER_CLI
export RUNTIME
export CONTAINER_RUNTIME_KIND
export GOPANEL_SHIM_DIR
export GOPANEL_RELEASE_SYNC_MARKER
export REAL_PODMAN_BIN
export REAL_DOCKER_BIN
export REAL_PODMAN_COMPOSE_BIN

cleanup_gopanel_shims() {
	rm -rf "$GOPANEL_SHIM_DIR"
}
trap cleanup_gopanel_shims EXIT

rm -rf "$GOPANEL_SHIM_DIR"
mkdir -p "$GOPANEL_SHIM_DIR"
export PATH="$GOPANEL_SHIM_DIR:$PATH"

gopanel_sync_release() {
	if [ -z "$PIPELINE_WORKSPACE_DIR" ] || [ -z "$PIPELINE_RELEASE_DIR" ]; then
		return 0
	fi
	echo "[GOPANEL] syncing workspace to release: $PIPELINE_WORKSPACE_DIR -> $PIPELINE_RELEASE_DIR"
	rm -rf "$PIPELINE_RELEASE_DIR"
	mkdir -p "$PIPELINE_RELEASE_DIR"
	if command -v tar >/dev/null 2>&1 && tar --help 2>/dev/null | grep -q -- '--exclude'; then
		(
			cd "$PIPELINE_WORKSPACE_DIR" && \
			tar \
				--exclude='./.git' \
				--exclude='./node_modules' \
				--exclude='./.gopanel_artifact' \
				--exclude='./.gopanel_shims' \
				--exclude='./__MACOSX' \
				-cf - .
		) | (
			cd "$PIPELINE_RELEASE_DIR" && tar xpf -
		)
	else
		cp -a "$PIPELINE_WORKSPACE_DIR"/. "$PIPELINE_RELEASE_DIR"/
		rm -rf \
			"$PIPELINE_RELEASE_DIR/.git" \
			"$PIPELINE_RELEASE_DIR/node_modules" \
			"$PIPELINE_RELEASE_DIR/.gopanel_artifact" \
			"$PIPELINE_RELEASE_DIR/.gopanel_shims" \
			"$PIPELINE_RELEASE_DIR/__MACOSX" 2>/dev/null || true
	fi
	date +%%s > "$GOPANEL_RELEASE_SYNC_MARKER" 2>/dev/null || true
}

gopanel_should_sync_release() {
	if [ "$#" -lt 2 ]; then
		return 1
	fi
	[ "$1" = "compose" ] && [ "$2" = "up" ]
}

# 兼容旧脚本：优先保留 docker 命令语义，只给 podman/podman-compose 做向后兼容别名
if [ "$CONTAINER_CLI" = "docker" ]; then
	podman() {
		if gopanel_should_sync_release "$@"; then
			gopanel_sync_release
		fi
		docker "$@"
	}
	cat > "$GOPANEL_SHIM_DIR/podman-compose" <<'EOF'
#!/bin/sh
if [ "$1" = "up" ]; then
	"$GOPANEL_SHIM_DIR/gopanel-sync-release"
fi
exec docker compose "$@"
EOF
	chmod +x "$GOPANEL_SHIM_DIR/podman-compose"
elif [ "$CONTAINER_CLI" = "podman" ]; then
	if command -v docker > /dev/null 2>&1; then
		podman() {
			if gopanel_should_sync_release "$@"; then
				gopanel_sync_release
			fi
			docker "$@"
		}
		cat > "$GOPANEL_SHIM_DIR/podman-compose" <<'EOF'
#!/bin/sh
if [ "$1" = "up" ]; then
	"$GOPANEL_SHIM_DIR/gopanel-sync-release"
fi
exec docker compose "$@"
EOF
		chmod +x "$GOPANEL_SHIM_DIR/podman-compose"
	fi
fi
cat > "$GOPANEL_SHIM_DIR/gopanel-sync-release" <<'EOF'
#!/bin/sh
if [ -z "$PIPELINE_WORKSPACE_DIR" ] || [ -z "$PIPELINE_RELEASE_DIR" ]; then
	exit 0
fi
rm -rf "$PIPELINE_RELEASE_DIR"
mkdir -p "$PIPELINE_RELEASE_DIR"
if command -v tar >/dev/null 2>&1 && tar --help 2>/dev/null | grep -q -- '--exclude'; then
	(
		cd "$PIPELINE_WORKSPACE_DIR" && \
		tar \
			--exclude='./.git' \
			--exclude='./node_modules' \
			--exclude='./.gopanel_artifact' \
			--exclude='./.gopanel_shims' \
			--exclude='./__MACOSX' \
			-cf - .
	) | (
		cd "$PIPELINE_RELEASE_DIR" && tar xpf -
	)
else
	cp -a "$PIPELINE_WORKSPACE_DIR"/. "$PIPELINE_RELEASE_DIR"/
	rm -rf \
		"$PIPELINE_RELEASE_DIR/.git" \
		"$PIPELINE_RELEASE_DIR/node_modules" \
		"$PIPELINE_RELEASE_DIR/.gopanel_artifact" \
		"$PIPELINE_RELEASE_DIR/.gopanel_shims" \
		"$PIPELINE_RELEASE_DIR/__MACOSX" 2>/dev/null || true
fi
date +%%s > "$GOPANEL_RELEASE_SYNC_MARKER" 2>/dev/null || true
EOF
chmod +x "$GOPANEL_SHIM_DIR/gopanel-sync-release"
cat > "$GOPANEL_SHIM_DIR/podman" <<'EOF'
#!/bin/sh
if [ "$1" = "compose" ] && [ "$2" = "up" ]; then
	"$GOPANEL_SHIM_DIR/gopanel-sync-release"
fi
if [ -n "$REAL_PODMAN_BIN" ]; then
	exec "$REAL_PODMAN_BIN" "$@"
fi
if [ -n "$REAL_DOCKER_BIN" ]; then
	exec "$REAL_DOCKER_BIN" "$@"
fi
echo "podman command not found" >&2
exit 127
EOF
chmod +x "$GOPANEL_SHIM_DIR/podman"
cat > "$GOPANEL_SHIM_DIR/docker" <<'EOF'
#!/bin/sh
if [ "$1" = "compose" ] && [ "$2" = "up" ]; then
	"$GOPANEL_SHIM_DIR/gopanel-sync-release"
fi
if [ -n "$REAL_DOCKER_BIN" ]; then
	exec "$REAL_DOCKER_BIN" "$@"
fi
if [ -n "$REAL_PODMAN_BIN" ]; then
	exec "$REAL_PODMAN_BIN" "$@"
fi
echo "docker command not found" >&2
exit 127
EOF
chmod +x "$GOPANEL_SHIM_DIR/docker"
cat > "$GOPANEL_SHIM_DIR/podman-compose" <<'EOF'
#!/bin/sh
if [ "$1" = "up" ]; then
	"$GOPANEL_SHIM_DIR/gopanel-sync-release"
fi
if [ -n "$REAL_PODMAN_COMPOSE_BIN" ]; then
	exec "$REAL_PODMAN_COMPOSE_BIN" "$@"
fi
if [ -n "$REAL_PODMAN_BIN" ]; then
	exec "$REAL_PODMAN_BIN" compose "$@"
fi
if [ -n "$REAL_DOCKER_BIN" ]; then
	exec "$REAL_DOCKER_BIN" compose "$@"
fi
echo "podman compose command not found" >&2
exit 127
EOF
chmod +x "$GOPANEL_SHIM_DIR/podman-compose"
echo "--- 使用运行时: $RUNTIME (类型: $CONTAINER_RUNTIME_KIND, 兼容别名: $CONTAINER_CLI) ---"
`, runtimeCLI, resolvedRuntime.Kind, runtimeCLI)
		// 注入 runtime 兼容层：历史脚本里写 docker ... 也可在 podman 环境运行
		fullScript := fmt.Sprintf("#!/bin/sh\nset -e\ncd \"%s\"\necho \"Current PWD: $(pwd)\"\n%s\n%s\nif [ ! -f \"$GOPANEL_RELEASE_SYNC_MARKER\" ]; then\n\tgopanel_sync_release\nfi\n", workspace, compatHeader, p.BuildScript)
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
			fmt.Sprintf("PIPELINE_RELEASE_DIR=%s", releaseDir),
			fmt.Sprintf("GOPANEL_RELEASE_DIR=%s", releaseDir),
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
		"-v", fmt.Sprintf("%s:/workspace", releaseDir),
		"-v", fmt.Sprintf("%s:/source:ro", workspace),
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
		"-e", "PIPELINE_WORKSPACE_DIR=/source",
		"-e", "GOPANEL_WORKSPACE_DIR=/source",
		"-e", "PIPELINE_RELEASE_DIR=/workspace",
		"-e", "GOPANEL_RELEASE_DIR=/workspace",
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
