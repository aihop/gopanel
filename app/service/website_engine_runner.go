package service

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/aihop/gopanel/global"
	"github.com/docker/docker/api/types/mount"
)

func parseRunnerConfig(raw map[string]interface{}) runnerConfig {
	rc := runnerConfig{Mode: "build_run", BaseImage: "node:20-alpine", WorkingDir: "/var/www/app", ContainerPort: "3000", StartCommand: "node .output/server/index.mjs", Env: map[string]string{}}
	if raw == nil {
		return rc
	}
	if v := strings.TrimSpace(asString(raw["mode"])); v != "" {
		rc.Mode = v
	}
	if v := strings.TrimSpace(asString(raw["baseImage"])); v != "" {
		rc.BaseImage = v
	}
	if v := strings.TrimSpace(asString(raw["workingDir"])); v != "" {
		rc.WorkingDir = v
		rc.HasCustomWorkingDir = true
	}
	if v := strings.TrimSpace(asNumberString(raw["containerPort"])); v != "" {
		rc.ContainerPort = v
	}
	if v := strings.TrimSpace(asNumberString(raw["hostPort"])); v != "" {
		rc.HostPort = v
	}
	if v := strings.TrimSpace(asString(raw["runnerUser"])); v != "" {
		rc.RunnerUser = v
	}
	if v, ok := raw["startCommand"]; ok {
		rc.StartCommand = strings.TrimSpace(asString(v))
	}
	if v := strings.TrimSpace(asString(raw["buildCommand"])); v != "" {
		rc.BuildCommand = v
	}
	if v := asString(raw["preStart"]); strings.TrimSpace(v) != "" {
		rc.PreStart = v
	}
	if envRaw, ok := raw["env"].(map[string]interface{}); ok {
		for k, v := range envRaw {
			rc.Env[k] = asString(v)
		}
	} else if envRaw, ok := raw["env"].(map[string]string); ok {
		for k, v := range envRaw {
			rc.Env[k] = v
		}
	}
	if pathsRaw, ok := raw["persistentPaths"].([]interface{}); ok {
		for _, item := range pathsRaw {
			if v := strings.TrimSpace(asString(item)); v != "" {
				rc.PersistentPaths = append(rc.PersistentPaths, v)
			}
		}
	} else if pathsRaw, ok := raw["persistentPaths"].([]string); ok {
		for _, item := range pathsRaw {
			if v := strings.TrimSpace(item); v != "" {
				rc.PersistentPaths = append(rc.PersistentPaths, v)
			}
		}
	}
	if networksRaw, ok := raw["extraNetworks"].([]interface{}); ok {
		for _, item := range networksRaw {
			if v := strings.TrimSpace(asString(item)); v != "" {
				rc.ExtraNetworks = append(rc.ExtraNetworks, v)
			}
		}
	} else if networksRaw, ok := raw["extraNetworks"].([]string); ok {
		for _, item := range networksRaw {
			if v := strings.TrimSpace(item); v != "" {
				rc.ExtraNetworks = append(rc.ExtraNetworks, v)
			}
		}
	}
	rc.ExtraNetworks = normalizeRunnerExtraNetworks(rc.ExtraNetworks)
	return rc
}
func resolveRunnerSourceMountDir(rc runnerConfig, workingDir string) string {
	wd := strings.TrimSpace(workingDir)
	if wd == "" {
		wd = "/var/www/app"
	}
	if rc.HasCustomWorkingDir {
		return wd
	}
	return runnerWorkspaceMountPath
}
func resolveRunnerPublishedHostPort(rc runnerConfig) string {
	if v := strings.TrimSpace(rc.HostPort); v != "" {
		return v
	}
	return "0"
}
func asString(v interface{}) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case fmt.Stringer:
		return t.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}
func asNumberString(v interface{}) string {
	switch t := v.(type) {
	case nil:
		return ""
	case int:
		return strconv.Itoa(t)
	case int64:
		return strconv.FormatInt(t, 10)
	case float64:
		return strconv.FormatInt(int64(t), 10)
	case float32:
		return strconv.FormatInt(int64(t), 10)
	case uint:
		return strconv.FormatUint(uint64(t), 10)
	case uint64:
		return strconv.FormatUint(t, 10)
	case string:
		return t
	default:
		return ""
	}
}
func mergeRunnerEnvs(base []string, rc runnerConfig, containerPort string, pipelineVersion string) []string {
	envMap := make(map[string]string)
	for _, e := range base {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}
	for k, v := range rc.Env {
		ks := strings.TrimSpace(k)
		if ks == "" {
			continue
		}
		envMap[ks] = v
	}
	if strings.TrimSpace(containerPort) != "" {
		envMap["PORT"] = strings.TrimSpace(containerPort)
	}
	if _, ok := envMap["HOST"]; !ok {
		envMap["HOST"] = "0.0.0.0"
	}
	if version := strings.TrimSpace(pipelineVersion); version != "" {
		envMap["GOPANEL_PIPELINE_VERSION"] = version
	}
	out := make([]string, 0, len(envMap))
	for k, v := range envMap {
		out = append(out, fmt.Sprintf("%s=%s", k, v))
	}
	return out
}
func buildRunnerPersistentBinds(pipelineKey string, rc runnerConfig, workingDir string) ([]string, error) {
	if len(rc.PersistentPaths) == 0 {
		return nil, nil
	}
	wd := strings.TrimSpace(workingDir)
	if wd == "" {
		wd = "/var/www/app"
	}
	key := strings.TrimSpace(pipelineKey)
	if key == "" {
		key = "pipeline-runtime"
	}
	baseDir := filepath.Join(global.CONF.System.BaseDir, "apps", key)
	binds := make([]string, 0, len(rc.PersistentPaths))
	for _, raw := range rc.PersistentPaths {
		subPath, target, ok := normalizeRunnerPersistentPath(wd, raw)
		if !ok {
			continue
		}
		hostDir := filepath.Join(baseDir, filepath.FromSlash(subPath))
		if err := os.MkdirAll(hostDir, 0755); err != nil {
			return nil, err
		}
		binds = append(binds, fmt.Sprintf("%s:%s", hostDir, target))
	}
	return binds, nil
}
func normalizeRunnerPersistentPath(workingDir string, raw string) (string, string, bool) {
	wd := path.Clean(strings.TrimSpace(workingDir))
	if wd == "." || wd == "/" || wd == "" {
		wd = "/var/www/app"
	}
	candidate := strings.TrimSpace(strings.ReplaceAll(raw, "\\", "/"))
	if candidate == "" {
		return "", "", false
	}
	if strings.HasPrefix(candidate, wd+"/") {
		candidate = strings.TrimPrefix(candidate, wd+"/")
	} else {
		candidate = strings.TrimPrefix(candidate, "/")
	}
	candidate = strings.TrimPrefix(path.Clean("/"+candidate), "/")
	if candidate == "" || candidate == "." {
		return "", "", false
	}
	target := path.Join(wd, candidate)
	return candidate, target, true
}
func collectRunnerPersistentSubpaths(rc runnerConfig, workingDir string) map[string]struct{} {
	if len(rc.PersistentPaths) == 0 {
		return nil
	}
	subpaths := make(map[string]struct{}, len(rc.PersistentPaths))
	for _, raw := range rc.PersistentPaths {
		subPath, _, ok := normalizeRunnerPersistentPath(workingDir, raw)
		if !ok {
			continue
		}
		subpaths[subPath] = struct{}{}
	}
	if len(subpaths) == 0 {
		return nil
	}
	return subpaths
}
func appendRunnerCleanupScript(b *strings.Builder, workingDir string, persistentSubpaths map[string]struct{}, candidates ...string) {
	toRemove := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		cleaned := strings.TrimPrefix(path.Clean("/"+strings.TrimSpace(candidate)), "/")
		if cleaned == "" || cleaned == "." {
			continue
		}
		if _, ok := persistentSubpaths[cleaned]; ok {
			continue
		}
		toRemove = append(toRemove, path.Join(workingDir, cleaned))
	}
	if len(toRemove) == 0 {
		b.WriteString("      echo \"[RUNNER] cleanup skipped: all transient dirs are persistent\"\n")
		return
	}
	b.WriteString("      rm -rf")
	for _, target := range toRemove {
		b.WriteString(" \"")
		b.WriteString(strings.ReplaceAll(target, "\"", "\\\""))
		b.WriteString("\"")
	}
	b.WriteString(" 2>/dev/null || true\n")
}
func buildRunnerEphemeralMounts(rc runnerConfig, workingDir, sourceDir string) []mount.Mount {
	if strings.ToLower(strings.TrimSpace(rc.Mode)) != "build_run" {
		return nil
	}
	if !runnerNeedsNodeModulesIsolation(rc, sourceDir) {
		return nil
	}
	wd := path.Clean(strings.TrimSpace(workingDir))
	if wd == "." || wd == "/" || wd == "" {
		wd = "/var/www/app"
	}
	for _, raw := range rc.PersistentPaths {
		subPath, _, ok := normalizeRunnerPersistentPath(wd, raw)
		if ok && subPath == "node_modules" {
			return nil
		}
	}
	return []mount.Mount{{Type: mount.TypeVolume, Target: path.Join(wd, "node_modules")}}
}
func runnerNeedsNodeModulesIsolation(rc runnerConfig, sourceDir string) bool {
	baseImage := strings.ToLower(strings.TrimSpace(rc.BaseImage))
	startCmd := strings.ToLower(strings.TrimSpace(rc.StartCommand))
	buildCmd := strings.ToLower(strings.TrimSpace(rc.BuildCommand))
	if strings.Contains(baseImage, "node") {
		return true
	}
	for _, token := range []string{"npm", "pnpm", "yarn", ".output/server/index.mjs", "node "} {
		if strings.Contains(startCmd, token) || strings.Contains(buildCmd, token) {
			return true
		}
	}
	if strings.TrimSpace(sourceDir) != "" {
		if _, err := os.Stat(filepath.Join(sourceDir, "package.json")); err == nil {
			return true
		}
	}
	return false
}
func detectRunnerProjectKind(sourceDir string, rc runnerConfig) string {
	if strings.TrimSpace(sourceDir) != "" {
		switch {
		case runnerDirHasAny(sourceDir, "go.mod", "main.go"):
			return "Go"
		case runnerDirHasAny(sourceDir, "requirements.txt", "pyproject.toml", "app.py", "manage.py"):
			return "Python"
		case runnerDirHasAny(sourceDir, "composer.json", "artisan", "public/index.php"):
			return "PHP"
		case runnerDirHasAny(sourceDir, ".output/server/index.mjs", ".next", "package.json", "server.js"):
			return "Node"
		case runnerDirHasAny(sourceDir, "dist/index.html", "index.html"):
			return "Static"
		}
	}
	baseImage := strings.ToLower(strings.TrimSpace(rc.BaseImage))
	startCmd := strings.ToLower(strings.TrimSpace(rc.StartCommand))
	buildCmd := strings.ToLower(strings.TrimSpace(rc.BuildCommand))
	switch {
	case strings.Contains(baseImage, "golang"), strings.Contains(buildCmd, "go build"), strings.Contains(startCmd, "./app"):
		return "Go"
	case strings.Contains(baseImage, "python"), strings.Contains(buildCmd, "pip "), strings.Contains(startCmd, "python "), strings.Contains(startCmd, "gunicorn"), strings.Contains(startCmd, "uvicorn"):
		return "Python"
	case strings.Contains(baseImage, "php"), strings.Contains(buildCmd, "composer "), strings.Contains(startCmd, "php "):
		return "PHP"
	case strings.Contains(baseImage, "node"), strings.Contains(buildCmd, "npm"), strings.Contains(buildCmd, "pnpm"), strings.Contains(buildCmd, "yarn"), strings.Contains(startCmd, "node "):
		return "Node"
	}
	return "Custom"
}
func runnerDirHasAny(dir string, names ...string) bool {
	for _, name := range names {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			return true
		}
	}
	return false
}
func normalizeRunnerExtraNetworks(items []string) []string {
	if len(items) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(items))
	out := make([]string, 0, len(items))
	for _, item := range items {
		name := strings.TrimSpace(item)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func buildRunnerScript(rc runnerConfig, sourceDir string) string {
	mode := strings.ToLower(strings.TrimSpace(rc.Mode))
	wd := strings.TrimSpace(rc.WorkingDir)
	if wd == "" {
		wd = "/var/www/app"
	}
	persistentSubpaths := collectRunnerPersistentSubpaths(rc, wd)
	srcDir := strings.TrimSpace(sourceDir)
	if srcDir == "" {
		srcDir = runnerWorkspaceMountPath
	}
	startCmd := strings.TrimSpace(rc.StartCommand)
	installCmd := strings.TrimSpace(rc.BuildCommand)
	var b strings.Builder
	b.WriteString("set -e\n")
	b.WriteString("mkdir -p \"")
	b.WriteString(strings.ReplaceAll(wd, "\"", "\\\""))
	b.WriteString("\"\n")
	if path.Clean(srcDir) == path.Clean(wd) {
		b.WriteString("echo \"[RUNNER] source mount already targets working dir, skip sync\"\n")
	} else {
		b.WriteString("if [ -d \"")
		b.WriteString(strings.ReplaceAll(srcDir, "\"", "\\\""))
		b.WriteString("\" ]; then\n")
		b.WriteString("  echo \"[RUNNER] syncing source into working dir\"\n")
		b.WriteString("  if command -v tar >/dev/null 2>&1; then\n")
		b.WriteString("    if tar --help 2>/dev/null | grep -q -- '--exclude'; then\n")
		b.WriteString("      echo \"[RUNNER] sync strategy: tar --exclude (skip node_modules/.git/.gopanel_artifact)\"\n")
		b.WriteString("      (cd \"")
		b.WriteString(strings.ReplaceAll(srcDir, "\"", "\\\""))
		b.WriteString("\" && tar --exclude='./node_modules' --exclude='./.git' --exclude='./.data' --exclude='./.gopanel_artifact' --exclude='./__MACOSX' -cf - .) | (cd \"")
		b.WriteString(strings.ReplaceAll(wd, "\"", "\\\""))
		b.WriteString("\" && tar xpf -)\n")
		b.WriteString("    else\n")
		b.WriteString("      echo \"[RUNNER] sync strategy: tar fallback + cleanup transient dirs\"\n")
		b.WriteString("      (cd \"")
		b.WriteString(strings.ReplaceAll(srcDir, "\"", "\\\""))
		b.WriteString("\" && tar cf - .) | (cd \"")
		b.WriteString(strings.ReplaceAll(wd, "\"", "\\\""))
		b.WriteString("\" && tar xpf -)\n")
		appendRunnerCleanupScript(&b, wd, persistentSubpaths, "node_modules", ".git", ".data", ".gopanel_artifact", "__MACOSX")
		b.WriteString("    fi\n")
		b.WriteString("  else\n")
		b.WriteString("    echo \"[RUNNER] sync strategy: cp -a fallback + cleanup transient dirs\"\n")
		b.WriteString("    cp -a \"")
		b.WriteString(strings.ReplaceAll(srcDir, "\"", "\\\""))
		b.WriteString("\"/. \"")
		b.WriteString(strings.ReplaceAll(wd, "\"", "\\\""))
		b.WriteString("\"/\n")
		appendRunnerCleanupScript(&b, wd, persistentSubpaths, "node_modules", ".git", ".data", ".gopanel_artifact", "__MACOSX")
		b.WriteString("  fi\n")
		b.WriteString("fi\n")
	}
	b.WriteString("cd \"")
	b.WriteString(strings.ReplaceAll(wd, "\"", "\\\""))
	b.WriteString("\"\n")
	if strings.TrimSpace(rc.PreStart) != "" {
		b.WriteString(rc.PreStart)
		if !strings.HasSuffix(rc.PreStart, "\n") {
			b.WriteString("\n")
		}
	}
	if mode == "build_run" {
		if installCmd != "" {
			b.WriteString("echo \"[BUILD+RUN] executing custom build command\"\n")
			b.WriteString(installCmd)
			b.WriteString("\n")
		} else {
			b.WriteString("if [ -f package.json ]; then\n")
			b.WriteString("  echo \"[BUILD+RUN] package.json detected, rebuilding app\"\n")
			b.WriteString("  if [ -f pnpm-lock.yaml ]; then pnpm install --frozen-lockfile; ")
			b.WriteString("elif [ -f yarn.lock ]; then yarn install --frozen-lockfile; ")
			b.WriteString("elif [ -f package-lock.json ]; then npm ci; else npm install; fi\n")
			b.WriteString("  npm run build\n")
			b.WriteString("elif [ -f .output/server/index.mjs ]; then\n")
			b.WriteString("  echo \"[RUN] detected standalone .output, start directly\"\n")
			b.WriteString("else\n")
			b.WriteString("  echo \"[BUILD+RUN] buildCommand 为空，且未识别到默认 Node 构建特征（package.json / .output）\" >&2\n")
			b.WriteString("  exit 1\n")
			b.WriteString("fi\n")
		}
	}
	if startCmd != "" {
		b.WriteString("exec ")
		b.WriteString(startCmd)
		b.WriteString("\n")
	} else {
		b.WriteString("echo \"[RUN] start command empty, skip auto start\"\n")
	}
	return b.String()
}
