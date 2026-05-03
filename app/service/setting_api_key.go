package service

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/common"
)

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
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: privateKeyBytes})
	return string(privateKeyPEM)
}
func exportPublicKeyToPEM(publicKey *rsa.PublicKey) (string, error) {
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return "", err
	}
	publicKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicKeyBytes})
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
