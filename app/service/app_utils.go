package service

import (
	"errors"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/buserr"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/utils/common"
	"math"
	"strconv"
)

var DatabaseKeys = map[string]uint{constant.AppMysql: 3306, constant.AppMariaDB: 3306, constant.AppPostgresql: 5432, constant.AppPostgres: 5432, constant.AppMongodb: 27017, constant.AppRedis: 6379, constant.AppMemcached: 11211}

func checkPort(key string, params map[string]interface{}) (int, error) {
	port, ok := params[key]
	if ok {
		portN := 0
		var err error
		switch p := port.(type) {
		case string:
			portN, err = strconv.Atoi(p)
			if err != nil {
				return portN, nil
			}
		case float64:
			portN = int(math.Ceil(p))
		case int:
			portN = p
		}
		oldInstalled, _ := appInstallRepo.ListBy(appInstallRepo.WithPort(portN))
		if len(oldInstalled) > 0 {
			var apps []string
			for _, install := range oldInstalled {
				apps = append(apps, install.App.Name)
			}
			return portN, buserr.WithMap(constant.ErrPortInOtherApp, map[string]interface{}{"port": portN, "apps": apps}, nil)
		}
		if common.ScanPort(portN) {
			return portN, buserr.WithDetail(constant.ErrPortInUsed, portN, nil)
		} else {
			return portN, nil
		}
	}
	return 0, nil
}
func checkPortExist(port int) error {
	errMap := make(map[string]interface{})
	errMap["port"] = port
	appInstall, _ := appInstallRepo.GetFirst(appInstallRepo.WithPort(port))
	if appInstall.ID > 0 {
		errMap["type"] = "TYPE_APP"
		errMap["name"] = appInstall.Name
		return errors.New(constant.ErrPortInUsed)
	}
	websiteDomainRepo := repo.NewWebsiteDomain()
	domain, _ := websiteDomainRepo.GetFirst(websiteDomainRepo.WithPort(port))
	if domain.ID > 0 {
		errMap["type"] = "TYPE_DOMAIN"
		errMap["name"] = domain.Domain
		return errors.New(constant.ErrPortInUsed)
	}
	if common.ScanPort(port) {
		return errors.New(constant.ErrPortInUsed)
	}
	return nil
}
