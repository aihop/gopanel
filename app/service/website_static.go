package service

import (
	"errors"
	"net"
	"path"
	"strings"

	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/utils/files"
)

func isIP(s string) bool {
	if s == "" {
		return false
	}
	if strings.Count(s, ":") >= 2 {
		return net.ParseIP(s) != nil
	}
	if idx := strings.Index(s, ":"); idx != -1 {
		s = s[:idx]
	}
	return net.ParseIP(s) != nil
}

func resolveWebsiteStaticRoot(alias string) string {
	alias = strings.TrimSpace(alias)
	if alias == "" {
		return ""
	}
	if strings.HasPrefix(alias, "/") {
		return alias
	}
	return path.Join(constant.AppInstallDir, "www", "sites", alias)
}

func ensureStaticWebsiteIndex(siteRoot string) error {
	fileOp := files.NewFileOp()
	siteRoot = strings.TrimSpace(siteRoot)
	if siteRoot == "" {
		return errors.New("静态网站目录不能为空")
	}
	if err := fileOp.CreateDir(siteRoot, 0755); err != nil {
		return err
	}
	indexPath := path.Join(siteRoot, "index.html")
	if fileOp.Stat(indexPath) {
		return nil
	}
	content := `<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>Welcome</title>
  <style>
    body {
      margin: 0;
      min-height: 100vh;
      display: flex;
      align-items: center;
      justify-content: center;
      background: linear-gradient(135deg, #eff6ff 0%, #ffffff 45%, #f8fafc 100%);
      color: #0f172a;
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
    }
    .card {
      max-width: 680px;
      margin: 24px;
      padding: 40px 36px;
      border: 1px solid rgba(148, 163, 184, 0.24);
      border-radius: 24px;
      background: rgba(255, 255, 255, 0.92);
      box-shadow: 0 24px 60px rgba(15, 23, 42, 0.08);
    }
    h1 {
      margin: 0 0 16px;
      font-size: 36px;
      line-height: 1.15;
    }
    p {
      margin: 0;
      font-size: 16px;
      line-height: 1.8;
      color: #475569;
    }
  </style>
</head>
<body>
  <div class="card">
    <h1>Welcome to GoPanel</h1>
    <p>静态网站已创建成功。你可以直接上传前端构建产物，或修改当前目录中的 index.html 开始部署页面。</p>
  </div>
</body>
</html>
`
	return fileOp.SaveFileWithByte(indexPath, []byte(content), 0644)
}
