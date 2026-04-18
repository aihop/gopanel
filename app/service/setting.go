package service

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/buserr"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/cmd"
	"github.com/aihop/gopanel/utils/common"
	"github.com/aihop/gopanel/utils/encrypt"
	"github.com/robfig/cron/v3"
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

func (u *SettingService) LoadFromCert() (*dto.SSLInfo, error) {
	ssl, err := settingRepo.Get(settingRepo.WithByKey("SSL"))
	if err != nil {
		return nil, err
	}
	if ssl.Value == "disable" {
		return &dto.SSLInfo{}, nil
	}
	sslType, err := settingRepo.Get(settingRepo.WithByKey("SSLType"))
	if err != nil {
		return nil, err
	}
	var data dto.SSLInfo
	switch sslType.Value {
	case "self":
		data, err = loadInfoFromCert()
		if err != nil {
			return nil, err
		}
	case "import-paste", "import-local":
		data, err = loadInfoFromCert()
		if err != nil {
			return nil, err
		}
		if _, err := os.Stat(path.Join(global.CONF.System.BaseDir, "gopanel/secret/server.crt")); err != nil {
			return nil, fmt.Errorf("load server.crt file failed, err: %v", err)
		}
		certFile, _ := os.ReadFile(path.Join(global.CONF.System.BaseDir, "gopanel/secret/server.crt"))
		data.Cert = string(certFile)

		if _, err := os.Stat(path.Join(global.CONF.System.BaseDir, "gopanel/secret/server.key")); err != nil {
			return nil, fmt.Errorf("load server.key file failed, err: %v", err)
		}
		keyFile, _ := os.ReadFile(path.Join(global.CONF.System.BaseDir, "gopanel/secret/server.key"))
		data.Key = string(keyFile)
	case "select":
		sslID, err := settingRepo.Get(settingRepo.WithByKey("SSLID"))
		if err != nil {
			return nil, err
		}
		id, _ := strconv.Atoi((sslID.Value))
		ssl, err := repo.NewSSL().GetFirst(commonRepo.WithByID(uint(id)))
		if err != nil {
			return nil, err
		}
		data.Domain = ssl.PrimaryDomain
		data.SSLID = uint(id)
		data.Timeout = ssl.ExpireDate.Format(constant.DateTimeLayout)
	}
	return &data, nil
}

func loadInfoFromCert() (dto.SSLInfo, error) {
	var info dto.SSLInfo
	certFile := path.Join(global.CONF.System.BaseDir, "gopanel/secret/server.crt")
	if _, err := os.Stat(certFile); err != nil {
		return info, err
	}
	certData, err := os.ReadFile(certFile)
	if err != nil {
		return info, err
	}
	certBlock, _ := pem.Decode(certData)
	if certBlock == nil {
		return info, err
	}
	certObj, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return info, err
	}
	var domains []string
	if len(certObj.IPAddresses) != 0 {
		for _, ip := range certObj.IPAddresses {
			domains = append(domains, ip.String())
		}
	}
	if len(certObj.DNSNames) != 0 {
		domains = append(domains, certObj.DNSNames...)
	}
	return dto.SSLInfo{
		Domain:   strings.Join(domains, ","),
		Timeout:  certObj.NotAfter.Format(constant.DateTimeLayout),
		RootPath: path.Join(global.CONF.System.BaseDir, "console/secret/server.crt"),
	}, nil
}

func checkCertValid() error {
	certificate, err := os.ReadFile(path.Join(global.CONF.System.BaseDir, "gopanel/secret/server.crt.tmp"))
	if err != nil {
		return err
	}
	key, err := os.ReadFile(path.Join(global.CONF.System.BaseDir, "gopanel/secret/server.key.tmp"))
	if err != nil {
		return err
	}
	if _, err = tls.X509KeyPair(certificate, key); err != nil {
		return err
	}
	certBlock, _ := pem.Decode(certificate)
	if certBlock == nil {
		return err
	}
	if _, err := x509.ParseCertificate(certBlock.Bytes); err != nil {
		return err
	}

	return nil
}

func (u *SettingService) GenerateApiKey() (string, error) {
	apiKey := common.RandStr(32)
	if err := settingRepo.Update("ApiKey", apiKey); err != nil {
		return global.CONF.System.ApiKey, err
	}
	global.CONF.System.ApiKey = apiKey
	return apiKey, nil
}

func (u *SettingService) UpdateApiConfig(req dto.ApiInterfaceConfig) error {
	if err := settingRepo.Update("ApiInterfaceStatus", req.ApiInterfaceStatus); err != nil {
		return err
	}
	global.CONF.System.ApiInterfaceStatus = req.ApiInterfaceStatus
	if err := settingRepo.Update("ApiKey", req.ApiKey); err != nil {
		return err
	}
	global.CONF.System.ApiKey = req.ApiKey
	if err := settingRepo.Update("IpWhiteList", req.IpWhiteList); err != nil {
		return err
	}
	global.CONF.System.IpWhiteList = req.IpWhiteList
	if err := settingRepo.Update("ApiKeyValidityTime", req.ApiKeyValidityTime); err != nil {
		return err
	}
	global.CONF.System.ApiKeyValidityTime = req.ApiKeyValidityTime
	return nil
}

func exportPrivateKeyToPEM(privateKey *rsa.PrivateKey) string {
	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	})
	return string(privateKeyPEM)
}

func exportPublicKeyToPEM(publicKey *rsa.PublicKey) (string, error) {
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return "", err
	}
	publicKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	})
	return string(publicKeyPEM), nil
}

func (u *SettingService) GenerateRSAKey() error {
	priKey, _ := settingRepo.Get(settingRepo.WithByKey("PASSWORD_PRIVATE_KEY"))
	pubKey, _ := settingRepo.Get(settingRepo.WithByKey("PASSWORD_PUBLIC_KEY"))
	if priKey.Value != "" && pubKey.Value != "" {
		return nil
	}
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}
	privateKeyPEM := exportPrivateKeyToPEM(privateKey)
	publicKeyPEM, err := exportPublicKeyToPEM(&privateKey.PublicKey)
	err = settingRepo.UpdateOrCreate("PASSWORD_PRIVATE_KEY", privateKeyPEM)
	if err != nil {
		return err
	}
	err = settingRepo.UpdateOrCreate("PASSWORD_PUBLIC_KEY", publicKeyPEM)
	if err != nil {
		return err
	}
	return nil
}
