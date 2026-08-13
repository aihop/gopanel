package api

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aihop/gopanel/app/model"
)

// 隔离会话的初始化是异步的：先落一条 initializing 会话，再后台同步仓库、建 Worktree。
// 这对「拉大仓库」这类慢操作是必要的，但认证失败、远端地址写错、源仓有未提交变更
// 这些秒级就能判定的问题，如果也留给异步阶段，每失败一次就会留下一条
// 没有任务、在任务列表里看不到、也没有任何入口能删掉的失败会话。
//
// 所以把「快速可判定」的部分提到创建请求里同步做：不通过就直接报错，不落库。
const (
	codeSessionRemoteProbeTimeout = 12 * time.Second
	codeSessionRemoteProbeTTL     = 30 * time.Second
	codeSessionRemoteProbeWorkers = 4
)

var codeSessionRemoteProbes = struct {
	sync.Mutex
	succeeded map[string]time.Time
}{succeeded: make(map[string]time.Time)}

func validateCodeSessionPrerequisites(project *model.AIProject, includeUncommitted bool) error {
	if project == nil {
		return nil
	}
	sourceDirs := codeProjectSourceDirs(project)
	candidates, err := discoverCodeProjectRepositoryCandidatesWithStatus(project, sourceDirs, !includeUncommitted)
	if err != nil {
		return err
	}
	if len(candidates) == 0 {
		return fmt.Errorf("项目目录中未发现 Git 仓库")
	}
	policy, err := codeProjectDeliveryPolicyWithCandidates(project, sourceDirs, candidates)
	if err != nil {
		return fmt.Errorf("项目交付策略无效：%w", err)
	}
	// 不允许快照未提交内容时，源仓一脏，异步阶段必然失败。
	if !includeUncommitted {
		for _, candidate := range candidates {
			if candidate.Dirty {
				return fmt.Errorf(
					"源仓库 %s 存在未提交变更，请先提交，或在创建会话时允许包含未提交内容",
					filepath.Base(candidate.SourceDir),
				)
			}
		}
	}
	return validateCodeRepositoryRemotesReachable(candidates, policy.GitCredentialID)
}

// validateCodeRepositoryRemotesReachable 用 ls-remote 探测远端与凭据。
// 纯本地仓库（没有配置远端）跳过，不阻断离线使用。
func validateCodeRepositoryRemotesReachable(candidates []codeRepositoryCandidate, credentialID uint) error {
	type remoteProbe struct {
		index      int
		candidate  codeRepositoryCandidate
		remoteName string
	}
	probes := make([]remoteProbe, 0, len(candidates))
	for index, candidate := range candidates {
		remoteName := codeRepositoryProbeRemote(candidate.SourceDir)
		if remoteName == "" {
			continue
		}
		probes = append(probes, remoteProbe{index: index, candidate: candidate, remoteName: remoteName})
	}
	if len(probes) == 0 {
		return nil
	}
	jobs := make(chan remoteProbe)
	errorsByIndex := make([]error, len(candidates))
	var workers sync.WaitGroup
	workers.Add(min(codeSessionRemoteProbeWorkers, len(probes)))
	for range min(codeSessionRemoteProbeWorkers, len(probes)) {
		go func() {
			defer workers.Done()
			for probe := range jobs {
				if err := validateCodeRepositoryRemoteReachable(probe.candidate, probe.remoteName, credentialID); err != nil {
					errorsByIndex[probe.index] = err
				}
			}
		}()
	}
	for _, probe := range probes {
		jobs <- probe
	}
	close(jobs)
	workers.Wait()
	for _, probeErr := range errorsByIndex {
		if probeErr != nil {
			return probeErr
		}
	}
	return nil
}

func validateCodeRepositoryRemoteReachable(candidate codeRepositoryCandidate, remoteName string, credentialID uint) error {
	cacheKey, err := codeSessionRemoteProbeCacheKey(candidate.SourceDir, remoteName, credentialID)
	if err != nil {
		return fmt.Errorf("仓库 %s 的远端配置不可用：%w", filepath.Base(candidate.SourceDir), err)
	}
	if codeSessionRemoteProbeCached(cacheKey, time.Now()) {
		return nil
	}
	if _, err := runCodeGitWithCredential(
		candidate.SourceDir, codeSessionRemoteProbeTimeout, credentialID,
		"-c", "credential.interactive=never", "ls-remote", "--heads", remoteName,
	); err != nil {
		return fmt.Errorf("仓库 %s 的远端不可访问：%w", filepath.Base(candidate.SourceDir), err)
	}
	codeSessionRemoteProbes.Lock()
	codeSessionRemoteProbes.succeeded[cacheKey] = time.Now().Add(codeSessionRemoteProbeTTL)
	codeSessionRemoteProbes.Unlock()
	return nil
}

func codeSessionRemoteProbeCacheKey(sourceDir, remoteName string, credentialID uint) (string, error) {
	remoteURL, err := runCodeGit(sourceDir, "remote", "get-url", "--", remoteName)
	if err != nil {
		return "", err
	}
	return strings.Join([]string{
		filepath.Clean(sourceDir), remoteName, strings.TrimSpace(remoteURL), strconv.FormatUint(uint64(credentialID), 10),
	}, "\x00"), nil
}

func codeSessionRemoteProbeCached(cacheKey string, now time.Time) bool {
	codeSessionRemoteProbes.Lock()
	defer codeSessionRemoteProbes.Unlock()
	expiresAt, exists := codeSessionRemoteProbes.succeeded[cacheKey]
	if !exists {
		return false
	}
	if !now.Before(expiresAt) {
		delete(codeSessionRemoteProbes.succeeded, cacheKey)
		return false
	}
	return true
}

func resetCodeSessionRemoteProbeCache() {
	codeSessionRemoteProbes.Lock()
	codeSessionRemoteProbes.succeeded = make(map[string]time.Time)
	codeSessionRemoteProbes.Unlock()
}

// codeRepositoryProbeRemote 选一个用于探测的远端：优先 origin，否则取第一个。
func codeRepositoryProbeRemote(sourceDir string) string {
	output, err := runCodeGit(sourceDir, "remote")
	if err != nil {
		return ""
	}
	names := strings.Fields(output)
	for _, name := range names {
		if name == "origin" {
			return name
		}
	}
	if len(names) > 0 {
		return names[0]
	}
	return ""
}
