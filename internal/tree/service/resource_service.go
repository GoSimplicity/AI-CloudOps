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
	DeleteResource(ctx context.Context, resourceType string, id int) error
	StartResource(ctx context.Context, resourceType string, id int) error
	StopResource(ctx context.Context, resourceType string, id int) error
	RestartResource(ctx context.Context, resourceType string, id int) error
}

type resourceService struct {
	logger          *zap.Logger
	dao             dao.ResourceDAO
	AliyunProvider  provider.AliyunProvider
	TencentProvider provider.TencentProvider
	HuaweiProvider  provider.HuaweiProvider
	AWSProvider     provider.AwsProvider
	AzureProvider   provider.AzureProvider
	GCPProvider     provider.GcpProvider
}

func NewResourceService(
	logger *zap.Logger,
	dao dao.ResourceDAO,
	aliyunProvider provider.AliyunProvider,
	tencentProvider provider.TencentProvider,
	huaweiProvider provider.HuaweiProvider,
	awsProvider provider.AwsProvider,
	azureProvider provider.AzureProvider,
	gcpProvider provider.GcpProvider,
) ResourceService {
	return &resourceService{
		logger:          logger,
		dao:             dao,
		AliyunProvider:  aliyunProvider,
		TencentProvider: tencentProvider,
		HuaweiProvider:  huaweiProvider,
		AWSProvider:     awsProvider,
		AzureProvider:   azureProvider,
		GCPProvider:     gcpProvider,
	}
}

// RestartResource 重启资源
func (r *resourceService) RestartResource(ctx context.Context, resourceType string, id int) error {
	// 首先停止资源
	err := r.StopResource(ctx, resourceType, id)
	if err != nil {
		r.logger.Error("重启资源失败：停止资源出错", 
			zap.String("resourceType", resourceType),
			zap.Int("id", id),
			zap.Error(err))
		return fmt.Errorf("停止资源失败: %w", err)
	}

	// 然后启动资源
	err = r.StartResource(ctx, resourceType, id)
	if err != nil {
		r.logger.Error("重启资源失败：启动资源出错", 
			zap.String("resourceType", resourceType),
			zap.Int("id", id),
			zap.Error(err))
		return fmt.Errorf("启动资源失败: %w", err)
	}

	return nil
}

// StartResource 启动资源
func (r *resourceService) StartResource(ctx context.Context, resourceType string, id int) error {
	// 获取资源信息
	resource, err := r.dao.GetResourceById(ctx, resourceType, id)
	if err != nil {
		r.logger.Error("启动资源失败：获取资源信息出错", 
			zap.String("resourceType", resourceType),
			zap.Int("id", id),
			zap.Error(err))
		return fmt.Errorf("获取资源信息失败: %w", err)
	}

	switch resourceType {
	case "ecs":
		return r.startEcsInstance(ctx, resource.Provider, resource.Region, resource.InstanceId)
	default:
		return fmt.Errorf("不支持的资源类型: %s", resourceType)
	}
}

// 启动ECS实例的具体实现
func (r *resourceService) startEcsInstance(ctx context.Context, provider model.CloudProvider, region string, instanceID string) error {
	var err error
	switch provider {
	case model.CloudProviderAliyun:
		err = r.AliyunProvider.StartInstance(ctx, region, instanceID)
	case model.CloudProviderTencent:
		err = r.TencentProvider.StartInstance(ctx, region, instanceID)
	case model.CloudProviderHuawei:
		err = r.HuaweiProvider.StartInstance(ctx, region, instanceID)
	case model.CloudProviderAWS:
		err = r.AWSProvider.StartInstance(ctx, region, instanceID)
	case model.CloudProviderAzure:
		err = r.AzureProvider.StartInstance(ctx, region, instanceID)
	case model.CloudProviderGCP:
		err = r.GCPProvider.StartInstance(ctx, region, instanceID)
	default:
		return fmt.Errorf("不支持的云提供商: %s", provider)
	}

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
func (r *resourceService) StopResource(ctx context.Context, resourceType string, id int) error {
	// 获取资源信息
	resource, err := r.dao.GetResourceById(ctx, resourceType, id)
	if err != nil {
		r.logger.Error("停止资源失败：获取资源信息出错", 
			zap.String("resourceType", resourceType),
			zap.Int("id", id),
			zap.Error(err))
		return fmt.Errorf("获取资源信息失败: %w", err)
	}

	switch resourceType {
	case "ecs":
		return r.stopEcsInstance(ctx, resource.Provider, resource.Region, resource.InstanceId)
	default:
		return fmt.Errorf("不支持的资源类型: %s", resourceType)
	}
}

// 停止ECS实例的具体实现
func (r *resourceService) stopEcsInstance(ctx context.Context, provider model.CloudProvider, region string, instanceID string) error {
	var err error
	switch provider {
	case model.CloudProviderAliyun:
		err = r.AliyunProvider.StopInstance(ctx, region, instanceID)
	case model.CloudProviderTencent:
		err = r.TencentProvider.StopInstance(ctx, region, instanceID)
	case model.CloudProviderHuawei:
		err = r.HuaweiProvider.StopInstance(ctx, region, instanceID)
	case model.CloudProviderAWS:
		err = r.AWSProvider.StopInstance(ctx, region, instanceID)
	case model.CloudProviderAzure:
		err = r.AzureProvider.StopInstance(ctx, region, instanceID)
	case model.CloudProviderGCP:
		err = r.GCPProvider.StopInstance(ctx, region, instanceID)
	default:
		return fmt.Errorf("不支持的云提供商: %s", provider)
	}

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
	// 同步ECS资源
	err := r.syncEcsResources(ctx, provider, region)
	if err != nil {
		r.logger.Error("同步ECS资源失败", 
			zap.String("provider", string(provider)),
			zap.String("region", region),
			zap.Error(err))
		return fmt.Errorf("同步ECS资源失败: %w", err)
	}

	//TODO: 同步其他资源(暂不支持)

	return nil
}

// 同步ECS资源的具体实现
func (r *resourceService) syncEcsResources(ctx context.Context, provider model.CloudProvider, region string) error {
	// 默认每页大小和起始页码
	pageSize := 50
	pageNumber := 1

	var instances []*model.ResourceECSResp
	var err error

	// 根据不同的云提供商获取实例列表
	switch provider {
	case model.CloudProviderAliyun:
		instances, err = r.AliyunProvider.ListInstances(ctx, region, pageSize, pageNumber)
	case model.CloudProviderTencent:
		instances, err = r.TencentProvider.ListInstances(ctx, region, pageSize, pageNumber)
	case model.CloudProviderHuawei:
		instances, err = r.HuaweiProvider.ListInstances(ctx, region, pageSize, pageNumber)
	case model.CloudProviderAWS:
		instances, err = r.AWSProvider.ListInstances(ctx, region, pageSize, pageNumber)
	case model.CloudProviderAzure:
		instances, err = r.AzureProvider.ListInstances(ctx, region, pageSize, pageNumber)
	case model.CloudProviderGCP:
		instances, err = r.GCPProvider.ListInstances(ctx, region, pageSize, pageNumber)
	default:
		return fmt.Errorf("不支持的云提供商: %s", provider)
	}

	if err != nil {
		r.logger.Error("获取实例列表失败", 
			zap.String("provider", string(provider)),
			zap.String("region", region),
			zap.Error(err))
		return fmt.Errorf("获取实例列表失败: %w", err)
	}

	// 将获取到的实例保存到数据库
	for _, instance := range instances {
		err = r.dao.SaveOrUpdateResource(ctx, instance)
		if err != nil {
			r.logger.Error("保存或更新ECS资源失败", 
				zap.String("instanceId", instance.InstanceId),
				zap.Error(err))
			// 继续处理其他实例，不中断整个同步过程
			continue
		}
	}

	r.logger.Info("同步ECS资源完成", 
		zap.String("provider", string(provider)),
		zap.String("region", region),
		zap.Int("count", len(instances)))

	return nil
}

// DeleteResource 删除资源
func (r *resourceService) DeleteResource(ctx context.Context, resourceType string, id int) error {
	// 获取资源信息
	resource, err := r.dao.GetResourceById(ctx, resourceType, id)
	if err != nil {
		r.logger.Error("删除资源失败：获取资源信息出错", 
			zap.String("resourceType", resourceType),
			zap.Int("id", id),
			zap.Error(err))
		return fmt.Errorf("获取资源信息失败: %w", err)
	}

	// 根据资源类型执行不同的删除逻辑
	switch resourceType {
	case "ecs":
		// 删除ECS实例
		err = r.deleteEcsInstance(ctx, resource.Provider, resource.Region, resource.InstanceId)
	// 可以添加其他资源类型的处理逻辑
	default:
		return fmt.Errorf("不支持的资源类型: %s", resourceType)
	}

	if err != nil {
		return err
	}

	// 从数据库中删除资源记录
	err = r.dao.DeleteResource(ctx, resourceType, id)
	if err != nil {
		r.logger.Error("从数据库删除资源记录失败", 
			zap.String("resourceType", resourceType),
			zap.Int("id", id),
			zap.Error(err))
		return fmt.Errorf("从数据库删除资源记录失败: %w", err)
	}

	return nil
}

// 删除ECS实例的具体实现
func (r *resourceService) deleteEcsInstance(ctx context.Context, provider model.CloudProvider, region string, instanceID string) error {
	var err error
	switch provider {
	case model.CloudProviderAliyun:
		err = r.AliyunProvider.DeleteInstance(ctx, region, instanceID)
	case model.CloudProviderTencent:
		err = r.TencentProvider.DeleteInstance(ctx, region, instanceID)
	case model.CloudProviderHuawei:
		err = r.HuaweiProvider.DeleteInstance(ctx, region, instanceID)
	case model.CloudProviderAWS:
		err = r.AWSProvider.DeleteInstance(ctx, region, instanceID)
	case model.CloudProviderAzure:
		err = r.AzureProvider.DeleteInstance(ctx, region, instanceID)
	case model.CloudProviderGCP:
		err = r.GCPProvider.DeleteInstance(ctx, region, instanceID)
	default:
		return fmt.Errorf("不支持的云提供商: %s", provider)
	}

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
