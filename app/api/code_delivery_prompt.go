package api

import (
	"strings"

	"github.com/aihop/gopanel/app/model"
)

const codeManagedGitDeliveryInstruction = `

[GoPanel Git 交付约束]
当前会话位于 GoPanel 管理的隔离 Git Worktree。你可以在当前 Worktree 中修改、暂存并提交代码，但不要执行 git push，不要直接更新远端目标分支，也不要切换、重置或删除项目目标分支。完成开发后只需保留本地提交；GoPanel 交付时仅按仓库依赖顺序安全合并到本地目标分支。需要发布时，用户会再单独执行“推送远端”。`

func codeManagedDeliveryPrompt(session *model.AIDevSession, prompt string) string {
	if session == nil || (strings.TrimSpace(session.WorktreeBranch) == "" && session.IsolationMode != codeIsolationMultiWorktree) {
		return prompt
	}
	return strings.TrimSpace(prompt) + codeManagedGitDeliveryInstruction
}
