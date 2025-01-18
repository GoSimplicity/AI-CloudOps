package utils

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
