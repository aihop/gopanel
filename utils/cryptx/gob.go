/*
 * @Author: Coller
 * @Date: 2022-01-22 19:21:29
 * @LastEditTime: 2024-01-05 11:43:56
 * @Desc: 加密解密
 */
package cryptx

import (
	"bytes"
	"encoding/gob"
)

/**
 * @desc: 解码
 * @return {*}
 */
func Decode(value string, r interface{}) error {
	network := bytes.NewBuffer([]byte(value))
	dec := gob.NewDecoder(network)
	return dec.Decode(r)
}

/**
 * @desc: 编码
 * @param undefined
 * @return {*}
 */
func Encode(value interface{}) (string, error) {
	network := bytes.NewBuffer(nil)
	enc := gob.NewEncoder(network)
	err := enc.Encode(value)
	if err != nil {
		return "", err
	}
	return network.String(), nil
}
