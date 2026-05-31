package service

import (
	"fmt"
	"github.com/aihop/gopanel/app/model"
	"strings"
)

func detectBuiltImageRef(p *model.Pipeline, version string, logs []string) string {
	outputImage := strings.TrimSpace(p.OutputImage)
	candidates := extractBuiltImageCandidates(logs)
	if outputImage != "" {
		for _, candidate := range candidates {
			if sameImageRepo(candidate, outputImage) && !strings.HasSuffix(candidate, ":latest") {
				return candidate
			}
		}
		for _, candidate := range candidates {
			if sameImageRepo(candidate, outputImage) {
				return candidate
			}
		}
		return fmt.Sprintf("%s:%s", outputImage, version)
	}
	for _, candidate := range candidates {
		if !strings.HasSuffix(candidate, ":latest") {
			return candidate
		}
	}
	if len(candidates) > 0 {
		return candidates[0]
	}
	if p.PipelineKey != "" {
		return fmt.Sprintf("%s:%s", p.PipelineKey, version)
	}
	if p.BuildImage != "host" && p.BuildImage != "" {
		return fmt.Sprintf("%s:%s", p.BuildImage, version)
	}
	return ""
}
func extractBuiltImageCandidates(logs []string) []string {
	candidates := make([]string, 0)
	seen := make(map[string]struct{})
	for i := len(logs) - 1; i >= 0; i-- {
		if imageRef := parseBuiltImageRef(logs[i]); imageRef != "" {
			if _, ok := seen[imageRef]; ok {
				continue
			}
			seen[imageRef] = struct{}{}
			candidates = append(candidates, imageRef)
		}
	}
	return candidates
}
func parseBuiltImageRef(line string) string {
	line = strings.TrimSpace(line)
	if idx := strings.Index(line, "naming to "); idx >= 0 {
		ref := strings.TrimSpace(line[idx+len("naming to "):])
		ref = strings.TrimSuffix(ref, " done")
		return normalizeBuiltImageRef(ref)
	}
	if idx := strings.Index(line, "Successfully tagged "); idx >= 0 {
		ref := strings.TrimSpace(line[idx+len("Successfully tagged "):])
		return normalizeBuiltImageRef(ref)
	}
	return ""
}
func normalizeBuiltImageRef(ref string) string {
	ref = strings.TrimSpace(ref)
	ref = strings.Trim(ref, "`\"'")
	ref = strings.TrimPrefix(ref, "docker.io/library/")
	return ref
}
func sameImageRepo(imageRef, outputImage string) bool {
	imageRef = normalizeBuiltImageRef(imageRef)
	outputImage = normalizeBuiltImageRef(outputImage)
	if imageRef == "" || outputImage == "" {
		return false
	}
	repo := imageRef
	if idx := strings.LastIndex(repo, ":"); idx > strings.LastIndex(repo, "/") {
		repo = repo[:idx]
	}
	return repo == outputImage || strings.HasSuffix(repo, "/"+outputImage)
}
