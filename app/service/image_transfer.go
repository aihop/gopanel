package service

import (
	"bufio"
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
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/pkg/archive"
	"io"
	"os"
	"path"
	"strings"
	"time"
)

func (u *ImageService) ImageBuild(req dto.ImageBuild) (string, error) {
	if docker.IsPodmanRuntime(context.Background()) {
		fileName := "Dockerfile"
		if req.From == "edit" {
			dir := fmt.Sprintf("%s/docker/build/%s", global.CONF.System.BaseDir, strings.ReplaceAll(req.Name, ":", "_"))
			if _, err := os.Stat(dir); err != nil && os.IsNotExist(err) {
				if err = os.MkdirAll(dir, os.ModePerm); err != nil {
					return "", err
				}
			}
			pathItem := fmt.Sprintf("%s/Dockerfile", dir)
			file, err := os.OpenFile(pathItem, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
			if err != nil {
				return "", err
			}
			defer file.Close()
			write := bufio.NewWriter(file)
			_, _ = write.WriteString(string(req.Dockerfile))
			_ = write.Flush()
			req.Dockerfile = dir
		} else {
			fileName = path.Base(req.Dockerfile)
			req.Dockerfile = path.Dir(req.Dockerfile)
		}
		dockerLogDir := path.Join(global.CONF.System.TmpDir, "docker_logs")
		if _, err := os.Stat(dockerLogDir); err != nil && os.IsNotExist(err) {
			if err = os.MkdirAll(dockerLogDir, os.ModePerm); err != nil {
				return "", err
			}
		}
		logItem := fmt.Sprintf("%s/image_build_%s_%s.log", dockerLogDir, strings.ReplaceAll(req.Name, ":", "_"), time.Now().Format(constant.DateTimeSlimLayout))
		file, err := os.OpenFile(logItem, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
		if err != nil {
			return "", err
		}
		go func() {
			defer file.Close()
			_ = docker.PodmanEnsureReady(context.Background())
			out, err := docker.PodmanBuild(context.Background(), req.Dockerfile, fileName, []string{req.Name}, stringsToMap(req.Tags))
			if err != nil {
				_, _ = file.WriteString(out + "\n" + err.Error())
				_, _ = file.WriteString("\nimage build failed!")
				return
			}
			_, _ = file.WriteString(out)
			_, _ = file.WriteString("\nimage build successful!")
		}()
		return path.Base(logItem), nil
	}
	client, err := docker.NewDockerClient()
	if err != nil {
		return "", err
	}
	defer client.Close()
	fileName := "Dockerfile"
	if req.From == "edit" {
		dir := fmt.Sprintf("%s/docker/build/%s", global.CONF.System.BaseDir, strings.ReplaceAll(req.Name, ":", "_"))
		if _, err := os.Stat(dir); err != nil && os.IsNotExist(err) {
			if err = os.MkdirAll(dir, os.ModePerm); err != nil {
				return "", err
			}
		}
		pathItem := fmt.Sprintf("%s/Dockerfile", dir)
		file, err := os.OpenFile(pathItem, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
		if err != nil {
			return "", err
		}
		defer file.Close()
		write := bufio.NewWriter(file)
		_, _ = write.WriteString(string(req.Dockerfile))
		write.Flush()
		req.Dockerfile = dir
	} else {
		fileName = path.Base(req.Dockerfile)
		req.Dockerfile = path.Dir(req.Dockerfile)
	}
	tar, err := archive.TarWithOptions(req.Dockerfile+"/", &archive.TarOptions{})
	if err != nil {
		return "", err
	}
	opts := types.ImageBuildOptions{Dockerfile: fileName, Tags: []string{req.Name}, Remove: true, Labels: stringsToMap(req.Tags)}
	dockerLogDir := path.Join(global.CONF.System.TmpDir, "docker_logs")
	if _, err := os.Stat(dockerLogDir); err != nil && os.IsNotExist(err) {
		if err = os.MkdirAll(dockerLogDir, os.ModePerm); err != nil {
			return "", err
		}
	}
	logItem := fmt.Sprintf("%s/image_build_%s_%s.log", dockerLogDir, strings.ReplaceAll(req.Name, ":", "_"), time.Now().Format(constant.DateTimeSlimLayout))
	file, err := os.OpenFile(logItem, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return "", err
	}
	go func() {
		defer file.Close()
		defer tar.Close()
		res, err := client.ImageBuild(context.Background(), tar, opts)
		if err != nil {
			global.LOG.Errorf("build image %s failed, err: %v", req.Name, err)
			_, _ = file.WriteString("image build failed!")
			return
		}
		defer res.Body.Close()
		body, err := io.ReadAll(res.Body)
		if err != nil {
			global.LOG.Errorf("build image %s failed, err: %v", req.Name, err)
			_, _ = file.WriteString(fmt.Sprintf("build image %s failed, err: %v", req.Name, err))
			_, _ = file.WriteString("image build failed!")
			return
		}
		if strings.Contains(string(body), "errorDetail") || strings.Contains(string(body), "error:") {
			global.LOG.Errorf("build image %s failed", req.Name)
			_, _ = file.Write(body)
			_, _ = file.WriteString("image build failed!")
			return
		}
		global.LOG.Infof("build image %s successful!", req.Name)
		_, _ = file.Write(body)
		_, _ = file.WriteString("image build successful!")
	}()
	return path.Base(logItem), nil
}
func (u *ImageService) ImagePull(req dto.ImagePull) (string, error) {
	if docker.IsPodmanRuntime(context.Background()) {
		imageRef, err := normalizeImageRefForPull(req.ImageName)
		if err != nil {
			return "", err
		}
		dockerLogDir := path.Join(global.CONF.System.TmpDir, "docker_logs")
		if _, err := os.Stat(dockerLogDir); err != nil && os.IsNotExist(err) {
			if err = os.MkdirAll(dockerLogDir, os.ModePerm); err != nil {
				return "", err
			}
		}
		imageItemName := strings.ReplaceAll(path.Base(imageRef), ":", "_")
		logItem := fmt.Sprintf("%s/image_pull_%s_%s.log", dockerLogDir, imageItemName, time.Now().Format(constant.DateTimeSlimLayout))
		file, err := os.OpenFile(logItem, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
		if err != nil {
			return "", err
		}
		go func() {
			defer file.Close()
			_ = docker.PodmanEnsureReady(context.Background())
			target := imageRef
			creds := ""
			if req.RepoID != 0 {
				repoItem, e := repo.NewIImageRepoRepo().Get(repo.NewCommonRepo().WithByID(req.RepoID))
				if e != nil {
					_, _ = file.WriteString(e.Error() + "\n")
					_, _ = file.WriteString("image pull failed!\n")
					return
				}
				target = repoItem.DownloadUrl + "/" + imageRef
				if repoItem.Auth {
					creds = strings.TrimSpace(repoItem.Username) + ":" + repoItem.Password
				}
			} else if hasCreds, authCreds := loadAuthCredentials(imageRef); hasCreds {
				creds = authCreds
			}
			out, err := docker.PodmanPull(context.Background(), target, creds)
			if err != nil {
				if strings.TrimSpace(out) != "" {
					_, _ = file.WriteString(out + "\n")
				}
				_, _ = file.WriteString(err.Error() + "\n")
				_, _ = file.WriteString("image pull failed!\n")
				return
			}
			if strings.TrimSpace(out) != "" {
				_, _ = file.WriteString(out + "\n")
			}
			_, _ = file.WriteString("image pull successful!\n")
		}()
		return path.Base(logItem), nil
	}
	imageRef, err := normalizeImageRefForPull(req.ImageName)
	if err != nil {
		return "", err
	}
	client, err := docker.NewDockerClient()
	if err != nil {
		return "", err
	}
	defer client.Close()
	dockerLogDir := path.Join(global.CONF.System.TmpDir, "docker_logs")
	if _, err := os.Stat(dockerLogDir); err != nil && os.IsNotExist(err) {
		if err = os.MkdirAll(dockerLogDir, os.ModePerm); err != nil {
			return "", err
		}
	}
	imageItemName := strings.ReplaceAll(path.Base(imageRef), ":", "_")
	logItem := fmt.Sprintf("%s/image_pull_%s_%s.log", dockerLogDir, imageItemName, time.Now().Format(constant.DateTimeSlimLayout))
	file, err := os.OpenFile(logItem, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return "", err
	}
	options := image.PullOptions{}
	if req.RepoID == 0 {
		hasAuth, authStr := loadAuthInfo(imageRef)
		if hasAuth {
			options.RegistryAuth = authStr
		}
		go func() {
			defer file.Close()
			out, err := client.ImagePull(context.TODO(), imageRef, options)
			if err != nil {
				global.LOG.Errorf("image %s pull failed, err: %v", imageRef, err)
				return
			}
			defer out.Close()
			global.LOG.Infof("pull image %s successful!", imageRef)
			_, _ = io.Copy(file, out)
		}()
		return path.Base(logItem), nil
	}
	repo, err := repo.NewIImageRepoRepo().Get(repo.NewCommonRepo().WithByID(req.RepoID))
	if err != nil {
		return "", err
	}
	if repo.Auth {
		authConfig := registry.AuthConfig{Username: repo.Username, Password: repo.Password}
		encodedJSON, err := json.Marshal(authConfig)
		if err != nil {
			return "", err
		}
		authStr := base64.StdEncoding.EncodeToString(encodedJSON)
		options.RegistryAuth = authStr
	}
	image := repo.DownloadUrl + "/" + imageRef
	go func() {
		defer file.Close()
		out, err := client.ImagePull(context.TODO(), image, options)
		if err != nil {
			_, _ = file.WriteString("image pull failed!\n")
			_, _ = file.WriteString(fmt.Sprintf("image %s pull failed, err: %v\n", image, err))
			return
		}
		defer out.Close()
		global.LOG.Infof("pull image %s successful!", imageRef)
		_, _ = io.Copy(file, out)
		_, _ = file.WriteString("\nimage pull successful!\n")
	}()
	return path.Base(logItem), nil
}
func (u *ImageService) ImageLoad(req dto.ImageLoad) error {
	if docker.IsPodmanRuntime(context.Background()) {
		_ = docker.PodmanEnsureReady(context.Background())
		_, err := docker.PodmanLoad(context.Background(), req.Path)
		return err
	}
	file, err := os.Open(req.Path)
	if err != nil {
		return err
	}
	defer file.Close()
	client, err := docker.NewDockerClient()
	if err != nil {
		return err
	}
	defer client.Close()
	res, err := client.ImageLoad(context.TODO(), file, nil)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	content, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	if strings.Contains(string(content), "Error") {
		return errors.New(string(content))
	}
	return nil
}
func (u *ImageService) ImageSave(req dto.ImageSave) error {
	if docker.IsPodmanRuntime(context.Background()) {
		_ = docker.PodmanEnsureReady(context.Background())
		outPath := path.Join(req.Path, req.Name)
		_, err := docker.PodmanSave(context.Background(), req.TagName, outPath)
		return err
	}
	client, err := docker.NewDockerClient()
	if err != nil {
		return err
	}
	defer client.Close()
	out, err := client.ImageSave(context.TODO(), []string{req.TagName})
	if err != nil {
		return err
	}
	defer out.Close()
	file, err := os.OpenFile(fmt.Sprintf("%s/%s.tar", req.Path, req.Name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0666)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err = io.Copy(file, out); err != nil {
		return err
	}
	return nil
}
func (u *ImageService) ImageTag(req dto.ImageTag) error {
	if docker.IsPodmanRuntime(context.Background()) {
		_ = docker.PodmanEnsureReady(context.Background())
		_, err := docker.PodmanTag(context.Background(), req.SourceID, req.TargetName)
		return err
	}
	client, err := docker.NewDockerClient()
	if err != nil {
		return err
	}
	defer client.Close()
	if err := client.ImageTag(context.TODO(), req.SourceID, req.TargetName); err != nil {
		return err
	}
	return nil
}
