package cryptx

import (
	"github.com/aihop/gopanel/utils/convertor"
	"golang.org/x/crypto/bcrypt"
)

/**
 * @desc: 密码加密
 * @param 加密的字符串
 * @return {*}
 */
func EncodePassword(rawPassword string) (string, error) {
	bytePassword, _ := convertor.ToBytes(rawPassword)
	hash, err := bcrypt.GenerateFromPassword(bytePassword, bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return convertor.ToString(hash), nil
}

/**
 * @desc: 密码验证
 * @param 验证的密码
 * @param 输入的密码
 * @return {*}
 */
func ValidatePassword(encodePassword, inputPassword string) bool {
	byteEncodePassword, _ := convertor.ToBytes(encodePassword)
	byteInputPassword, _ := convertor.ToBytes(inputPassword)
	err := bcrypt.CompareHashAndPassword(byteEncodePassword, byteInputPassword)
	return err == nil
}
