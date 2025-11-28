/*
 * MIT License
 *
 * Copyright (c) 2024 Bamboo
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 * THE SOFTWARE.
 *
 */

package base

import (
	"encoding/base64"
	"errors"
)

// Base64Encrypt 用于加密明文密码
func Base64Encrypt(password string) string {
	return base64.StdEncoding.EncodeToString([]byte(password))
}

// Base64Decrypt 用于解密加密后的密码
func Base64Decrypt(encryptedPassword string) (string, error) {
	decoded, err := base64.StdEncoding.DecodeString(encryptedPassword)
	if err != nil {
		return "", errors.New("解密失败: " + err.Error())
	}
	return string(decoded), nil
}

// Base64EncryptWithMagic 加密
// 通过先添加特定盐值、反转字符串再进行base64编码
func Base64EncryptWithMagic(password string) string {
	const salt = "CloudOps@2024#Security!"
	// 添加盐值并反转字符串
	reversed := reverseString(password + salt)
	// 多次编码增加复杂度
	encoded := base64.StdEncoding.EncodeToString([]byte(reversed))
	encoded = base64.StdEncoding.EncodeToString([]byte(encoded + salt))
	return encoded
}

// Base64DecryptWithMagic 解密
// 与加密过程相反的步骤还原原始密码
func Base64DecryptWithMagic(encryptedPassword string) (string, error) {
	const salt = "CloudOps@2024#Security!"
	// 第一次解码
	decoded, err := base64.StdEncoding.DecodeString(encryptedPassword)
	if err != nil {
		return "", errors.New("解密失败: " + err.Error())
	}

	// 移除盐值
	decodedStr := string(decoded)
	if len(decodedStr) <= len(salt) {
		return "", errors.New("解密失败: 无效的加密数据")
	}
	decodedStr = decodedStr[:len(decodedStr)-len(salt)]

	// 第二次解码
	finalDecoded, err := base64.StdEncoding.DecodeString(decodedStr)
	if err != nil {
		return "", errors.New("解密失败: " + err.Error())
	}

	// 反转并移除盐值
	reversed := reverseString(string(finalDecoded))
	if len(reversed) <= len(salt) {
		return "", errors.New("解密失败: 无效的加密数据")
	}

	return reversed[:len(reversed)-len(salt)], nil
}

// 反转字符串
func reverseString(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}
