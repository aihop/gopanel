package middleware

import "strings"

func IsMobileNodeProxyTargetAllowed(method, path string) bool {
	clean := strings.Trim(strings.TrimSpace(path), "/")
	clean = strings.TrimPrefix(clean, "api/")
	method = strings.ToUpper(strings.TrimSpace(method))

	switch clean {
	case "mobile/app/containers":
		return method == "GET"
	case "mobile/app/containers/operate", "mobile/app/containers/publish-website":
		return method == "POST"
	case "mobile/app/resources/websites", "mobile/app/resources/databases", "mobile/app/resources/ssl", "mobile/app/resources/apps":
		return method == "GET"
	case "mobile/app/resources/websites/domains":
		return method == "POST"
	}

	parts := strings.Split(clean, "/")
	return method == "GET" && len(parts) == 5 && parts[0] == "mobile" && parts[1] == "app" &&
		parts[2] == "containers" && parts[3] != "" && parts[4] == "publish-options"
}
