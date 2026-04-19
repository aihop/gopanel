package service

import (
	"context"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/utils/docker"
)

func (u *ImageService) pagePodman(req dto.SearchWithPage) (int64, interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	images, err := docker.PodmanListImages(ctx)
	if err != nil {
		return 0, nil, err
	}

	containers, _ := docker.PodmanListContainers(ctx, true)
	used := make(map[string]struct{}, len(containers))
	for _, c := range containers {
		if c.ImageID != "" {
			used[c.ImageID] = struct{}{}
		}
	}

	var records []dto.ImageInfo
	for _, img := range images {
		if strings.TrimSpace(req.Info) != "" {
			ok := false
			for _, tag := range img.Tags {
				if strings.Contains(tag, req.Info) {
					ok = true
					break
				}
			}
			if !ok {
				continue
			}
		}
		_, isUsed := used[img.ID]
		records = append(records, dto.ImageInfo{
			ID:        img.ID,
			Tags:      img.Tags,
			IsUsed:    isUsed,
			CreatedAt: img.Created,
			Size:      img.Size,
		})
	}

	total := len(records)
	page := req.Page
	if page <= 0 {
		page = 1
	}
	limit := req.Limit
	if limit <= 0 {
		limit = 10
	}
	start := (page - 1) * limit
	end := page * limit
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}
	return int64(total), records[start:end], nil
}
