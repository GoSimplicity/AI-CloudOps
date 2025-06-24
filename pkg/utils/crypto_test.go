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
	"testing"

	"go.uber.org/zap"
)

func TestCryptoManager(t *testing.T) {
	// 创建测试密钥
	testKey := make([]byte, 32)
	for i := range testKey {
		testKey[i] = byte(i)
	}

	// 创建logger
	logger, _ := zap.NewDevelopment()

	// 创建加密管理器
	cm := NewCryptoManager(testKey, logger)

	// 测试数据
	testSecretKey := "test-secret-key-12345"

	t.Run("TestEncryptDecrypt", func(t *testing.T) {
		// 加密
		encrypted, err := cm.EncryptSecretKey(testSecretKey)
		if err != nil {
			t.Fatalf("加密失败: %v", err)
		}

		// 验证加密结果不为空且不等于原文
		if encrypted == "" {
			t.Fatal("加密结果为空")
		}
		if encrypted == testSecretKey {
			t.Fatal("加密结果与原文相同")
		}

		// 解密
		decrypted, err := cm.DecryptSecretKey(encrypted)
		if err != nil {
			t.Fatalf("解密失败: %v", err)
		}

		// 验证解密结果
		if decrypted != testSecretKey {
			t.Fatalf("解密结果不匹配，期望: %s, 实际: %s", testSecretKey, decrypted)
		}
	})

	t.Run("TestBatchEncryptDecrypt", func(t *testing.T) {
		// 批量测试数据
		secretKeys := []string{
			"secret1",
			"secret2",
			"secret3",
		}

		// 批量加密
		encrypted, err := cm.EncryptBatch(secretKeys)
		if err != nil {
			t.Fatalf("批量加密失败: %v", err)
		}

		// 验证加密结果数量
		if len(encrypted) != len(secretKeys) {
			t.Fatalf("批量加密结果数量不匹配，期望: %d, 实际: %d", len(secretKeys), len(encrypted))
		}

		// 批量解密
		decrypted, err := cm.DecryptBatch(encrypted)
		if err != nil {
			t.Fatalf("批量解密失败: %v", err)
		}

		// 验证解密结果
		if len(decrypted) != len(secretKeys) {
			t.Fatalf("批量解密结果数量不匹配，期望: %d, 实际: %d", len(secretKeys), len(decrypted))
		}

		// 验证每个结果
		for i, expected := range secretKeys {
			if decrypted[i] != expected {
				t.Fatalf("批量解密结果不匹配，索引: %d, 期望: %s, 实际: %s", i, expected, decrypted[i])
			}
		}
	})

	t.Run("TestKeyRotation", func(t *testing.T) {
		// 原始密钥加密
		originalEncrypted, err := cm.EncryptSecretKey(testSecretKey)
		if err != nil {
			t.Fatalf("原始密钥加密失败: %v", err)
		}

		// 生成新密钥
		newKey := make([]byte, 32)
		for i := range newKey {
			newKey[i] = byte(i + 100)
		}

		// 轮换密钥
		err = cm.RotateKey(newKey)
		if err != nil {
			t.Fatalf("密钥轮换失败: %v", err)
		}

		// 使用新密钥加密
		newEncrypted, err := cm.EncryptSecretKey(testSecretKey)
		if err != nil {
			t.Fatalf("新密钥加密失败: %v", err)
		}

		// 验证新加密结果与原始不同
		if newEncrypted == originalEncrypted {
			t.Fatal("新密钥加密结果与原始相同")
		}

		// 验证新密钥可以解密新加密的数据
		decrypted, err := cm.DecryptSecretKey(newEncrypted)
		if err != nil {
			t.Fatalf("新密钥解密失败: %v", err)
		}

		if decrypted != testSecretKey {
			t.Fatalf("新密钥解密结果不匹配，期望: %s, 实际: %s", testSecretKey, decrypted)
		}
	})

	t.Run("TestValidation", func(t *testing.T) {
		// 测试有效数据
		validData, _ := cm.EncryptSecretKey("test")
		err := cm.ValidateEncryptedData(validData)
		if err != nil {
			t.Fatalf("有效数据验证失败: %v", err)
		}

		// 测试无效数据
		invalidData := "invalid-base64-data!"
		err = cm.ValidateEncryptedData(invalidData)
		if err == nil {
			t.Fatal("无效数据验证应该失败")
		}

		// 测试空数据
		err = cm.ValidateEncryptedData("")
		if err == nil {
			t.Fatal("空数据验证应该失败")
		}
	})

	t.Run("TestKeyInfo", func(t *testing.T) {
		info := cm.GetKeyInfo()

		// 验证必要字段
		requiredFields := []string{"version", "createdAt", "keyLength", "algorithm"}
		for _, field := range requiredFields {
			if _, exists := info[field]; !exists {
				t.Fatalf("密钥信息缺少字段: %s", field)
			}
		}

		// 验证字段值
		if info["keyLength"] != 32 {
			t.Fatalf("密钥长度不正确，期望: 32, 实际: %v", info["keyLength"])
		}

		if info["algorithm"] != "AES-256-GCM" {
			t.Fatalf("算法不正确，期望: AES-256-GCM, 实际: %v", info["algorithm"])
		}
	})
}

func TestGenerateRandomKey(t *testing.T) {
	// 生成随机密钥
	key1, err := GenerateRandomKey()
	if err != nil {
		t.Fatalf("生成随机密钥失败: %v", err)
	}

	// 验证密钥长度
	if len(key1) != 32 {
		t.Fatalf("密钥长度不正确，期望: 32, 实际: %d", len(key1))
	}

	// 生成另一个随机密钥
	key2, err := GenerateRandomKey()
	if err != nil {
		t.Fatalf("生成第二个随机密钥失败: %v", err)
	}

	// 验证两个密钥不同
	if string(key1) == string(key2) {
		t.Fatal("两个随机密钥相同")
	}
}

func TestGenerateKeyFromPassword(t *testing.T) {
	password := "test-password"
	salt := make([]byte, 16)
	for i := range salt {
		salt[i] = byte(i)
	}

	// 生成密钥
	key, err := GenerateKeyFromPassword(password, salt)
	if err != nil {
		t.Fatalf("从密码生成密钥失败: %v", err)
	}

	// 验证密钥长度
	if len(key) != 32 {
		t.Fatalf("密钥长度不正确，期望: 32, 实际: %d", len(key))
	}

	// 测试盐值长度不足
	shortSalt := make([]byte, 8)
	_, err = GenerateKeyFromPassword(password, shortSalt)
	if err == nil {
		t.Fatal("短盐值应该导致错误")
	}
}
