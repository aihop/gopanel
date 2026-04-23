package service

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"

	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/buserr"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/cmd"
	"github.com/aihop/gopanel/utils/compose"
	"github.com/aihop/gopanel/utils/docker"
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
		composeItemLocal := dto.ComposeInfo{
			ContainerNumber: 0,
			CreatedAt:       localItem.CreatedAt.Format(constant.DateTimeLayout),
			ConfigFile:      localItem.Path,
			Workdir:         strings.TrimSuffix(localItem.Path, "/docker-compose.yml"),
		}
		composeItemLocal.CreatedBy = "GoPanel"
		composeItemLocal.Path = localItem.Path
		composeLocalMap[localItem.Name] = composeItemLocal
	}

	composeMap := make(map[string]dto.ComposeInfo)
	for _, container := range list {
		name, ok := firstLabel(container.Labels, composeProjectLabel, podmanComposeProjectLabel)
		if ok {
			containerItem := dto.ComposeContainer{
				ContainerID: container.ID,
				Name:        container.Names[0][1:],
				State:       container.State,
				CreateTime:  time.Unix(container.Created, 0).Format(constant.DateTimeLayout),
			}
			if compose, has := composeMap[name]; has {
				compose.ContainerNumber++
				compose.Containers = append(compose.Containers, containerItem)
				composeMap[name] = compose
			} else {
				config, _ := firstLabel(container.Labels, composeConfigLabel, podmanComposeConfigLabel)
				workdir, _ := firstLabel(container.Labels, composeWorkdirLabel, podmanComposeWorkdirLabel)
				composeItem := dto.ComposeInfo{
					ContainerNumber: 1,
					CreatedAt:       time.Unix(container.Created, 0).Format(constant.DateTimeLayout),
					ConfigFile:      config,
					Workdir:         workdir,
					Containers:      []dto.ComposeContainer{containerItem},
				}
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

		// key的大小写问题
		var existingItem dto.ComposeInfo
		var exists bool
		var mainKey string
		for keyLocal, itemLocal := range composeLocalMap {
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

		// if existingItem, exists := mergedMap[key]; exists {
		// 	if item.ContainerNumber > 0 {
		// 		if existingItem.ContainerNumber <= 0 {
		// 			mergedMap[key] = item
		// 		}
		// 	}
		// } else {
		// 	mergedMap[key] = item
		// }
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

func (u *ContainerService) TestCompose(req *dto.ComposeCreate) (bool, error) {
	if cmd.CheckIllegal(req.Path) {
		return false, buserr.New(constant.ErrCmdIllegal)
	}
	composeRepo := repo.NewIComposeTemplateRepo()
	commonRepo := repo.NewCommonRepo()
	composeItem, _ := composeRepo.GetRecord(commonRepo.WithByName(req.Name))
	if composeItem.ID != 0 {
		return false, constant.ErrRecordExist
	}
	if err := u.loadPath(req); err != nil {
		return false, err
	}
	if err := newComposeEnv(req.Path, req.Env); err != nil {
		return false, err
	}
	c, err := compose.Command(context.Background(), "-f", req.Path, "config")
	if err != nil {
		return false, err
	}
	stdout, err := c.CombinedOutput()
	if err != nil {
		return false, errors.New(string(stdout))
	}
	return true, nil
}

func (u *ContainerService) CreateCompose(req *dto.ComposeCreate) (string, error) {
	if cmd.CheckIllegal(req.Name, req.Path) {
		return "", buserr.New(constant.ErrCmdIllegal)
	}
	if err := u.loadPath(req); err != nil {
		return "", err
	}
	global.LOG.Infof("docker-compose.yml %s create success, start to docker-compose up", req.Name)

	if req.From == "path" {
		req.Name = path.Base(path.Dir(req.Path))
	}

	dockerLogDir := path.Join(global.CONF.System.TmpDir, "docker_logs")
	if _, err := os.Stat(dockerLogDir); err != nil && os.IsNotExist(err) {
		if err = os.MkdirAll(dockerLogDir, os.ModePerm); err != nil {
			return "", err
		}
	}
	logItem := fmt.Sprintf("%s/compose_create_%s_%s.log", dockerLogDir, req.Name, time.Now().Format(constant.DateTimeSlimLayout))
	file, err := os.OpenFile(logItem, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return "", err
	}
	if err := newComposeEnv(req.Path, req.Env); err != nil {
		return "", err
	}
	go func() {
		defer file.Close()
		c, err := compose.Command(context.Background(), "-f", req.Path, "up", "-d")
		if err != nil {
			global.LOG.Errorf("docker-compose.yml path :%v", req.Path)
			global.LOG.Errorf("docker-compose up %s failed, err: %v", req.Name, err)
			_, _ = compose.Down(req.Path)
			_, _ = file.WriteString("docker-compose up failed!")
			return
		}
		multiWriter := io.MultiWriter(os.Stdout, file)
		c.Stdout = multiWriter
		c.Stderr = multiWriter
		if err := c.Run(); err != nil {
			global.LOG.Errorf("docker-compose.yml path :%v", req.Path)
			global.LOG.Errorf("docker-compose up %s failed, err: %v", req.Name, err)
			_, _ = compose.Down(req.Path)
			_, _ = file.WriteString("docker-compose up failed!")
			return
		}
		composeRepo := repo.NewIComposeTemplateRepo()
		global.LOG.Infof("docker-compose up %s successful!", req.Name)
		_ = composeRepo.CreateRecord(&model.Compose{Name: req.Name, Path: req.Path})
		_, _ = file.WriteString("docker-compose up successful!")
	}()

	return path.Base(logItem), nil
}

func (u *ContainerService) ComposeOperation(req *dto.ComposeOperation) error {
	composeRepo := repo.NewIComposeTemplateRepo()
	commonRepo := repo.NewCommonRepo()
	if len(req.Path) == 0 && req.Operation == "delete" {
		_ = composeRepo.DeleteRecord(commonRepo.WithByName(req.Name))
		return nil
	}
	if cmd.CheckIllegal(req.Path, req.Operation) {
		return buserr.New(constant.ErrCmdIllegal)
	}
	if _, err := os.Stat(req.Path); err != nil {
		return fmt.Errorf("load file with path %s failed, %v", req.Path, err)
	}
	if req.Operation == "delete" {
		if stdout, err := compose.Operate(req.Path, "down"); err != nil {
			return errors.New(string(stdout))
		}
		if req.WithFile {
			_ = os.RemoveAll(path.Dir(req.Path))
		}
		_ = composeRepo.DeleteRecord(commonRepo.WithByName(req.Name))
		return nil
	}
	if req.Operation == "up" {
		if stdout, err := compose.Up(req.Path); err != nil {
			return errors.New(string(stdout))
		}
	} else {
		if stdout, err := compose.Operate(req.Path, req.Operation); err != nil {
			return errors.New(string(stdout))
		}
	}
	global.LOG.Infof("docker-compose %s %s successful", req.Operation, req.Name)
	return nil
}

func (u *ContainerService) ComposeUpdate(req *dto.ComposeUpdate) error {
	if cmd.CheckIllegal(req.Name, req.Path) {
		return buserr.New(constant.ErrCmdIllegal)
	}
	oldFile, err := os.ReadFile(req.Path)
	if err != nil {
		return fmt.Errorf("load file with path %s failed, %v", req.Path, err)
	}
	file, err := os.OpenFile(req.Path, os.O_WRONLY|os.O_TRUNC, 0640)
	if err != nil {
		return err
	}
	defer file.Close()
	write := bufio.NewWriter(file)
	_, _ = write.WriteString(req.Content)
	write.Flush()

	global.LOG.Infof("docker-compose.yml %s has been replaced, now start to docker-compose restart", req.Path)
	if err := newComposeEnv(req.Path, req.Env); err != nil {
		return err
	}

	if stdout, err := compose.Up(req.Path); err != nil {
		if err := recreateCompose(string(oldFile), req.Path); err != nil {
			return fmt.Errorf("update failed when handle compose up, err: %s, recreate failed: %v", string(stdout), err)
		}
		return fmt.Errorf("update failed when handle compose up, err: %s", string(stdout))
	}

	return nil
}

func (u *ContainerService) loadPath(req *dto.ComposeCreate) error {
	if req.From == "template" || req.From == "edit" {
		dir := fmt.Sprintf("%s/docker/compose/%s", global.CONF.System.BaseDir, req.Name)
		if _, err := os.Stat(dir); err != nil && os.IsNotExist(err) {
			if err = os.MkdirAll(dir, os.ModePerm); err != nil {
				return err
			}
		}

		path := fmt.Sprintf("%s/docker-compose.yml", dir)
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
		if err != nil {
			return err
		}
		defer file.Close()
		write := bufio.NewWriter(file)
		_, _ = write.WriteString(string(req.File))
		write.Flush()
		req.Path = path
	}
	return nil
}

func recreateCompose(content, path string) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_TRUNC, 0640)
	if err != nil {
		return err
	}
	defer file.Close()
	write := bufio.NewWriter(file)
	_, _ = write.WriteString(content)
	write.Flush()

	if stdout, err := compose.Up(path); err != nil {
		return errors.New(string(stdout))
	}
	return nil
}

func newComposeEnv(pathItem string, env []string) error {
	if len(env) == 0 {
		return nil
	}
	envFilePath := path.Join(path.Dir(pathItem), ".env")

	file, err := os.OpenFile(envFilePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		global.LOG.Errorf("failed to create env file: %v", err)
		return err
	}
	defer file.Close()
	for _, env := range env {
		envItem := strings.TrimSpace(env)
		if _, err := file.WriteString(fmt.Sprintf("%s\n", envItem)); err != nil {
			global.LOG.Errorf("failed to write env to file: %v", err)
			return err
		}
	}
	global.LOG.Infof(".env file successfully created or updated with env variables in %s", envFilePath)
	return nil
}
