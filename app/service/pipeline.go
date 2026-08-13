package service

import (
	"bytes"
	"context"
	"fmt"
	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/buserr"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"gorm.io/gorm"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var (
	pipelineCancels        sync.Map
	pipelineExecutionLocks sync.Map
	pipelineMutationMu     sync.Mutex
)

func pipelineExecutionLock(pipelineID uint) *sync.Mutex {
	lock, _ := pipelineExecutionLocks.LoadOrStore(pipelineID, &sync.Mutex{})
	return lock.(*sync.Mutex)
}

func StopPipeline(recordID uint) {
	if cancel, ok := pipelineCancels.Load(recordID); ok {
		if cancelFunc, isFunc := cancel.(context.CancelFunc); isFunc {
			cancelFunc()
		}
	}
	repo.NewPipelineRecord(global.DB).UpdateStatus(recordID, "failed", "用户手动强制终止")
}

type PipelineService struct {
	db         *gorm.DB
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
	return &PipelineService{db: db, repo: repo.NewPipeline(db), recordRepo: repo.NewPipelineRecord(db)}
}
func (s *PipelineService) DetectRunnerPreset(ctx context.Context, req request.PipelineDetect) (*RunnerPresetDetectResult, error) {
	if strings.EqualFold(strings.TrimSpace(req.SourceType), "code") {
		if req.CodeProjectID == 0 {
			return nil, buserr.New(constant.ErrPipelineCodeProjectRequired)
		}
		var project model.AIProject
		if err := s.db.First(&project, req.CodeProjectID).Error; err != nil {
			return nil, buserr.New(constant.ErrPipelineCodeProjectNotFound)
		}
		detectDir := strings.TrimSpace(project.PrimaryRepository)
		if detectDir == "" && len(project.SourceDirs) > 0 {
			detectDir = strings.TrimSpace(project.SourceDirs[0])
		}
		if detectDir == "" {
			return nil, fmt.Errorf("Code 项目没有可探测的源目录")
		}
		result := detectRunnerPresetFromDir(detectDir)
		if result.Preset == "" {
			result.Preset = "custom"
		}
		if result.Hits == nil {
			result.Hits = []string{}
		}
		return &result, nil
	}
	repoURL := strings.TrimSpace(req.RepoUrl)
	branch := strings.TrimSpace(req.Branch)
	if repoURL == "" {
		return nil, fmt.Errorf("仓库地址不能为空")
	}
	if branch == "" {
		branch = "main"
	}
	if err := ensurePipelineClonePrerequisites(repoURL); err != nil {
		return nil, err
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
	markers := []marker{{path: ".output/server/index.mjs", label: "Nuxt SSR"}, {path: ".next/standalone/server.js", label: "Next.js standalone"}, {path: ".next", label: "Next.js"}, {path: "server.js", label: "Node server.js"}, {path: "go.mod", label: "Go module"}, {path: "main.go", label: "Go main"}, {path: "requirements.txt", label: "Python requirements"}, {path: "pyproject.toml", label: "Python pyproject"}, {path: "app.py", label: "Python app.py"}, {path: "composer.json", label: "PHP Composer"}, {path: "artisan", label: "Laravel artisan"}, {path: "public/index.php", label: "PHP public entry"}, {path: "dist/index.html", label: "静态站 dist"}, {path: "index.html", label: "静态站 root index"}, {path: "package.json", label: "Node package"}}
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
