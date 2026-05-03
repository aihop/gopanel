package service

import (
	"context"
	"errors"
	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/utils/docker"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"runtime"
	"strings"
	"time"
)

type ImageService struct{}
type IImageService interface {
	Page(req dto.SearchWithPage) (int64, interface{}, error)
	List() ([]dto.Options, error)
	ListAll() ([]dto.ImageInfo, error)
	ImageBuild(req dto.ImageBuild) (string, error)
	ImagePull(req dto.ImagePull) (string, error)
	ImageLoad(req dto.ImageLoad) error
	ImageSave(req dto.ImageSave) error
	ImagePush(req dto.ImagePush) (string, error)
	ImageRemove(req dto.BatchDelete) error
	ImageTag(req dto.ImageTag) error
}

func NewIImageService() IImageService {
	return &ImageService{}
}
func normalizeImageRefForPull(input string) (string, error) {
	s := strings.TrimSpace(input)
	if s == "" {
		return "", errors.New("image name is empty")
	}
	fields := strings.Fields(s)
	if len(fields) >= 3 {
		first := strings.ToLower(fields[0])
		second := strings.ToLower(fields[1])
		if (first == "docker" || first == "podman") && second == "pull" {
			s = fields[len(fields)-1]
		}
	}
	s = strings.TrimSpace(s)
	if strings.ContainsAny(s, " \t\r\n") {
		return "", errors.New("invalid image name")
	}
	return s, nil
}
func loadImagesWithRuntimeView(ctx context.Context) ([]image.Summary, []types.Container, error) {
	if docker.IsPodmanRuntime(ctx) && runtime.GOOS == "linux" {
		return docker.ListImagesMerged(ctx)
	}
	client, err := docker.NewDockerClient()
	if err != nil {
		return nil, nil, err
	}
	defer client.Close()
	list, err := client.ImageList(ctx, image.ListOptions{})
	if err != nil {
		return nil, nil, err
	}
	containers, _ := client.ContainerList(ctx, container.ListOptions{All: true})
	return list, containers, nil
}
func (u *ImageService) Page(req dto.SearchWithPage) (int64, interface{}, error) {
	if docker.IsPodmanRuntime(context.Background()) && runtime.GOOS == "darwin" {
		return u.pagePodman(req)
	}
	var (
		list       []image.Summary
		containers []types.Container
		records    []dto.ImageInfo
		backDatas  []dto.ImageInfo
	)
	ctx := context.Background()
	var err error
	list, containers, err = loadImagesWithRuntimeView(ctx)
	if err != nil {
		return 0, nil, err
	}
	if len(req.Info) != 0 {
		length, count := len(list), 0
		for count < length {
			hasTag := false
			for _, tag := range list[count].RepoTags {
				if strings.Contains(tag, req.Info) {
					hasTag = true
					break
				}
			}
			if !hasTag {
				list = append(list[:count], list[(count+1):]...)
				length--
			} else {
				count++
			}
		}
	}
	for _, image := range list {
		size := formatFileSize(image.Size)
		records = append(records, dto.ImageInfo{ID: image.ID, Tags: image.RepoTags, IsUsed: checkUsed(image.ID, containers), CreatedAt: time.Unix(image.Created, 0), Size: size})
	}
	total, start, end := len(records), (req.Page-1)*req.Limit, req.Page*req.Limit
	if start > total {
		backDatas = make([]dto.ImageInfo, 0)
	} else {
		if end >= total {
			end = total
		}
		backDatas = records[start:end]
	}
	return int64(total), backDatas, nil
}
func (u *ImageService) ListAll() ([]dto.ImageInfo, error) {
	if docker.IsPodmanRuntime(context.Background()) && runtime.GOOS == "darwin" {
		_, data, err := u.pagePodman(dto.SearchWithPage{PageInfo: dto.PageInfo{Page: 1, Limit: 1000000}})
		if err != nil {
			return nil, err
		}
		if items, ok := data.([]dto.ImageInfo); ok {
			return items, nil
		}
		return nil, errors.New("invalid response")
	}
	var records []dto.ImageInfo
	ctx := context.Background()
	var (
		list       []image.Summary
		containers []types.Container
	)
	var err error
	list, containers, err = loadImagesWithRuntimeView(ctx)
	if err != nil {
		return nil, err
	}
	for _, image := range list {
		size := formatFileSize(image.Size)
		records = append(records, dto.ImageInfo{ID: image.ID, Tags: image.RepoTags, IsUsed: checkUsed(image.ID, containers), CreatedAt: time.Unix(image.Created, 0), Size: size})
	}
	return records, nil
}
func (u *ImageService) List() ([]dto.Options, error) {
	if docker.IsPodmanRuntime(context.Background()) && runtime.GOOS == "darwin" {
		images, err := docker.PodmanListImages(context.Background())
		if err != nil {
			return nil, err
		}
		var backDatas []dto.Options
		for _, img := range images {
			for _, tag := range img.Tags {
				backDatas = append(backDatas, dto.Options{Option: tag})
			}
		}
		return backDatas, nil
	}
	var (
		list      []image.Summary
		backDatas []dto.Options
	)
	ctx := context.Background()
	var err error
	list, _, err = loadImagesWithRuntimeView(ctx)
	if err != nil {
		return nil, err
	}
	for _, image := range list {
		for _, tag := range image.RepoTags {
			backDatas = append(backDatas, dto.Options{Option: tag})
		}
	}
	return backDatas, nil
}

type dockerConfig struct {
	Auths map[string]authConfig `json:"auths"`
}
type authConfig struct {
	Auth string `json:"auth"`
}
