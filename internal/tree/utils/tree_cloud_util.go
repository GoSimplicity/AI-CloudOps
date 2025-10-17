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

// AliyunSyncConfig 阿里云同步配置
type AliyunSyncConfig struct {
	AccessKey      string
	SecretKey      string
	Region         string
	CloudAccountID int
	ResourceType   model.CloudResourceType
	InstanceIDs    []string
	SyncMode       model.SyncMode
}

// SyncAliyunResources 同步阿里云资源
func SyncAliyunResources(ctx context.Context, config *AliyunSyncConfig, logger *zap.Logger) ([]*model.TreeCloudResource, error) {
	logger.Info("开始同步阿里云资源",
		zap.Int("cloudAccountID", config.CloudAccountID),
		zap.String("region", config.Region),
		zap.String("syncMode", string(config.SyncMode)),
		zap.Int("specifiedInstanceCount", len(config.InstanceIDs)))

	// 创建阿里云客户端
	client, err := NewAliyunClient(config.AccessKey, config.SecretKey, config.Region, logger)
	if err != nil {
		logger.Error("创建阿里云客户端失败", zap.Error(err))
		return nil, err
	}

	// 目前只支持ECS资源类型的同步
	if config.ResourceType != 0 && config.ResourceType != model.ResourceTypeECS {
		return nil, fmt.Errorf("暂不支持该资源类型的同步: %d", config.ResourceType)
	}

	// 获取ECS实例列表
	resources, err := client.ListECSInstances(ctx, config.InstanceIDs)
	if err != nil {
		logger.Error("获取阿里云ECS实例失败", zap.Error(err))
		return nil, err
	}

	logger.Info("阿里云API返回资源数量", zap.Int("count", len(resources)))

	// 为每个资源设置云账户ID
	for _, resource := range resources {
		resource.CloudAccountID = config.CloudAccountID
	}

	logger.Info("同步阿里云资源成功",
		zap.String("syncMode", string(config.SyncMode)),
		zap.Int("count", len(resources)))

	return resources, nil
}

// SyncTencentResources 同步腾讯云资源
func SyncTencentResources(ctx context.Context, req *model.SyncTreeCloudResourceReq, logger *zap.Logger) error {
	logger.Warn("腾讯云资源同步功能暂未实现")
	return fmt.Errorf("腾讯云资源同步功能暂未实现")
}

// SyncAWSResources 同步AWS资源
func SyncAWSResources(ctx context.Context, req *model.SyncTreeCloudResourceReq, logger *zap.Logger) error {
	logger.Warn("AWS资源同步功能暂未实现")
	return fmt.Errorf("AWS资源同步功能暂未实现")
}

// SyncHuaweiResources 同步华为云资源
func SyncHuaweiResources(ctx context.Context, req *model.SyncTreeCloudResourceReq, logger *zap.Logger) error {
	logger.Warn("华为云资源同步功能暂未实现")
	return fmt.Errorf("华为云资源同步功能暂未实现")
}

// SyncAzureResources 同步Azure资源
func SyncAzureResources(ctx context.Context, req *model.SyncTreeCloudResourceReq, logger *zap.Logger) error {
	logger.Warn("Azure资源同步功能暂未实现")
	return fmt.Errorf("Azure资源同步功能暂未实现")
}

// SyncGCPResources 同步GCP资源
func SyncGCPResources(ctx context.Context, req *model.SyncTreeCloudResourceReq, logger *zap.Logger) error {
	logger.Warn("GCP资源同步功能暂未实现")
	return fmt.Errorf("GCP资源同步功能暂未实现")
}
