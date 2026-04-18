package token

import (
	"errors"
	"time"

	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/cryptx"
	"github.com/duke-git/lancet/v2/convertor"

	"github.com/golang-jwt/jwt/v5"
)

type CustomClaims struct {
	Role        string
	UserId      uint
	StoreId     uint
	SaltId      string
	FileBaseDir string
	jwt.RegisteredClaims
}

/**
* @desc: 生成Token
* @param entId 企业ID
* @param userId 用户ID
* @param saltId 盐值
* @param hour 时间
* @return {*}
 */
func Create(userId uint, role, saltId, fileBaseDir string, hour time.Duration) (string, error) {
	claimsData := CustomClaims{
		UserId:      userId,
		Role:        role,
		SaltId:      saltId,
		FileBaseDir: fileBaseDir,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(hour * time.Hour)),
			Issuer:    "",
		},
	}
	tokenClaims := jwt.NewWithClaims(jwt.SigningMethodHS256, &claimsData)
	signKey, _ := convertor.ToBytes(global.CONF.System.EncryptKey)
	token, err := tokenClaims.SignedString(signKey)
	if err != nil {
		return "", err
	}
	return cryptx.AesEncrypt(token, "")
}

func Parse(tokenStr string) (*CustomClaims, error) {
	aesStr, err := cryptx.AesDecrypt(tokenStr, "")
	if err != nil {
		return nil, errors.New("token error")
	}
	token, err := jwt.ParseWithClaims(aesStr, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		signKey, _ := convertor.ToBytes(global.CONF.System.EncryptKey)
		return signKey, nil
	})
	if err != nil || token == nil {
		return nil, errors.New("token error")
	}
	if claims, ok := token.Claims.(*CustomClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, errors.New("token error")
}
