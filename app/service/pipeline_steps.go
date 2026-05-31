package service

import (
	"bytes"
	"context"
	"fmt"
	"github.com/aihop/gopanel/app/model"
	udocker "github.com/aihop/gopanel/utils/docker"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

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
	gitDir := filepath.Join(workspace, ".git")
	if _, err := os.Stat(gitDir); !os.IsNotExist(err) {
		logger.Info("检测到本地缓存，正在执行 git pull (分支: %s)...", p.Branch)
		remoteCmd := exec.CommandContext(ctx, "git", "remote", "set-url", "origin", repoUrl)
		remoteCmd.Dir = workspace
		_ = runGitCommand(remoteCmd, "Git remote")
		checkoutCmd := exec.CommandContext(ctx, "git", "checkout", p.Branch)
		checkoutCmd.Dir = workspace
		if err := runGitCommand(checkoutCmd, "Git checkout"); err != nil {
			return "", err
		}
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
	hashCmd := exec.CommandContext(ctx, "git", "rev-parse", "HEAD")
	hashCmd.Dir = workspace
	if hashBytes, err := hashCmd.Output(); err == nil {
		commitHash := strings.TrimSpace(string(hashBytes))
		logger.Info("代码拉取成功, Commit Hash: %s", commitHash)
		return commitHash, nil
	} else {
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
	if p.BuildImage == "" {
		return fmt.Errorf("构建镜像不能为空，请在流水线配置中设置 BuildImage")
	}
	if p.BuildImage == "host" {
		logger.Info("选择宿主机本地环境构建 (版本: v%s)，注意：构建环境不可重复！", version)
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

gopanel_validate_release_paths() {
	if [ -z "$PIPELINE_BASE_DIR" ] || [ -z "$PIPELINE_WORKSPACE_DIR" ] || [ -z "$PIPELINE_RELEASE_DIR" ]; then
		echo "[GOPANEL] missing pipeline sync paths" >&2
		return 1
	fi
	case "$PIPELINE_WORKSPACE_DIR" in
		"$PIPELINE_BASE_DIR"/*) ;;
		*) echo "[GOPANEL] workspace path escaped base dir: $PIPELINE_WORKSPACE_DIR" >&2; return 1 ;;
	esac
	case "$PIPELINE_RELEASE_DIR" in
		"$PIPELINE_BASE_DIR"/*) ;;
		*) echo "[GOPANEL] release path escaped base dir: $PIPELINE_RELEASE_DIR" >&2; return 1 ;;
	esac
	if [ "$PIPELINE_RELEASE_DIR" = "$PIPELINE_BASE_DIR" ] || [ "$PIPELINE_RELEASE_DIR" = "/" ]; then
		echo "[GOPANEL] invalid release path: $PIPELINE_RELEASE_DIR" >&2
		return 1
	fi
	return 0
}

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
	gopanel_validate_release_paths || return 1
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
if [ -z "$PIPELINE_BASE_DIR" ]; then
	echo "[GOPANEL] missing pipeline base dir" >&2
	exit 1
fi
case "$PIPELINE_WORKSPACE_DIR" in
	"$PIPELINE_BASE_DIR"/*) ;;
	*) echo "[GOPANEL] workspace path escaped base dir: $PIPELINE_WORKSPACE_DIR" >&2; exit 1 ;;
esac
case "$PIPELINE_RELEASE_DIR" in
	"$PIPELINE_BASE_DIR"/*) ;;
	*) echo "[GOPANEL] release path escaped base dir: $PIPELINE_RELEASE_DIR" >&2; exit 1 ;;
esac
if [ "$PIPELINE_RELEASE_DIR" = "$PIPELINE_BASE_DIR" ] || [ "$PIPELINE_RELEASE_DIR" = "/" ]; then
	echo "[GOPANEL] invalid release path: $PIPELINE_RELEASE_DIR" >&2
	exit 1
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
		fullScript := fmt.Sprintf("#!/bin/sh\nset -e\ncd \"%s\"\necho \"Current PWD: $(pwd)\"\n%s\n%s\nif [ ! -f \"$GOPANEL_RELEASE_SYNC_MARKER\" ]; then\n\tgopanel_sync_release\nfi\n", workspace, compatHeader, p.BuildScript)
		_ = os.WriteFile(scriptPath, []byte(fullScript), 0755)
		defer os.Remove(scriptPath)
		cmd := exec.CommandContext(ctx, "sh", scriptPath)
		cmd.Dir = workspace
		cmd.Env = os.Environ()
		cmd.Env = append(cmd.Env, fmt.Sprintf("PIPELINE_VERSION=%s", version), fmt.Sprintf("VERSION=%s", version), fmt.Sprintf("CONTAINER_CLI=%s", runtimeCLI), fmt.Sprintf("PIPELINE_BASE_DIR=%s", filepath.Dir(workspace)), fmt.Sprintf("GOPANEL_BASE_DIR=%s", filepath.Dir(workspace)), fmt.Sprintf("PIPELINE_WORKSPACE_DIR=%s", workspace), fmt.Sprintf("GOPANEL_WORKSPACE_DIR=%s", workspace), fmt.Sprintf("PIPELINE_RELEASE_DIR=%s", releaseDir), fmt.Sprintf("GOPANEL_RELEASE_DIR=%s", releaseDir))
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
	cmdArgs := []string{"run", "-i", "--rm", "-v", fmt.Sprintf("%s:/workspace", releaseDir), "-v", fmt.Sprintf("%s:/source:ro", workspace)}
	resolved := udocker.ResolveRuntime(ctx)
	if strings.HasPrefix(resolved.Host, "unix://") {
		hostSock := strings.TrimPrefix(resolved.Host, "unix://")
		if hostSock != "" {
			cmdArgs = append(cmdArgs, "-v", fmt.Sprintf("%s:/var/run/docker.sock", hostSock))
		}
	}
	homeDir, err := os.UserHomeDir()
	if err == nil {
		sshDir := filepath.Join(homeDir, ".ssh")
		if _, err := os.Stat(sshDir); !os.IsNotExist(err) {
			cmdArgs = append(cmdArgs, "-v", fmt.Sprintf("%s:/root/.ssh:ro", sshDir))
		}
	}
	cmdArgs = append(cmdArgs, "-e", fmt.Sprintf("PIPELINE_VERSION=%s", version), "-e", fmt.Sprintf("VERSION=%s", version), "-e", fmt.Sprintf("CONTAINER_CLI=%s", runtimeCLI), "-e", "PIPELINE_WORKSPACE_DIR=/source", "-e", "GOPANEL_WORKSPACE_DIR=/source", "-e", "PIPELINE_RELEASE_DIR=/workspace", "-e", "GOPANEL_RELEASE_DIR=/workspace", "-e", "DOCKER_HOST=unix:///var/run/docker.sock", "-w", "/workspace", p.BuildImage, "sh")
	cmd, cerr := udocker.RuntimeCommand(ctx, cmdArgs...)
	if cerr != nil {
		return cerr
	}
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
