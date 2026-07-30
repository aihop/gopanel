package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aihop/gopanel/utils/files"
	"github.com/aihop/gopanel/utils/gpc"
)

type ContainerRuntimeInstallTask struct {
	ID          string    `json:"id"`
	Runtime     string    `json:"runtime"`
	Status      string    `json:"status"`
	Stage       string    `json:"stage"`
	Message     string    `json:"message,omitempty"`
	NeedsAction string    `json:"needsAction,omitempty"`
	Output      string    `json:"output,omitempty"`
	StartedAt   time.Time `json:"startedAt"`
	FinishedAt  time.Time `json:"finishedAt,omitempty"`
}

type containerRuntimeInstallManager struct {
	mu      sync.RWMutex
	tasks   map[string]*ContainerRuntimeInstallTask
	current string
}

var runtimeInstallManager = &containerRuntimeInstallManager{tasks: make(map[string]*ContainerRuntimeInstallTask)}

func StartContainerRuntimeInstall(runtimeKind string) (*ContainerRuntimeInstallTask, error) {
	runtimeKind = strings.ToLower(strings.TrimSpace(runtimeKind))
	if runtimeKind != "docker" && runtimeKind != "podman" {
		return nil, errors.New("runtime must be docker or podman")
	}
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		return nil, errors.New("container runtime installation is not supported on this platform")
	}
	if commandAvailable("docker") || commandAvailable("podman") {
		return nil, errors.New("a container runtime is already installed")
	}

	runtimeInstallManager.mu.Lock()
	defer runtimeInstallManager.mu.Unlock()
	for id, existing := range runtimeInstallManager.tasks {
		if existing.Status != "running" && time.Since(existing.FinishedAt) > time.Hour {
			delete(runtimeInstallManager.tasks, id)
		}
	}
	if current := runtimeInstallManager.tasks[runtimeInstallManager.current]; current != nil && current.Status == "running" {
		if current.Runtime != runtimeKind {
			return nil, fmt.Errorf("%s installation is already running", current.Runtime)
		}
		return cloneRuntimeInstallTask(current), nil
	}
	task := &ContainerRuntimeInstallTask{
		ID: fmt.Sprintf("runtime-%d", time.Now().UnixNano()), Runtime: runtimeKind, Status: "running", Stage: "installing", StartedAt: time.Now(),
	}
	runtimeInstallManager.tasks[task.ID] = task
	runtimeInstallManager.current = task.ID
	go runContainerRuntimeInstall(task.ID)
	return cloneRuntimeInstallTask(task), nil
}

func GetContainerRuntimeInstallTask(id string) (*ContainerRuntimeInstallTask, error) {
	runtimeInstallManager.mu.RLock()
	defer runtimeInstallManager.mu.RUnlock()
	task := runtimeInstallManager.tasks[strings.TrimSpace(id)]
	if task == nil {
		return nil, errors.New("runtime install task not found")
	}
	return cloneRuntimeInstallTask(task), nil
}

func runContainerRuntimeInstall(id string) {
	params := currentRuntimeUserParams()
	runtimeInstallManager.mu.RLock()
	runtimeKind := runtimeInstallManager.tasks[id].Runtime
	runtimeInstallManager.mu.RUnlock()
	params["runtime"] = runtimeKind

	ctx, cancel := context.WithTimeout(context.Background(), 21*time.Minute)
	defer cancel()
	response, err := gpc.Do(ctx, "CONTAINER_RUNTIME_INSTALL", params)
	result := struct {
		Message     string `json:"message"`
		NeedsAction string `json:"needsAction"`
		Output      string `json:"output"`
	}{}
	if response != nil {
		_ = json.Unmarshal([]byte(response.Output), &result)
	}

	runtimeInstallManager.mu.Lock()
	defer runtimeInstallManager.mu.Unlock()
	task := runtimeInstallManager.tasks[id]
	if task == nil {
		return
	}
	task.FinishedAt = time.Now()
	if err != nil {
		task.Status = "failed"
		task.Stage = "failed"
		task.Message = err.Error()
		if response != nil {
			task.Output = strings.TrimSpace(response.Output)
		}
		return
	}
	task.Status = "success"
	task.Stage = "completed"
	task.Message = strings.TrimSpace(result.Message)
	task.NeedsAction = strings.TrimSpace(result.NeedsAction)
	task.Output = strings.TrimSpace(result.Output)
}

func currentRuntimeUserParams() map[string]interface{} {
	params := map[string]interface{}{
		"uid": os.Getuid(), "gid": os.Getgid(), "group": files.GetGroup(os.Getgid()), "rootless": runtime.GOOS == "linux" && os.Geteuid() != 0,
	}
	if current, err := user.Current(); err == nil && current != nil {
		params["username"] = strings.TrimSpace(current.Username)
		if uid, err := strconv.Atoi(current.Uid); err == nil {
			params["uid"] = uid
		}
		if gid, err := strconv.Atoi(current.Gid); err == nil {
			params["gid"] = gid
			params["group"] = files.GetGroup(gid)
		}
	}
	if runtime.GOOS == "darwin" && params["uid"] == 0 {
		consoleUID := strings.TrimSpace(os.Getenv("SUDO_UID"))
		if consoleUID == "" {
			if output, err := exec.Command("stat", "-f", "%u", "/dev/console").Output(); err == nil {
				consoleUID = strings.TrimSpace(string(output))
			}
		}
		if consoleUser, err := user.LookupId(consoleUID); err == nil && consoleUser != nil {
			if uid, err := strconv.Atoi(consoleUser.Uid); err == nil && uid > 0 {
				params["uid"] = uid
				params["username"] = consoleUser.Username
			}
		}
	}
	return params
}

func cloneRuntimeInstallTask(task *ContainerRuntimeInstallTask) *ContainerRuntimeInstallTask {
	copy := *task
	return &copy
}

var commandAvailable = func(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
