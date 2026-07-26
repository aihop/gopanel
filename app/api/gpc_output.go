package api

import (
	"strings"

	"github.com/aihop/gopanel/utils/gpc"
)

// gpcOutput 安全地取 gpc 响应里的 output。
// gpc.Do 现在保证返回非 nil，但错误分支里读响应字段的写法太容易被复制，
// 这里给一个不会炸的取值口子。
func gpcOutput(resp *gpc.Response) string {
	if resp == nil {
		return ""
	}
	return strings.TrimSpace(resp.Output)
}
