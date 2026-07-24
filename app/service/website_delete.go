package service

import (
	"context"
	"errors"
	"strings"

	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/docker"
)

func (s WebsiteService) Delete(ctx context.Context, id uint) error {
	website, err := s.repo.GetFirst(commonRepo.WithByID(id))
	if err != nil {
		return errors.New("网站不存在")
	}
	tx := global.DB.Begin()
	defer func() {
		if tx != nil {
			tx.Rollback()
		}
	}()
	txCtx := context.WithValue(ctx, constant.DB, tx)
	if website.Type == constant.Proxy || website.Type == constant.WebApp || website.Type == constant.Static || website.Type == constant.Redirect {
		var otherDomains []string
		if website.Domains != nil {
			for _, d := range website.Domains {
				if normalizeWebsiteDomainForCompare(d.Domain) != normalizeWebsiteDomainForCompare(website.PrimaryDomain) {
					otherDomains = append(otherDomains, d.Domain)
				}
			}
		}

		targetDomain := website.PrimaryDomain
		if strings.EqualFold(website.Protocol, constant.ProtocolHTTP) || strings.EqualFold(website.Protocol, constant.Http) {
			targetDomain = "http://" + website.PrimaryDomain
		} else if strings.EqualFold(website.Protocol, constant.ProtocolHTTPS) {
			targetDomain = "https://" + website.PrimaryDomain
		}

		_, err := CaddyRemoveServerBlock(ctx, targetDomain, strings.Join(otherDomains, "\n"))
		if err != nil {
			return err
		}
	}
	domainRepo := repo.NewWebsiteDomain()
	if err := domainRepo.DeleteByWebsiteId(txCtx, id); err != nil {
		return err
	}
	if err := repo.NewWebsiteUpstream().DeleteByWebsiteID(txCtx, id); err != nil {
		return err
	}
	if err := s.repo.DeleteBy(txCtx, commonRepo.WithByID(id)); err != nil {
		return err
	}
	if err := tx.Commit().Error; err != nil {
		return err
	}
	tx = nil
	if website.Type == constant.WebApp && website.ContainerID != "" {
		cli, err := docker.NewDockerClient()
		if err == nil {
			defer cli.Close()
			err = RemoveEngineContainer(context.Background(), cli, website.ContainerID)
			if err != nil {
				global.LOG.Errorf("Failed to remove engine container %s: %v", website.ContainerID, err)
			}
		}
	}
	return nil
}
