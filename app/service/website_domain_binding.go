package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/buserr"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
)

func (s WebsiteService) UpdateDomainBindings(
	ctx context.Context,
	websiteID uint,
	primaryDomain string,
	otherDomains string,
	redirectDomainsToPrimary bool,
) error {
	website, err := s.repo.GetFirst(commonRepo.WithByID(websiteID))
	if err != nil {
		return buserr.New(constant.ErrWebsiteNotFound)
	}

	primaryDomain, otherDomains = normalizeWebsiteBindingInput(primaryDomain, otherDomains)
	if primaryDomain == "" {
		return buserr.New(constant.ErrWebsiteDomainPrimaryRequired)
	}
	domains, _, _, err := getWebsiteDomains(strings.Join(nonEmptyStrings(primaryDomain, otherDomains), "\n"), 80, website.ID)
	if err != nil {
		return err
	}
	if len(domains) == 0 {
		return buserr.New(constant.ErrWebsiteDomainAtLeastOne)
	}

	website.PrimaryDomain = formatWebsiteBindingDomain(domains[0])
	website.RedirectDomainsToPrimary = redirectDomainsToPrimary
	for index := range domains {
		domains[index].WebsiteID = website.ID
		if index == 0 {
			domains[index].IsPrimary = 20
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
	domainRepo := repo.NewWebsiteDomain()
	if err := domainRepo.DeleteByWebsiteId(txCtx, website.ID); err != nil {
		return err
	}
	if err := domainRepo.BatchCreate(txCtx, domains); err != nil {
		return err
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

func normalizeWebsiteBindingInput(primaryDomain, otherDomains string) (string, string) {
	primary := normalizeWebsiteDomainForCompare(primaryDomain)
	lines := strings.FieldsFunc(strings.ReplaceAll(otherDomains, ",", "\n"), func(char rune) bool {
		return char == '\n' || char == '\r'
	})
	seen := map[string]struct{}{strings.ToLower(primary): {}}
	normalized := make([]string, 0, len(lines))
	for _, line := range lines {
		domain := normalizeWebsiteDomainForCompare(line)
		key := strings.ToLower(domain)
		if domain == "" {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		normalized = append(normalized, domain)
	}
	return primary, strings.Join(normalized, "\n")
}

func nonEmptyStrings(values ...string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			result = append(result, value)
		}
	}
	return result
}

func formatWebsiteBindingDomain(domain model.WebsiteDomain) string {
	if domain.Port == 80 {
		return domain.Domain
	}
	return fmt.Sprintf("%s:%d", domain.Domain, domain.Port)
}
