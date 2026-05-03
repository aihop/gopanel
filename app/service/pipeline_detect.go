package service

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

func detectRunnerPresetFromDir(dir string) RunnerPresetDetectResult {
	hits := make([]string, 0, 6)
	hasFile := func(rel string) bool {
		_, err := os.Stat(filepath.Join(dir, rel))
		return err == nil
	}
	appendHit := func(hit string) {
		hits = append(hits, hit)
	}
	if hasFile("go.mod") {
		appendHit("go.mod")
	}
	if hasFile("main.go") {
		appendHit("main.go")
	}
	if hasFile("requirements.txt") {
		appendHit("requirements.txt")
	}
	if hasFile("pyproject.toml") {
		appendHit("pyproject.toml")
	}
	if hasFile("app.py") {
		appendHit("app.py")
	}
	if hasFile("manage.py") {
		appendHit("manage.py")
	}
	if hasFile("composer.json") {
		appendHit("composer.json")
	}
	if hasFile("artisan") {
		appendHit("artisan")
	}
	if hasFile("public/index.php") {
		appendHit("public/index.php")
	}
	if hasFile("package.json") {
		appendHit("package.json")
	}
	if hasFile("go.mod") || hasFile("main.go") || dirHasEntry(filepath.Join(dir, "cmd")) {
		return RunnerPresetDetectResult{Preset: "go", Hits: hits}
	}
	if hasFile("requirements.txt") || hasFile("pyproject.toml") || hasFile("app.py") || hasFile("manage.py") {
		return RunnerPresetDetectResult{Preset: "python", Hits: hits}
	}
	if hasFile("composer.json") || hasFile("artisan") || hasFile("public/index.php") {
		return RunnerPresetDetectResult{Preset: "php", Hits: hits}
	}
	packageJSONPath := filepath.Join(dir, "package.json")
	if data, err := os.ReadFile(packageJSONPath); err == nil {
		var pkg pipelinePackageJSON
		if json.Unmarshal(data, &pkg) == nil {
			if hasPackageDep(pkg, "nuxt") {
				appendHit("package.json:nuxt")
				return RunnerPresetDetectResult{Preset: "nuxt", Hits: hits}
			}
			if hasPackageDep(pkg, "next") {
				appendHit("package.json:next")
				return RunnerPresetDetectResult{Preset: "next", Hits: hits}
			}
			return RunnerPresetDetectResult{Preset: "node", Hits: hits}
		}
	}
	return RunnerPresetDetectResult{Preset: "custom", Hits: hits}
}
func hasPackageDep(pkg pipelinePackageJSON, name string) bool {
	if _, ok := pkg.Dependencies[name]; ok {
		return true
	}
	_, ok := pkg.DevDependencies[name]
	return ok
}
func dirHasEntry(dir string) bool {
	entries, err := os.ReadDir(dir)
	return err == nil && len(entries) > 0
}
func buildPipelineRepoURL(repoURL, authType, authData string) string {
	repoURL = strings.TrimSpace(repoURL)
	if authType == "token" && authData != "" {
		tokenEncoded := url.QueryEscape(authData)
		if strings.HasPrefix(repoURL, "https://") {
			repoURL = strings.Replace(repoURL, "https://", fmt.Sprintf("https://%s@", tokenEncoded), 1)
		} else if strings.HasPrefix(repoURL, "http://") {
			repoURL = strings.Replace(repoURL, "http://", fmt.Sprintf("http://%s@", tokenEncoded), 1)
		}
	} else if authType == "password" && authData != "" {
		parts := strings.SplitN(authData, ":", 2)
		if len(parts) == 2 {
			username := url.QueryEscape(parts[0])
			password := url.QueryEscape(parts[1])
			authString := fmt.Sprintf("%s:%s", username, password)
			if strings.HasPrefix(repoURL, "https://") {
				repoURL = strings.Replace(repoURL, "https://", fmt.Sprintf("https://%s@", authString), 1)
			} else if strings.HasPrefix(repoURL, "http://") {
				repoURL = strings.Replace(repoURL, "http://", fmt.Sprintf("http://%s@", authString), 1)
			}
		} else {
			if strings.HasPrefix(repoURL, "https://") {
				repoURL = strings.Replace(repoURL, "https://", fmt.Sprintf("https://%s@", authData), 1)
			} else if strings.HasPrefix(repoURL, "http://") {
				repoURL = strings.Replace(repoURL, "http://", fmt.Sprintf("http://%s@", authData), 1)
			}
		}
	}
	return repoURL
}
