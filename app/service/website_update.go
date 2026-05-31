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
	"github.com/aihop/gopanel/utils/files"
)

func (s WebsiteService) Update(ctx context.Context, req *request.WebsiteUpdate) error {
	website, err := s.repo.GetFirst(commonRepo.WithByID(req.ID))
	if err != nil {
		return errors.New("网站不存在")
	}
	if err := ensurePipelineExists(req.PipelineId); err != nil {
		return err
	}
	oldProxy := website.Proxy
	originalDomains := website.Domains
	if normalizedPrimaryDomain := sanitizeWebsitePrimaryDomain(req.PrimaryDomain); normalizedPrimaryDomain != "" {
		website.PrimaryDomain = normalizedPrimaryDomain
	}
	if strings.TrimSpace(req.Protocol) != "" {
		if normalizedProtocol := normalizeWebsiteProtocol(req.Protocol); normalizedProtocol != "" {
			website.Protocol = normalizedProtocol
		}
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

	website.AntiCrawler = req.AntiCrawler
	website.AntiLeech = req.AntiLeech
	website.RateLimitMode = req.RateLimitMode
	website.WafEnable = req.WafEnable
	website.BlockSensitive = req.BlockSensitive
	website.IPAllowlist = strings.TrimSpace(req.IPAllowlist)
	website.IPBlocklist = strings.TrimSpace(req.IPBlocklist)
	website.SecurityHeader = req.SecurityHeader
	website.HstsEnabled = req.HstsEnabled
	website.HttpConfig = req.HttpConfig
	website.RedirectCode = req.RedirectCode
	website.RedirectDomainsToPrimary = req.RedirectDomainsToPrimary

	var newContent, updatedContent string
	var domains []model.WebsiteDomain
	var isUpdateOtherDomains bool
	var shouldRewriteCaddy bool
	var targetOtherDomains string
	var oldDomain, newDomain []string
	if req.OtherDomains != "" && website.PrimaryDomain != req.OtherDomains {
		fileUtil := files.NewFileOp()
		content, err := fileUtil.GetContent(CaddyFilePath())
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
			newContent, err = CaddyUpdateOtherDomains(ctx, string(content), website.PrimaryDomain, otherDomains, newOtherDomains)
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
		if newContent != "" {
			fileUtil := files.NewFileOp()
			content, err := fileUtil.GetContent(CaddyFilePath())
			if err != nil {
				if os.IsNotExist(err) {
					shouldRewriteCaddy = true
				} else {
					return err
				}
			}
			updatedContent = string(content)
		}
		if !shouldRewriteCaddy {
			newContent, _ = CaddyUpdateProxy(updatedContent, website.PrimaryDomain, req.Proxy)
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
	if shouldEnsureWebsiteCaddyConfig(&website) {
		shouldRewriteCaddy = true
	}
	if shouldRewriteCaddy || newContent != "" {
		if targetOtherDomains == "" && len(website.Domains) > 0 {
			targetOtherDomains = buildWebsiteOtherDomains(&website)
		}
		_ = targetOtherDomains
		return ApplyCaddyFromDB(ctx)
	}
	return ApplyCaddyFromDB(ctx)
}

func shouldEnsureWebsiteCaddyConfig(website *model.Website) bool {
	if website == nil {
		return false
	}
	switch website.Type {
	case constant.Static, constant.Proxy, constant.WebApp, constant.Redirect:
	default:
		return false
	}
	domain := BuildWebsiteCaddyDomain(website.PrimaryDomain, website.Protocol)
	if domain == "" {
		return false
	}
	exists, err := CaddyExistAddress(domain)
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
