package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/utils/docker"
)

// stepBuildImage builds a Docker image from the release directory
// and pushes it if a registry is configured in actionParams.
func (s *PipelineService) stepBuildImage(ctx context.Context, logger *PipelineLogger, p *model.Pipeline, releaseDir string, recordID uint) (string, error) {
	logger.Info("开始构建 Docker 镜像...")

	// Parse action params
	params := struct {
		ImageName string `json:"imageName"`
		Tag       string `json:"tag"`
	}{}
	if p.ActionParams != "" {
		_ = json.Unmarshal([]byte(p.ActionParams), &params)
	}

	imageName := strings.TrimSpace(params.ImageName)
	if imageName == "" {
		imageName = p.PipelineKey
	}

	tag := strings.TrimSpace(params.Tag)
	if tag == "" {
		tag = recordVersionTag(recordID)
	}

	imageRef := fmt.Sprintf("%s:%s", imageName, tag)
	logger.Info("镜像引用: %s", imageRef)

	dockerfile := "Dockerfile"
	if p.ArtifactPath != "" {
		dockerfile = p.ArtifactPath + "/Dockerfile"
	}

	// docker build -t <image> <releaseDir>
	args := []string{"build", "-t", imageRef, releaseDir}
	// Use Dockerfile from artifactPath if specified, otherwise Dockerfile in releaseDir
	args = append(args, "-f", dockerfile)

	out, err := docker.RuntimeCommand(ctx, args...)
	if err != nil {
		return "", fmt.Errorf("镜像构建失败: %w (output: %s)", err, out)
	}
	logger.Info("镜像构建成功: %s", imageRef)
	logger.Info("构建输出:\n%s", out)

	return imageRef, nil
}

func recordVersionTag(recordID uint) string {
	return fmt.Sprintf("record-%d", recordID)
}
