package api

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type codeQualityPubspec struct {
	Dependencies map[string]any `yaml:"dependencies"`
}

func detectDartQualityChecks(workDir, displayRoot string) []codeQualityCheck {
	content, err := os.ReadFile(filepath.Join(workDir, "pubspec.yaml"))
	if err != nil {
		return nil
	}
	var pubspec codeQualityPubspec
	if yaml.Unmarshal(content, &pubspec) != nil {
		return nil
	}
	executable := "dart"
	labelPrefix := "Dart"
	if _, usesFlutter := pubspec.Dependencies["flutter"]; usesFlutter || fileExists(filepath.Join(workDir, ".metadata")) {
		executable = "flutter"
		labelPrefix = "Flutter"
	}
	checks := []codeQualityCheck{
		newCodeQualityCheck("lint", labelPrefix+" analyze", workDir, displayRoot, executable, "analyze"),
	}
	if info, statErr := os.Stat(filepath.Join(workDir, "test")); statErr == nil && info.IsDir() {
		checks = append(checks, newCodeQualityCheck("test", labelPrefix+" test", workDir, displayRoot, executable, "test"))
	}
	applyDartQualityToolchain(checks, workDir)
	return checks
}

func applyDartQualityToolchain(checks []codeQualityCheck, toolchainRoot string) {
	for index := range checks {
		check := &checks[index]
		if check.Executable != "flutter" && check.Executable != "dart" {
			continue
		}
		candidate := filepath.Join(toolchainRoot, ".toolchains", "flutter", "bin", check.Executable)
		if fileExists(candidate) {
			check.Executable = candidate
		}
	}
}
