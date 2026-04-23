//go:build darwin
// +build darwin

package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/pelletier/go-toml/v2"

	"github.com/aihop/gopanel/utils/docker"
)

func podmanMachineRegistriesGet(ctx context.Context) ([]string, error) {
	if runtime.GOOS != "darwin" {
		return nil, fmt.Errorf("unsupported platform")
	}
	if err := docker.PodmanEnsureReady(ctx); err != nil {
		return nil, err
	}
	data, _, err := podmanMachineReadFile(ctx, "/etc/containers/registries.conf")
	if err != nil {
		return nil, err
	}
	var config map[string]interface{}
	if err := toml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	var mirrors []string
	if registries := anyToMapSlice(config["registry"]); len(registries) > 0 {
		for _, reg := range registries {
			location, _ := reg["location"].(string)
			prefix, _ := reg["prefix"].(string)
			if location == "docker.io" || prefix == "docker.io" {
				for _, mirror := range anyToMapSlice(reg["mirror"]) {
					if loc, ok := mirror["location"].(string); ok {
						mirrors = append(mirrors, loc)
					}
				}
				break
			}
		}
	}
	return mirrors, nil
}

func podmanMachineRegistriesSet(ctx context.Context, mirrors []string) error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("unsupported platform")
	}
	if err := docker.PodmanEnsureReady(ctx); err != nil {
		return err
	}
	data, _, err := podmanMachineReadFile(ctx, "/etc/containers/registries.conf")
	if err != nil {
		return err
	}
	var config map[string]interface{}
	if err := toml.Unmarshal(data, &config); err != nil {
		return err
	}
	if config == nil {
		config = make(map[string]interface{})
	}

	searchList := anyToStringSlice(config["unqualified-search-registries"])
	foundDockerSearch := false
	for _, s := range searchList {
		if s == "docker.io" {
			foundDockerSearch = true
			break
		}
	}
	if !foundDockerSearch {
		searchList = append(searchList, "docker.io")
	}
	config["unqualified-search-registries"] = searchList

	foundReg := false
	var newRegistries []map[string]interface{}
	for _, reg := range anyToMapSlice(config["registry"]) {
		location, _ := reg["location"].(string)
		prefix, _ := reg["prefix"].(string)
		if location == "docker.io" || prefix == "docker.io" {
			foundReg = true
			var newMirrors []map[string]interface{}
			for _, m := range mirrors {
				m = strings.TrimSpace(m)
				m = strings.TrimPrefix(m, "https://")
				m = strings.TrimPrefix(m, "http://")
				m = strings.TrimSuffix(m, "/")
				if m == "" {
					continue
				}
				newMirrors = append(newMirrors, map[string]interface{}{"location": m})
			}
			reg["mirror"] = newMirrors
			if location == "" {
				reg["location"] = "docker.io"
			}
			if prefix == "" {
				reg["prefix"] = "docker.io"
			}
		}
		newRegistries = append(newRegistries, reg)
	}
	if !foundReg && len(mirrors) > 0 {
		newReg := map[string]interface{}{
			"location": "docker.io",
			"prefix":   "docker.io",
		}
		var newMirrors []map[string]interface{}
		for _, m := range mirrors {
			m = strings.TrimSpace(m)
			m = strings.TrimPrefix(m, "https://")
			m = strings.TrimPrefix(m, "http://")
			m = strings.TrimSuffix(m, "/")
			if m == "" {
				continue
			}
			newMirrors = append(newMirrors, map[string]interface{}{"location": m})
		}
		newReg["mirror"] = newMirrors
		newRegistries = append(newRegistries, newReg)
	}
	config["registry"] = newRegistries

	outData, err := toml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal registries.conf: %w", err)
	}
	_, err = podmanMachineWriteFile(ctx, "/etc/containers/registries.conf", outData)
	return err
}

func podmanMachineReadFile(ctx context.Context, path string) ([]byte, string, error) {
	candidates := podmanMachineCandidates(ctx)
	var lastOut []byte
	var lastMachine string
	for _, m := range candidates {
		out, err := podmanMachineReadFileOn(ctx, m, path)
		if err == nil {
			return out, m, nil
		}
		lastOut = out
		lastMachine = m
	}
	return nil, "", fmt.Errorf("podman machine read file failed (machine=%s): %s", strings.TrimSpace(lastMachine), strings.TrimSpace(string(lastOut)))
}

func podmanMachineWriteFile(ctx context.Context, path string, data []byte) (string, error) {
	candidates := podmanMachineCandidates(ctx)
	var lastOut []byte
	var lastMachine string
	for _, m := range candidates {
		out, err := podmanMachineWriteFileOn(ctx, m, path, data)
		if err == nil {
			return m, nil
		}
		lastOut = out
		lastMachine = m
	}
	return "", fmt.Errorf("podman machine write file failed (machine=%s): %s", strings.TrimSpace(lastMachine), strings.TrimSpace(string(lastOut)))
}

func podmanMachineReadFileOn(ctx context.Context, machine string, path string) ([]byte, error) {
	out, err := podmanMachineSSH(ctx, machine, "cat", path)
	if err == nil {
		return out, nil
	}
	out, err2 := podmanMachineSSH(ctx, machine, "sudo", "cat", path)
	if err2 == nil {
		return out, nil
	}
	return out, fmt.Errorf("%s", strings.TrimSpace(string(out)))
}

func podmanMachineWriteFileOn(ctx context.Context, machine string, path string, data []byte) ([]byte, error) {
	encoded := base64.StdEncoding.EncodeToString(data)
	cmd := "echo '" + encoded + "' | base64 -d | sudo tee " + shellEscapeSimple(path) + " >/dev/null"
	out, err := podmanMachineSSH(ctx, machine, "sh", "-lc", cmd)
	if err != nil {
		return out, fmt.Errorf("%s", strings.TrimSpace(string(out)))
	}
	return out, nil
}

func podmanMachineCandidates(ctx context.Context) []string {
	out, err := exec.CommandContext(ctx, "podman", "machine", "list", "--format", "json").CombinedOutput()
	if err != nil {
		return []string{""}
	}
	var raw []map[string]interface{}
	if err := json.Unmarshal(out, &raw); err != nil {
		return []string{""}
	}
	type item struct {
		name    string
		def     bool
		running bool
	}
	var items []item
	for _, m := range raw {
		name, _ := m["Name"].(string)
		if name == "" {
			name, _ = m["name"].(string)
		}
		if strings.TrimSpace(name) == "" {
			continue
		}
		def := getAnyBool(m, "Default", "default")
		running := getAnyBool(m, "Running", "running")
		items = append(items, item{name: name, def: def, running: running})
	}
	if len(items) == 0 {
		return []string{""}
	}
	add := func(dst []string, name string) []string {
		for _, s := range dst {
			if s == name {
				return dst
			}
		}
		return append(dst, name)
	}
	var res []string
	for _, it := range items {
		if it.def {
			res = add(res, it.name)
		}
	}
	for _, it := range items {
		if it.running {
			res = add(res, it.name)
		}
	}
	for _, it := range items {
		res = add(res, it.name)
	}
	return res
}

func getAnyBool(m map[string]interface{}, keys ...string) bool {
	for _, k := range keys {
		v, ok := m[k]
		if !ok || v == nil {
			continue
		}
		switch x := v.(type) {
		case bool:
			return x
		case string:
			s := strings.ToLower(strings.TrimSpace(x))
			return s == "true" || s == "1" || s == "yes"
		default:
			s := strings.ToLower(strings.TrimSpace(fmt.Sprint(x)))
			return s == "true" || s == "1" || s == "yes"
		}
	}
	return false
}

func anyToStringSlice(v interface{}) []string {
	switch x := v.(type) {
	case []string:
		return compactStringsLocal(x)
	case []interface{}:
		var out []string
		for _, it := range x {
			out = append(out, strings.TrimSpace(fmt.Sprint(it)))
		}
		return compactStringsLocal(out)
	default:
		return nil
	}
}

func compactStringsLocal(in []string) []string {
	var out []string
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		out = append(out, s)
	}
	return out
}

func anyToMapSlice(v interface{}) []map[string]interface{} {
	switch x := v.(type) {
	case []map[string]interface{}:
		return x
	case []interface{}:
		var out []map[string]interface{}
		for _, it := range x {
			if m, ok := it.(map[string]interface{}); ok {
				out = append(out, m)
			}
		}
		return out
	default:
		return nil
	}
}

func podmanMachineSSH(ctx context.Context, machine string, args ...string) ([]byte, error) {
	base := []string{"machine", "ssh"}
	if strings.TrimSpace(machine) != "" {
		base = append(base, machine)
	}
	base = append(base, "--")
	base = append(base, args...)
	return exec.CommandContext(ctx, "podman", base...).CombinedOutput()
}

func shellEscapeSimple(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}
