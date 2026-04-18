package cryptx

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
	"strings"

	"github.com/aihop/gopanel/global"
)

func getSKey() string {
	if len(global.CONF.System.EncryptKey) >= 16 {
		return global.CONF.System.EncryptKey[:16]
	}
	// 如果没有配置足够强度的 EncryptKey，系统直接抛出 panic 拒绝启动或加解密
	// 这样可以逼迫用户在安装阶段必须生成安全的 16 位随机密钥，而不是回退到一个已公开的硬编码密钥
	panic("Security Error: global.CONF.System.EncryptKey is missing or less than 16 characters. Please configure a secure encrypt key in conf.yaml")
}

func pkcs7Padding(data []byte, blockSize int) []byte {
	//判断缺少几位长度。最少1，最多 blockSize
	padding := blockSize - len(data)%blockSize
	//补足位数。把切片[]byte{byte(padding)}复制padding个
	padText := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(data, padText...)
}

// pkcs7UnPadding 填充的反向操作
func pkcs7UnPadding(data []byte) ([]byte, error) {
	length := len(data)
	if length == 0 {
		return nil, errors.New("strings error")
	}
	//获取填充的个数
	unPadding := int(data[length-1])
	if unPadding > length {
		return nil, errors.New("strings error")
	}
	return data[:(length - unPadding)], nil
}

func validateLength(data []byte, blockSize int) error {
	if len(data)%blockSize != 0 {
		return fmt.Errorf("data length must be a multiple of block size (%d)", blockSize)
	}
	return nil
}

// AesEncrypt 加密
func AesEncrypt(str, iKey string) (d string, err error) {
	if str == "" {
		return "", errors.New("aes encrypted content cannot ne empty")
	}
	if iKey == "" {
		iKey = getSKey()
	}
	data := []byte(str)
	key := []byte(iKey)
	//创建加密实例
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	//判断加密块的大小
	blockSize := block.BlockSize()
	//填充
	encryptBytes := pkcs7Padding(data, blockSize)
	//初始化加密数据接收切片
	crypted := make([]byte, len(encryptBytes))
	//使用cbc加密模式
	blockMode := cipher.NewCBCEncrypter(block, key[:blockSize])
	//执行加密
	blockMode.CryptBlocks(crypted, encryptBytes)
	datas := base64.StdEncoding.EncodeToString(crypted)
	datas = strings.Replace(datas, "+", "-", -1)
	datas = strings.Replace(datas, "/", "_", -1)
	return datas, nil
}

// AesDecrypt 解密
func AesDecrypt(str string, iKey string) (string, error) {
	if str == "" {
		return "", errors.New("aes decrypted content cannot be empty")
	}
	str = strings.Replace(str, "-", "+", -1)
	str = strings.Replace(str, "_", "/", -1)
	if iKey == "" {
		iKey = getSKey()
	}
	n, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		return "", err
	}
	data := []byte(n)
	key := []byte(iKey)
	//创建实例
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	//获取块的大小
	blockSize := block.BlockSize()
	// 校验数据长度
	if err := validateLength(data, blockSize); err != nil {
		return "", err
	}
	//使用cbc
	blockMode := cipher.NewCBCDecrypter(block, key[:blockSize])
	//初始化解密数据接收切片
	crypted := make([]byte, len(data))
	//执行解密
	blockMode.CryptBlocks(crypted, data)
	//去除填充
	crypted, err = pkcs7UnPadding(crypted)
	if err != nil {
		return "", err
	}
	return string(crypted), nil
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
	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(ciphertext, ciphertext)
	ciphertext = pkcs7Unpad(ciphertext)
	return ciphertext, nil
}

func pkcs7Unpad(data []byte) []byte {
	length := len(data)
	padLength := int(data[length-1])
	return data[:length-padLength]
}
