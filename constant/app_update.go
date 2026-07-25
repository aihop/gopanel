package constant

import "os"

var (
	UpgradeUrl = getEnvOrDefault("GOPANEL_UPGRADE_URL", "https://gopanel.cn/api/panel/upgrade")
	// TrackUrl 面板事件上报地址，与 install.sh 的 API_TRACK_URL 同一个接口
	TrackUrl = getEnvOrDefault("GOPANEL_TRACK_URL", "https://gopanel.cn/api/panel/installs/track")
)

func getEnvOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
