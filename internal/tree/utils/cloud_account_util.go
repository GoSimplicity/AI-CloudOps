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
	"context"
	"fmt"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
)

// VerifyAliyunCredentials 验证阿里云凭证
func VerifyAliyunCredentials(ctx context.Context, req *model.VerifyCloudCredentialsReq, logger *zap.Logger) error {
	client, err := NewAliyunClient(req.AccessKey, req.SecretKey, req.Region, logger)
	if err != nil {
		logger.Error("创建阿里云客户端失败", zap.Error(err))
		return err
	}

	if err := client.VerifyCredentials(ctx); err != nil {
		logger.Error("验证阿里云凭证失败", zap.Error(err))
		return err
	}

	return nil
}

// VerifyTencentCredentials 验证腾讯云凭证
func VerifyTencentCredentials(ctx context.Context, req *model.VerifyCloudCredentialsReq, logger *zap.Logger) error {
	logger.Warn("腾讯云凭证验证功能暂未实现")
	return fmt.Errorf("腾讯云凭证验证功能暂未实现")
}

// VerifyAWSCredentials 验证AWS凭证
func VerifyAWSCredentials(ctx context.Context, req *model.VerifyCloudCredentialsReq, logger *zap.Logger) error {
	logger.Warn("AWS凭证验证功能暂未实现")
	return fmt.Errorf("AWS凭证验证功能暂未实现")
}

// VerifyHuaweiCredentials 验证华为云凭证
func VerifyHuaweiCredentials(ctx context.Context, req *model.VerifyCloudCredentialsReq, logger *zap.Logger) error {
	logger.Warn("华为云凭证验证功能暂未实现")
	return fmt.Errorf("华为云凭证验证功能暂未实现")
}

// VerifyAzureCredentials 验证Azure凭证
func VerifyAzureCredentials(ctx context.Context, req *model.VerifyCloudCredentialsReq, logger *zap.Logger) error {
	logger.Warn("Azure凭证验证功能暂未实现")
	return fmt.Errorf("Azure凭证验证功能暂未实现")
}

// VerifyGCPCredentials 验证GCP凭证
func VerifyGCPCredentials(ctx context.Context, req *model.VerifyCloudCredentialsReq, logger *zap.Logger) error {
	logger.Warn("GCP凭证验证功能暂未实现")
	return fmt.Errorf("GCP凭证验证功能暂未实现")
}
