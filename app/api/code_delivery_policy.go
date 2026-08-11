package api

import (
	"errors"
	"path/filepath"
	"sort"
	"strings"

	"github.com/aihop/gopanel/app/model"
)

const defaultCodeDeliveryBranch = "main"

type codeDeliveryPolicy struct {
	PrimaryRepository string
	DeliveryBranch    string
	GitCredentialID   uint
}

func normalizeCodeDeliveryPolicy(sourceDirs []string, primaryRepository, deliveryBranch string) (codeDeliveryPolicy, error) {
	candidates, err := discoverCodeRepositoryCandidates(sourceDirs)
	if err != nil {
		return codeDeliveryPolicy{}, err
	}
	return normalizeCodeDeliveryPolicyWithCandidates(sourceDirs, candidates, primaryRepository, deliveryBranch)
}

func normalizeCodeDeliveryPolicyWithCandidates(sourceDirs []string, candidates []codeRepositoryCandidate, primaryRepository, deliveryBranch string) (codeDeliveryPolicy, error) {
	if len(candidates) == 0 {
		return codeDeliveryPolicy{}, errors.New("项目目录中未发现 Git 仓库")
	}
	primaryRepository = strings.TrimSpace(primaryRepository)
	if primaryRepository != "" {
		resolved, resolveErr := filepath.EvalSymlinks(filepath.Clean(primaryRepository))
		if resolveErr != nil {
			return codeDeliveryPolicy{}, errors.New("主交付仓库不可访问，请重新选择项目内的 Git 仓库")
		}
		primaryRepository = resolved
	}
	if primaryRepository == "" {
		primaryRepository = selectPrimaryCodeRepository(sourceDirs, candidates)
	}
	if !containsCodeRepository(candidates, primaryRepository) {
		return codeDeliveryPolicy{}, errors.New("主交付仓库不属于当前项目")
	}
	configuredBranch := strings.TrimSpace(deliveryBranch)
	deliveryBranch = configuredBranch
	if deliveryBranch == "" {
		deliveryBranch = defaultCodeDeliveryBranch
	}
	if _, err := runCodeGit(primaryRepository, "check-ref-format", "--branch", deliveryBranch); err != nil {
		return codeDeliveryPolicy{}, errors.New("交付分支名称无效")
	}
	if !codeRepositoryBranchExists(primaryRepository, deliveryBranch) {
		if configuredBranch != "" {
			return codeDeliveryPolicy{}, errors.New("主交付仓库不存在配置的交付分支")
		}
		fallback := codeRepositoryDefaultBranch(primaryRepository)
		if fallback == "" {
			fallback, _ = runCodeGit(primaryRepository, "branch", "--show-current")
		}
		if strings.TrimSpace(fallback) == "" {
			return codeDeliveryPolicy{}, errors.New("主交付仓库没有可用的目标分支")
		}
		deliveryBranch = strings.TrimSpace(fallback)
	}
	return codeDeliveryPolicy{PrimaryRepository: primaryRepository, DeliveryBranch: deliveryBranch}, nil
}

func codeProjectDeliveryPolicy(project *model.AIProject, sourceDirs []string) (codeDeliveryPolicy, error) {
	if project == nil {
		return codeDeliveryPolicy{}, errors.New("项目交付策略不可用")
	}
	policy, err := normalizeCodeDeliveryPolicy(sourceDirs, project.PrimaryRepository, project.DeliveryBranch)
	policy.GitCredentialID = project.GitCredentialID
	return policy, err
}

func codeProjectDeliveryPolicyWithCandidates(project *model.AIProject, sourceDirs []string, candidates []codeRepositoryCandidate) (codeDeliveryPolicy, error) {
	if project == nil {
		return codeDeliveryPolicy{}, errors.New("项目交付策略不可用")
	}
	policy, err := normalizeCodeDeliveryPolicyWithCandidates(sourceDirs, candidates, project.PrimaryRepository, project.DeliveryBranch)
	policy.GitCredentialID = project.GitCredentialID
	return policy, err
}

func applyCodeProjectDeliveryPolicy(project *model.AIProject, sourceDirs []string) error {
	if project == nil {
		return errors.New("项目交付策略不可用")
	}
	candidates, err := discoverCodeProjectRepositoryCandidates(project, sourceDirs)
	if err != nil {
		return err
	}
	if len(candidates) == 0 {
		project.PrimaryRepository = ""
		if strings.TrimSpace(project.DeliveryBranch) == "" {
			project.DeliveryBranch = defaultCodeDeliveryBranch
		}
		return nil
	}
	policy, err := normalizeCodeDeliveryPolicyWithCandidates(sourceDirs, candidates, project.PrimaryRepository, project.DeliveryBranch)
	if err != nil {
		return err
	}
	project.PrimaryRepository = policy.PrimaryRepository
	project.DeliveryBranch = policy.DeliveryBranch
	return nil
}

func selectPrimaryCodeRepository(sourceDirs []string, candidates []codeRepositoryCandidate) string {
	for _, sourceDir := range sourceDirs {
		resolved, err := filepath.EvalSymlinks(filepath.Clean(sourceDir))
		if err != nil {
			continue
		}
		for _, candidate := range candidates {
			if candidate.SourceDir == resolved {
				return candidate.SourceDir
			}
		}
	}
	ordered := append([]codeRepositoryCandidate(nil), candidates...)
	sort.SliceStable(ordered, func(i, j int) bool {
		leftDepth := strings.Count(filepath.Clean(ordered[i].SourceDir), string(filepath.Separator))
		rightDepth := strings.Count(filepath.Clean(ordered[j].SourceDir), string(filepath.Separator))
		if leftDepth != rightDepth {
			return leftDepth < rightDepth
		}
		return ordered[i].SourceDir < ordered[j].SourceDir
	})
	return ordered[0].SourceDir
}

func containsCodeRepository(candidates []codeRepositoryCandidate, sourceDir string) bool {
	for _, candidate := range candidates {
		if candidate.SourceDir == sourceDir {
			return true
		}
	}
	return false
}

func codeRepositoryBranchExists(sourceDir, branch string) bool {
	for _, ref := range []string{"refs/heads/" + branch, "refs/remotes/origin/" + branch} {
		if _, err := runCodeGit(sourceDir, "show-ref", "--verify", ref); err == nil {
			return true
		}
	}
	return false
}

func codeRepositoryDefaultBranch(sourceDir string) string {
	remoteHead, _ := runCodeGit(sourceDir, "symbolic-ref", "--short", "refs/remotes/origin/HEAD")
	if remoteHead = strings.TrimSpace(remoteHead); strings.HasPrefix(remoteHead, "origin/") {
		return strings.TrimPrefix(remoteHead, "origin/")
	}
	return ""
}
