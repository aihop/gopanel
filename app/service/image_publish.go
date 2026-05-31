package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/docker"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/registry"
	"io"
	"os"
	"path"
	"strings"
	"time"
)

func (u *ImageService) ImagePush(req dto.ImagePush) (string, error) {
	if docker.IsPodmanRuntime(context.Background()) {
		repoItem, err := repo.NewIImageRepoRepo().Get(repo.NewCommonRepo().WithByID(req.RepoID))
		if err != nil {
			return "", err
		}
		dockerLogDir := path.Join(global.CONF.System.TmpDir, "docker_logs")
		if _, err := os.Stat(dockerLogDir); err != nil && os.IsNotExist(err) {
			if err = os.MkdirAll(dockerLogDir, os.ModePerm); err != nil {
				return "", err
			}
		}
		imageItemName := strings.ReplaceAll(path.Base(req.TagName), ":", "_")
		logItem := fmt.Sprintf("%s/image_push_%s_%s.log", dockerLogDir, imageItemName, time.Now().Format(constant.DateTimeSlimLayout))
		file, err := os.OpenFile(logItem, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
		if err != nil {
			return "", err
		}
		go func() {
			defer file.Close()
			_ = docker.PodmanEnsureReady(context.Background())
			targetName := fmt.Sprintf("%s/%s", repoItem.DownloadUrl, req.Name)
			if targetName != req.TagName {
				if _, err := docker.PodmanTag(context.Background(), req.TagName, targetName); err != nil {
					_, _ = file.WriteString(err.Error())
					_, _ = file.WriteString("\nimage push failed!")
					return
				}
			}
			creds := ""
			if repoItem.Auth {
				creds = strings.TrimSpace(repoItem.Username) + ":" + repoItem.Password
			}
			out, err := docker.PodmanPush(context.Background(), targetName, creds)
			if err != nil {
				if strings.TrimSpace(out) != "" {
					_, _ = file.WriteString(out + "\n")
				}
				_, _ = file.WriteString(err.Error())
				_, _ = file.WriteString("\nimage push failed!")
				return
			}
			_, _ = file.WriteString(out)
			_, _ = file.WriteString("\nimage push successful!")

			// 推送完成后删除临时标签，避免本地残留 registry 地址镜像
			if targetName != req.TagName {
				_, _ = file.WriteString("\n清理临时标签: " + targetName)
				_, _ = docker.PodmanRemoveImage(context.Background(), targetName)
			}
		}()
		return path.Base(logItem), nil
	}
	client, err := docker.NewDockerClient()
	if err != nil {
		return "", err
	}
	defer client.Close()
	repo, err := repo.NewIImageRepoRepo().Get(repo.NewCommonRepo().WithByID(req.RepoID))
	if err != nil {
		return "", err
	}
	options := image.PushOptions{All: true}
	authConfig := registry.AuthConfig{Username: repo.Username, Password: repo.Password}
	encodedJSON, err := json.Marshal(authConfig)
	if err != nil {
		return "", err
	}
	authStr := base64.URLEncoding.EncodeToString(encodedJSON)
	options.RegistryAuth = authStr
	newName := fmt.Sprintf("%s/%s", repo.DownloadUrl, req.Name)
	if newName != req.TagName {
		if err := client.ImageTag(context.TODO(), req.TagName, newName); err != nil {
			return "", err
		}
	}
	dockerLogDir := global.CONF.System.TmpDir + "/docker_logs"
	if _, err := os.Stat(dockerLogDir); err != nil && os.IsNotExist(err) {
		if err = os.MkdirAll(dockerLogDir, os.ModePerm); err != nil {
			return "", err
		}
	}
	imageItemName := strings.ReplaceAll(path.Base(req.Name), ":", "_")
	logItem := fmt.Sprintf("%s/image_push_%s_%s.log", dockerLogDir, imageItemName, time.Now().Format(constant.DateTimeSlimLayout))
	file, err := os.OpenFile(logItem, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return "", err
	}
	go func() {
		defer file.Close()
		out, err := client.ImagePush(context.TODO(), newName, options)
		if err != nil {
			global.LOG.Errorf("image %s push failed, err: %v", req.TagName, err)
			_, _ = file.WriteString("image push failed!")
			return
		}
		defer out.Close()
		global.LOG.Infof("push image %s successful!", req.Name)
		_, _ = io.Copy(file, out)
		_, _ = file.WriteString("image push successful!")
	}()
	return path.Base(logItem), nil
}
func (u *ImageService) ImageRemove(req dto.BatchDelete) error {
	if docker.IsPodmanRuntime(context.Background()) {
		_ = docker.PodmanEnsureReady(context.Background())
		for _, name := range req.Names {
			if strings.TrimSpace(name) == "" {
				continue
			}
			if _, err := docker.PodmanRemoveImage(context.Background(), name); err != nil {
				return err
			}
		}
		return nil
	}
	client, err := docker.NewDockerClient()
	if err != nil {
		return err
	}
	defer client.Close()
	for _, id := range req.Names {
		if _, err := client.ImageRemove(context.TODO(), id, image.RemoveOptions{Force: req.Force, PruneChildren: true}); err != nil {
			if strings.Contains(err.Error(), "image is being used") || strings.Contains(err.Error(), "is using") {
				if strings.Contains(id, "sha256:") {
					return errors.New(constant.ErrObjectInUsed)
				}
				return fmt.Errorf("%s: %s", constant.ErrInUsed, id)
			}
			if strings.Contains(err.Error(), "image has dependent") {
				return errors.New(constant.ErrObjectBeDependent)
			}
			return err
		}
	}
	return nil
}
