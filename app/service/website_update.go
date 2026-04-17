package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/init/caddy"
	"github.com/aihop/gopanel/utils/files"
)

func (s WebsiteService) Update(req *request.WebsiteUpdate) error {
	website, err := s.repo.GetFirst(commonRepo.WithByID(req.ID))
	if err != nil {
		return errors.New("网站不存在")
	}
	oldProxy := website.Proxy
	originalDomains := website.Domains
	if normalizedPrimaryDomain := sanitizeWebsitePrimaryDomain(req.PrimaryDomain); normalizedPrimaryDomain != "" {
		website.PrimaryDomain = normalizedPrimaryDomain
	}
	if strings.TrimSpace(req.Protocol) != "" {
		website.Protocol = strings.TrimSpace(req.Protocol)
	}
	website.Remark = req.Remark
	website.IPV6 = req.IPV6
	if strings.TrimSpace(req.Proxy) != "" {
		website.Proxy = strings.TrimSpace(req.Proxy)
	}
	website.PipelineID = req.PipelineId
	if req.CodeSource != "" {
		website.CodeSource = req.CodeSource
	}
	var newContent, updatedContent []byte
	caddyService := NewCaddy()
	var domains []model.WebsiteDomain
	var isUpdateOtherDomains bool
	var shouldRewriteCaddy bool
	var targetOtherDomains string
	var oldDomain, newDomain []string
	if req.OtherDomains != "" && website.PrimaryDomain != req.OtherDomains {
		fileUtil := files.NewFileOp()
		content, err := fileUtil.GetContent(caddy.CaddyFilePath())
		if err != nil {
			if os.IsNotExist(err) {
				shouldRewriteCaddy = true
			} else {
				return err
			}
		}
		defaultHttpPort := 80
		domains, _, _, _ = getWebsiteDomains(req.OtherDomains, defaultHttpPort, website.ID)
		var otherDomains, newOtherDomains string
		if len(domains) > 0 {
			for _, v := range domains {
				newDomain = append(newDomain, v.Domain)
				if v.Port != 443 {
					newOtherDomains += fmt.Sprintf("http://%s\n", v.Domain)
				}
			}
		}
		if len(website.Domains) > 0 {
			for _, d := range website.Domains {
				oldDomain = append(oldDomain, d.Domain)
				if normalizeWebsiteDomainForCompare(d.Domain) != normalizeWebsiteDomainForCompare(website.PrimaryDomain) {
					otherDomains += fmt.Sprintf("%s\n", d.Domain)
				}
			}
		}
		otherDomains = strings.TrimSuffix(otherDomains, "\n")
		newOtherDomains = strings.TrimSuffix(newOtherDomains, "\n")
		targetOtherDomains = newOtherDomains

		if !shouldRewriteCaddy && isDomainChanged(oldDomain, newDomain) {
			newContent, err = caddyService.UpdateOtherDomains(content, website.PrimaryDomain, otherDomains, newOtherDomains)
			if err != nil {
				return err
			}
			isUpdateOtherDomains = true
		} else if isDomainChanged(oldDomain, newDomain) {
			isUpdateOtherDomains = true
		}

		if isUpdateOtherDomains {
			domainRepo := repo.NewWebsiteDomain()
			if err := domainRepo.DeleteByWebsiteIdNotIsPrimary(context.Background(), website.ID); err != nil {
				return err
			}
		}
	}
	if oldProxy != req.Proxy {
		if newContent != nil {
			fileUtil := files.NewFileOp()
			content, err := fileUtil.GetContent(caddy.CaddyFilePath())
			if err != nil {
				if os.IsNotExist(err) {
					shouldRewriteCaddy = true
				} else {
					return err
				}
			}
			updatedContent = content
		}
		if !shouldRewriteCaddy {
			newContent, _ = caddyService.UpdateProxy(updatedContent, website.PrimaryDomain, req.Proxy)
		}
	}
	if err := s.repo.Save(context.Background(), &website); err != nil {
		return err
	}
	if isUpdateOtherDomains {
		domainRepo := repo.NewWebsiteDomain()
		if err := domainRepo.BatchCreate(context.Background(), domains); err != nil {
			return err
		}
		website.Domains = make([]*model.WebsiteDomain, 0, len(domains))
		for i := range domains {
			domain := domains[i]
			website.Domains = append(website.Domains, &domain)
		}
	}
	if !isUpdateOtherDomains {
		website.Domains = originalDomains
	}
	if shouldEnsureWebsiteCaddyConfig(&website, caddyService) {
		shouldRewriteCaddy = true
	}
	if shouldRewriteCaddy {
		if targetOtherDomains == "" && len(website.Domains) > 0 {
			targetOtherDomains = buildWebsiteOtherDomains(&website)
		}
		return rewriteWebsiteCaddyConfig(&website, targetOtherDomains)
	}
	if newContent != nil {
		return caddyService.SaveContent(newContent)
	}
	return nil
}

func shouldEnsureWebsiteCaddyConfig(website *model.Website, caddyService *CaddyService) bool {
	if website == nil {
		return false
	}
	switch website.Type {
	case constant.Static, constant.Proxy, constant.WebApp:
	default:
		return false
	}
	domain := BuildWebsiteCaddyDomain(website.PrimaryDomain, website.Protocol)
	if domain == "" {
		return false
	}
	exists, err := caddyService.ExistAddress(domain)
	if err != nil {
		return true
	}
	return !exists
}

func buildWebsiteOtherDomains(website *model.Website) string {
	if len(website.Domains) == 0 {
		return ""
	}
	var domains []string
	for _, d := range website.Domains {
		if d.Domain == "" || normalizeWebsiteDomainForCompare(d.Domain) == normalizeWebsiteDomainForCompare(website.PrimaryDomain) {
			continue
		}
		domains = append(domains, d.Domain)
	}
	return strings.Join(domains, "\n")
}

func rewriteWebsiteCaddyConfig(website *model.Website, otherDomains string) error {
	domain := BuildWebsiteCaddyDomain(website.PrimaryDomain, website.Protocol)
	caddyService := NewCaddy()
	switch website.Type {
	case constant.Static:
		_, err := caddyService.ReplaceStaticServerBlock(domain, website.SiteDir, otherDomains, website.Protocol)
		return err
	case constant.Proxy, constant.WebApp:
		_, err := caddyService.ReplaceServerBlock(domain, website.Proxy, otherDomains, website.Protocol)
		return err
	default:
		return nil
	}
}
