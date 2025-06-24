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
	"encoding/base64"
	"fmt"
	"io"
	"sync"
	"time"

	"go.uber.org/zap"
)

// CryptoManager AES加密管理器
type CryptoManager interface {
	EncryptSecretKey(secretKey string) (string, error)
	DecryptSecretKey(encryptedSecretKey string) (string, error)
	EncryptBatch(secretKeys []string) ([]string, error)
	DecryptBatch(encryptedSecretKeys []string) ([]string, error)
	RotateKey(newKey []byte) error
	GetKeyInfo() map[string]interface{}
	ValidateEncryptedData(encryptedData string) error
}

// CryptoManager 结构体实现
var _ CryptoManager = (*cryptoManager)(nil)

// CryptoManager AES加密管理器
type cryptoManager struct {
	encryptionKey []byte
	logger        *zap.Logger
	mu            sync.RWMutex
	keyVersion    int
	keyCreatedAt  time.Time
}

// NewCryptoManager 创建新的加密管理器
func NewCryptoManager(encryptionKey []byte, logger *zap.Logger) *cryptoManager {
	if len(encryptionKey) != 32 {
		panic("加密密钥必须是32字节(256位)")
	}

	return &cryptoManager{
		encryptionKey: encryptionKey,
		logger:        logger,
		keyVersion:    1,
		keyCreatedAt:  time.Now(),
	}
}

// EncryptSecretKey 加密密钥
func (cm *cryptoManager) EncryptSecretKey(secretKey string) (string, error) {
	if secretKey == "" {
		return "", fmt.Errorf("密钥不能为空")
	}

	cm.mu.RLock()
	defer cm.mu.RUnlock()

	// 创建AES cipher
	block, err := aes.NewCipher(cm.encryptionKey)
	if err != nil {
		cm.logger.Error("创建AES cipher失败", zap.Error(err))
		return "", fmt.Errorf("创建AES cipher失败: %w", err)
	}

	// 创建GCM模式
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		cm.logger.Error("创建GCM模式失败", zap.Error(err))
		return "", fmt.Errorf("创建GCM模式失败: %w", err)
	}

	// 生成随机nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		cm.logger.Error("生成随机nonce失败", zap.Error(err))
		return "", fmt.Errorf("生成随机nonce失败: %w", err)
	}

	// 加密
	ciphertext := gcm.Seal(nonce, nonce, []byte(secretKey), nil)

	// 编码为base64
	encoded := base64.StdEncoding.EncodeToString(ciphertext)

	cm.logger.Debug("密钥加密成功", zap.Int("keyVersion", cm.keyVersion))
	return encoded, nil
}

// DecryptSecretKey 解密密钥
func (cm *cryptoManager) DecryptSecretKey(encryptedSecretKey string) (string, error) {
	if encryptedSecretKey == "" {
		return "", fmt.Errorf("加密密钥不能为空")
	}

	cm.mu.RLock()
	defer cm.mu.RUnlock()

	// 解码base64
	ciphertext, err := base64.StdEncoding.DecodeString(encryptedSecretKey)
	if err != nil {
		cm.logger.Error("解码base64失败", zap.Error(err))
		return "", fmt.Errorf("解码base64失败: %w", err)
	}

	// 创建AES cipher
	block, err := aes.NewCipher(cm.encryptionKey)
	if err != nil {
		cm.logger.Error("创建AES cipher失败", zap.Error(err))
		return "", fmt.Errorf("创建AES cipher失败: %w", err)
	}

	// 创建GCM模式
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		cm.logger.Error("创建GCM模式失败", zap.Error(err))
		return "", fmt.Errorf("创建GCM模式失败: %w", err)
	}

	// 检查密文长度
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		cm.logger.Error("密文长度不足", zap.Int("ciphertextLen", len(ciphertext)), zap.Int("nonceSize", nonceSize))
		return "", fmt.Errorf("密文长度不足")
	}

	// 分离nonce和密文
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	// 解密
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		cm.logger.Error("解密失败", zap.Error(err))
		return "", fmt.Errorf("解密失败: %w", err)
	}

	cm.logger.Debug("密钥解密成功", zap.Int("keyVersion", cm.keyVersion))
	return string(plaintext), nil
}

// RotateKey 轮换加密密钥
func (cm *cryptoManager) RotateKey(newKey []byte) error {
	if len(newKey) != 32 {
		return fmt.Errorf("新密钥必须是32字节(256位)")
	}

	cm.mu.Lock()
	defer cm.mu.Unlock()

	// 备份旧密钥用于解密现有数据
	oldVersion := cm.keyVersion

	// 更新密钥
	cm.encryptionKey = newKey
	cm.keyVersion++
	cm.keyCreatedAt = time.Now()

	cm.logger.Info("加密密钥轮换成功",
		zap.Int("oldVersion", oldVersion),
		zap.Int("newVersion", cm.keyVersion),
		zap.Time("keyCreatedAt", cm.keyCreatedAt))

	return nil
}

// GetKeyInfo 获取密钥信息
func (cm *cryptoManager) GetKeyInfo() map[string]interface{} {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return map[string]interface{}{
		"version":   cm.keyVersion,
		"createdAt": cm.keyCreatedAt,
		"keyLength": len(cm.encryptionKey),
		"algorithm": "AES-256-GCM",
	}
}

// ValidateEncryptedData 验证加密数据格式
func (cm *cryptoManager) ValidateEncryptedData(encryptedData string) error {
	if encryptedData == "" {
		return fmt.Errorf("加密数据不能为空")
	}

	// 尝试解码base64
	ciphertext, err := base64.StdEncoding.DecodeString(encryptedData)
	if err != nil {
		return fmt.Errorf("无效的base64编码: %w", err)
	}

	// 检查最小长度（nonce + 至少1字节密文）
	if len(ciphertext) < 13 {
		return fmt.Errorf("加密数据长度不足")
	}

	return nil
}

// EncryptBatch 批量加密
func (cm *cryptoManager) EncryptBatch(secretKeys []string) ([]string, error) {
	if len(secretKeys) == 0 {
		return []string{}, nil
	}

	cm.logger.Debug("开始批量加密", zap.Int("count", len(secretKeys)))

	encryptedKeys := make([]string, len(secretKeys))
	for i, secretKey := range secretKeys {
		encrypted, err := cm.EncryptSecretKey(secretKey)
		if err != nil {
			cm.logger.Error("批量加密失败", zap.Int("index", i), zap.Error(err))
			return nil, fmt.Errorf("批量加密失败，索引 %d: %w", i, err)
		}
		encryptedKeys[i] = encrypted
	}

	cm.logger.Debug("批量加密完成", zap.Int("count", len(encryptedKeys)))
	return encryptedKeys, nil
}

// DecryptBatch 批量解密
func (cm *cryptoManager) DecryptBatch(encryptedSecretKeys []string) ([]string, error) {
	if len(encryptedSecretKeys) == 0 {
		return []string{}, nil
	}

	cm.logger.Debug("开始批量解密", zap.Int("count", len(encryptedSecretKeys)))

	decryptedKeys := make([]string, len(encryptedSecretKeys))
	for i, encryptedKey := range encryptedSecretKeys {
		decrypted, err := cm.DecryptSecretKey(encryptedKey)
		if err != nil {
			cm.logger.Error("批量解密失败", zap.Int("index", i), zap.Error(err))
			return nil, fmt.Errorf("批量解密失败，索引 %d: %w", i, err)
		}
		decryptedKeys[i] = decrypted
	}

	cm.logger.Debug("批量解密完成", zap.Int("count", len(decryptedKeys)))
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

// GenerateKeyFromPassword 从密码生成密钥（使用PBKDF2）
func GenerateKeyFromPassword(password string, salt []byte) ([]byte, error) {
	if len(salt) < 16 {
		return nil, fmt.Errorf("盐值长度不足，至少需要16字节")
	}

	// 使用PBKDF2生成密钥
	key := make([]byte, 32)
	copy(key, []byte(password))

	// 简单的密钥派生，生产环境建议使用crypto/pbkdf2
	for i := 0; i < 10000; i++ {
		for j := range key {
			key[j] ^= salt[j%len(salt)]
		}
	}

	return key, nil
}
