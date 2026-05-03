package constant

import "os"

var (
	UpgradeUrl = getEnvOrDefault("GOPANEL_UPGRADE_URL", "https://gopanel.cn/api/panel/upgrade")
)

func getEnvOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
