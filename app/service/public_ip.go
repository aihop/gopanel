package service

import (
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/aihop/gopanel/init/geo"
)

var (
	publicIPMu     sync.Mutex
	publicIPCache  string
	publicIPExpire time.Time
	ipv4Regexp     = regexp.MustCompile(`(\d{1,3}\.){3}\d{1,3}`)
)

// ResolvePublicIP 返回服务器真实的公网出口 IP。
//
// 注意：GetOutboundIP 取的是本机路由源地址（conn.LocalAddr），NAT 环境（国内云主机绝大多数如此）
// 下拿到的是 10.x/172.x/192.168.x 内网 IP，无法用于地理定位。这里通过外部 echo 服务获取真实公网 IP，
// 优先国内可访问的端点，保证国内服务器能拿到结果；结果缓存 1 小时。失败返回空字符串。
func ResolvePublicIP() string {
	publicIPMu.Lock()
	defer publicIPMu.Unlock()
	if publicIPCache != "" && time.Now().Before(publicIPExpire) {
		return publicIPCache
	}
	// 优先国内可访问端点（保证国内服务器一定能拿到公网 IP），最后附一个国际端点兜底。
	endpoints := []string{
		"https://myip.ipip.net",
		"https://4.ipw.cn",
		"https://ip.3322.net",
		"https://api.ipify.org",
	}
	client := &http.Client{Timeout: 3 * time.Second}
	for _, url := range endpoints {
		resp, err := client.Get(url)
		if err != nil {
			continue
		}
		body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
		resp.Body.Close()
		if err != nil {
			continue
		}
		if ip := ipv4Regexp.FindString(string(body)); ip != "" {
			publicIPCache = ip
			publicIPExpire = time.Now().Add(time.Hour)
			return ip
		}
	}
	return ""
}

// IsChinaMainlandServer 判断当前服务器是否位于中国大陆，用于选择下载镜像源（国内 gitcode / 海外 github）。
func IsChinaMainlandServer() bool {
	ip := ResolvePublicIP()
	if ip == "" {
		// 拿不到公网 IP 时退回本机地址：直连公网 IP（未经 NAT）的机器仍可正确判断。
		ip = GetOutboundIP()
	}
	return strings.Contains(geo.Region(ip), "中国")
}
