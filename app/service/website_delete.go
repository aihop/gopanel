package service

import (
	"context"
	"errors"
	"strings"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/docker"
)

func (s WebsiteService) Delete(id uint) error {
	website, err := s.repo.GetFirst(commonRepo.WithByID(id))
	if err != nil {
		return errors.New("网站不存在")
	}
	if website.Type == constant.Proxy || website.Type == constant.WebApp || website.Type == constant.Static {
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

		_, err := NewCaddy().RemoveServerBlock(targetDomain, strings.Join(otherDomains, "\n"))
		if err != nil {
			return err
		}
	}

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

	if err := s.repo.DeleteBy(context.Background(), commonRepo.WithByID(id)); err != nil {
		return err
	}

	_ = repo.NewWebsiteDomain().DeleteByWebsiteIdNotIsPrimary(context.Background(), id)

	db := global.DB.Where("website_id = ?", id).Delete(&model.WebsiteDomain{})
	if db.Error != nil {
		global.LOG.Errorf("Failed to delete website domains for website %d: %v", id, db.Error)
	}
	return nil
}
