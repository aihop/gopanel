/*
 * @Author: Coller
 * @Date: 2022-05-17 12:38:10
 * @LastEditTime: 2024-04-21 17:17:40
 * @Desc: 字符串处理
 */
package stringx

import (
	"encoding/hex"
	"fmt"

	"github.com/duke-git/lancet/v2/convertor"

	"golang.org/x/crypto/scrypt"
)

/**
 * @desc: 根据明文密码和加盐值生成密码
 * @param password
 * @param salt 盐值
 * @return {*}
 */
func GetPassword(password string, salt string) (verify string, err error) {
	var rb []byte
	pw, _ := convertor.ToBytes(password)
	sa, _ := convertor.ToBytes(salt)
	rb, err = scrypt.Key(pw, sa, 16384, 8, 1, 32)
	if err != nil {
		return
	}
	verify = hex.EncodeToString(rb)
	return
}

/**
 *  @Description: 字符串补零
 *  @param str :需要操作的字符串
 *  @param resultLen 结果字符串的长度
 *  @param reverse true 为前置补零，false 为后置补零
 *  @return string
 */
func ZeroFillByStr(str string, resultLen int, reverse bool) string {
	if len(str) > resultLen || resultLen <= 0 {
		return str
	}
	if reverse {
		return fmt.Sprintf("%0*s", resultLen, str) // 不足前置补零
	}
	result := str
	for i := 0; i < resultLen-len(str); i++ {
		result += "0"
	}
	return result
}
