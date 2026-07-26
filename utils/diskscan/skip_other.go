//go:build !darwin

package diskscan

// platformSkipDirs 非 macOS 平台没有 firmlink 那类需要额外跳过的目录
var platformSkipDirs []string
