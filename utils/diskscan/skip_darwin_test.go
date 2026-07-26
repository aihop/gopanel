//go:build darwin

package diskscan

import "testing"

// macOS APFS 用 firmlink 把数据卷同时暴露在 / 和 /System/Volumes/Data 下，
// 两边设备号相同（实测均为同一个 dev），"不跨文件系统"拦不住，
// 会导致同一份数据被走两遍。这里锁死跳过规则，防止以后被改掉。
func TestDarwinSkipsSystemVolumes(t *testing.T) {
	mustSkip := []string{
		"/System/Volumes",
		"/System/Volumes/Data",
		"/System/Volumes/Data/Users/someone/big.dmg",
		"/System/Volumes/VM/swapfile0",
	}
	for _, p := range mustSkip {
		if !isSkipped(p, platformSkipDirs) {
			t.Errorf("%s 应被跳过（firmlink 重复遍历）", p)
		}
	}

	mustKeep := []string{"/Users/someone/big.dmg", "/Applications/Xcode.app", "/private/var/log/system.log", "/Systemx/data"}
	for _, p := range mustKeep {
		if isSkipped(p, platformSkipDirs) {
			t.Errorf("%s 不应被跳过", p)
		}
	}
}
