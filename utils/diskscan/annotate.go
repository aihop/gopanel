package diskscan

import "os"

// AnnotateRemovable 给扫描结果标注「能不能清理」。
//
// 只对最终 Top-N 结果做（最多几百条），不在遍历时做：遍历时每个目录都
// statfs 一次成本太高，而绝大多数文件根本不会进入结果。
//
// canElevate 表示删除操作能否以 root 身份执行（rootless 部署下由 gpc 代劳）。
// 为 true 时属主检查就没意义了——root 谁都删得动；只读卷仍然拦不住。
func AnnotateRemovable(files []FileItem, baseDir string, euid int, canElevate bool) {
	roCache := make(map[string]bool, 8)
	isRO := func(path string) bool {
		dir := parentDir(path)
		if v, ok := roCache[dir]; ok {
			return v
		}
		v := IsReadOnlyFS(dir)
		roCache[dir] = v
		return v
	}

	for i := range files {
		f := &files[i]
		switch {
		case IsProtected(f.Path, baseDir):
			f.Removable, f.Reason = false, "系统关键路径，禁止清理"
		case f.IsContainer:
			f.Removable, f.Reason = false, "容器存储层，请到容器页面执行 prune"
		case IsJournalInternal(f.Path):
			f.Removable, f.Reason = false, "journald 内部文件，请用 journalctl --vacuum-size 清理"
		case isRO(f.Path):
			f.Removable, f.Reason = false, "位于只读挂载点，任何权限都无法删除"
		default:
			if !canElevate && euid != 0 {
				if uid, ok := ownerUID(f.Path); ok && uid != euid {
					f.Removable = false
					f.Reason = "属主不是面板运行用户，需要安装 gpc 提权"
					continue
				}
			}
			f.Removable, f.Reason = true, ""
		}
	}
}

func parentDir(path string) string {
	for i := len(path) - 1; i > 0; i-- {
		if path[i] == os.PathSeparator {
			return path[:i]
		}
	}
	return "/"
}
