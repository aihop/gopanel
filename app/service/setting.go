package service

import (
	"encoding/json"
	"errors"
	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/buserr"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/cmd"
	"github.com/aihop/gopanel/utils/common"
	"github.com/aihop/gopanel/utils/encrypt"
	"github.com/robfig/cron/v3"
	"net"
	"strconv"
	"time"
)

type SettingService struct{}

func NewSetting() *SettingService {
	return &SettingService{}
}

type ISettingService interface {
	GetInfo() (*dto.SettingInfo, error)
	LoadInterfaceAddr() ([]string, error)
	Update(key, value string) error
	UpdateProxy(req dto.ProxyUpdate) error
	UpdatePort(port uint) error
	UpdateBindInfo(req dto.BindInfo) error
	LoadFromCert() (*dto.SSLInfo, error)
	GenerateApiKey() (string, error)
	UpdateApiConfig(req dto.ApiInterfaceConfig) error
	GenerateRSAKey() error
}

func NewISettingService() ISettingService {
	return &SettingService{}
}
func (u *SettingService) GetInfo() (*dto.SettingInfo, error) {
	setting, err := settingRepo.GetList()
	if err != nil {
		return nil, errors.New(constant.ErrRecordNotFound)
	}
	settingMap := make(map[string]string)
	for _, set := range setting {
		settingMap[set.Key] = set.Value
	}
	var info dto.SettingInfo
	arr, err := json.Marshal(settingMap)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(arr, &info); err != nil {
		return nil, err
	}
	if info.ProxyPasswdKeep != constant.StatusEnable {
		info.ProxyPasswd = ""
	} else {
		info.ProxyPasswd, _ = encrypt.StringDecrypt(info.ProxyPasswd)
	}
	info.LocalTime = time.Now().Format("2006-01-02 15:04:05 MST -0700")
	return &info, err
}
func (u *SettingService) Update(key, value string) error {
	switch key {
	case "MonitorStatus":
		if value == "enable" && global.MonitorCronID == 0 {
			interval, err := settingRepo.Get(settingRepo.WithByKey("MonitorInterval"))
			if err != nil {
				return err
			}
			if err := StartMonitor(false, interval.Value); err != nil {
				return err
			}
		}
		if value == "disable" && global.MonitorCronID != 0 {
			monitorCancel()
			global.Cron.Remove(cron.EntryID(global.MonitorCronID))
			global.MonitorCronID = 0
		}
	case "MonitorInterval":
		status, err := settingRepo.Get(settingRepo.WithByKey("MonitorStatus"))
		if err != nil {
			return err
		}
		if status.Value == "enable" && global.MonitorCronID != 0 {
			if err := StartMonitor(true, value); err != nil {
				return err
			}
		}
	case "AppStoreLastModified":
		exist, _ := settingRepo.Get(settingRepo.WithByKey("AppStoreLastModified"))
		if exist.ID == 0 {
			_ = settingRepo.Create("AppStoreLastModified", value)
			return nil
		}
	}
	if err := settingRepo.Update(key, value); err != nil {
		return err
	}
	switch key {
	case "ExpirationDays":
		timeout, err := strconv.Atoi(value)
		if err != nil {
			return err
		}
		if err := settingRepo.Update("ExpirationTime", time.Now().AddDate(0, 0, timeout).Format(constant.DateTimeLayout)); err != nil {
			return err
		}
	case "BindDomain":
		if len(value) != 0 {
			_ = global.SESSION.Clean()
		}
	case "UserName", "Password":
		_ = global.SESSION.Clean()
	}
	return nil
}
func (u *SettingService) LoadInterfaceAddr() ([]string, error) {
	addrMap := make(map[string]struct{})
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil, err
	}
	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if ok && ipNet.IP.To16() != nil {
			addrMap[ipNet.IP.String()] = struct{}{}
		}
	}
	var data []string
	for key := range addrMap {
		data = append(data, key)
	}
	return data, nil
}
func (u *SettingService) UpdateBindInfo(req dto.BindInfo) error {
	if err := settingRepo.Update("Ipv6", req.Ipv6); err != nil {
		return err
	}
	if err := settingRepo.Update("BindAddress", req.BindAddress); err != nil {
		return err
	}
	go func() {
		time.Sleep(1 * time.Second)
		if err := cmd.RestartGoPanel(); err != nil {
			global.LOG.Errorf("restart system with new bind info failed, err: %v", err)
		}
	}()
	return nil
}
func (u *SettingService) UpdateProxy(req dto.ProxyUpdate) error {
	if err := settingRepo.Update("ProxyUrl", req.ProxyUrl); err != nil {
		return err
	}
	if err := settingRepo.Update("ProxyType", req.ProxyType); err != nil {
		return err
	}
	if err := settingRepo.Update("ProxyPort", req.ProxyPort); err != nil {
		return err
	}
	if err := settingRepo.Update("ProxyUser", req.ProxyUser); err != nil {
		return err
	}
	pass, _ := encrypt.StringEncrypt(req.ProxyPasswd)
	if err := settingRepo.Update("ProxyPasswd", pass); err != nil {
		return err
	}
	if err := settingRepo.Update("ProxyPasswdKeep", req.ProxyPasswdKeep); err != nil {
		return err
	}
	return nil
}
func (u *SettingService) UpdatePort(port uint) error {
	if common.ScanPort(int(port)) {
		return buserr.WithDetail(constant.ErrPortInUsed, port, nil)
	}
	serverPort, err := settingRepo.Get(settingRepo.WithByKey("ServerPort"))
	if err != nil {
		return err
	}
	portValue, _ := strconv.Atoi(serverPort.Value)
	if err := OperateFirewallPort([]int{portValue}, []int{int(port)}); err != nil {
		global.LOG.Errorf("set system firewall ports failed, err: %v", err)
	}
	if err := settingRepo.Update("ServerPort", strconv.Itoa(int(port))); err != nil {
		return err
	}
	go func() {
		time.Sleep(1 * time.Second)
		if err := cmd.RestartGoPanel(); err != nil {
			global.LOG.Errorf("restart system port failed, err: %v", err)
		}
	}()
	return nil
}
