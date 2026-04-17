package service

import (
	"context"
	"strings"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
)

func (s *WebsiteService) SyncFromCaddyfile() error {
	hosts, err := NewCaddy().GetAllHosts()
	if err != nil {
		global.LOG.Errorf("Failed to get hosts from Caddyfile: %v", err)
		return err
	}

	for _, host := range hosts {
		if host == "" {
			continue
		}
		if strings.Contains(host, "localhost") {
			continue
		}

		protocol := "http"
		primaryDomain := host
		if strings.HasPrefix(host, "https://") {
			protocol = "https"
			primaryDomain = strings.TrimPrefix(host, "https://")
		} else if strings.HasPrefix(host, "http://") {
			protocol = "http"
			primaryDomain = strings.TrimPrefix(host, "http://")
		}

		_, err = s.repo.GetFirst(s.repo.WithDomain(primaryDomain))
		if err == nil {
			continue
		}

		newWeb := &model.Website{
			PrimaryDomain: primaryDomain,
			Protocol:      protocol,
			Type:          constant.Static,
			Alias:         strings.ReplaceAll(primaryDomain, ".", "_"),
			Remark:        "自动从配置文件同步",
			Status:        "Running",
		}
		if err := s.repo.Create(context.Background(), newWeb); err != nil {
			global.LOG.Errorf("Failed to sync host %s from Caddyfile to DB: %v", host, err)
		} else {
			domainRepo := repo.NewWebsiteDomain()
			_ = domainRepo.BatchCreate(context.Background(), []model.WebsiteDomain{{
				WebsiteID: newWeb.ID,
				Domain:    primaryDomain,
				Port:      80,
			}})
			global.LOG.Infof("Successfully synced missing host %s from Caddyfile to DB", host)
		}
	}
	return nil
}
