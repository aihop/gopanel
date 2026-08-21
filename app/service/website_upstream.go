package service

import (
	"net/url"
	"strings"

	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/app/dto/response"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/buserr"
	"github.com/aihop/gopanel/constant"
)

func normalizeWebsiteUpstreamScheme(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "https":
		return "https"
	default:
		return "http"
	}
}

func normalizeWebsiteUpstreamTransport(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "fastcgi":
		return "fastcgi"
	case "unix":
		return "unix"
	case "https":
		return "https"
	default:
		return "http"
	}
}

func parseLegacyWebsiteProxy(proxy string) (request.WebsiteUpstream, bool) {
	proxy = strings.TrimSpace(proxy)
	if proxy == "" {
		return request.WebsiteUpstream{}, false
	}
	if strings.HasPrefix(proxy, "unix//") {
		return request.WebsiteUpstream{Address: strings.TrimPrefix(proxy, "unix//"), Transport: "unix"}, true
	}
	if strings.HasPrefix(proxy, "http://") || strings.HasPrefix(proxy, "https://") {
		u, err := url.Parse(proxy)
		if err == nil && u.Host != "" {
			return request.WebsiteUpstream{
				Address: u.Host,
				Scheme:  normalizeWebsiteUpstreamScheme(u.Scheme),
			}, true
		}
	}
	return request.WebsiteUpstream{Address: proxy, Scheme: "http"}, true
}

func normalizeWebsiteUpstreams(items []request.WebsiteUpstream, legacyProxy string) ([]model.WebsiteUpstream, error) {
	if len(items) == 0 {
		if fallback, ok := parseLegacyWebsiteProxy(legacyProxy); ok {
			items = []request.WebsiteUpstream{fallback}
		}
	}
	result := make([]model.WebsiteUpstream, 0, len(items))
	for index, item := range items {
		address := strings.TrimSpace(item.Address)
		if address == "" {
			continue
		}
		upstream := model.WebsiteUpstream{
			Address:        address,
			Scheme:         normalizeWebsiteUpstreamScheme(item.Scheme),
			Weight:         item.Weight,
			Enabled:        item.Enabled,
			Backup:         item.Backup,
			HealthURI:      strings.TrimSpace(item.HealthURI),
			HealthInterval: strings.TrimSpace(item.HealthInterval),
			HealthTimeout:  strings.TrimSpace(item.HealthTimeout),
			Transport:      normalizeWebsiteUpstreamTransport(item.Transport),
			Sort:           index,
		}
		if upstream.Weight <= 0 {
			upstream.Weight = 1
		}
		if !item.Enabled {
			upstream.Enabled = false
		} else {
			upstream.Enabled = true
		}
		result = append(result, upstream)
	}
	return result, nil
}

func ensureWebsiteProxyUpstreams(items []request.WebsiteUpstream, legacyProxy string) ([]model.WebsiteUpstream, error) {
	upstreams, err := normalizeWebsiteUpstreams(items, legacyProxy)
	if err != nil {
		return nil, err
	}
	if len(upstreams) == 0 {
		return nil, buserr.New(constant.ErrWebsiteUpstreamAtLeastOne)
	}
	for _, item := range upstreams {
		if item.Enabled {
			return upstreams, nil
		}
	}

	return nil, buserr.New(constant.ErrWebsiteUpstreamAtLeastOneEnabled)
}
func buildWebsiteUpstreamDial(item model.WebsiteUpstream) string {
	address := strings.TrimSpace(item.Address)
	if address == "" {
		return ""
	}
	switch normalizeWebsiteUpstreamTransport(item.Transport) {
	case "unix":
		if strings.HasPrefix(address, "unix//") {
			return address
		}
		if strings.HasPrefix(address, "/") {
			return "unix//" + address
		}
		return "unix//" + address
	case "https":
		if strings.HasPrefix(address, "https://") {
			return address
		}
		if strings.HasPrefix(address, "http://") {
			return "https://" + strings.TrimPrefix(address, "http://")
		}
		return "https://" + address
	default:
		if strings.HasPrefix(address, "http://") || strings.HasPrefix(address, "https://") {
			return address
		}
		if normalizeWebsiteUpstreamScheme(item.Scheme) == "https" {
			return "https://" + address
		}
		return address
	}
}

func websiteProxyFromUpstreams(items []model.WebsiteUpstream, fallback string) string {
	for _, item := range items {
		if !item.Enabled {
			continue
		}
		if dial := buildWebsiteUpstreamDial(item); dial != "" {
			return dial
		}
	}
	if fallbackItem, ok := parseLegacyWebsiteProxy(fallback); ok {
		return buildWebsiteUpstreamDial(model.WebsiteUpstream{
			Address:   fallbackItem.Address,
			Scheme:    fallbackItem.Scheme,
			Transport: fallbackItem.Transport,
			Enabled:   true,
		})
	}
	return strings.TrimSpace(fallback)
}

func responseWebsiteUpstreams(items []*model.WebsiteUpstream, fallback string) []response.WebsiteUpstream {
	result := make([]response.WebsiteUpstream, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		result = append(result, response.WebsiteUpstream{
			ID:             item.ID,
			CreatedAt:      item.CreatedAt,
			UpdatedAt:      item.UpdatedAt,
			Address:        item.Address,
			Scheme:         item.Scheme,
			Weight:         item.Weight,
			Enabled:        item.Enabled,
			Backup:         item.Backup,
			HealthURI:      item.HealthURI,
			HealthInterval: item.HealthInterval,
			HealthTimeout:  item.HealthTimeout,
			Transport:      item.Transport,
			Sort:           item.Sort,
		})
	}
	if len(result) > 0 {
		return result
	}
	if fallbackItem, ok := parseLegacyWebsiteProxy(fallback); ok {
		return []response.WebsiteUpstream{{
			Address:   fallbackItem.Address,
			Scheme:    normalizeWebsiteUpstreamScheme(fallbackItem.Scheme),
			Weight:    1,
			Enabled:   true,
			Transport: normalizeWebsiteUpstreamTransport(fallbackItem.Transport),
		}}
	}
	return []response.WebsiteUpstream{}
}

func reverseProxyDialList(website model.Website) []string {
	var dials []string
	for _, item := range website.Upstreams {
		if item == nil || !item.Enabled {
			continue
		}
		if dial := buildWebsiteUpstreamDial(*item); dial != "" {
			dials = append(dials, dial)
		}
	}
	if len(dials) > 0 {
		return dials
	}
	if proxy := strings.TrimSpace(website.Proxy); proxy != "" {
		return []string{proxy}
	}
	return nil
}

func reverseProxyHealthSettings(website model.Website) (uri, interval, timeout string) {
	for _, item := range website.Upstreams {
		if item == nil || !item.Enabled {
			continue
		}
		if uri == "" {
			uri = strings.TrimSpace(item.HealthURI)
		}
		if interval == "" {
			interval = strings.TrimSpace(item.HealthInterval)
		}
		if timeout == "" {
			timeout = strings.TrimSpace(item.HealthTimeout)
		}
	}
	return
}
