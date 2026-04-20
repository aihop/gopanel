package service

import (
	"context"
	"strings"

	"github.com/aihop/gopanel/utils/docker"
	"gopkg.in/yaml.v3"
)

func shouldSkipComposePull(ctx context.Context, composeYml string, envContent string) (bool, []string) {
	images := extractComposeImages(composeYml, envContent)
	if len(images) == 0 {
		return false, nil
	}
	client, err := docker.NewDockerClient()
	if err != nil {
		return false, nil
	}
	defer client.Close()

	var missing []string
	for _, img := range images {
		if img == "" {
			continue
		}
		_, _, err := client.ImageInspectWithRaw(ctx, img)
		if err != nil {
			missing = append(missing, img)
		}
	}
	return len(missing) == 0, missing
}

func extractComposeImages(composeYml string, envContent string) []string {
	envMap := parseEnvMap(envContent)
	m := make(map[string]interface{})
	if err := yaml.Unmarshal([]byte(composeYml), &m); err != nil {
		return extractComposeImagesFallback(composeYml, envMap)
	}
	servicesVal, ok := m["services"]
	if !ok || servicesVal == nil {
		return nil
	}
	servicesMap, ok := servicesVal.(map[string]interface{})
	if !ok {
		return nil
	}
	var out []string
	for _, v := range servicesMap {
		svc, ok := v.(map[string]interface{})
		if !ok {
			continue
		}
		raw, ok := svc["image"]
		if !ok || raw == nil {
			continue
		}
		img, ok := raw.(string)
		if !ok {
			continue
		}
		img = strings.TrimSpace(img)
		if img == "" {
			continue
		}
		img = expandEnvVars(img, envMap)
		if img == "" || strings.Contains(img, "${") {
			continue
		}
		out = append(out, img)
	}
	return dedupeStrings(out)
}

func extractComposeImagesFallback(composeYml string, envMap map[string]string) []string {
	lines := strings.Split(composeYml, "\n")
	var out []string
	for _, line := range lines {
		ltrim := strings.TrimSpace(line)
		if !strings.HasPrefix(ltrim, "image:") {
			continue
		}
		rest := strings.TrimSpace(strings.TrimPrefix(ltrim, "image:"))
		if rest == "" {
			continue
		}
		rest = strings.Trim(rest, `"'`)
		rest = expandEnvVars(rest, envMap)
		if rest == "" || strings.Contains(rest, "${") {
			continue
		}
		out = append(out, rest)
	}
	return dedupeStrings(out)
}

func parseEnvMap(envContent string) map[string]string {
	m := make(map[string]string)
	lines := strings.Split(envContent, "\n")
	for _, line := range lines {
		trim := strings.TrimSpace(line)
		if trim == "" || strings.HasPrefix(trim, "#") || !strings.Contains(trim, "=") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		if key == "" {
			continue
		}
		val = strings.Trim(val, `"'`)
		m[key] = val
	}
	return m
}

func expandEnvVars(s string, envMap map[string]string) string {
	if !strings.Contains(s, "${") {
		return s
	}
	out := s
	for k, v := range envMap {
		out = strings.ReplaceAll(out, "${"+k+"}", v)
	}
	return out
}

func dedupeStrings(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	var out []string
	for _, s := range in {
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

