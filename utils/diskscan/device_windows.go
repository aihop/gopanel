//go:build windows

package diskscan

// deviceOf Windows 上没有设备号概念，返回 false 让调用方退化为允许跨设备。
func deviceOf(path string) (uint64, bool) {
	return 0, false
}
