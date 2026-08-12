package api

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
	"gorm.io/gorm"
)

// 会话残留的分类。分类只是给界面看的预期，真正的删除仍然走
// cleanupDeliveredCodeSessionWorktrees 那套守卫（在管理目录内、是有效 worktree、
// 无未提交变更、分支已合并），所以即使这里判错也删不掉不该删的东西。
const (
	codeResidueStateSafe     = "safe"     // 可安全清理
	codeResidueStateDirty    = "dirty"    // 有未提交变更
	codeResidueStateUnmerged = "unmerged" // 有未合并提交
	codeResidueStateActive   = "active"   // 会话仍在使用
	codeResidueStateOrphan   = "orphan"   // 目录还在，会话记录已不存在
)

type codeWorktreeResidue struct {
	SessionID     uint     `json:"sessionId"`
	SessionTitle  string   `json:"sessionTitle,omitempty"`
	ProjectID     uint     `json:"projectId,omitempty"`
	SessionStatus string   `json:"sessionStatus,omitempty"`
	State         string   `json:"state"`
	Reason        string   `json:"reason,omitempty"`
	Directories   []string `json:"directories"`
	Branches      []string `json:"branches,omitempty"`
	DiskBytes     int64    `json:"diskBytes"`
}

// codeResidueDirectoryKind 区分同一会话在管理目录下留下的三种目录：
// 开发用的 session_N、单仓交付用的 delivery_N、多仓集成用的 delivery_N_multi。
func parseCodeResidueDirectory(name string) (uint, bool) {
	trimmed := strings.TrimSuffix(name, "_multi")
	for _, prefix := range []string{"session_", "delivery_"} {
		if !strings.HasPrefix(trimmed, prefix) {
			continue
		}
		id, err := strconv.ParseUint(strings.TrimPrefix(trimmed, prefix), 10, 64)
		if err != nil || id == 0 {
			return 0, false
		}
		return uint(id), true
	}
	return 0, false
}

// scanCodeWorktreeResidues 枚举某个用户管理目录下的全部会话残留。
// 按会话聚合而不是按目录：一个会话可能同时留下开发、交付、集成三份目录，
// 分开列会让用户以为要清三次。
func scanCodeWorktreeResidues(userID uint) ([]codeWorktreeResidue, error) {
	root := aiProjectWorktreeRoot(userID)
	entries, err := os.ReadDir(root)
	if errors.Is(err, os.ErrNotExist) {
		return []codeWorktreeResidue{}, nil
	}
	if err != nil {
		return nil, err
	}
	grouped := make(map[uint][]string)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		sessionID, ok := parseCodeResidueDirectory(entry.Name())
		if !ok {
			continue
		}
		grouped[sessionID] = append(grouped[sessionID], filepath.Join(root, entry.Name()))
	}
	residues := make([]codeWorktreeResidue, 0, len(grouped))
	for sessionID, directories := range grouped {
		sort.Strings(directories)
		residues = append(residues, classifyCodeSessionResidue(sessionID, directories))
	}
	sort.SliceStable(residues, func(left, right int) bool {
		return residues[left].SessionID > residues[right].SessionID
	})
	return residues, nil
}

func classifyCodeSessionResidue(sessionID uint, directories []string) codeWorktreeResidue {
	residue := codeWorktreeResidue{
		SessionID:   sessionID,
		Directories: directories,
		DiskBytes:   codeResidueDiskUsage(directories),
	}
	// 数据库不可用时无从判断会话是否还在跑，只能一律按「在用」处理：
	// 这个接口唯一的下游动作是删除，判不准的时候必须偏向不删。
	if global.DB == nil {
		residue.State = codeResidueStateActive
		residue.Reason = "会话状态不可读，保守起见不清理"
		return residue
	}
	var session model.AIDevSession
	if err := global.DB.First(&session, sessionID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			residue.State = codeResidueStateOrphan
			residue.Reason = "会话记录已不存在，目录是上一次异常退出的残留"
			return residue
		}
		residue.State = codeResidueStateActive
		residue.Reason = "会话状态读取失败，保守起见不清理"
		return residue
	}
	residue.SessionTitle, residue.ProjectID = session.Title, session.ProjectID
	residue.SessionStatus = session.Status
	residue.Branches = codeResidueBranches(&session)

	switch session.Status {
	case codeSessionStatusInitializing, codeSessionStatusActive, codeSessionStatusDelivering:
		residue.State = codeResidueStateActive
		residue.Reason = "会话仍在使用中"
		return residue
	}
	state, reason := inspectCodeResidueSafety(&session, directories)
	residue.State, residue.Reason = state, reason
	return residue
}

// inspectCodeResidueSafety 用与实际清理相同的判据预判结果：
// 有未提交变更、或分支还有没合进源仓的提交，都不能清。
func inspectCodeResidueSafety(session *model.AIDevSession, directories []string) (string, string) {
	repositories, err := loadCodeSessionRepositories(session.ID)
	if err != nil {
		return codeResidueStateActive, "会话仓库信息读取失败，保守起见不清理"
	}
	type worktreeCheck struct{ workDir, sourceDir, branch, label string }
	checks := make([]worktreeCheck, 0, len(repositories)+1)
	if len(repositories) > 0 {
		for index := range repositories {
			repository := &repositories[index]
			checks = append(checks, worktreeCheck{
				workDir: repository.WorktreeDir, sourceDir: repository.SourceDir,
				branch: repository.Branch, label: "仓库 " + repository.LinkName,
			})
		}
	} else if strings.TrimSpace(session.SourceWorkDir) != "" {
		checks = append(checks, worktreeCheck{
			workDir: session.WorkDir, sourceDir: session.SourceWorkDir,
			branch: session.WorktreeBranch, label: "会话工作区",
		})
	}
	for _, check := range checks {
		if _, err := os.Lstat(check.workDir); errors.Is(err, os.ErrNotExist) {
			continue
		}
		if !isCodeGitWorktree(check.workDir) {
			continue
		}
		status, statusErr := runCodeGit(check.workDir, "status", "--porcelain")
		if statusErr != nil {
			return codeResidueStateActive, check.label + " 状态读取失败，保守起见不清理"
		}
		if strings.TrimSpace(status) != "" {
			return codeResidueStateDirty, check.label + " 仍有未提交变更"
		}
		if strings.TrimSpace(check.branch) == "" || !isCodeGitWorktree(check.sourceDir) {
			continue
		}
		if _, err := runCodeGit(check.sourceDir, "merge-base", "--is-ancestor", check.branch, "HEAD"); err != nil {
			return codeResidueStateUnmerged, check.label + " 的分支 " + check.branch + " 还有未合入的提交"
		}
	}
	if len(checks) == 0 && len(directories) > 0 {
		return codeResidueStateOrphan, "会话已无仓库绑定，目录是残留"
	}
	return codeResidueStateSafe, ""
}

func codeResidueBranches(session *model.AIDevSession) []string {
	branches := make([]string, 0, 2)
	if branch := strings.TrimSpace(session.WorktreeBranch); branch != "" {
		branches = append(branches, branch)
	}
	repositories, err := loadCodeSessionRepositories(session.ID)
	if err != nil {
		return branches
	}
	for index := range repositories {
		if branch := strings.TrimSpace(repositories[index].Branch); branch != "" {
			branches = append(branches, branch)
		}
	}
	sort.Strings(branches)
	return slicesCompactCodeBranches(branches)
}

func slicesCompactCodeBranches(branches []string) []string {
	if len(branches) < 2 {
		return branches
	}
	compacted := branches[:1]
	for _, branch := range branches[1:] {
		if branch != compacted[len(compacted)-1] {
			compacted = append(compacted, branch)
		}
	}
	return compacted
}

// removeCodeResidueDirectories 删除孤儿残留目录。
//
// 孤儿是「会话记录没了、目录还在」，没有会话可以交给守卫函数去判，
// 所以这里自己把守边界：每个目录都必须落在该用户的管理目录下，
// 且必须是 parseCodeResidueDirectory 认得的命名。少一道校验，
// 一个被污染的路径就能变成任意目录删除。
func removeCodeResidueDirectories(userID uint, directories []string) error {
	root := aiProjectWorktreeRoot(userID)
	for _, directory := range directories {
		cleaned := filepath.Clean(directory)
		if !isPathInside(cleaned, root) || filepath.Dir(cleaned) != filepath.Clean(root) {
			return errors.New("残留目录不在 GoPanel 管理目录中，已跳过")
		}
		if _, ok := parseCodeResidueDirectory(filepath.Base(cleaned)); !ok {
			return errors.New("残留目录命名不受管理，已跳过")
		}
		info, err := os.Lstat(cleaned)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		// 符号链接不能跟着删：链接指向哪里不受管理目录约束。
		if info.Mode()&os.ModeSymlink != 0 {
			return errors.New("残留目录是符号链接，已跳过")
		}
		if err := os.RemoveAll(cleaned); err != nil {
			return err
		}
	}
	return nil
}

// codeResidueDiskUsage 只是给用户一个「清掉能省多少」的量级参考，
// 统计失败不该让整个列表接口失败，所以出错就当 0。
func codeResidueDiskUsage(directories []string) int64 {
	var total int64
	for _, directory := range directories {
		_ = filepath.WalkDir(directory, func(path string, entry os.DirEntry, err error) error {
			if err != nil || entry.IsDir() {
				return nil
			}
			if info, infoErr := entry.Info(); infoErr == nil {
				total += info.Size()
			}
			return nil
		})
	}
	return total
}
