package api

import (
	"errors"
	"path/filepath"
	"strings"

	"github.com/aihop/gopanel/app/model"
)

// 项目级仓库排除。
//
// 项目目录本身是个仓库、里面又嵌了一堆子仓库时（典型如 app/themes/*），
// 发现逻辑会沿 gitlink 递归把它们全部带进来。这里给用户一个开关把不想要的摘掉。
//
// 过滤放在 discoverCodeRepositoryCandidates 的出口，因为那是所有路径的唯一咽喉：
// 建 worktree、交付策略、本地主仓同步、质量检查、会话预检全都经过它。
// 只在某一条路径（比如本地主仓同步）里屏蔽的话，被排除的仓库照样会被快照进工作区、
// 照样参与交付和推送 —— 那是治标不治本。

// normalizeCodeExcludedRepositories 归一化排除清单：清理路径、去空、去重。
func normalizeCodeExcludedRepositories(paths []string) []string {
	seen := make(map[string]struct{}, len(paths))
	normalized := make([]string, 0, len(paths))
	for _, path := range paths {
		cleaned := strings.TrimSpace(path)
		if cleaned == "" {
			continue
		}
		cleaned = filepath.Clean(cleaned)
		if resolved, err := filepath.EvalSymlinks(cleaned); err == nil {
			cleaned = filepath.Clean(resolved)
		}
		if _, exists := seen[cleaned]; exists {
			continue
		}
		seen[cleaned] = struct{}{}
		normalized = append(normalized, cleaned)
	}
	return normalized
}

// isCodeRepositoryExcluded 判断某个仓库是否被排除。
// 命中排除项本身，或位于某个排除项之内（排除一个仓库自然排除它的嵌套子仓库）。
func isCodeRepositoryExcluded(sourceDir string, excluded []string) bool {
	cleaned := filepath.Clean(strings.TrimSpace(sourceDir))
	if cleaned == "" || cleaned == "." {
		return false
	}
	for _, item := range excluded {
		if cleaned == item || isPathInside(cleaned, item) {
			return true
		}
	}
	return false
}

// filterExcludedCodeRepositories 摘掉被排除的仓库。
//
// 被排除仓库如果是别人的 gitlink 子仓库，父仓库那条记录本身不受影响 ——
// 父仓库照常参与开发，只是这个子目录不再被单独当成一个仓库管理。
func filterExcludedCodeRepositories(
	candidates []codeRepositoryCandidate,
	excluded []string,
) []codeRepositoryCandidate {
	if len(excluded) == 0 || len(candidates) == 0 {
		return candidates
	}
	kept := make([]codeRepositoryCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		if isCodeRepositoryExcluded(candidate.SourceDir, excluded) {
			continue
		}
		kept = append(kept, candidate)
	}
	return kept
}

// discoverCodeProjectRepositoryCandidates 按项目配置发现仓库，已经摘掉排除项。
// 项目编辑页要列出全部仓库供用户勾选，那里应当直接用 discoverCodeRepositoryCandidates。
func discoverCodeProjectRepositoryCandidates(
	project *model.AIProject,
	sourceDirs []string,
) ([]codeRepositoryCandidate, error) {
	return discoverCodeProjectRepositoryCandidatesWithStatus(project, sourceDirs, true)
}

func discoverCodeProjectRepositoryCandidatesWithStatus(
	project *model.AIProject,
	sourceDirs []string,
	inspectStatus bool,
) ([]codeRepositoryCandidate, error) {
	candidates, err := discoverCodeRepositoryCandidatesWithStatus(sourceDirs, inspectStatus)
	if err != nil {
		return nil, err
	}
	if project == nil {
		return candidates, nil
	}
	return filterExcludedCodeRepositories(candidates, normalizeCodeExcludedRepositories(project.ExcludedRepositories)), nil
}

// validateCodeExcludedRepositories 校验排除清单。
// 主交付仓库被排除会让交付无处落地，必须挡在保存这一步，不能等到交付时才炸。
func validateCodeExcludedRepositories(primaryRepository string, excluded []string) error {
	primary := filepath.Clean(strings.TrimSpace(primaryRepository))
	if primary == "" || primary == "." {
		return nil
	}
	if isCodeRepositoryExcluded(primary, excluded) {
		return errors.New("主交付仓库不能被排除，请先更换主交付仓库")
	}
	return nil
}
