package service

import (
	"context"
	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/utils/docker"
	"github.com/docker/docker/api/types/container"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const composeProjectLabel = "com.docker.compose.project"
const composeConfigLabel = "com.docker.compose.project.config_files"
const composeWorkdirLabel = "com.docker.compose.project.working_dir"
const podmanComposeProjectLabel = "io.podman.compose.project"
const podmanComposeConfigLabel = "io.podman.compose.project.config_files"
const podmanComposeWorkdirLabel = "io.podman.compose.project.working_dir"
const composeCreatedBy = "createdBy"

type DockerCompose struct {
	Version  string                            `yaml:"version"`
	Services map[string]map[string]interface{} `yaml:"services"`
	Networks map[string]interface{}            `yaml:"networks"`
}

func firstLabel(labels map[string]string, keys ...string) (string, bool) {
	for _, k := range keys {
		if v, ok := labels[k]; ok && len(v) > 0 {
			return v, true
		}
	}
	return "", false
}
func (u *ContainerService) PageCompose(req *dto.SearchWithPage) (int64, interface{}, error) {
	var (
		records   []dto.ComposeInfo
		BackDatas []dto.ComposeInfo
	)
	options := container.ListOptions{All: true}
	list, err := docker.ListContainersMerged(context.Background(), options)
	if err != nil {
		return 0, nil, err
	}
	composeCreatedByLocal, _ := repo.NewIComposeTemplateRepo().ListRecord()
	composeLocalMap := make(map[string]dto.ComposeInfo)
	for _, localItem := range composeCreatedByLocal {
		composeItemLocal := dto.ComposeInfo{ContainerNumber: 0, CreatedAt: localItem.CreatedAt.Format(constant.DateTimeLayout), ConfigFile: localItem.Path, Workdir: strings.TrimSuffix(localItem.Path, "/docker-compose.yml")}
		composeItemLocal.CreatedBy = "GoPanel"
		composeItemLocal.Path = localItem.Path
		composeLocalMap[localItem.Name] = composeItemLocal
	}
	composeMap := make(map[string]dto.ComposeInfo)
	for _, container := range list {
		name, ok := firstLabel(container.Labels, composeProjectLabel, podmanComposeProjectLabel)
		if ok {
			containerItem := dto.ComposeContainer{ContainerID: container.ID, Name: container.Names[0][1:], State: container.State, CreateTime: time.Unix(container.Created, 0).Format(constant.DateTimeLayout)}
			if compose, has := composeMap[name]; has {
				compose.ContainerNumber++
				compose.Containers = append(compose.Containers, containerItem)
				composeMap[name] = compose
			} else {
				config, _ := firstLabel(container.Labels, composeConfigLabel, podmanComposeConfigLabel)
				workdir, _ := firstLabel(container.Labels, composeWorkdirLabel, podmanComposeWorkdirLabel)
				composeItem := dto.ComposeInfo{ContainerNumber: 1, CreatedAt: time.Unix(container.Created, 0).Format(constant.DateTimeLayout), ConfigFile: config, Workdir: workdir, Containers: []dto.ComposeContainer{containerItem}}
				createdBy, ok := container.Labels[composeCreatedBy]
				if ok {
					composeItem.CreatedBy = createdBy
				}
				if len(config) != 0 && len(workdir) != 0 && strings.Contains(config, workdir) {
					composeItem.Path = config
				} else {
					composeItem.Path = workdir
				}
				for i := 0; i < len(composeCreatedByLocal); i++ {
					if composeCreatedByLocal[i].Name == name {
						composeItem.CreatedBy = "GoPanel"
						composeCreatedByLocal = append(composeCreatedByLocal[:i], composeCreatedByLocal[i+1:]...)
						break
					}
				}
				composeMap[name] = composeItem
			}
		}
	}
	mergedMap := make(map[string]dto.ComposeInfo)
	for key, localItem := range composeLocalMap {
		mergedMap[key] = localItem
	}
	for key, item := range composeMap {
		var existingItem dto.ComposeInfo
		var exists bool
		var mainKey string
		for keyLocal, itemLocal := range // key的大小写问题
		composeLocalMap {
			if strings.EqualFold(key, keyLocal) {
				exists = true
				mainKey = keyLocal
				existingItem = itemLocal
			}
		}
		if exists {
			if item.ContainerNumber > 0 {
				if existingItem.ContainerNumber <= 0 {
					mergedMap[mainKey] = item
				}
			}
		} else {
			mergedMap[key] = item
		}
	}
	for key, value := range mergedMap {
		value.Name = key
		records = append(records, value)
	}
	if len(req.Info) != 0 {
		length, count := len(records), 0
		for count < length {
			if !strings.Contains(records[count].Name, req.Info) {
				records = append(records[:count], records[(count+1):]...)
				length--
			} else {
				count++
			}
		}
	}
	sort.Slice(records, func(i, j int) bool {
		return records[i].CreatedAt > records[j].CreatedAt
	})
	total, start, end := len(records), (req.Page-1)*req.Limit, req.Page*req.Limit
	if start > total {
		BackDatas = make([]dto.ComposeInfo, 0)
	} else {
		if end >= total {
			end = total
		}
		BackDatas = records[start:end]
	}
	listItem := loadEnv(BackDatas)
	return int64(total), listItem, nil
}
func loadEnv(list []dto.ComposeInfo) []dto.ComposeInfo {
	for i := 0; i < len(list); i++ {
		envFilePath := filepath.Join(path.Dir(list[i].Path), "gopanel.env")
		file, err := os.ReadFile(envFilePath)
		if err != nil {
			continue
		}
		lines := strings.Split(string(file), "\n")
		for _, line := range lines {
			lineItem := strings.TrimSpace(line)
			if len(lineItem) != 0 && !strings.HasPrefix(lineItem, "#") {
				list[i].Env = append(list[i].Env, lineItem)
			}
		}
	}
	return list
}
