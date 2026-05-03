package service

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"os"
	"path"
	"strconv"
	"strings"
)

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
	return dto.SSLInfo{Domain: strings.Join(domains, ","), Timeout: certObj.NotAfter.Format(constant.DateTimeLayout), RootPath: path.Join(global.CONF.System.BaseDir, "console/secret/server.crt")}, nil
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
