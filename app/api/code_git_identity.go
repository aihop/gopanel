package api

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
)

// 面板产出的提交统一用这个身份落库，git log 里能一眼把人和面板分开。
// 收成常量而不是在每个调用点写字面量：之前十几处各写一份，
// 改邮箱域名时就漏掉了 code_delivery_conflict_complete.go 那一处。
const (
	codeGitAuthorName  = "GoPanel Code"
	codeGitAuthorEmail = "code@gopanel.cn"
)

// 合并提交首行的长度上限。任务标题是用户自由填写的，
// 超长首行会把 git log --oneline 和平台的提交列表撑坏。
const codeCommitSubjectMaxRunes = 72

// codeGitAuthorArgs 返回面板提交所需的身份参数。
func codeGitAuthorArgs() []string {
	return []string{
		"-c", "user.name=" + codeGitAuthorName,
		"-c", "user.email=" + codeGitAuthorEmail,
	}
}

// codeGitAuthoredArgs 把身份参数拼在具体的 git 子命令前面。
// 每次都返回新切片，调用点可以安全地继续 append。
func codeGitAuthoredArgs(args ...string) []string {
	return append(codeGitAuthorArgs(), args...)
}

// codeResolvedMergeCommitArgs 是冲突解决后收尾提交用的参数。
//
// --cleanup=strip 不能省：冲突后复用的 MERGE_MSG 里带着 Git 追加的
// 「# Conflicts:」注释块，而 --no-edit 不走编辑器时默认 cleanup 是 whitespace，
// 注释会原样留在提交正文里。
func codeResolvedMergeCommitArgs() []string {
	return codeGitAuthoredArgs("-c", "commit.gpgsign=false", "commit", "--no-edit", "--cleanup=strip")
}

// codeDeliveryMergeMessage 生成交付合并提交的信息。
//
// Git 默认的 `merge --no-edit` 在这里产出的是 "Merge commit 'sha' into HEAD"，
// 回看历史时既看不出合的是哪个会话，也看不出这次交付做了什么。
// 仓库名只在多仓交付时带上，单仓场景没有歧义，不必占首行长度。
func codeDeliveryMergeMessage(session *model.AIDevSession, repositoryName string) string {
	subject := "merge: " + codeDeliverySubject(session)
	if session != nil {
		scope := fmt.Sprintf("session #%d", session.ID)
		if name := codeTrailerValue(repositoryName); name != "" {
			scope += ", " + name
		}
		subject += " (" + scope + ")"
	}
	return codeAppendCommitTrailers(subject, session)
}

// codeCommitMessageWithTrailers 在用户填写的提交说明后面补上可追溯信息。
// 必须在提交说明校验之后调用：500 字符的上限约束的是用户输入，不含这些 trailer。
func codeCommitMessageWithTrailers(message string, session *model.AIDevSession) string {
	return codeAppendCommitTrailers(strings.TrimRight(message, "\n"), session)
}

func codeAppendCommitTrailers(message string, session *model.AIDevSession) string {
	trailers := codeCommitTrailers(session)
	if len(trailers) == 0 {
		return message
	}
	return message + "\n\n" + strings.Join(trailers, "\n")
}

// codeCommitTrailers 给面板产出的提交补可追溯信息。
//
// 作者固定成 GoPanel Code 之后，git log --author 只能看出「是面板提交的」，
// 看不出是谁发起、哪个会话、哪个执行器和模型写的。这些信息放在 trailer 里，
// git interpret-trailers 和主流代码托管平台都能直接解析。
func codeCommitTrailers(session *model.AIDevSession) []string {
	if session == nil || session.ID == 0 {
		return nil
	}
	trailers := []string{fmt.Sprintf("Session-Id: %d", session.ID)}
	if session.LastTaskID > 0 {
		trailers = append(trailers, fmt.Sprintf("Task-Id: %d", session.LastTaskID))
	}
	if executor := codeTrailerValue(session.AgentName); executor != "" {
		trailers = append(trailers, "Executor: "+executor)
	}
	if providerModel := codeTrailerValue(session.ProviderModel); providerModel != "" {
		trailers = append(trailers, "Model: "+providerModel)
	}
	if author := codeSessionAuthorTrailer(session.UserID); author != "" {
		trailers = append(trailers, "Co-Authored-By: "+author)
	}
	return trailers
}

// codeSessionAuthorTrailer 把发起会话的真人写进 Co-Authored-By。
// 查不到用户就返回空：可追溯信息缺一条，不该让整个交付失败。
func codeSessionAuthorTrailer(userID uint) string {
	if userID == 0 || global.DB == nil {
		return ""
	}
	var user model.User
	if err := global.DB.Select("nick_name", "email").First(&user, userID).Error; err != nil {
		return ""
	}
	email := codeTrailerValue(user.Email)
	if email == "" {
		return ""
	}
	name := codeTrailerValue(user.NickName)
	if name == "" {
		name = email
	}
	return fmt.Sprintf("%s <%s>", name, email)
}

// codeDeliverySessionForMessage 为生成提交信息加载会话。
// 只服务于提交信息，查不到就返回 nil 让调用方退回不带会话信息的版本——
// 交付已经走到合并这一步，不该因为凑不齐一条日志就失败。
func codeDeliverySessionForMessage(sessionID uint) *model.AIDevSession {
	if sessionID == 0 || global.DB == nil {
		return nil
	}
	var session model.AIDevSession
	if err := global.DB.First(&session, sessionID).Error; err != nil {
		return nil
	}
	return &session
}

// codeDeliverySubject 取交付合并提交的首行。
// 优先用任务标题——那是用户自己写的，最能说明这次交付做了什么；
// 任务不可用时退回会话标题，都没有才用会话编号兜底。
func codeDeliverySubject(session *model.AIDevSession) string {
	if session == nil {
		return "交付会话变更"
	}
	if subject := codeCommitSubjectText(codeSessionTaskTitle(session)); subject != "" {
		return subject
	}
	if subject := codeCommitSubjectText(session.Title); subject != "" {
		return subject
	}
	return fmt.Sprintf("交付会话 #%d 的变更", session.ID)
}

// codeSessionTaskTitle 取会话当前任务的标题。
// CurrentTaskTitle 是列表接口顺带填好的内存字段，有值就不必再查库。
func codeSessionTaskTitle(session *model.AIDevSession) string {
	if title := strings.TrimSpace(session.CurrentTaskTitle); title != "" {
		return title
	}
	if session.LastTaskID == 0 || global.DB == nil {
		return ""
	}
	var task model.AITask
	if err := global.DB.Select("title").First(&task, session.LastTaskID).Error; err != nil {
		return ""
	}
	return task.Title
}

// codeCommitSubjectText 把任意标题压成合法的提交首行：
// 只取第一行、折叠空白、按 rune 截断，避免多字节字符被截半。
func codeCommitSubjectText(text string) string {
	if index := strings.IndexAny(text, "\r\n"); index >= 0 {
		text = text[:index]
	}
	text = strings.Join(strings.Fields(text), " ")
	if utf8.RuneCountInString(text) <= codeCommitSubjectMaxRunes {
		return text
	}
	return strings.TrimSpace(string([]rune(text)[:codeCommitSubjectMaxRunes])) + "…"
}

// codeTrailerValue 清掉换行和尖括号后再写进 trailer。
// 昵称、执行器名都是外部可控的，带换行的值能凭空伪造出额外的 trailer 行，
// 带尖括号的值能把 Co-Authored-By 的邮箱地址撑坏。
//
// 换行换成空格而不是直接删掉：删掉会把上下两行的词粘成一个，
// 让本就异常的值更难看懂。
func codeTrailerValue(value string) string {
	cleaned := strings.Map(func(character rune) rune {
		switch character {
		case '\r', '\n':
			return ' '
		case '<', '>':
			return -1
		}
		return character
	}, value)
	return strings.Join(strings.Fields(cleaned), " ")
}
