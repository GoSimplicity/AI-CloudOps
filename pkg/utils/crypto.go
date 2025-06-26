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
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"

	"golang.org/x/crypto/pbkdf2"
)

// EncryptSecretKey 使用指定密钥加密数据
func EncryptSecretKey(secretKey string, encryptionKey []byte) (string, error) {
	if secretKey == "" {
		return "", fmt.Errorf("密钥不能为空")
	}
	if len(encryptionKey) != 32 {
		return "", fmt.Errorf("加密密钥必须是32字节(256位)")
	}

	// 创建AES cipher
	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return "", fmt.Errorf("创建AES cipher失败: %w", err)
	}

	// 创建GCM模式
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("创建GCM模式失败: %w", err)
	}

	// 生成随机nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("生成随机nonce失败: %w", err)
	}

	// 加密
	ciphertext := gcm.Seal(nonce, nonce, []byte(secretKey), nil)

	// 编码为base64
	encoded := base64.StdEncoding.EncodeToString(ciphertext)

	return encoded, nil
}

// DecryptSecretKey 使用指定密钥解密数据
func DecryptSecretKey(encryptedSecretKey string, encryptionKey []byte) (string, error) {
	if encryptedSecretKey == "" {
		return "", fmt.Errorf("加密密钥不能为空")
	}
	if len(encryptionKey) != 32 {
		return "", fmt.Errorf("加密密钥必须是32字节(256位)")
	}

	// 解码base64
	ciphertext, err := base64.StdEncoding.DecodeString(encryptedSecretKey)
	if err != nil {
		return "", fmt.Errorf("解码base64失败: %w", err)
	}

	// 创建AES cipher
	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return "", fmt.Errorf("创建AES cipher失败: %w", err)
	}

	// 创建GCM模式
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("创建GCM模式失败: %w", err)
	}

	// 检查密文长度
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("密文长度不足")
	}

	// 分离nonce和密文
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	// 解密
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("解密失败: %w", err)
	}

	return string(plaintext), nil
}

// EncryptBatch 批量加密
func EncryptBatch(secretKeys []string, encryptionKey []byte) ([]string, error) {
	if len(secretKeys) == 0 {
		return []string{}, nil
	}
	if len(encryptionKey) != 32 {
		return nil, fmt.Errorf("加密密钥必须是32字节(256位)")
	}

	encryptedKeys := make([]string, len(secretKeys))
	for i, secretKey := range secretKeys {
		encrypted, err := EncryptSecretKey(secretKey, encryptionKey)
		if err != nil {
			return nil, fmt.Errorf("批量加密失败，索引 %d: %w", i, err)
		}
		encryptedKeys[i] = encrypted
	}

	return encryptedKeys, nil
}

// DecryptBatch 批量解密
func DecryptBatch(encryptedSecretKeys []string, encryptionKey []byte) ([]string, error) {
	if len(encryptedSecretKeys) == 0 {
		return []string{}, nil
	}
	if len(encryptionKey) != 32 {
		return nil, fmt.Errorf("加密密钥必须是32字节(256位)")
	}

	decryptedKeys := make([]string, len(encryptedSecretKeys))
	for i, encryptedKey := range encryptedSecretKeys {
		decrypted, err := DecryptSecretKey(encryptedKey, encryptionKey)
		if err != nil {
			return nil, fmt.Errorf("批量解密失败，索引 %d: %w", i, err)
		}
		decryptedKeys[i] = decrypted
	}

	return decryptedKeys, nil
}

// GenerateRandomKey 生成随机密钥
func GenerateRandomKey() ([]byte, error) {
	key := make([]byte, 32) // AES-256
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("生成随机密钥失败: %w", err)
	}
	return key, nil
}

// GenerateRandomSalt 生成随机盐值
func GenerateRandomSalt() ([]byte, error) {
	salt := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("生成随机盐值失败: %w", err)
	}
	return salt, nil
}

// GenerateKeyFromPassword 从密码生成密钥（使用PBKDF2）
func GenerateKeyFromPassword(password string, salt []byte) ([]byte, error) {
	if password == "" {
		return nil, fmt.Errorf("密码不能为空")
	}
	if len(salt) < 16 {
		return nil, fmt.Errorf("盐值长度不足，至少需要16字节")
	}

	// 使用PBKDF2生成密钥，增加迭代次数提高安全性
	const iterations = 100000
	key := pbkdf2.Key([]byte(password), salt, iterations, 32, sha256.New)

	return key, nil
}

// ValidateEncryptedData 验证加密数据格式
func ValidateEncryptedData(encryptedData string) error {
	if encryptedData == "" {
		return fmt.Errorf("加密数据不能为空")
	}

	// 尝试解码base64
	ciphertext, err := base64.StdEncoding.DecodeString(encryptedData)
	if err != nil {
		return fmt.Errorf("无效的base64编码: %w", err)
	}

	// 检查最小长度（nonce + 至少1字节密文 + GCM标签16字节）
	if len(ciphertext) < 12+1+16 {
		return fmt.Errorf("加密数据长度不足")
	}

	return nil
}

// SecureZeroMemory 安全清零内存
func SecureZeroMemory(data []byte) {
	for i := range data {
		data[i] = 0
	}
}
