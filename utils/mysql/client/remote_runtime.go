package client

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	udocker "github.com/aihop/gopanel/utils/docker"
	"github.com/aihop/gopanel/utils/gpc"
)

func ensureHostDumpCmd(preferred string) (string, error) {
	if cmd := lookupMysqlCommand(preferred); cmd != "" {
		return cmd, nil
	}
	if runtime.GOOS != "linux" {
		return "", nil
	}
	if err := ensureMysqlClientInstalled(); err != nil {
		return "", err
	}
	if cmd := lookupMysqlCommand(preferred); cmd != "" {
		return cmd, nil
	}
	return "", errors.New("mysql client is not installed on host (need mysql/mysqldump or mariadb/mariadb-dump)")
}

func ensureHostMysqlCmd(preferred string) (string, error) {
	if cmd := lookupMysqlCommand(preferred); cmd != "" {
		return cmd, nil
	}
	if runtime.GOOS != "linux" {
		return "", nil
	}
	if err := ensureMysqlClientInstalled(); err != nil {
		return "", err
	}
	if cmd := lookupMysqlCommand(preferred); cmd != "" {
		return cmd, nil
	}
	return "", errors.New("mysql client is not installed on host (need mysql/mysqldump or mariadb/mariadb-dump)")
}

func lookupMysqlCommand(preferred string) string {
	candidates := []string{preferred}
	switch preferred {
	case "mariadb-dump":
		candidates = append(candidates, "mysqldump")
	case "mysqldump":
		candidates = append(candidates, "mariadb-dump")
	case "mariadb":
		candidates = append(candidates, "mysql")
	case "mysql":
		candidates = append(candidates, "mariadb")
	}
	for _, item := range candidates {
		if strings.TrimSpace(item) == "" {
			continue
		}
		if _, err := exec.LookPath(item); err == nil {
			return item
		}
	}
	return ""
}

func ensureMysqlClientInstalled() error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	resp, err := gpc.Do(ctx, "MYSQL_CLIENT_INSTALL", nil)
	if err != nil {
		global.LOG.Warnf("ensure mysql client via gpc failed: %v", err)
		return errors.New("mysql client auto-install failed via gpc: " + err.Error())
	}
	if resp != nil && strings.TrimSpace(resp.Output) != "" {
		global.LOG.Infof("ensure mysql client via gpc: %s", strings.TrimSpace(resp.Output))
	}
	return nil
}

func ensureDockerImage(image string, policy string, timeout uint) error {
	policy = strings.TrimSpace(strings.ToLower(policy))
	if policy == "" {
		policy = "missing"
	}
	if policy != "missing" && policy != "always" && policy != "never" {
		policy = "missing"
	}

	exists := dockerImageExists(image)
	if exists && policy != "always" {
		return nil
	}
	if !exists && policy == "never" {
		return fmt.Errorf("docker image not found locally: %s", image)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()
	cmd, err := runtimeCommandForDBTool(ctx, "pull", image)
	if err != nil {
		return err
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return errors.New(constant.ErrExecTimeOut)
		}
		return fmt.Errorf("pull %s failed: %s", image, strings.TrimSpace(string(out)))
	}
	return nil
}

func dockerImageExists(image string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd, err := runtimeCommandForDBTool(ctx, "image", "inspect", image)
	if err != nil {
		return false
	}
	if err := cmd.Run(); err != nil {
		return false
	}
	return true
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func runtimeCommandForDBTool(ctx context.Context, args ...string) (*exec.Cmd, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if runtime.GOOS == "darwin" && !strings.EqualFold(strings.TrimSpace(global.CONF.System.ContainerRuntime), "docker") {
		podmanPath, err := udocker.PodmanBinaryPath()
		if err != nil {
			return nil, errors.New("podman binary not found for mysql database tools on darwin")
		}
		if err := udocker.PodmanEnsureReady(ctx); err != nil {
			return nil, err
		}
		return exec.CommandContext(ctx, podmanPath, args...), nil
	}
	return udocker.RuntimeCommand(ctx, args...)
}

func loadImage(dbType, version string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	images, _, err := udocker.ListImagesMerged(ctx)
	if err != nil {
		return "", err
	}

	var candidates []string
	for _, image := range images {
		for _, tag := range image.RepoTags {
			tag = strings.TrimSpace(tag)
			if tag == "" || tag == "<none>:<none>" {
				continue
			}
			if !strings.HasPrefix(tag, dbType+":") {
				continue
			}
			candidates = append(candidates, tag)
		}
	}

	if version == "" {
		if best, ok := pickBestTag(candidates); ok {
			return best, nil
		}
		return loadVersion(dbType, version), nil
	}

	for _, tag := range candidates {
		if dbType == "mariadb" {
			return tag, nil
		}
		if strings.HasPrefix(version, "5.6") && strings.HasPrefix(tag, "mysql:5.6") {
			return tag, nil
		}
		if strings.HasPrefix(version, "5.7") && strings.HasPrefix(tag, "mysql:5.7") {
			return tag, nil
		}
		if strings.HasPrefix(version, "8.") && strings.HasPrefix(tag, "mysql:8.") {
			return tag, nil
		}
	}

	if best, ok := pickBestTag(candidates); ok {
		return best, nil
	}
	return loadVersion(dbType, version), nil
}

func loadVersion(dbType string, version string) string {
	if dbType == "mariadb" {
		return "mariadb:11.3.2"
	}
	if strings.HasPrefix(version, "5.6") {
		return "mysql:5.6.51"
	}
	if strings.HasPrefix(version, "5.7") {
		return "mysql:5.7.44"
	}
	return "mysql:8.2.0"
}

func pickBestTag(tags []string) (string, bool) {
	if len(tags) == 0 {
		return "", false
	}
	best := tags[0]
	for _, t := range tags[1:] {
		if compareDockerTagVersion(t, best) > 0 {
			best = t
		}
	}
	return best, true
}

func compareDockerTagVersion(a, b string) int {
	av := parseTagVersion(a)
	bv := parseTagVersion(b)
	n := len(av)
	if len(bv) > n {
		n = len(bv)
	}
	for i := 0; i < n; i++ {
		ai := 0
		if i < len(av) {
			ai = av[i]
		}
		bi := 0
		if i < len(bv) {
			bi = bv[i]
		}
		if ai != bi {
			if ai > bi {
				return 1
			}
			return -1
		}
	}
	return 0
}

func parseTagVersion(tag string) []int {
	parts := strings.SplitN(tag, ":", 2)
	if len(parts) != 2 {
		return nil
	}
	v := parts[1]
	if i := strings.IndexAny(v, "-+"); i >= 0 {
		v = v[:i]
	}
	raw := strings.Split(v, ".")
	out := make([]int, 0, len(raw))
	for _, s := range raw {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		n, err := strconv.Atoi(s)
		if err != nil {
			return out
		}
		out = append(out, n)
	}
	return out
}

func sslSkip(version, dbType string) string {
	if dbType == constant.AppMariaDB || strings.HasPrefix(version, "5.6") || strings.HasPrefix(version, "5.7") {
		return "--skip-ssl"
	}
	return "--ssl-mode=DISABLED"
}
