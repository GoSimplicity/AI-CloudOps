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

package service

import (
	"context"
	"fmt"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/provider"
	"go.uber.org/zap"
)

type ResourceService interface {
	SyncResources(ctx context.Context, provider model.CloudProvider, region string) error
	DeleteResource(ctx context.Context, resourceType string, id string) error
	StartResource(ctx context.Context, resourceType string, id string) error
	StopResource(ctx context.Context, resourceType string, id string) error
	RestartResource(ctx context.Context, resourceType string, id string) error
}

type resourceService struct {
	logger          *zap.Logger
	dao             dao.ResourceDAO
	providerFactory *provider.ProviderFactory
}

func NewResourceService(
	logger *zap.Logger,
	dao dao.ResourceDAO,
	providerFactory *provider.ProviderFactory,
) ResourceService {
	return &resourceService{
		logger:          logger,
		dao:             dao,
		providerFactory: providerFactory,
	}
}

// RestartResource 重启资源
func (r *resourceService) RestartResource(ctx context.Context, resourceType string, id string) error {
	// 首先停止资源
	err := r.StopResource(ctx, resourceType, id)
	if err != nil {
		r.logger.Error("重启资源失败：停止资源出错",
			zap.String("resourceType", resourceType),
			zap.String("id", id),
			zap.Error(err))
		return fmt.Errorf("停止资源失败: %w", err)
	}

	// 然后启动资源
	err = r.StartResource(ctx, resourceType, id)
	if err != nil {
		r.logger.Error("重启资源失败：启动资源出错",
			zap.String("resourceType", resourceType),
			zap.String("id", id),
			zap.Error(err))
		return fmt.Errorf("启动资源失败: %w", err)
	}

	return nil
}

// StartResource 启动资源
func (r *resourceService) StartResource(ctx context.Context, resourceType string, id string) error {
	// 获取资源信息
	resource, err := r.dao.GetResourceById(ctx, resourceType, id)
	if err != nil {
		r.logger.Error("启动资源失败：获取资源信息出错",
			zap.String("resourceType", resourceType),
			zap.String("id", id),
			zap.Error(err))
		return fmt.Errorf("获取资源信息失败: %w", err)
	}

	switch resourceType {
	case "ecs":
		return r.startEcsInstance(ctx, resource.Provider, resource.RegionId, resource.InstanceId)
	default:
		return fmt.Errorf("不支持的资源类型: %s", resourceType)
	}
}

// 启动ECS实例的具体实现
func (r *resourceService) startEcsInstance(ctx context.Context, provider model.CloudProvider, region string, instanceID string) error {
	cloudProvider, err := r.providerFactory.GetProvider(provider)
	if err != nil {
		return fmt.Errorf("获取云提供商失败: %w", err)
	}

	err = cloudProvider.StartInstance(ctx, region, instanceID)
	if err != nil {
		r.logger.Error("启动实例失败",
			zap.String("provider", string(provider)),
			zap.String("region", region),
			zap.String("instanceID", instanceID),
			zap.Error(err))
		return fmt.Errorf("启动实例失败: %w", err)
	}

	return nil
}

// StopResource 停止资源
func (r *resourceService) StopResource(ctx context.Context, resourceType string, id string) error {
	// 获取资源信息
	resource, err := r.dao.GetResourceById(ctx, resourceType, id)
	if err != nil {
		r.logger.Error("停止资源失败：获取资源信息出错",
			zap.String("resourceType", resourceType),
			zap.String("id", id),
			zap.Error(err))
		return fmt.Errorf("获取资源信息失败: %w", err)
	}

	switch resourceType {
	case "ecs":
		return r.stopEcsInstance(ctx, resource.Provider, resource.RegionId, resource.InstanceId)
	default:
		return fmt.Errorf("不支持的资源类型: %s", resourceType)
	}
}

// 停止ECS实例的具体实现
func (r *resourceService) stopEcsInstance(ctx context.Context, provider model.CloudProvider, region string, instanceID string) error {
	cloudProvider, err := r.providerFactory.GetProvider(provider)
	if err != nil {
		return fmt.Errorf("获取云提供商失败: %w", err)
	}

	err = cloudProvider.StopInstance(ctx, region, instanceID)
	if err != nil {
		r.logger.Error("停止实例失败",
			zap.String("provider", string(provider)),
			zap.String("region", region),
			zap.String("instanceID", instanceID),
			zap.Error(err))
		return fmt.Errorf("停止实例失败: %w", err)
	}

	return nil
}

// SyncResources 同步资源
func (r *resourceService) SyncResources(ctx context.Context, provider model.CloudProvider, region string) error {
	syncErr := make(chan error, 1)

	// 异步同步资源
	go func() {
		// 同步ECS资源
		err := r.syncEcsResources(ctx, provider, region)
		syncErr <- err // 无论是否出错，都要发送信号
	}()

	// 等待同步完成或超时
	select {
	case err := <-syncErr:
		// 同步完成
		if err != nil {
			return fmt.Errorf("同步ECS资源失败: %v", err)
		}
		return nil

	case <-ctx.Done():
		// 上下文取消（如超时或手动取消）
		return fmt.Errorf("同步取消: %v", ctx.Err())
	}
}

// 同步ECS资源的具体实现
func (r *resourceService) syncEcsResources(ctx context.Context, provider model.CloudProvider, region string) error {
	cloudProvider, err := r.providerFactory.GetProvider(provider)
	if err != nil {
		return fmt.Errorf("获取云提供商失败: %w", err)
	}

	err = cloudProvider.SyncResources(ctx, region)
	if err != nil {
		r.logger.Error("获取实例列表失败",
			zap.String("provider", string(provider)),
			zap.String("region", region),
			zap.Error(err))
		return fmt.Errorf("获取实例列表失败: %w", err)
	}

	r.logger.Info("同步ECS资源完成",
		zap.String("provider", string(provider)),
		zap.String("region", region))

	return nil
}

// DeleteResource 删除资源
func (r *resourceService) DeleteResource(ctx context.Context, resourceType string, id string) error {
	// 获取资源信息
	resource, err := r.dao.GetResourceById(ctx, resourceType, id)
	if err != nil {
		r.logger.Error("删除资源失败：获取资源信息出错",
			zap.String("resourceType", resourceType),
			zap.String("id", id),
			zap.Error(err))
		return fmt.Errorf("获取资源信息失败: %w", err)
	}

	switch resourceType {
	case "ecs":
		return r.deleteEcsInstance(ctx, resource.Provider, resource.RegionId, resource.InstanceId)
	default:
		return fmt.Errorf("不支持的资源类型: %s", resourceType)
	}
}

// 删除ECS实例的具体实现
func (r *resourceService) deleteEcsInstance(ctx context.Context, provider model.CloudProvider, region string, instanceID string) error {
	cloudProvider, err := r.providerFactory.GetProvider(provider)
	if err != nil {
		return fmt.Errorf("获取云提供商失败: %w", err)
	}

	err = cloudProvider.DeleteInstance(ctx, region, instanceID)
	if err != nil {
		r.logger.Error("删除实例失败",
			zap.String("provider", string(provider)),
			zap.String("region", region),
			zap.String("instanceID", instanceID),
			zap.Error(err))
		return fmt.Errorf("删除实例失败: %w", err)
	}

	return nil
}
