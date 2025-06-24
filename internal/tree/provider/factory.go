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

package provider

import (
	"fmt"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	huawei "github.com/GoSimplicity/AI-CloudOps/internal/tree/provider/huawei"
	"go.uber.org/zap"
)

// ProviderFactory 支持动态创建多云多账户 Provider 实例
// 推荐由 service 层先解密 SecretKey 后传入
type ProviderFactory struct {
	logger *zap.Logger
}

func NewProviderFactory(logger *zap.Logger) *ProviderFactory {
	return &ProviderFactory{logger: logger}
}

// CreateProvider 根据 CloudAccount 和解密后的 SecretKey 动态创建 Provider 实例
func (f *ProviderFactory) CreateProvider(account *model.CloudAccount, decryptedSecret string) (Provider, error) {
	if account == nil {
		return nil, fmt.Errorf("CloudAccount 不能为空")
	}
	acc := *account // 拷贝，避免外部副作用
	acc.EncryptedSecret = decryptedSecret

	switch acc.Provider {
	case model.CloudProviderAliyun:
		return NewAliyunProvider(f.logger, &acc), nil
	case model.CloudProviderHuawei:
		return huawei.NewHuaweiProvider(f.logger, &acc), nil
	default:
		return nil, fmt.Errorf("不支持的云提供商: %s", acc.Provider)
	}
}
