package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
)

// Update 更新网站
// @param ctx 上下文
// @param req 更新网站请求
// @return error 错误
func (s WebsiteService) Update(ctx context.Context, req *request.WebsiteUpdate) error {
	website, err := s.repo.GetFirst(commonRepo.WithByID(req.ID))
	if err != nil {
		return errors.New("网站不存在")
	}
	originalDomains := website.Domains
	var upstreams []model.WebsiteUpstream
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
	if website.Type == constant.Proxy {
		proxyFallback := strings.TrimSpace(req.Proxy)
		if proxyFallback == "" {
			proxyFallback = website.Proxy
		}
		upstreams, err = ensureWebsiteProxyUpstreams(req.Upstreams, proxyFallback)
		if err != nil {
			return err
		}
		website.Proxy = websiteProxyFromUpstreams(upstreams, website.Proxy)
	} else if strings.TrimSpace(req.Proxy) != "" {
		website.Proxy = strings.TrimSpace(req.Proxy)
	}
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

	var domains []model.WebsiteDomain
	var isUpdateOtherDomains bool
	var oldDomain, newDomain []string
	if req.OtherDomains != "" && website.PrimaryDomain != req.OtherDomains {
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

		if isDomainChanged(oldDomain, newDomain) {
			isUpdateOtherDomains = true
		}
	}
	tx := global.DB.Begin()
	defer func() {
		if tx != nil {
			tx.Rollback()
		}
	}()
	txCtx := context.WithValue(ctx, constant.DB, tx)
	if err := s.repo.Save(txCtx, &website); err != nil {
		return err
	}
	if website.Type == constant.Proxy {
		for i := range upstreams {
			upstreams[i].WebsiteID = website.ID
		}
		if err := repo.NewWebsiteUpstream().ReplaceByWebsiteID(txCtx, website.ID, upstreams); err != nil {
			return err
		}
	}
	if isUpdateOtherDomains {
		domainRepo := repo.NewWebsiteDomain()
		if err := domainRepo.DeleteByWebsiteIdNotIsPrimary(txCtx, website.ID); err != nil {
			return err
		}
		if err := domainRepo.BatchCreate(txCtx, domains); err != nil {
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
	if err := ApplyCaddyFromDB(txCtx); err != nil {
		return err
	}
	if err := tx.Commit().Error; err != nil {
		return err
	}
	tx = nil
	return nil
}
