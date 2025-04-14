package service

import (
	"context"
	"fmt"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/provider"
	"go.uber.org/zap"
)

type EcsService interface {
	// 资源管理
	ListEcsResources(ctx context.Context, req *model.ListEcsResourcesReq) (*model.PageResp, error)
	GetEcsResourceById(ctx context.Context, id int) (*model.ResourceECSResp, error)
	CreateEcsResource(ctx context.Context, params *model.EcsCreationParams) error
	StartEcsResource(ctx context.Context, provider model.CloudProvider, region string, instanceID string) error
	StopEcsResource(ctx context.Context, provider model.CloudProvider, region string, instanceID string) error
	
	// 磁盘管理
	ListDisks(ctx context.Context, provider model.CloudProvider, region string, pageSize int, pageNumber int) (*model.PageResp, error)
	CreateDisk(ctx context.Context, provider model.CloudProvider, region string, params *model.DiskCreationParams) error
	DeleteDisk(ctx context.Context, provider model.CloudProvider, region string, diskID string) error
	AttachDisk(ctx context.Context, provider model.CloudProvider, region string, diskID string, instanceID string) error
	DetachDisk(ctx context.Context, provider model.CloudProvider, region string, diskID string, instanceID string) error
}

type ecsService struct {
	AliyunProvider provider.AliyunProvider
	TencentProvider provider.TencentProvider
	HuaweiProvider provider.HuaweiProvider
	AWSProvider provider.AwsProvider
	AzureProvider provider.AzureProvider
	GCPProvider provider.GcpProvider
	logger *zap.Logger
	dao    dao.EcsDAO
}


func NewEcsService(logger *zap.Logger, dao dao.EcsDAO, AliyunProvider provider.AliyunProvider, TencentProvider provider.TencentProvider, HuaweiProvider provider.HuaweiProvider, AWSProvider provider.AwsProvider, AzureProvider provider.AzureProvider, GCPProvider provider.GcpProvider) EcsService {
	return &ecsService{
		logger: logger,
		dao:    dao,
		AliyunProvider: AliyunProvider,
		TencentProvider: TencentProvider,
		HuaweiProvider: HuaweiProvider,
		AWSProvider: AWSProvider,
		AzureProvider: AzureProvider,
		GCPProvider: GCPProvider,
	}
}

// CreateEcsResource 创建ECS资源
func (e *ecsService) CreateEcsResource(ctx context.Context, params *model.EcsCreationParams) error {
	if params.Provider == model.CloudProviderLocal {
		err := e.dao.CreateEcsResource(ctx, params)
		if err != nil {
			e.logger.Error("[CreateEcsResource] 创建ECS资源失败", zap.Error(err))
			return err
		}
		return nil
	}

	var err error
	switch params.Provider {
	case model.CloudProviderAliyun:
		err = e.AliyunProvider.CreateInstance(ctx, params.Region, params)
	case model.CloudProviderTencent:
		err = e.TencentProvider.CreateInstance(ctx, params.Region, params)
	case model.CloudProviderHuawei:
		err = e.HuaweiProvider.CreateInstance(ctx, params.Region, params)
	case model.CloudProviderAWS:
		err = e.AWSProvider.CreateInstance(ctx, params.Region, params)
	case model.CloudProviderAzure:
		err = e.AzureProvider.CreateInstance(ctx, params.Region, params)
	case model.CloudProviderGCP:
		err = e.GCPProvider.CreateInstance(ctx, params.Region, params)
	default:
		return fmt.Errorf("[CreateEcsResource] 不支持的云提供商: %s", params.Provider)
	}

	if err != nil {
		e.logger.Error("[CreateEcsResource] 创建云实例失败", 
			zap.String("provider", string(params.Provider)),
			zap.String("region", params.Region),
			zap.Error(err))
		return fmt.Errorf("[CreateEcsResource] 创建云实例失败: %w", err)
	}

	return nil
}

// StartEcsResource 启动ECS资源
func (e *ecsService) StartEcsResource(ctx context.Context, provider model.CloudProvider, region string, instanceID string) error {
	var err error
	switch provider {
	case model.CloudProviderAliyun:
		err = e.AliyunProvider.StartInstance(ctx, region, instanceID)
	case model.CloudProviderTencent:
		err = e.TencentProvider.StartInstance(ctx, region, instanceID)
	case model.CloudProviderHuawei:
		err = e.HuaweiProvider.StartInstance(ctx, region, instanceID)
	case model.CloudProviderAWS:
		err = e.AWSProvider.StartInstance(ctx, region, instanceID)
	case model.CloudProviderAzure:
		err = e.AzureProvider.StartInstance(ctx, region, instanceID)
	case model.CloudProviderGCP:
		err = e.GCPProvider.StartInstance(ctx, region, instanceID)
	default:
		return fmt.Errorf("[StartEcsResource] 不支持的云提供商: %s", provider)
	}

	if err != nil {
		e.logger.Error("[StartEcsResource] 启动云实例失败", 
			zap.String("provider", string(provider)),
			zap.String("region", region),
			zap.String("instanceID", instanceID),
			zap.Error(err))
		return fmt.Errorf("[StartEcsResource] 启动云实例失败: %w", err)
	}

	return nil
}

// StopEcsResource 停止ECS资源
func (e *ecsService) StopEcsResource(ctx context.Context, provider model.CloudProvider, region string, instanceID string) error {
	var err error
	switch provider {
	case model.CloudProviderAliyun:
		err = e.AliyunProvider.StopInstance(ctx, region, instanceID)
	case model.CloudProviderTencent:
		err = e.TencentProvider.StopInstance(ctx, region, instanceID)
	case model.CloudProviderHuawei:
		err = e.HuaweiProvider.StopInstance(ctx, region, instanceID)
	case model.CloudProviderAWS:
		err = e.AWSProvider.StopInstance(ctx, region, instanceID)
	case model.CloudProviderAzure:
		err = e.AzureProvider.StopInstance(ctx, region, instanceID)
	case model.CloudProviderGCP:
		err = e.GCPProvider.StopInstance(ctx, region, instanceID)
	default:
		return fmt.Errorf("[StopEcsResource] 不支持的云提供商: %s", provider)
	}

	if err != nil {
		e.logger.Error("[StopEcsResource] 停止云实例失败", 
			zap.String("provider", string(provider)),
			zap.String("region", region),
			zap.String("instanceID", instanceID),
			zap.Error(err))
		return fmt.Errorf("[StopEcsResource] 停止云实例失败: %w", err)
	}

	return nil
}

// GetEcsResourceById 获取ECS资源详情
func (e *ecsService) GetEcsResourceById(ctx context.Context, id int) (*model.ResourceECSResp, error) {
	resource, err := e.dao.GetEcsResourceById(ctx, id)
	if err != nil {
		e.logger.Error("[GetEcsResourceById] 获取ECS资源详情失败", zap.Error(err))
		return nil, err
	}
	return resource, nil
}

// ListEcsResources 获取ECS资源列表
func (e *ecsService) ListEcsResources(ctx context.Context, req *model.ListEcsResourcesReq) (*model.PageResp, error) {
	resources, err := e.dao.ListEcsResources(ctx, req)
	if err != nil {
		e.logger.Error("[ListEcsResources] 获取ECS资源列表失败", zap.Error(err))
		return nil, err
	}
	return resources, nil
}

// ListDisks 获取磁盘列表
func (e *ecsService) ListDisks(ctx context.Context, provider model.CloudProvider, region string, pageSize int, pageNumber int) (*model.PageResp, error) {
	var (
		result []*model.PageResp
		err    error
	)

	switch provider {
	case model.CloudProviderAliyun:
		result, err = e.AliyunProvider.ListDisks(ctx, region, pageSize, pageNumber)
	case model.CloudProviderTencent:
		result, err = e.TencentProvider.ListDisks(ctx, region, pageSize, pageNumber)
	case model.CloudProviderHuawei:
		result, err = e.HuaweiProvider.ListDisks(ctx, region, pageSize, pageNumber)
	case model.CloudProviderAWS:
		result, err = e.AWSProvider.ListDisks(ctx, region, pageSize, pageNumber)
	case model.CloudProviderAzure:
		result, err = e.AzureProvider.ListDisks(ctx, region, pageSize, pageNumber)
	case model.CloudProviderGCP:
		result, err = e.GCPProvider.ListDisks(ctx, region, pageSize, pageNumber)
	default:
		return nil, fmt.Errorf("[ListDisks] 不支持的云提供商: %s", provider)
	}

	if err != nil {
		e.logger.Error("[ListDisks] 获取磁盘列表失败", 
			zap.String("provider", string(provider)),
			zap.String("region", region),
			zap.Error(err))
		return nil, fmt.Errorf("[ListDisks] 获取磁盘列表失败: %w", err)
	}

	if len(result) > 0 {
		return result[0], nil
	}
	return &model.PageResp{}, nil
}

// CreateDisk 创建磁盘
func (e *ecsService) CreateDisk(ctx context.Context, provider model.CloudProvider, region string, params *model.DiskCreationParams) error {
	var err error
	switch provider {
	case model.CloudProviderAliyun:
		err = e.AliyunProvider.CreateDisk(ctx, region, params)
	case model.CloudProviderTencent:
		err = e.TencentProvider.CreateDisk(ctx, region, params)
	case model.CloudProviderHuawei:
		err = e.HuaweiProvider.CreateDisk(ctx, region, params)
	case model.CloudProviderAWS:
		err = e.AWSProvider.CreateDisk(ctx, region, params)
	case model.CloudProviderAzure:
		err = e.AzureProvider.CreateDisk(ctx, region, params)
	case model.CloudProviderGCP:
		err = e.GCPProvider.CreateDisk(ctx, region, params)
	default:
		return fmt.Errorf("[CreateDisk] 不支持的云提供商: %s", provider)
	}

	if err != nil {
		e.logger.Error("[CreateDisk] 创建磁盘失败", 
			zap.String("provider", string(provider)),
			zap.String("region", region),
			zap.Error(err))
		return fmt.Errorf("[CreateDisk] 创建磁盘失败: %w", err)
	}

	return nil
}

// DeleteDisk 删除磁盘
func (e *ecsService) DeleteDisk(ctx context.Context, provider model.CloudProvider, region string, diskID string) error {
	var err error
	switch provider {
	case model.CloudProviderAliyun:
		err = e.AliyunProvider.DeleteDisk(ctx, region, diskID)
	case model.CloudProviderTencent:
		err = e.TencentProvider.DeleteDisk(ctx, region, diskID)
	case model.CloudProviderHuawei:
		err = e.HuaweiProvider.DeleteDisk(ctx, region, diskID)
	case model.CloudProviderAWS:
		err = e.AWSProvider.DeleteDisk(ctx, region, diskID)
	case model.CloudProviderAzure:
		err = e.AzureProvider.DeleteDisk(ctx, region, diskID)
	case model.CloudProviderGCP:
		err = e.GCPProvider.DeleteDisk(ctx, region, diskID)
	default:
		return fmt.Errorf("[DeleteDisk] 不支持的云提供商: %s", provider)
	}

	if err != nil {
		e.logger.Error("[DeleteDisk] 删除磁盘失败", 
			zap.String("provider", string(provider)),
			zap.String("region", region),
			zap.String("diskID", diskID),
			zap.Error(err))
		return fmt.Errorf("[DeleteDisk] 删除磁盘失败: %w", err)
	}

	return nil
}

// AttachDisk 挂载磁盘
func (e *ecsService) AttachDisk(ctx context.Context, provider model.CloudProvider, region string, diskID string, instanceID string) error {
	var err error
	switch provider {
	case model.CloudProviderAliyun:
		err = e.AliyunProvider.AttachDisk(ctx, region, diskID, instanceID)
	case model.CloudProviderTencent:
		err = e.TencentProvider.AttachDisk(ctx, region, diskID, instanceID)
	case model.CloudProviderHuawei:
		err = e.HuaweiProvider.AttachDisk(ctx, region, diskID, instanceID)
	case model.CloudProviderAWS:
		err = e.AWSProvider.AttachDisk(ctx, region, diskID, instanceID)
	case model.CloudProviderAzure:
		err = e.AzureProvider.AttachDisk(ctx, region, diskID, instanceID)
	case model.CloudProviderGCP:
		err = e.GCPProvider.AttachDisk(ctx, region, diskID, instanceID)
	default:
		return fmt.Errorf("[AttachDisk] 不支持的云提供商: %s", provider)
	}

	if err != nil {
		e.logger.Error("[AttachDisk] 挂载磁盘失败", 
			zap.String("provider", string(provider)),
			zap.String("region", region),
			zap.String("diskID", diskID),
			zap.String("instanceID", instanceID),
			zap.Error(err))
		return fmt.Errorf("[AttachDisk] 挂载磁盘失败: %w", err)
	}

	return nil
}

// DetachDisk 卸载磁盘
func (e *ecsService) DetachDisk(ctx context.Context, provider model.CloudProvider, region string, diskID string, instanceID string) error {
	var err error
	switch provider {
	case model.CloudProviderAliyun:
		err = e.AliyunProvider.DetachDisk(ctx, region, diskID, instanceID)
	case model.CloudProviderTencent:
		err = e.TencentProvider.DetachDisk(ctx, region, diskID, instanceID)
	case model.CloudProviderHuawei:
		err = e.HuaweiProvider.DetachDisk(ctx, region, diskID, instanceID)
	case model.CloudProviderAWS:
		err = e.AWSProvider.DetachDisk(ctx, region, diskID, instanceID)
	case model.CloudProviderAzure:
		err = e.AzureProvider.DetachDisk(ctx, region, diskID, instanceID)
	case model.CloudProviderGCP:
		err = e.GCPProvider.DetachDisk(ctx, region, diskID, instanceID)
	default:
		return fmt.Errorf("[DetachDisk] 不支持的云提供商: %s", provider)
	}

	if err != nil {
		e.logger.Error("[DetachDisk] 卸载磁盘失败", 
			zap.String("provider", string(provider)),
			zap.String("region", region),
			zap.String("diskID", diskID),
			zap.String("instanceID", instanceID),
			zap.Error(err))
		return fmt.Errorf("[DetachDisk] 卸载磁盘失败: %w", err)
	}

	return nil
}
