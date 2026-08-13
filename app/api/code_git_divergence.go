package api

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

// codeRepositoryDivergenceError 说清「本地和远端差在哪、该怎么办」。
//
// 原本只说「请先处理分支差异」——用户不知道该 pull 还是 push 还是 commit，
// 实测这条在会话创建里失败了 4 次。差异有三种形态，对应三种完全不同的动作，
// 混成一句话等于没给建议。
func codeRepositoryDivergenceError(sourceDir, targetBranch, remoteRef string) error {
	name := filepath.Base(sourceDir)
	ahead, behind, ok := codeRepositoryAheadBehind(sourceDir, targetBranch, remoteRef)
	if !ok {
		return fmt.Errorf("源仓库 %s 存在未提交变更，且本地分支 %s 与远端不一致；请先同步分支后再创建会话", name, targetBranch)
	}
	switch {
	case ahead > 0 && behind > 0:
		return fmt.Errorf(
			"源仓库 %s 存在未提交变更，且本地 %s 与远端已分叉（领先 %d 个提交、落后 %d 个）；请先合并或变基，再创建会话",
			name, targetBranch, ahead, behind,
		)
	case behind > 0:
		return fmt.Errorf(
			"源仓库 %s 存在未提交变更，且本地 %s 落后远端 %d 个提交；请先拉取（git pull）后再创建会话",
			name, targetBranch, behind,
		)
	case ahead > 0:
		return fmt.Errorf(
			"源仓库 %s 存在未提交变更，且本地 %s 领先远端 %d 个提交；请先推送（git push）后再创建会话",
			name, targetBranch, ahead,
		)
	default:
		return fmt.Errorf("源仓库 %s 存在未提交变更，且本地分支 %s 与远端不一致；请先同步分支后再创建会话", name, targetBranch)
	}
}

// codeRepositoryAheadBehind 数出本地相对远端领先/落后多少个提交。
// 取不到就返回 ok=false，让调用方退回不带数字的说法——
// 少一个数字总好过报一个错的数字。
func codeRepositoryAheadBehind(sourceDir, targetBranch, remoteRef string) (int, int, bool) {
	if strings.TrimSpace(targetBranch) == "" || strings.TrimSpace(remoteRef) == "" {
		return 0, 0, false
	}
	output, err := runCodeGit(sourceDir, "rev-list", "--left-right", "--count", remoteRef+"..."+targetBranch)
	if err != nil {
		return 0, 0, false
	}
	fields := strings.Fields(strings.TrimSpace(output))
	if len(fields) != 2 {
		return 0, 0, false
	}
	// --left-right 的左边是 remoteRef 独有的（本地落后的部分），右边是本地独有的。
	behind, behindErr := strconv.Atoi(fields[0])
	ahead, aheadErr := strconv.Atoi(fields[1])
	if behindErr != nil || aheadErr != nil {
		return 0, 0, false
	}
	return ahead, behind, true
}
