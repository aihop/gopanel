//go:build !darwin && !linux

package diskscan

// isReadOnlyFS 其他平台不做判断，一律当作可写；标注不准好过误报不可删
func isReadOnlyFS(path string) bool {
	return false
}
