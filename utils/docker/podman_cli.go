package docker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type PodmanContainer struct {
	ID      string
	Name    string
	Image   string
	ImageID string
	State   string
	Status  string
	Created time.Time
	Labels  map[string]string
	Ports   []string
}

type PodmanImage struct {
	ID      string
	Tags    []string
	Created time.Time
	Size    string
}

func ParsePodmanImageSizeBytes(raw string) (int64, bool) {
	s := strings.TrimSpace(strings.ToUpper(raw))
	if s == "" {
		return 0, false
	}
	if n, err := strconv.ParseInt(s, 10, 64); err == nil {
		return n, true
	}

	s = strings.ReplaceAll(s, ",", "")
	s = strings.ReplaceAll(s, "IB", "")
	s = strings.ReplaceAll(s, "I", "")
	s = strings.ReplaceAll(s, "B", "")
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}

	unit := ""
	switch {
	case strings.HasSuffix(s, "K"):
		unit = "K"
		s = strings.TrimSuffix(s, "K")
	case strings.HasSuffix(s, "M"):
		unit = "M"
		s = strings.TrimSuffix(s, "M")
	case strings.HasSuffix(s, "G"):
		unit = "G"
		s = strings.TrimSuffix(s, "G")
	case strings.HasSuffix(s, "T"):
		unit = "T"
		s = strings.TrimSuffix(s, "T")
	case strings.HasSuffix(s, "P"):
		unit = "P"
		s = strings.TrimSuffix(s, "P")
	}

	v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return 0, false
	}
	switch unit {
	case "K":
		v *= 1024
	case "M":
		v *= 1024 * 1024
	case "G":
		v *= 1024 * 1024 * 1024
	case "T":
		v *= 1024 * 1024 * 1024 * 1024
	case "P":
		v *= 1024 * 1024 * 1024 * 1024 * 1024
	}
	return int64(v), true
}

func PodmanEnsureReady(ctx context.Context) error {
	if runtime.GOOS != "darwin" {
		return nil
	}
	if _, err := runtimeBinaryPath("podman"); err != nil {
		return err
	}
	base := ctx
	if base == nil {
		base = context.Background()
	}
	ensureCtx, cancel := context.WithTimeout(base, 60*time.Second)
	defer cancel()

	out, err := runPodman(ensureCtx, "machine", "list", "--format", "json")
	if err != nil {
		return err
	}
	var items []map[string]interface{}
	_ = json.Unmarshal([]byte(out), &items)
	if len(items) == 0 {
		if _, err := runPodman(ensureCtx, "machine", "init"); err != nil {
			lower := strings.ToLower(err.Error())
			if !strings.Contains(lower, "already exists") && !strings.Contains(lower, "already been created") {
				return err
			}
		}
	}
	_, err = runPodman(ensureCtx, "machine", "start")
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "already running") {
			return nil
		}
		return err
	}
	return nil
}

func PodmanListContainers(ctx context.Context, all bool) ([]PodmanContainer, error) {
	if _, err := runtimeBinaryPath("podman"); err != nil {
		return nil, err
	}
	if runtime.GOOS == "darwin" {
		_ = PodmanEnsureReady(ctx)
	}
	args := []string{"ps", "--format", "json"}
	if all {
		args = []string{"ps", "-a", "--format", "json"}
	}
	out, err := runPodman(ctx, args...)
	if err != nil {
		return nil, err
	}
	var raw []map[string]interface{}
	if err := json.Unmarshal([]byte(out), &raw); err != nil {
		return nil, err
	}
	var res []PodmanContainer
	for _, m := range raw {
		c := PodmanContainer{
			ID:      getAnyString(m, "Id", "ID", "id"),
			Image:   getAnyString(m, "Image", "image"),
			ImageID: getAnyString(m, "ImageID", "ImageId", "imageID", "imageId"),
			State:   getAnyString(m, "State", "state"),
			Status:  getAnyString(m, "Status", "status"),
			Labels:  getAnyStringMap(m, "Labels", "labels"),
		}
		c.Name = getFirstString(m, "Names", "names", "Name", "name")
		c.Created = getAnyTime(m, "CreatedAt", "Created", "createdAt", "created")
		c.Ports = getPodmanPorts(m)
		res = append(res, c)
	}
	return res, nil
}

func PodmanListImages(ctx context.Context) ([]PodmanImage, error) {
	if _, err := runtimeBinaryPath("podman"); err != nil {
		return nil, err
	}
	if runtime.GOOS == "darwin" {
		_ = PodmanEnsureReady(ctx)
	}
	out, err := runPodman(ctx, "images", "--format", "json")
	if err != nil {
		return nil, err
	}
	var raw []map[string]interface{}
	if err := json.Unmarshal([]byte(out), &raw); err != nil {
		return nil, err
	}
	var res []PodmanImage
	for _, m := range raw {
		id := getAnyString(m, "Id", "ID", "id")
		if id == "" {
			id = getAnyString(m, "ImageID", "ImageId", "imageID", "imageId")
		}
		repo := getAnyString(m, "Repository", "repository")
		tag := getAnyString(m, "Tag", "tag")
		var tags []string
		if names := getStringSlice(m, "Names", "names", "RepoTags", "repoTags"); len(names) > 0 {
			tags = append(tags, names...)
		} else if repo != "" {
			if tag == "" || tag == "<none>" {
				tags = []string{repo}
			} else {
				tags = []string{repo + ":" + tag}
			}
		}
		size := getAnyString(m, "Size", "size")
		if size == "" {
			size = getAnyString(m, "SizeBytes", "sizeBytes")
		}
		res = append(res, PodmanImage{
			ID:      id,
			Tags:    tags,
			Created: getAnyTime(m, "CreatedAt", "Created", "createdAt", "created"),
			Size:    size,
		})
	}
	return res, nil
}

func PodmanPull(ctx context.Context, imageName string, creds string) (string, error) {
	args := []string{"pull"}
	if strings.TrimSpace(creds) != "" {
		args = append(args, "--creds", creds)
	}
	args = append(args, imageName)
	return runPodman(ctx, args...)
}

func PodmanTag(ctx context.Context, sourceID string, targetName string) (string, error) {
	return runPodman(ctx, "tag", sourceID, targetName)
}

func PodmanRemoveImage(ctx context.Context, id string) (string, error) {
	return runPodman(ctx, "rmi", "-f", id)
}

func PodmanSave(ctx context.Context, tagName string, outPath string) (string, error) {
	return runPodman(ctx, "save", "-o", outPath, tagName)
}

func PodmanLoad(ctx context.Context, inPath string) (string, error) {
	return runPodman(ctx, "load", "-i", inPath)
}

func PodmanPush(ctx context.Context, tagName string, creds string) (string, error) {
	args := []string{"push"}
	if strings.TrimSpace(creds) != "" {
		args = append(args, "--creds", creds)
	}
	args = append(args, tagName)
	return runPodman(ctx, args...)
}

func PodmanBuild(ctx context.Context, contextDir string, dockerfileName string, tags []string, labels map[string]string) (string, error) {
	args := []string{"build"}
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		args = append(args, "-t", tag)
	}
	if strings.TrimSpace(dockerfileName) != "" {
		args = append(args, "-f", dockerfileName)
	}
	for key, value := range labels {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		args = append(args, "--label", key+"="+value)
	}
	args = append(args, contextDir)
	return runPodman(ctx, args...)
}

func PodmanRunBuild(ctx context.Context, contextDir string, dockerfileName string, tag string) (string, error) {
	return PodmanBuild(ctx, contextDir, dockerfileName, []string{tag}, nil)
}

func PodmanVersion(ctx context.Context) (string, error) {
	out, err := runPodman(ctx, "version", "--format", "json")
	if err != nil {
		return "", err
	}
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		return "", err
	}
	if client, ok := m["Client"].(map[string]interface{}); ok {
		if v, ok := client["Version"]; ok {
			return strings.TrimSpace(fmt.Sprint(v)), nil
		}
	}
	if v, ok := m["Version"]; ok {
		return strings.TrimSpace(fmt.Sprint(v)), nil
	}
	return "", nil
}

func runPodman(ctx context.Context, args ...string) (string, error) {
	base := ctx
	if base == nil {
		base = context.Background()
	}
	if _, ok := base.Deadline(); !ok {
		var cancel context.CancelFunc
		base, cancel = context.WithTimeout(base, 45*time.Second)
		defer cancel()
	}
	podmanPath, err := runtimeBinaryPath("podman")
	if err != nil {
		return "", err
	}
	resolved := ResolveRuntime(base)
	if resolved.Kind == RuntimePodman {
		host := strings.TrimSpace(resolved.Host)
		if host != "" && host != "podman-cli" && host != "podman://local" {
			args = append([]string{"--url", host}, args...)
		}
	}
	c := exec.CommandContext(base, podmanPath, args...)
	if extraEnv := podmanCommandEnv(resolved); len(extraEnv) > 0 {
		c.Env = append(os.Environ(), extraEnv...)
	}
	out, err := c.CombinedOutput()
	s := strings.TrimSpace(string(out))
	if err != nil {
		if s != "" {
			return "", errors.New(s)
		}
		return "", err
	}
	return s, nil
}

func podmanCommandEnv(resolved ResolvedRuntime) []string {
	if resolved.Kind != RuntimePodman {
		return nil
	}
	host := strings.TrimSpace(resolved.Host)
	if host == "" || host == "podman-cli" || host == "podman://local" {
		return nil
	}
	extraEnv := []string{
		"CONTAINER_HOST=" + host,
		"DOCKER_HOST=" + host,
		"PODMAN_HOST=" + host,
	}
	if runtime.GOOS != "linux" || !IsRootlessPodmanHost(host) {
		return extraEnv
	}
	runtimeDir := podmanRuntimeDirFromHost(host)
	if runtimeDir == "" {
		return extraEnv
	}
	extraEnv = append(extraEnv, "XDG_RUNTIME_DIR="+runtimeDir)
	if busPath := filepath.Join(runtimeDir, "bus"); busPath != "" {
		extraEnv = append(extraEnv, "DBUS_SESSION_BUS_ADDRESS=unix:path="+busPath)
	}
	return extraEnv
}

func podmanRuntimeDirFromHost(host string) string {
	host = strings.TrimSpace(host)
	if !strings.HasPrefix(host, "unix://") {
		return ""
	}
	sockPath := strings.TrimPrefix(host, "unix://")
	if !strings.HasSuffix(sockPath, "/podman/podman.sock") {
		return ""
	}
	return filepath.Dir(filepath.Dir(sockPath))
}

func getAnyString(m map[string]interface{}, keys ...string) string {
	for _, k := range keys {
		v, ok := m[k]
		if !ok || v == nil {
			continue
		}
		switch x := v.(type) {
		case string:
			return x
		default:
			return strings.TrimSpace(fmt.Sprint(x))
		}
	}
	return ""
}

func getFirstString(m map[string]interface{}, keys ...string) string {
	for _, k := range keys {
		v, ok := m[k]
		if !ok || v == nil {
			continue
		}
		switch x := v.(type) {
		case []string:
			if len(x) > 0 {
				return x[0]
			}
		case []interface{}:
			if len(x) > 0 {
				if s, ok := x[0].(string); ok {
					return s
				}
				return strings.TrimSpace(fmt.Sprint(x[0]))
			}
		case string:
			return x
		}
	}
	return ""
}

func getStringSlice(m map[string]interface{}, keys ...string) []string {
	for _, k := range keys {
		v, ok := m[k]
		if !ok || v == nil {
			continue
		}
		switch x := v.(type) {
		case []string:
			return x
		case []interface{}:
			var out []string
			for _, it := range x {
				out = append(out, strings.TrimSpace(fmt.Sprint(it)))
			}
			out = compactStrings(out)
			if len(out) > 0 {
				return out
			}
		}
	}
	return nil
}

func getAnyStringMap(m map[string]interface{}, keys ...string) map[string]string {
	for _, k := range keys {
		v, ok := m[k]
		if !ok || v == nil {
			continue
		}
		switch x := v.(type) {
		case map[string]string:
			return x
		case map[string]interface{}:
			out := make(map[string]string, len(x))
			for kk, vv := range x {
				out[kk] = strings.TrimSpace(fmt.Sprint(vv))
			}
			return out
		}
	}
	return nil
}

func getAnyTime(m map[string]interface{}, keys ...string) time.Time {
	for _, k := range keys {
		v, ok := m[k]
		if !ok || v == nil {
			continue
		}
		switch x := v.(type) {
		case float64:
			if x > 0 {
				return time.Unix(int64(x), 0)
			}
		case int64:
			if x > 0 {
				return time.Unix(x, 0)
			}
		case json.Number:
			n, _ := x.Int64()
			if n > 0 {
				return time.Unix(n, 0)
			}
		case string:
			s := strings.TrimSpace(x)
			if s == "" {
				continue
			}
			if n, err := strconv.ParseInt(s, 10, 64); err == nil && n > 0 {
				return time.Unix(n, 0)
			}
			if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
				return t
			}
			if t, err := time.Parse(time.RFC3339, s); err == nil {
				return t
			}
		}
	}
	return time.Time{}
}

func getPodmanPorts(m map[string]interface{}) []string {
	v, ok := m["Ports"]
	if !ok || v == nil {
		v, ok = m["ports"]
		if !ok || v == nil {
			return nil
		}
	}
	switch x := v.(type) {
	case []string:
		return compactStrings(x)
	case []interface{}:
		var out []string
		for _, it := range x {
			switch p := it.(type) {
			case string:
				out = append(out, p)
			case map[string]interface{}:
				out = append(out, formatPodmanPortMap(p))
			}
		}
		return compactStrings(out)
	}
	return nil
}

func formatPodmanPortMap(m map[string]interface{}) string {
	hostIP := getAnyString(m, "host_ip", "HostIP", "hostIP")
	hostPort := getAnyString(m, "host_port", "HostPort", "hostPort")
	containerPort := getAnyString(m, "container_port", "ContainerPort", "containerPort")
	proto := strings.ToLower(getAnyString(m, "protocol", "Protocol", "proto"))
	if hostPort == "" || containerPort == "" {
		return ""
	}
	if hostIP == "" {
		hostIP = "0.0.0.0"
	}
	if proto == "" {
		proto = "tcp"
	}
	return hostIP + ":" + hostPort + "->" + containerPort + "/" + proto
}

func compactStrings(in []string) []string {
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
