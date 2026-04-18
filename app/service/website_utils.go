package service

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/buserr"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/utils/common"
)

func getDomain(domainStr string, defaultPort int) (model.WebsiteDomain, error) {
	var (
		err    error
		domain = model.WebsiteDomain{}
		portN  int
	)
	domainArray := strings.Split(domainStr, ":")
	if len(domainArray) == 1 {
		domain.Domain, err = handleChineseDomain(domainArray[0])
		if err != nil {
			return domain, err
		}
		domain.Port = defaultPort
		return domain, nil
	}
	if len(domainArray) > 1 {
		domain.Domain, err = handleChineseDomain(domainArray[0])
		if err != nil {
			return domain, err
		}
		portStr := domainArray[1]
		portN, err = strconv.Atoi(portStr)
		if err != nil {
			return domain, buserr.WithName("ErrTypePort", portStr)
		}
		if portN <= 0 || portN > 65535 {
			return domain, buserr.New("ErrTypePortRange")
		}
		domain.Port = portN
		return domain, nil
	}
	return domain, nil
}

func handleChineseDomain(domain string) (string, error) {
	if common.ContainsChinese(domain) {
		return common.PunycodeEncode(domain)
	}
	return domain, nil
}

func getWebsiteDomains(domains string, defaultPort int, websiteID uint) (domainModels []model.WebsiteDomain, addPorts []int, addDomains []string, err error) {
	var (
		ports = make(map[int]struct{})
	)
	domainArray := strings.Split(domains, "\n")
	websiteDomainRepo := repo.NewWebsiteDomain()
	websiteRepo := repo.NewWebsite()
	for _, domain := range domainArray {
		domain = strings.TrimSpace(domain)
		if domain == "" {
			continue
		}
		if strings.HasPrefix(domain, "https://") {
			err = errors.New("域名仅支持 http:// 前缀，不支持 https:// 前缀")
			return
		}
		domain = strings.TrimPrefix(domain, "http://")
		if !common.IsValidDomain(domain) {
			err = errors.New("ErrDomainFormat" + domain)
			return
		}
		var domainModel model.WebsiteDomain
		domainModel, err = getDomain(domain, defaultPort)
		if err != nil {
			return
		}
		if reflect.DeepEqual(domainModel, model.WebsiteDomain{}) {
			continue
		}
		domainModel.WebsiteID = websiteID
		domainModels = append(domainModels, domainModel)
		if domainModel.Port != defaultPort {
			ports[domainModel.Port] = struct{}{}
		}
		if exist, _ := websiteDomainRepo.GetFirst(websiteDomainRepo.WithDomain(domainModel.Domain), websiteDomainRepo.WithWebsiteId(websiteID)); exist.ID == 0 {
			addDomains = append(addDomains, domainModel.Domain)
		}
	}
	for _, domain := range domainModels {
		if exist, _ := websiteDomainRepo.GetFirst(websiteDomainRepo.WithDomain(domain.Domain), websiteDomainRepo.WithPort(domain.Port)); exist.ID > 0 {
			website, _ := websiteRepo.GetFirst(commonRepo.WithByID(exist.WebsiteID))
			err = errors.New(constant.ErrDomainIsUsed + website.PrimaryDomain)
			return
		}
	}

	for port := range ports {
		if existPorts, _ := websiteDomainRepo.GetBy(websiteDomainRepo.WithPort(port)); len(existPorts) == 0 {
			errMap := make(map[string]interface{})
			errMap["port"] = port
			appInstall, _ := appInstallRepo.GetFirst(appInstallRepo.WithPort(port))
			if appInstall.ID > 0 {
				errMap["type"] = "TYPE_APP"
				errMap["name"] = appInstall.Name
				err = errors.New("ErrPortExist" + fmt.Sprintf("%v", errMap))
				return
			}
			if common.ScanPort(port) {
				err = errors.New("ErrPortInUsed" + fmt.Sprintf("%v", port))
				return
			}
		}
		if existPorts, _ := websiteDomainRepo.GetBy(websiteDomainRepo.WithWebsiteId(websiteID), websiteDomainRepo.WithPort(port)); len(existPorts) == 0 {
			addPorts = append(addPorts, port)
		}
	}

	return
}

func isDomainChanged(newDomain, oldDomain []string) bool {
	if len(newDomain) != len(oldDomain) {
		return true
	}
	// 转为 map，去重
	newMap := make(map[string]struct{}, len(newDomain))
	oldMap := make(map[string]struct{}, len(oldDomain))
	for _, d := range newDomain {
		newMap[d] = struct{}{}
	}
	for _, d := range oldDomain {
		oldMap[d] = struct{}{}
	}
	// 比较内容
	for d := range newMap {
		if _, ok := oldMap[d]; !ok {
			return true
		}
	}
	for d := range oldMap {
		if _, ok := newMap[d]; !ok {
			return true
		}
	}
	return false
}
