package service

import (
	"bytes"
	"context"
	"fmt"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/utils/docker"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	dockerclient "github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"io"
	"runtime"
	"sort"
	"strings"
)

func syncAppInstallStatus(appInstall *model.AppInstall, force bool) error {
	if appInstall.Status == constant.Installing || appInstall.Status == constant.Rebuilding || appInstall.Status == constant.Upgrading {
		return nil
	}
	resolved := docker.ResolveRuntime(context.Background())
	if resolved.Kind == docker.RuntimePodman && runtime.GOOS == "darwin" {
		containers, err := docker.PodmanListContainers(context.Background(), true)
		if err != nil {
			return err
		}
		containersMap := make(map[string]types.Container)
		for _, c := range containers {
			name := strings.TrimPrefix(strings.TrimSpace(c.Name), "/")
			if name == "" {
				continue
			}
			containersMap["/"+name] = types.Container{Names: []string{"/" + name}, State: c.State}
		}
		synAppInstall(containersMap, appInstall, force)
		return nil
	}
	if resolved.Kind == docker.RuntimePodman {
		options := container.ListOptions{All: true}
		containers, _, err := docker.ListContainersMergedWithSource(context.Background(), options)
		if err != nil {
			return err
		}
		containersMap := matchInstallContainers(appInstall, containers)
		synAppInstall(containersMap, appInstall, force)
		return nil
	}
	cli, err := docker.NewRuntimeClient()
	if err != nil {
		return err
	}
	defer cli.Close()
	var (
		containers    []types.Container
		containersMap map[string]types.Container
	)
	containers, err = cli.ListAllContainers()
	if err != nil {
		return err
	}
	containersMap = matchInstallContainers(appInstall, containers)
	synAppInstall(containersMap, appInstall, force)
	return nil
}
func loadInstallMatchedContainers(appInstall *model.AppInstall) (map[string]types.Container, error) {
	resolved := docker.ResolveRuntime(context.Background())
	if resolved.Kind == docker.RuntimePodman && runtime.GOOS == "darwin" {
		items, err := docker.PodmanListContainers(context.Background(), true)
		if err != nil {
			return nil, err
		}
		containers := make([]types.Container, 0, len(items))
		for _, c := range items {
			name := normalizeContainerName(c.Name)
			if name == "" {
				continue
			}
			containers = append(containers, types.Container{Names: []string{"/" + name}, State: strings.ToLower(strings.TrimSpace(c.State)), Status: strings.TrimSpace(c.Status), Labels: c.Labels})
		}
		return matchInstallContainers(appInstall, containers), nil
	}
	if resolved.Kind == docker.RuntimePodman {
		items, _, err := docker.ListContainersMergedWithSource(context.Background(), container.ListOptions{All: true})
		if err != nil {
			return nil, err
		}
		return matchInstallContainers(appInstall, items), nil
	}
	cli, err := docker.NewRuntimeClient()
	if err != nil {
		return nil, err
	}
	defer cli.Close()
	items, err := cli.ListAllContainers()
	if err != nil {
		return nil, err
	}
	return matchInstallContainers(appInstall, items), nil
}
func describeInstallContainerState(appInstall *model.AppInstall) (string, error) {
	containersMap, err := loadInstallMatchedContainers(appInstall)
	if err != nil {
		return "", err
	}
	containerNames := splitContainerNames(appInstall.ContainerName)
	if len(containerNames) == 0 {
		return "", nil
	}
	grouped := map[string][]string{"running": {}, "created": {}, "restarting": {}, "exited": {}, "paused": {}, "dead": {}, "not_found": {}, "unknown": {}}
	for _, name := range containerNames {
		con, ok := containersMap["/"+name]
		if !ok {
			grouped["not_found"] = append(grouped["not_found"], name)
			continue
		}
		state := strings.ToLower(strings.TrimSpace(con.State))
		switch state {
		case "running":
			grouped["running"] = append(grouped["running"], name)
		case "created", "configured":
			grouped["created"] = append(grouped["created"], name)
		case "restarting":
			grouped["restarting"] = append(grouped["restarting"], name)
		case "exited", "stopped":
			grouped["exited"] = append(grouped["exited"], name)
		case "paused":
			grouped["paused"] = append(grouped["paused"], name)
		case "dead":
			grouped["dead"] = append(grouped["dead"], name)
		default:
			grouped["unknown"] = append(grouped["unknown"], fmt.Sprintf("%s(%s)", name, state))
		}
	}
	var parts []string
	appendPart := func(key, label string) {
		if len(grouped[key]) == 0 {
			return
		}
		sort.Strings(grouped[key])
		parts = append(parts, label+": "+strings.Join(grouped[key], ","))
	}
	appendPart("running", "running")
	appendPart("created", "created")
	appendPart("restarting", "restarting")
	appendPart("exited", "exited")
	appendPart("paused", "paused")
	appendPart("dead", "dead")
	appendPart("not_found", "not found")
	appendPart("unknown", "unknown")
	if len(parts) == 0 {
		return "", nil
	}
	return "container states -> " + strings.Join(parts, "; "), nil
}
func describeInstallContainerDiagnostics(appInstall *model.AppInstall) (string, error) {
	containerNames := splitContainerNames(appInstall.ContainerName)
	if len(containerNames) == 0 {
		return "", nil
	}
	cli, err := docker.NewRuntimeAPIClient()
	if err != nil {
		return "", err
	}
	defer cli.Close()
	parts := make([]string, 0, len(containerNames))
	for _, name := range containerNames {
		inspect, err := cli.ContainerInspect(context.Background(), name)
		if err != nil {
			continue
		}
		if inspect.ContainerJSONBase == nil || inspect.ContainerJSONBase.State == nil {
			continue
		}
		state := inspect.ContainerJSONBase.State
		status := strings.ToLower(strings.TrimSpace(state.Status))
		if status == "running" {
			continue
		}
		detail := []string{fmt.Sprintf("%s(status=%s", name, status)}
		if state.ExitCode != 0 {
			detail = append(detail, fmt.Sprintf("exitCode=%d", state.ExitCode))
		}
		if msg := strings.TrimSpace(state.Error); msg != "" {
			detail = append(detail, fmt.Sprintf("error=%s", msg))
		}
		if logs := readInstallContainerLogs(context.Background(), cli, name); logs != "" {
			detail = append(detail, fmt.Sprintf("logs=%s", logs))
		}
		parts = append(parts, strings.Join(detail, ", ")+")")
	}
	if len(parts) == 0 {
		return "", nil
	}
	return "container diagnostics -> " + strings.Join(parts, "; "), nil
}
func readInstallContainerLogs(ctx context.Context, cli *dockerclient.Client, containerName string) string {
	reader, err := cli.ContainerLogs(ctx, containerName, container.LogsOptions{ShowStdout: true, ShowStderr: true, Tail: "50"})
	if err != nil {
		return ""
	}
	defer reader.Close()
	var stdoutBuf bytes.Buffer
	var stderrBuf bytes.Buffer
	if _, err := stdcopy.StdCopy(&stdoutBuf, &stderrBuf, reader); err != nil {
		content, readErr := io.ReadAll(reader)
		if readErr != nil {
			return ""
		}
		logs := strings.TrimSpace(string(content))
		return summarizeInstallLogs(logs)
	}
	logs := strings.TrimSpace(stdoutBuf.String())
	if errLogs := strings.TrimSpace(stderrBuf.String()); errLogs != "" {
		if logs != "" {
			logs += "\n"
		}
		logs += errLogs
	}
	return summarizeInstallLogs(logs)
}
func summarizeInstallLogs(logs string) string {
	logs = strings.TrimSpace(logs)
	if logs == "" {
		return ""
	}
	logs = strings.ReplaceAll(logs, "\n", " | ")
	if len(logs) > 600 {
		logs = logs[:600] + "..."
	}
	return logs
}
func matchInstallContainers(appInstall *model.AppInstall, containers []types.Container) map[string]types.Container {
	containersMap := make(map[string]types.Container)
	expectedNames := splitContainerNames(appInstall.ContainerName)
	expectedSet := make(map[string]struct{}, len(expectedNames))
	for _, name := range expectedNames {
		expectedSet[name] = struct{}{}
	}
	actualNames := make(map[string]struct{})
	for _, con := range containers {
		matched := false
		for _, rawName := range con.Names {
			name := normalizeContainerName(rawName)
			if name == "" {
				continue
			}
			if _, ok := expectedSet[name]; ok {
				matched = true
			}
		}
		if !matched && !containerBelongsToInstall(con, appInstall) {
			continue
		}
		for _, rawName := range con.Names {
			name := normalizeContainerName(rawName)
			if name == "" {
				continue
			}
			containersMap["/"+name] = con
			actualNames[name] = struct{}{}
		}
	}
	if len(actualNames) > 0 {
		var names []string
		for name := range actualNames {
			names = append(names, name)
		}
		sort.Strings(names)
		appInstall.ContainerName = strings.Join(names, ",")
	}
	return containersMap
}
func containerBelongsToInstall(con types.Container, appInstall *model.AppInstall) bool {
	labels := con.Labels
	if len(labels) == 0 {
		return false
	}
	if v, ok := firstLabel(labels, composeProjectLabel, podmanComposeProjectLabel); ok && strings.TrimSpace(v) == strings.TrimSpace(appInstall.Name) {
		return true
	}
	if v, ok := firstLabel(labels, composeWorkdirLabel, podmanComposeWorkdirLabel); ok && sameInstallPath(v, appInstall.GetPath()) {
		return true
	}
	if v, ok := firstLabel(labels, composeConfigLabel, podmanComposeConfigLabel); ok {
		composePath := strings.TrimSpace(appInstall.GetComposePath())
		for _, item := range strings.Split(v, ",") {
			if sameInstallPath(item, composePath) {
				return true
			}
		}
	}
	if serviceName := strings.TrimSpace(appInstall.ServiceName); serviceName != "" {
		serviceLabel, serviceOK := firstLabel(labels, composeServiceLabel, podmanComposeServiceLabel)
		projectLabel, projectOK := firstLabel(labels, composeProjectLabel, podmanComposeProjectLabel)
		if serviceOK && projectOK && strings.EqualFold(strings.TrimSpace(serviceLabel), serviceName) {
			for _, alias := range installComposeProjectAliases(appInstall) {
				if strings.EqualFold(strings.TrimSpace(projectLabel), alias) {
					return true
				}
			}
		}
	}
	return false
}
func sameInstallPath(left, right string) bool {
	left = strings.TrimSpace(strings.TrimSuffix(left, "/"))
	right = strings.TrimSpace(strings.TrimSuffix(right, "/"))
	return left != "" && right != "" && left == right
}
func splitContainerNames(raw string) []string {
	var names []string
	for _, item := range strings.Split(raw, ",") {
		name := normalizeContainerName(item)
		if name == "" {
			continue
		}
		names = append(names, name)
	}
	return names
}
func normalizeContainerName(raw string) string {
	return strings.TrimPrefix(strings.TrimSpace(raw), "/")
}
