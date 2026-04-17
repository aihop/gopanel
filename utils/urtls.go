package utils

import (
	"os"
	"path/filepath"
	"runtime"
)

func GetConfigPath(path, file string) string {
	switch runtime.GOOS {
	case "linux":
		return filepath.Join(path, file)
	default:
		return filepath.Join(os.Getenv("HOME"), ".gopanel", file)
	}
}

func RemoveDuplicateStrings(input []string) []string {
	seen := make(map[string]struct{})
	var result []string
	for _, str := range input {
		if _, ok := seen[str]; !ok {
			seen[str] = struct{}{}
			result = append(result, str)
		}
	}
	return result
}
