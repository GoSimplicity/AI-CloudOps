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

package utils

import (
	"errors"

	pkgUtils "github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"github.com/spf13/viper"
)

// EncryptPassword 加密密码
// 使用配置文件中的加密密钥对密码进行加密
func EncryptPassword(password string) (string, error) {
	if password == "" {
		return "", nil
	}

	encryptionKey := viper.GetString("tree.password_encryption_key")
	if encryptionKey == "" {
		return "", errors.New("未配置密码加密密钥")
	}
	if len(encryptionKey) != 32 {
		return "", errors.New("密码加密密钥长度必须为32字节")
	}

	return pkgUtils.EncryptSecretKey(password, []byte(encryptionKey))
}

// DecryptPassword 解密密码
// 使用配置文件中的加密密钥对加密后的密码进行解密
func DecryptPassword(encryptedPassword string) (string, error) {
	if encryptedPassword == "" {
		return "", nil
	}

	encryptionKey := viper.GetString("tree.password_encryption_key")
	if encryptionKey == "" {
		return "", errors.New("未配置密码加密密钥")
	}
	if len(encryptionKey) != 32 {
		return "", errors.New("密码加密密钥长度必须为32字节")
	}

	return pkgUtils.DecryptSecretKey(encryptedPassword, []byte(encryptionKey))
}
