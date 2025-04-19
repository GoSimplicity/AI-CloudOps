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
)

type ProviderFactory struct {
	providers map[model.CloudProvider]Provider
}

func NewProviderFactory(
	aliyun *AliyunProviderImpl,
	tencent *TencentProviderImpl,
	huawei *HuaweiProviderImpl,
	aws *AWSProviderImpl,
	azure *AzureProviderImpl,
	gcp *GCPProviderImpl,
) *ProviderFactory {
	return &ProviderFactory{
		providers: map[model.CloudProvider]Provider{
			model.CloudProviderAliyun:  aliyun,
			model.CloudProviderTencent: tencent,
			model.CloudProviderHuawei:  huawei,
			model.CloudProviderAWS:     aws,
			model.CloudProviderAzure:   azure,
			model.CloudProviderGCP:     gcp,
		},
	}
}

func (f *ProviderFactory) GetProvider(providerType model.CloudProvider) (Provider, error) {
	provider, ok := f.providers[providerType]
	if !ok {
		return nil, fmt.Errorf("不支持的云提供商: %s", providerType)
	}
	return provider, nil
}
