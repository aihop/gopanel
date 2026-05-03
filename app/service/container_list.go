package service

import (
	"context"
	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/docker"
	"github.com/docker/docker/api/types"
	"sort"
	"strings"
	"time"
)

func (u *ContainerService) Page(req *dto.PageContainer) (int64, interface{}, error) {
	var (
		records []types.Container
		list    []types.Container
	)
	ctx := context.Background()
	resolved := docker.ResolveRuntime(ctx)
	isPodman := docker.IsPodmanRuntime(ctx)
	containers, sourceByID, err := getContainerListView(ctx)
	if err != nil {
		return 0, nil, err
	}
	if req.ExcludeAppStore {
		for _, item := range containers {
			if created, ok := item.Labels[composeCreatedBy]; ok && created == "Apps" {
				continue
			}
			list = append(list, item)
		}
	} else {
		list = containers
	}
	if strings.TrimSpace(req.Filters) != "" {
		k, v, ok := splitLabelFilter(req.Filters)
		if ok {
			k = normalizeContainerLabelFilter(k, isPodman)
			var filtered []types.Container
			for _, item := range list {
				if item.Labels == nil {
					continue
				}
				if lv, ok := item.Labels[k]; ok && (v == "" || lv == v) {
					filtered = append(filtered, item)
				}
			}
			list = filtered
		}
	}
	if len(req.Name) != 0 {
		length, count := len(list), 0
		for count < length {
			if !strings.Contains(containerPrimaryName(list[count]), req.Name) {
				list = append(list[:count], list[(count+1):]...)
				length--
			} else {
				count++
			}
		}
	}
	if req.State != "all" {
		length, count := len(list), 0
		for count < length {
			if list[count].State != req.State {
				list = append(list[:count], list[(count+1):]...)
				length--
			} else {
				count++
			}
		}
	}
	switch req.OrderBy {
	case "name":
		sort.Slice(list, func(i, j int) bool {
			if req.Order == constant.OrderAsc {
				return containerPrimaryName(list[i]) < containerPrimaryName(list[j])
			}
			return containerPrimaryName(list[i]) > containerPrimaryName(list[j])
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
				return list[i].Created < list[j].Created
			}
			return list[i].Created > list[j].Created
		})
	}
	total, start, end := len(list), (req.Page-1)*req.Limit, req.Page*req.Limit
	if start > total {
		records = make([]types.Container, 0)
	} else {
		if end >= total {
			end = total
		}
		records = list[start:end]
	}
	backDatas := make([]dto.ContainerInfo, len(records))
	relatedMeta, err := preloadContainerPageMeta(records)
	if err != nil {
		return 0, nil, err
	}
	for i := 0; i < len(records); i++ {
		item := records[i]
		IsFromCompose := false
		if _, ok := firstLabel(item.Labels, composeProjectLabel, podmanComposeProjectLabel); ok {
			IsFromCompose = true
		}
		IsFromApp := false
		if created, ok := item.Labels[composeCreatedBy]; ok && created == "Apps" {
			IsFromApp = true
		}
		exposePorts := transPortToStr(records[i].Ports)
		name := ""
		if len(item.Names) > 0 {
			name = strings.TrimPrefix(item.Names[0], "/")
		}
		imageID := item.ImageID
		if parts := strings.Split(imageID, ":"); len(parts) > 1 {
			imageID = parts[len(parts)-1]
		}
		info := dto.ContainerInfo{ContainerID: item.ID, CreateTime: time.Unix(item.Created, 0).Format(constant.DateTimeLayout), Name: name, ImageId: imageID, ImageName: item.Image, State: item.State, RunTime: item.Status, Ports: exposePorts, IsFromApp: IsFromApp, IsFromCompose: IsFromCompose, RuntimeKind: string(resolved.Kind), RuntimeMode: inferContainerRuntimeMode(string(resolved.Kind), ""), SourceType: "manual"}
		if isPodman && sourceByID != nil {
			info.RuntimeHost = strings.TrimSpace(sourceByID[item.ID])
		}
		info.RuntimeMode = inferContainerRuntimeMode(info.RuntimeKind, info.RuntimeHost)
		if install, ok := relatedMeta.installByContainerName[info.Name]; ok && install.ID > 0 {
			info.AppInstallName = install.Name
			info.SourceType = "app"
			info.Websites = append(info.Websites, relatedMeta.websiteDomainsByInstallID[install.ID]...)
		}
		if info.SourceType == "manual" {
			if website, ok := relatedMeta.websiteByContainerID[item.ID]; ok && website.ID > 0 {
				if strings.TrimSpace(website.PrimaryDomain) != "" {
					info.Websites = append(info.Websites, website.PrimaryDomain)
				}
				if website.PipelineID > 0 {
					info.SourceType = "pipeline"
				} else {
					info.SourceType = "website"
				}
			} else if strings.HasPrefix(strings.ToLower(info.Name), "pipeline-") {
				info.SourceType = "pipeline"
			}
		}
		if info.SourceType == "manual" && info.IsFromCompose {
			info.SourceType = "compose"
		}
		backDatas[i] = info
		if item.NetworkSettings != nil && len(item.NetworkSettings.Networks) > 0 {
			networks := make([]string, 0, len(item.NetworkSettings.Networks))
			for key := range item.NetworkSettings.Networks {
				if ip := strings.TrimSpace(item.NetworkSettings.Networks[key].IPAddress); ip != "" {
					networks = append(networks, ip)
				}
			}
			backDatas[i].Network = normalizeContainerIPList(networks)
		}
	}
	return int64(total), backDatas, nil
}
func preloadContainerPageMeta(records []types.Container) (*containerPageRelatedMeta, error) {
	meta := &containerPageRelatedMeta{installByContainerName: make(map[string]model.AppInstall), websiteDomainsByInstallID: make(map[uint][]string), websiteByContainerID: make(map[string]model.Website)}
	if len(records) == 0 {
		return meta, nil
	}
	containerNames := make([]string, 0, len(records))
	containerIDs := make([]string, 0, len(records))
	seenNames := make(map[string]struct{}, len(records))
	seenIDs := make(map[string]struct{}, len(records))
	for _, item := range records {
		name := normalizeContainerName(containerPrimaryName(item))
		if name != "" {
			if _, ok := seenNames[name]; !ok {
				seenNames[name] = struct{}{}
				containerNames = append(containerNames, name)
			}
		}
		id := strings.TrimSpace(item.ID)
		if id != "" {
			if _, ok := seenIDs[id]; !ok {
				seenIDs[id] = struct{}{}
				containerIDs = append(containerIDs, id)
			}
		}
	}
	if len(containerNames) > 0 {
		var installs []model.AppInstall
		query := global.DB.Model(&model.AppInstall{}).Select("id", "name", "container_name").Where("1 = 0")
		for _, name := range containerNames {
			query = query.Or("container_name = ? OR container_name LIKE ? OR container_name LIKE ? OR container_name LIKE ?", name, name+",%", "%,"+name+",%", "%,"+name)
		}
		if err := query.Find(&installs).Error; err != nil {
			return nil, err
		}
		for _, install := range installs {
			for _, name := range splitContainerNames(install.ContainerName) {
				if _, ok := meta.installByContainerName[name]; !ok {
					meta.installByContainerName[name] = install
				}
			}
		}
	}
	installIDs := make([]uint, 0, len(meta.installByContainerName))
	seenInstallIDs := make(map[uint]struct{}, len(meta.installByContainerName))
	for _, install := range meta.installByContainerName {
		if install.ID == 0 {
			continue
		}
		if _, ok := seenInstallIDs[install.ID]; ok {
			continue
		}
		seenInstallIDs[install.ID] = struct{}{}
		installIDs = append(installIDs, install.ID)
	}
	if len(installIDs) == 0 && len(containerIDs) == 0 {
		return meta, nil
	}
	var websites []model.Website
	query := global.DB.Model(&model.Website{}).Select("id", "primary_domain", "app_install_id", "container_id", "pipeline_id").Order("id asc").Where("1 = 0")
	if len(installIDs) > 0 {
		query = query.Or("app_install_id IN ?", installIDs)
	}
	if len(containerIDs) > 0 {
		query = query.Or("container_id IN ?", containerIDs)
	}
	if err := query.Find(&websites).Error; err != nil {
		return nil, err
	}
	for _, website := range websites {
		if website.AppInstallID > 0 {
			domain := strings.TrimSpace(website.PrimaryDomain)
			if domain != "" {
				meta.websiteDomainsByInstallID[website.AppInstallID] = append(meta.websiteDomainsByInstallID[website.AppInstallID], domain)
			}
		}
		containerID := strings.TrimSpace(website.ContainerID)
		if containerID != "" {
			if _, ok := meta.websiteByContainerID[containerID]; !ok {
				meta.websiteByContainerID[containerID] = website
			}
		}
	}
	return meta, nil
}
func (u *ContainerService) List() ([]string, error) {
	ctx := context.Background()
	containers, _, err := getContainerListView(ctx)
	if err != nil {
		return nil, err
	}
	var datas []string
	for _, container := range containers {
		for _, name := range container.Names {
			if len(name) != 0 {
				datas = append(datas, strings.TrimPrefix(name, "/"))
			}
		}
	}
	return datas, nil
}
