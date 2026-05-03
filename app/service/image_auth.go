package service

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/pkg/homedir"
	"os"
	"path"
	"strings"
)

func formatFileSize(fileSize int64) (size string) {
	if fileSize < 1024 {
		return fmt.Sprintf("%.2fB", float64(fileSize)/float64(1))
	} else if fileSize < (1024 * 1024) {
		return fmt.Sprintf("%.2fKB", float64(fileSize)/float64(1024))
	} else if fileSize < (1024 * 1024 * 1024) {
		return fmt.Sprintf("%.2fMB", float64(fileSize)/float64(1024*1024))
	} else if fileSize < (1024 * 1024 * 1024 * 1024) {
		return fmt.Sprintf("%.2fGB", float64(fileSize)/float64(1024*1024*1024))
	} else if fileSize < (1024 * 1024 * 1024 * 1024 * 1024) {
		return fmt.Sprintf("%.2fTB", float64(fileSize)/float64(1024*1024*1024*1024))
	} else {
		return fmt.Sprintf("%.2fEB", float64(fileSize)/float64(1024*1024*1024*1024*1024))
	}
}
func normalizeImageID(id string) string {
	id = strings.TrimSpace(id)
	return strings.TrimPrefix(id, "sha256:")
}
func checkUsed(imageID string, containers []types.Container) bool {
	imageID = normalizeImageID(imageID)
	for _, container := range containers {
		if normalizeImageID(container.ImageID) == imageID {
			return true
		}
	}
	return false
}
func loadAuthInfo(image string) (bool, string) {
	hasCreds, creds := loadAuthCredentials(image)
	if !hasCreds {
		return false, ""
	}
	parts := strings.SplitN(creds, ":", 2)
	if len(parts) != 2 {
		return false, ""
	}
	authConfig := registry.AuthConfig{Username: parts[0], Password: parts[1]}
	encodedJSON, err := json.Marshal(authConfig)
	if err != nil {
		return false, ""
	}
	authStr := base64.URLEncoding.EncodeToString(encodedJSON)
	return true, authStr
}
func loadAuthCredentials(image string) (bool, string) {
	if !strings.Contains(image, "/") {
		return false, ""
	}
	homeDir := homedir.Get()
	confPath := path.Join(homeDir, ".docker/config.json")
	configFileBytes, err := os.ReadFile(confPath)
	if err != nil {
		return false, ""
	}
	var config dockerConfig
	if err = json.Unmarshal(configFileBytes, &config); err != nil {
		return false, ""
	}
	var (
		user   string
		passwd string
	)
	imagePrefix := strings.Split(image, "/")[0]
	if val, ok := config.Auths[imagePrefix]; ok {
		itemByte, _ := base64.StdEncoding.DecodeString(val.Auth)
		itemStr := string(itemByte)
		if strings.Contains(itemStr, ":") {
			user = strings.Split(itemStr, ":")[0]
			passwd = strings.Split(itemStr, ":")[1]
		}
	}
	if user == "" {
		return false, ""
	}
	return true, user + ":" + passwd
}
