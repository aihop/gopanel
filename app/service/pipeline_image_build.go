package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/utils/docker"
)

// stepBuildImage builds a Docker image from the release directory
// and pushes it if a registry is configured in actionParams.
type pipelineImageArtifact struct {
	Tag          string
	ID           string
	Digest       string
	RepoDigest   string
	ImmutableRef string
}

func (s *PipelineService) stepBuildImage(ctx context.Context, logger *PipelineLogger, p *model.Pipeline, releaseDir string, recordID uint) (pipelineImageArtifact, error) {
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

	cmd, err := docker.RuntimeCommand(ctx, args...)
	if err != nil {
		return pipelineImageArtifact{}, fmt.Errorf("镜像构建失败: %w", err)
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return pipelineImageArtifact{}, fmt.Errorf("镜像构建失败: %w (output: %s)", err, strings.TrimSpace(string(out)))
	}
	logger.Info("镜像构建成功: %s", imageRef)
	if output := strings.TrimSpace(string(out)); output != "" {
		logger.Info("构建输出:\n%s", output)
	}

	artifact, err := inspectPipelineImageArtifact(ctx, imageRef)
	if err != nil {
		return pipelineImageArtifact{}, fmt.Errorf("读取镜像不可变身份失败: %w", err)
	}
	logger.Info("镜像不可变引用: %s", artifact.ImmutableRef)
	return artifact, nil
}

func inspectPipelineImageArtifact(ctx context.Context, imageRef string) (pipelineImageArtifact, error) {
	cmd, err := docker.RuntimeCommand(ctx, "image", "inspect", imageRef)
	if err != nil {
		return pipelineImageArtifact{}, err
	}
	output, err := cmd.Output()
	if err != nil {
		return pipelineImageArtifact{}, err
	}
	return parsePipelineImageInspect(imageRef, output)
}

func parsePipelineImageInspect(imageRef string, output []byte) (pipelineImageArtifact, error) {
	var rows []struct {
		ID          string   `json:"Id"`
		Digest      string   `json:"Digest"`
		RepoDigests []string `json:"RepoDigests"`
	}
	if err := json.Unmarshal(output, &rows); err != nil || len(rows) == 0 {
		if err == nil {
			err = fmt.Errorf("empty image inspect result")
		}
		return pipelineImageArtifact{}, err
	}
	row := rows[0]
	repoDigests := append([]string(nil), row.RepoDigests...)
	sort.Strings(repoDigests)
	repoDigest := ""
	if len(repoDigests) > 0 {
		repoDigest = strings.TrimSpace(repoDigests[0])
	}
	digest := strings.TrimSpace(row.Digest)
	if at := strings.LastIndex(repoDigest, "@"); digest == "" && at >= 0 {
		digest = strings.TrimSpace(repoDigest[at+1:])
	}
	imageID := strings.TrimSpace(row.ID)
	if digest == "" {
		digest = imageID
	}
	immutableRef := repoDigest
	if immutableRef == "" {
		immutableRef = imageID
	}
	if digest == "" || immutableRef == "" {
		return pipelineImageArtifact{}, fmt.Errorf("image inspect did not return a content digest")
	}
	return pipelineImageArtifact{Tag: strings.TrimSpace(imageRef), ID: imageID, Digest: digest, RepoDigest: repoDigest, ImmutableRef: immutableRef}, nil
}

func recordVersionTag(recordID uint) string {
	return fmt.Sprintf("record-%d", recordID)
}
