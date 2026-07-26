//go:build darwin

package diskscan

// platformSkipDirs macOS 上必须跳过 /System/Volumes。
//
// APFS 用 firmlink 把数据卷挂在 /System/Volumes/Data，同时把 /Users、/Applications、
// /private 等目录“投影”到根上。实测这些路径的设备号完全相同（都是 16777234），
// 所以“不跨文件系统”那道判断拦不住它——遍历 / 时会把 /Users 和
// /System/Volumes/Data/Users 各走一遍，同一个文件被统计两次，
// 扫描耗时翻倍，Top-N 列表里还会出现同一文件的两条不同路径。
//
// 该目录被显式指定为扫描根时不跳过（见 Scan 里的 path != root 判断），
// 否则用户想扫数据卷会直接扫出空结果。
var platformSkipDirs = []string{"/System/Volumes"}
