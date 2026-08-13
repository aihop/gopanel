package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/aihop/gopanel/app/model"
)

type pipelineReleaseArtifact struct {
	sourceType       string
	imageTag         string
	imageDigest      string
	archiveFile      string
	releaseDir       string
	artifactDigest   string
	artifactManifest string
	artifactMeta     string
}

func buildPipelineReleaseArtifact(ctx context.Context, pipeline *model.Pipeline, record *model.PipelineRecord) (pipelineReleaseArtifact, error) {
	if pipeline == nil || record == nil {
		return pipelineReleaseArtifact{}, fmt.Errorf("发布版本失败：缺少流水线或构建记录信息")
	}
	result := pipelineReleaseArtifact{
		sourceType:  "release_dir",
		imageTag:    strings.TrimSpace(record.ImageTag),
		archiveFile: strings.TrimSpace(record.ArchiveFile),
		releaseDir:  strings.TrimSpace(record.RunnerReleaseDir),
	}
	if result.releaseDir == "" {
		result.releaseDir = pipelineReleaseDir(pipeline)
	}

	manifest := model.ArtifactManifest{
		SchemaVersion:    model.ArtifactManifestSchemaVersion,
		PipelineID:       pipeline.ID,
		PipelineRecordID: record.ID,
		Commit:           strings.TrimSpace(record.CommitHash),
		SourceType:       strings.TrimSpace(record.SourceType),
		SourceID:         record.SourceID,
		SourceDigest:     strings.TrimSpace(record.SourceDigest),
		Runtime:          pipelineArtifactRuntime(pipeline, record),
	}

	switch {
	case result.imageTag != "":
		result.sourceType = "image"
		imageArtifact, err := pipelineImageArtifactFromRecord(ctx, record)
		if err != nil {
			return pipelineReleaseArtifact{}, err
		}
		result.imageDigest = imageArtifact.Digest
		result.artifactDigest = imageArtifact.Digest
		manifest.Type = model.ArtifactTypeContainerImage
		manifest.Digest = imageArtifact.Digest
		manifest.Image = &model.ArtifactImageManifest{Tag: imageArtifact.Tag, ID: imageArtifact.ID, RepoDigest: imageArtifact.RepoDigest, ImmutableRef: imageArtifact.ImmutableRef}
		result.archiveFile = ""
		result.releaseDir = ""
	case result.archiveFile != "":
		result.sourceType = "archive"
		digest, size, err := pipelineFileDigest(result.archiveFile)
		if err != nil {
			return pipelineReleaseArtifact{}, fmt.Errorf("计算归档制品摘要失败: %w", err)
		}
		result.artifactDigest = digest
		manifest.Type = model.ArtifactTypeStaticArchive
		manifest.Digest = digest
		manifest.SizeBytes = size
		manifest.Archive = &model.ArtifactFileManifest{Path: result.archiveFile}
		result.releaseDir = ""
	default:
		releaseDir, err := snapshotPipelineReleaseDir(pipeline, record, result.releaseDir)
		if err != nil {
			return pipelineReleaseArtifact{}, err
		}
		result.releaseDir = releaseDir
		digest, size, err := pipelineDirectoryDigest(releaseDir)
		if err != nil {
			return pipelineReleaseArtifact{}, fmt.Errorf("计算目录制品摘要失败: %w", err)
		}
		result.artifactDigest = digest
		manifest.Type = model.ArtifactTypeReleaseDirectory
		manifest.Digest = digest
		manifest.SizeBytes = size
		manifest.Directory = &model.ArtifactFileManifest{Path: releaseDir}
	}

	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		return pipelineReleaseArtifact{}, err
	}
	metaJSON, err := json.Marshal(map[string]interface{}{
		"artifactPath": strings.TrimSpace(pipeline.ArtifactPath), "buildImage": strings.TrimSpace(pipeline.BuildImage),
		"pipelineKey": strings.TrimSpace(pipeline.PipelineKey), "runnerMode": strings.TrimSpace(pipeline.RunnerMode),
		"runnerHostPort": record.RunnerHostPort, "runnerContainerId": strings.TrimSpace(record.RunnerContainerID),
	})
	if err != nil {
		return pipelineReleaseArtifact{}, err
	}
	result.artifactManifest = string(manifestJSON)
	result.artifactMeta = string(metaJSON)
	return result, nil
}

func (artifact pipelineReleaseArtifact) release(pipeline *model.Pipeline, record *model.PipelineRecord) *model.Release {
	return &model.Release{
		PipelineID: pipeline.ID, PipelineRecordID: record.ID, Version: strings.TrimSpace(record.Version),
		CommitHash: strings.TrimSpace(record.CommitHash), Changelog: strings.TrimSpace(record.Changelog),
		SourceType: artifact.sourceType, ImageTag: artifact.imageTag, ImageDigest: artifact.imageDigest,
		ArchiveFile: artifact.archiveFile, ReleaseDir: artifact.releaseDir, ArtifactDigest: artifact.artifactDigest,
		ArtifactManifest: artifact.artifactManifest, ArtifactMeta: artifact.artifactMeta, Status: "ready",
	}
}

func pipelineImageArtifactFromRecord(ctx context.Context, record *model.PipelineRecord) (pipelineImageArtifact, error) {
	imageArtifact := pipelineImageArtifact{
		Tag: strings.TrimSpace(record.ImageTag), ID: strings.TrimSpace(record.ImageID),
		Digest: strings.TrimSpace(record.ImageDigest), ImmutableRef: strings.TrimSpace(record.ImageRef),
	}
	if strings.Contains(imageArtifact.ImmutableRef, "@") {
		imageArtifact.RepoDigest = imageArtifact.ImmutableRef
	}
	if imageArtifact.Digest != "" && imageArtifact.ImmutableRef != "" {
		return imageArtifact, nil
	}
	inspected, err := inspectPipelineImageArtifact(ctx, imageArtifact.Tag)
	if err != nil {
		return pipelineImageArtifact{}, fmt.Errorf("发布镜像版本失败：无法确认镜像不可变身份: %w", err)
	}
	return inspected, nil
}

func pipelineArtifactRuntime(pipeline *model.Pipeline, record *model.PipelineRecord) model.ArtifactRuntimeManifest {
	runtimeManifest := model.ArtifactRuntimeManifest{Mode: strings.TrimSpace(pipeline.RunnerMode)}
	if strings.TrimSpace(pipeline.RunnerConfig) != "" {
		var raw map[string]interface{}
		if json.Unmarshal([]byte(pipeline.RunnerConfig), &raw) == nil {
			runner := parseRunnerConfig(raw)
			runtimeManifest.StartCommand = strings.TrimSpace(runner.StartCommand)
			runtimeManifest.WorkingDir = strings.TrimSpace(runner.WorkingDir)
			if port, err := strconv.Atoi(strings.TrimSpace(runner.ContainerPort)); err == nil {
				runtimeManifest.Port = port
			}
		}
	}
	if runtimeManifest.Port == 0 && strings.TrimSpace(pipeline.ActionParams) != "" {
		var actionParams struct {
			ExposePort int `json:"exposePort"`
		}
		if json.Unmarshal([]byte(pipeline.ActionParams), &actionParams) == nil {
			runtimeManifest.Port = actionParams.ExposePort
		}
	}
	return runtimeManifest
}

func pipelineFileDigest(path string) (string, int64, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return "", 0, err
	}
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", 0, err
	}
	return "sha256:" + hex.EncodeToString(hash.Sum(nil)), info.Size(), nil
}

func pipelineDirectoryDigest(root string) (string, int64, error) {
	entries := make([]string, 0)
	err := filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == root {
			return nil
		}
		entries = append(entries, path)
		return nil
	})
	if err != nil {
		return "", 0, err
	}
	sort.Strings(entries)
	hash := sha256.New()
	var size int64
	for _, path := range entries {
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return "", 0, err
		}
		info, err := os.Lstat(path)
		if err != nil {
			return "", 0, err
		}
		_, _ = io.WriteString(hash, filepath.ToSlash(relative)+"\x00"+info.Mode().String()+"\x00")
		if info.IsDir() {
			continue
		}
		if info.Mode()&os.ModeSymlink != 0 {
			target, err := os.Readlink(path)
			if err != nil {
				return "", 0, err
			}
			_, _ = io.WriteString(hash, "link\x00"+target+"\x00")
			continue
		}
		file, err := os.Open(path)
		if err != nil {
			return "", 0, err
		}
		written, copyErr := io.Copy(hash, file)
		closeErr := file.Close()
		if copyErr != nil {
			return "", 0, copyErr
		}
		if closeErr != nil {
			return "", 0, closeErr
		}
		size += written
		_, _ = io.WriteString(hash, "\x00")
	}
	return "sha256:" + hex.EncodeToString(hash.Sum(nil)), size, nil
}
