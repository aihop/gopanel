package service

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// changelogMaxCommits / changelogMaxBytes 更新说明只做展示，超长的没人看，也别把库撑爆
const (
	changelogMaxCommits  = 50
	changelogMaxBytes    = 8000
	changelogFallbackNum = 10
)

// collectPipelineChangelog 取「上次成功构建的 commit..HEAD」之间的提交标题，一行一条。
// sinceCommit 为空（首次构建）或该 commit 在本地不存在（浅克隆截断、强推、变基）时，
// 退回到最近 changelogFallbackNum 条，保证发布记录里总有内容可看。
func collectPipelineChangelog(ctx context.Context, logger *PipelineLogger, workspace string, sinceCommit string) string {
	runGitLog := func(args ...string) (string, bool) {
		cmd := exec.CommandContext(ctx, "git", append([]string{"log", "--no-merges", "--pretty=format:%s"}, args...)...)
		cmd.Dir = workspace
		out, err := cmd.Output()
		if err != nil {
			return "", false
		}
		return string(out), true
	}

	since := strings.TrimSpace(sinceCommit)
	if since != "" {
		// 先确认这个 commit 本地存在，否则 git log A..HEAD 会直接报错
		checkCmd := exec.CommandContext(ctx, "git", "cat-file", "-e", since+"^{commit}")
		checkCmd.Dir = workspace
		if err := checkCmd.Run(); err != nil {
			logger.Info("上次构建的提交 %s 在本地历史中不存在（浅克隆或强推），改取最近 %d 条提交", shortCommit(since), changelogFallbackNum)
			since = ""
		}
	}

	var raw string
	if since != "" {
		out, ok := runGitLog(fmt.Sprintf("%s..HEAD", since))
		if !ok {
			logger.Info("提取提交记录失败，跳过更新说明")
			return ""
		}
		raw = out
		if strings.TrimSpace(raw) == "" {
			logger.Info("与上次构建相比没有新提交")
			return ""
		}
	} else {
		out, ok := runGitLog(fmt.Sprintf("-n%d", changelogFallbackNum))
		if !ok {
			logger.Info("提取提交记录失败，跳过更新说明")
			return ""
		}
		raw = out
	}

	changelog := normalizeChangelog(raw)
	if changelog == "" {
		return ""
	}
	logger.Info("已提取 %d 条提交作为本次更新说明", len(strings.Split(changelog, "\n")))
	return changelog
}

// normalizeChangelog 清掉空行、去重、限制条数与总长度。
// 截断按 rune 走，避免把中文切成半个字。
func normalizeChangelog(raw string) string {
	seen := make(map[string]struct{})
	lines := make([]string, 0, changelogMaxCommits)
	total := 0
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if _, ok := seen[line]; ok {
			continue
		}
		seen[line] = struct{}{}
		line = truncateRunes(line, 200)
		if total+len(line)+1 > changelogMaxBytes {
			lines = append(lines, "…（更新说明过长，已截断）")
			break
		}
		total += len(line) + 1
		lines = append(lines, line)
		if len(lines) >= changelogMaxCommits {
			lines = append(lines, "…（提交过多，已截断）")
			break
		}
	}
	return strings.Join(lines, "\n")
}

func truncateRunes(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "…"
}

func shortCommit(hash string) string {
	h := strings.TrimSpace(hash)
	if len(h) > 12 {
		return h[:12]
	}
	return h
}
