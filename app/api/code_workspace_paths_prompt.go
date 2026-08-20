package api

import (
	"fmt"
	"strings"

	"github.com/aihop/gopanel/app/model"
)

// 多源目录项目的工作区是 GoPanel 合成的：目录本身只放一份清单，
// 每个源仓库以符号链接的形式挂在下面。
//
// 权限没有问题——真实路径经 EvalSymlinks 解析后已作为 --add-dir 传给了执行器，
// 穿过链接写主仓是允许的。别扭的是路径身份：模型看到的是
// .../project_2/apay/src/x.ts，而工具一解析链接又变回 /Users/hugh/.../apay/src/x.ts，
// 两套路径来回横跳，模型容易把它们当成两个地方。
//
// 这里把映射直接摆给它看，让它知道两者是同一份文件、可以直接用真实路径。
const codeWorkspacePathsHeader = `

[GoPanel 工作区路径]
当前工作目录是 GoPanel 为多仓库项目合成的容器，下面每一项都是指向真实仓库的符号链接，两侧是同一份文件，可直接使用真实路径读写：`

func codeWorkspacePathsPrompt(session *model.AIDevSession, prompt string) string {
	mapping := codeWorkspacePathMapping(session)
	if mapping == "" {
		return prompt
	}
	return strings.TrimSpace(prompt) + codeWorkspacePathsHeader + mapping
}

// codeWorkspacePathMapping 生成「链接名 → 真实路径」清单。
// 不是合成工作区（单仓项目直接以源目录为工作目录、或 Worktree 隔离）时返回空，
// 那些情况下模型看到的本来就是唯一且真实的路径，再加一段只会是噪音。
func codeWorkspacePathMapping(session *model.AIDevSession) string {
	if session == nil || strings.TrimSpace(session.WorkDir) == "" {
		return ""
	}
	if session.IsolationMode == codeIsolationMultiWorktree || strings.TrimSpace(session.WorktreeBranch) != "" {
		return ""
	}
	manifest, err := readAIProjectWorkspaceManifest(session.WorkDir)
	if err != nil || len(manifest.Sources) == 0 {
		return ""
	}
	var builder strings.Builder
	for _, source := range manifest.Sources {
		linkName := strings.TrimSpace(source.LinkName)
		path := strings.TrimSpace(source.Path)
		if linkName == "" || path == "" {
			continue
		}
		builder.WriteString(fmt.Sprintf("\n- %s/ → %s", linkName, path))
	}
	return builder.String()
}
