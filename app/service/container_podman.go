package service

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/utils/docker"
)

func (u *ContainerService) pagePodman(req *dto.PageContainer) (int64, interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	containers, err := docker.PodmanListContainers(ctx, true)
	if err != nil {
		return 0, nil, err
	}

	var list []docker.PodmanContainer
	if req.ExcludeAppStore {
		for _, c := range containers {
			if c.Labels != nil {
				if created, ok := c.Labels[composeCreatedBy]; ok && created == "Apps" {
					continue
				}
			}
			list = append(list, c)
		}
	} else {
		list = containers
	}

	if strings.TrimSpace(req.Filters) != "" {
		k, v, ok := splitLabelFilter(req.Filters)
		if ok {
			k = normalizeContainerLabelFilter(k, true)
			var filtered []docker.PodmanContainer
			for _, c := range list {
				if c.Labels == nil {
					continue
				}
				if lv, ok := c.Labels[k]; ok && (v == "" || lv == v) {
					filtered = append(filtered, c)
				}
			}
			list = filtered
		}
	}

	if strings.TrimSpace(req.Name) != "" {
		var filtered []docker.PodmanContainer
		for _, c := range list {
			if strings.Contains(c.Name, req.Name) {
				filtered = append(filtered, c)
			}
		}
		list = filtered
	}
	if req.State != "all" {
		var filtered []docker.PodmanContainer
		for _, c := range list {
			if c.State == req.State {
				filtered = append(filtered, c)
			}
		}
		list = filtered
	}

	switch req.OrderBy {
	case "name":
		sort.Slice(list, func(i, j int) bool {
			if req.Order == constant.OrderAsc {
				return list[i].Name < list[j].Name
			}
			return list[i].Name > list[j].Name
		})
	case "state":
		sort.Slice(list, func(i, j int) bool {
			if req.Order == constant.OrderAsc {
				return list[i].State < list[j].State
			}
			return list[i].State > list[j].State
		})
	default:
		sort.Slice(list, func(i, j int) bool {
			if req.Order == constant.OrderAsc {
				return list[i].Created.Before(list[j].Created)
			}
			return list[i].Created.After(list[j].Created)
		})
	}

	total := len(list)
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

	records := list[start:end]
	backDatas := make([]dto.ContainerInfo, len(records))
	for i := 0; i < len(records); i++ {
		item := records[i]
		isFromCompose := false
		if item.Labels != nil {
			if _, ok := firstLabel(item.Labels, composeProjectLabel, podmanComposeProjectLabel); ok {
				isFromCompose = true
			}
		}
		isFromApp := false
		if item.Labels != nil {
			if created, ok := item.Labels[composeCreatedBy]; ok && created == "Apps" {
				isFromApp = true
			}
		}

		imageID := item.ImageID
		if parts := strings.Split(imageID, ":"); len(parts) > 1 {
			imageID = parts[len(parts)-1]
		}

		info := dto.ContainerInfo{
			ContainerID:   item.ID,
			CreateTime:    item.Created.Format(constant.DateTimeLayout),
			Name:          strings.TrimPrefix(item.Name, "/"),
			ImageId:       imageID,
			ImageName:     item.Image,
			State:         item.State,
			RunTime:       item.Status,
			Ports:         item.Ports,
			IsFromApp:     isFromApp,
			IsFromCompose: isFromCompose,
		}

		appInstallRepo := repo.NewIAppInstallRepo()
		websiteRepo := repo.NewWebsite()
		install, _ := appInstallRepo.GetFirst(appInstallRepo.WithContainerName(info.Name))
		if install.ID > 0 {
			info.AppInstallName = install.Name
			info.AppName = "namemem"
			websites, _ := websiteRepo.GetBy(websiteRepo.WithAppInstallId(install.ID))
			for _, website := range websites {
				info.Websites = append(info.Websites, website.PrimaryDomain)
			}
		}

		backDatas[i] = info
	}

	return int64(total), backDatas, nil
}

func splitLabelFilter(filter string) (string, string, bool) {
	s := strings.TrimSpace(filter)
	if s == "" {
		return "", "", false
	}
	if strings.Contains(s, "=") {
		parts := strings.SplitN(s, "=", 2)
		return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), true
	}
	return s, "", true
}
