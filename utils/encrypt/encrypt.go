package encrypt

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
)

func StringEncrypt(text string) (string, error) {
	if len(text) == 0 {
		return "", nil
	}
	if len(global.CONF.System.EncryptKey) == 0 {
		var encryptSetting model.Setting
		if err := global.DB.Where("key = ?", "EncryptKey").First(&encryptSetting).Error; err != nil {
			return "", err
		}
		global.CONF.System.EncryptKey = encryptSetting.Value
	}
	key := global.CONF.System.EncryptKey
	pass := []byte(text)
	xpass, err := aesEncryptWithSalt([]byte(key), pass)
	if err == nil {
		pass64 := base64.StdEncoding.EncodeToString(xpass)
		return pass64, err
	}
	return "", err
}

func StringDecrypt(text string) (string, error) {
	if len(text) == 0 {
		return "", nil
	}
	if len(global.CONF.System.EncryptKey) == 0 {
		var encryptSetting model.Setting
		if err := global.DB.Where("key = ?", "EncryptKey").First(&encryptSetting).Error; err != nil {
			return "", err
		}
		global.CONF.System.EncryptKey = encryptSetting.Value
	}
	key := global.CONF.System.EncryptKey
	bytesPass, err := base64.StdEncoding.DecodeString(text)
	if err != nil {
		return "", err
	}
	var tpass []byte
	tpass, err = aesDecryptWithSalt([]byte(key), bytesPass)
	if err == nil {
		result := string(tpass[:])
		return result, err
	}
	return "", err
}

func padding(plaintext []byte, blockSize int) []byte {
	padding := blockSize - len(plaintext)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(plaintext, padtext...)
}

// unPadding 去除 PKCS#7 填充。
// 密文用错误的 key 解出来是随机字节，末位可能是任何值——不校验直接切片
// 会 panic（实际发生过：节点令牌换 key 后残留旧密文，定时采集一解密就把
// 整个面板进程打崩，每分钟一次）。解密坏数据只允许返回错误，不允许 panic。
func unPadding(origData []byte) ([]byte, error) {
	length := len(origData)
	if length == 0 {
		return nil, errors.New("解密结果为空，无法去除填充")
	}
	unpadding := int(origData[length-1])
	if unpadding <= 0 || unpadding > length {
		return nil, errors.New("填充字节非法，密文可能已损坏或加密密钥不匹配")
	}
	return origData[:(length - unpadding)], nil
}

func aesEncryptWithSalt(key, plaintext []byte) ([]byte, error) {
	plaintext = padding(plaintext, aes.BlockSize)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	ciphertext := make([]byte, aes.BlockSize+len(plaintext))
	iv := ciphertext[0:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}
	cbc := cipher.NewCBCEncrypter(block, iv)
	cbc.CryptBlocks(ciphertext[aes.BlockSize:], plaintext)
	return ciphertext, nil
}
func aesDecryptWithSalt(key, ciphertext []byte) ([]byte, error) {
	var block cipher.Block
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	if len(ciphertext) < aes.BlockSize {
		return nil, fmt.Errorf("iciphertext too short")
	}
	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]
	// CryptBlocks 遇到非整块长度会 panic，截断的密文必须在这里挡下
	if len(ciphertext)%aes.BlockSize != 0 {
		return nil, errors.New("密文长度不是块大小的整数倍，可能已损坏")
	}
	cbc := cipher.NewCBCDecrypter(block, iv)
	cbc.CryptBlocks(ciphertext, ciphertext)
	return unPadding(ciphertext)
}

func ParseRSAPrivateKey(privateKeyPEM string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(privateKeyPEM))
	if block == nil || block.Type != "RSA PRIVATE KEY" {
		return nil, errors.New("failed to decode PEM block containing the private key")
	}
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	return privateKey, nil
}

func aesDecrypt(ciphertext, key, iv []byte) ([]byte, error) {
	if len(key) != 16 && len(key) != 24 && len(key) != 32 {
		return nil, errors.New("invalid AES key length: must be 16, 24, or 32 bytes")
	}
	if len(iv) != aes.BlockSize {
		return nil, errors.New("invalid IV length: must be 16 bytes")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	if len(ciphertext)%aes.BlockSize != 0 {
		return nil, errors.New("密文长度不是块大小的整数倍，可能已损坏")
	}
	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(ciphertext, ciphertext)
	return unPadding(ciphertext)
}

func DecryptPassword(encryptedData string, privateKey *rsa.PrivateKey) (string, error) {
	parts := strings.Split(encryptedData, ":")
	if len(parts) != 3 {
		return "", errors.New("encrypted data format error")
	}
	keyCipher := parts[0]
	ivBase64 := parts[1]
	ciphertextBase64 := parts[2]

	encryptedAESKey, err := base64.StdEncoding.DecodeString(keyCipher)
	if err != nil {
		return "", errors.New("failed to decode keyCipher")
	}

	aesKey, err := rsa.DecryptPKCS1v15(rand.Reader, privateKey, encryptedAESKey)
	if err != nil {
		return "", errors.New("failed to decode AES Key")
	}

	ciphertext, err := base64.StdEncoding.DecodeString(ciphertextBase64)
	if err != nil {
		return "", errors.New("failed to decrypt the encrypted data")
	}
	iv, err := base64.StdEncoding.DecodeString(ivBase64)
	if err != nil {
		return "", errors.New("failed to decode the IV")
	}

	password, err := aesDecrypt(ciphertext, aesKey, iv)
	if err != nil {
		return "", err
	}

	return string(password), nil
}
