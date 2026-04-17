/*
 * @Author: Coller
 * @Date: 2021-09-24 12:30:08
 * @LastEditTime: 2024-04-21 14:37:44
 * @Desc: 密码验证
 */
package cryptx

import (
	"fmt"

	"github.com/duke-git/lancet/v2/convertor"
	"golang.org/x/crypto/bcrypt"
)

/**
 * @desc: 密码加密
 * @param 加密的字符串
 * @return {*}
 */
func EncodePassword(rawPassword string) string {
	bytePassword, _ := convertor.ToBytes(rawPassword)
	hash, err := bcrypt.GenerateFromPassword(bytePassword, bcrypt.DefaultCost)
	if err != nil {
		fmt.Println(err)
	}
	return convertor.ToString(hash)
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
