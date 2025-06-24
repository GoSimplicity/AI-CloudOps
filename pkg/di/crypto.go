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

package di

import (
	"encoding/base64"
	"os"

	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"go.uber.org/zap"
)

// InitCryptoManager 初始化加密管理器
func InitCryptoManager(logger *zap.Logger) utils.CryptoManager {
	// 尝试从环境变量获取加密密钥
	encryptionKeyStr := os.Getenv("ENCRYPTION_KEY")

	var encryptionKey []byte
	var err error

	if encryptionKeyStr != "" {
		// 如果环境变量存在，尝试解码base64
		encryptionKey, err = base64.StdEncoding.DecodeString(encryptionKeyStr)
		if err != nil {
			logger.Error("解码环境变量中的加密密钥失败", zap.Error(err))
			// 如果解码失败，生成新的密钥
			encryptionKey, err = utils.GenerateRandomKey()
			if err != nil {
				logger.Fatal("生成随机加密密钥失败", zap.Error(err))
			}
		}
	} else {
		// 如果环境变量不存在，生成新的密钥
		encryptionKey, err = utils.GenerateRandomKey()
		if err != nil {
			logger.Fatal("生成随机加密密钥失败", zap.Error(err))
		}

		// 将生成的密钥编码为base64并输出到日志，方便用户设置环境变量
		encodedKey := base64.StdEncoding.EncodeToString(encryptionKey)
		logger.Info("生成新的加密密钥，建议设置环境变量 ENCRYPTION_KEY",
			zap.String("key", encodedKey),
			zap.String("env_var", "ENCRYPTION_KEY="+encodedKey))
	}

	// 验证密钥长度
	if len(encryptionKey) != 32 {
		logger.Fatal("加密密钥长度不正确", zap.Int("expected", 32), zap.Int("actual", len(encryptionKey)))
	}

	logger.Info("加密管理器初始化成功", zap.Int("keyLength", len(encryptionKey)))

	return utils.NewCryptoManager(encryptionKey, logger)
}
