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
	"strconv"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/provider"
	"go.uber.org/zap"
)

type TreeEcsService interface {
	// 资源管理
	ListEcsResources(ctx context.Context, req *model.ListEcsResourcesReq) (model.ListResp[*model.ResourceEcs], error)
	GetEcsResourceById(ctx context.Context, req *model.GetEcsDetailReq) (*model.ResourceECSDetailResp, error)
	CreateEcsResource(ctx context.Context, params *model.CreateEcsResourceReq) error
	StartEcsResource(ctx context.Context, req *model.StartEcsReq) error
	StopEcsResource(ctx context.Context, req *model.StopEcsReq) error
	RestartEcsResource(ctx context.Context, req *model.RestartEcsReq) error
	DeleteEcsResource(ctx context.Context, req *model.DeleteEcsReq) error
	ListEcsResourceOptions(ctx context.Context, req *model.ListEcsResourceOptionsReq) (model.ListResp[*model.ListEcsResourceOptionsResp], error)

	// 磁盘管理
	ListDisks(ctx context.Context, provider model.CloudProvider, region string, pageSize int, pageNumber int) (model.ListResp[*model.ResourceDisk], error)
	CreateDisk(ctx context.Context, provider model.CloudProvider, region string, params *model.DiskCreationParams) error
	DeleteDisk(ctx context.Context, provider model.CloudProvider, region string, diskID string) error
	AttachDisk(ctx context.Context, provider model.CloudProvider, region string, diskID string, instanceID string) error
	DetachDisk(ctx context.Context, provider model.CloudProvider, region string, diskID string, instanceID string) error
}

type treeEcsService struct {
	providerFactory *provider.ProviderFactory
	logger          *zap.Logger
	dao             dao.TreeEcsDAO
}

func NewTreeEcsService(logger *zap.Logger, dao dao.TreeEcsDAO, providerFactory *provider.ProviderFactory) TreeEcsService {
	return &treeEcsService{
		logger:          logger,
		dao:             dao,
		providerFactory: providerFactory,
	}
}

// CreateEcsResource 创建ECS资源
func (e *treeEcsService) CreateEcsResource(ctx context.Context, params *model.CreateEcsResourceReq) error {
	if params.Provider == model.CloudProviderLocal {
		err := e.dao.CreateEcsResource(ctx, &model.ResourceEcs{
			ResourceBase: model.ResourceBase{
				Description:  params.Description,
				TreeNodeID:   params.TreeNodeId,
				Tags:         params.Tags,
				LastSyncTime: time.Now(),
				Provider:     params.Provider,
				InstanceName: params.InstanceName,
			},
			InstanceType: params.InstanceType,
			HostName:     params.Hostname,
			Password:     params.Password,
			IpAddr:       params.IpAddr,
			Port:         params.Port,
			AuthMode:     params.AuthMode,
			Key:          params.Key,
			OsType:       params.OsType,
			ImageName:    params.ImageName,
		})
		if err != nil {
			e.logger.Error("[CreateEcsResource] 创建ECS资源失败", zap.Error(err))
			return err
		}
		return nil
	}

	cloudProvider, err := e.providerFactory.GetProvider(params.Provider)
	if err != nil {
		return fmt.Errorf("[CreateEcsResource] 获取云提供商失败: %w", err)
	}

	err = cloudProvider.CreateInstance(ctx, params.Region, params)
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
func (e *treeEcsService) StartEcsResource(ctx context.Context, req *model.StartEcsReq) error {
	cloudProvider, err := e.providerFactory.GetProvider(req.Provider)
	if err != nil {
		return fmt.Errorf("[StartEcsResource] 获取云提供商失败: %w", err)
	}

	err = cloudProvider.StartInstance(ctx, req.Region, req.InstanceId)
	if err != nil {
		e.logger.Error("[StartEcsResource] 启动云实例失败",
			zap.String("provider", string(req.Provider)),
			zap.String("region", req.Region),
			zap.String("instanceID", req.InstanceId),
			zap.Error(err))
		return fmt.Errorf("[StartEcsResource] 启动云实例失败: %w", err)
	}

	return nil
}

// StopEcsResource 停止ECS资源
func (e *treeEcsService) StopEcsResource(ctx context.Context, req *model.StopEcsReq) error {
	cloudProvider, err := e.providerFactory.GetProvider(req.Provider)
	if err != nil {
		return fmt.Errorf("[StopEcsResource] 获取云提供商失败: %w", err)
	}

	err = cloudProvider.StopInstance(ctx, req.Region, req.InstanceId)
	if err != nil {
		e.logger.Error("[StopEcsResource] 停止云实例失败",
			zap.String("provider", string(req.Provider)),
			zap.String("region", req.Region),
			zap.String("instanceID", req.InstanceId),
			zap.Error(err))
		return fmt.Errorf("[StopEcsResource] 停止云实例失败: %w", err)
	}

	return nil
}

// RestartEcsResource 重启ECS资源
func (e *treeEcsService) RestartEcsResource(ctx context.Context, req *model.RestartEcsReq) error {
	cloudProvider, err := e.providerFactory.GetProvider(req.Provider)
	if err != nil {
		return fmt.Errorf("[RestartEcsResource] 获取云提供商失败: %w", err)
	}

	err = cloudProvider.RestartInstance(ctx, req.Region, req.InstanceId)
	if err != nil {
		e.logger.Error("[RestartEcsResource] 重启云实例失败",
			zap.String("provider", string(req.Provider)),
			zap.String("region", req.Region),
			zap.String("instanceID", req.InstanceId),
			zap.Error(err))
		return fmt.Errorf("[RestartEcsResource] 重启云实例失败: %w", err)
	}

	return nil
}

// DeleteEcsResource 删除ECS资源
func (e *treeEcsService) DeleteEcsResource(ctx context.Context, req *model.DeleteEcsReq) error {
	if req.Provider == model.CloudProviderLocal {
		err := e.dao.DeleteEcsResource(ctx, req.InstanceId)
		if err != nil {
			e.logger.Error("[DeleteEcsResource] 删除ECS资源失败", zap.Error(err))
			return fmt.Errorf("[DeleteEcsResource] 删除ECS资源失败: %w", err)
		}
		return nil
	}

	cloudProvider, err := e.providerFactory.GetProvider(req.Provider)
	if err != nil {
		return fmt.Errorf("[DeleteEcsResource] 获取云提供商失败: %w", err)
	}

	err = cloudProvider.DeleteInstance(ctx, req.Region, req.InstanceId)
	if err != nil {
		e.logger.Error("[DeleteEcsResource] 删除云实例失败",
			zap.String("provider", string(req.Provider)),
			zap.String("region", req.Region),
			zap.String("instanceID", req.InstanceId),
			zap.Error(err))
		return fmt.Errorf("[DeleteEcsResource] 删除云实例失败: %w", err)
	}

	return nil
}

// GetEcsResourceById 获取ECS资源详情
func (e *treeEcsService) GetEcsResourceById(ctx context.Context, req *model.GetEcsDetailReq) (*model.ResourceECSDetailResp, error) {
	if req.Provider == model.CloudProviderLocal {
		intId, err := strconv.ParseInt(req.InstanceId, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("[GetEcsResourceById] 转换实例ID失败: %w", err)
		}
		resource, err := e.dao.GetEcsResourceById(ctx, int(intId))
		if err != nil {
			e.logger.Error("[GetEcsResourceById] 获取ECS资源失败", zap.Error(err))
			return nil, fmt.Errorf("[GetEcsResourceById] 获取ECS资源失败: %w", err)
		}
		return &model.ResourceECSDetailResp{
			Data: resource,
		}, nil
	}

	cloudProvider, err := e.providerFactory.GetProvider(req.Provider)
	if err != nil {
		return nil, fmt.Errorf("[GetEcsResourceById] 获取云提供商失败: %w", err)
	}

	result, err := cloudProvider.GetInstance(ctx, req.Region, req.InstanceId)
	if err != nil {
		e.logger.Error("[GetEcsResourceById] 获取ECS资源详情失败", zap.Error(err))
		return nil, fmt.Errorf("[GetEcsResourceById] 获取ECS资源详情失败: %w", err)
	}

	return &model.ResourceECSDetailResp{
		Data: result,
	}, nil
}

// ListEcsResources 获取ECS资源列表
func (e *treeEcsService) ListEcsResources(ctx context.Context, req *model.ListEcsResourcesReq) (model.ListResp[*model.ResourceEcs], error) {
	if req.Provider == model.CloudProviderLocal {
		resources, err := e.dao.ListEcsResources(ctx, req)
		if err != nil {
			return model.ListResp[*model.ResourceEcs]{
				Total: 0,
				Items: []*model.ResourceEcs{},
			}, err
		}
		return model.ListResp[*model.ResourceEcs]{
			Total: int64(len(resources)),
			Items: resources,
		}, nil
	}

	cloudProvider, err := e.providerFactory.GetProvider(req.Provider)
	if err != nil {
		return model.ListResp[*model.ResourceEcs]{
			Total: 0,
			Items: []*model.ResourceEcs{},
		}, fmt.Errorf("[ListEcsResources] 获取云提供商失败: %w", err)
	}

	resources, err := cloudProvider.ListInstances(ctx, req.Region, req.Size, req.Page)
	if err != nil {
		e.logger.Error("[ListEcsResources] 获取ECS资源列表失败", zap.Error(err))
		return model.ListResp[*model.ResourceEcs]{
			Total: 0,
			Items: []*model.ResourceEcs{},
		}, err
	}

	return model.ListResp[*model.ResourceEcs]{
		Total: int64(len(resources)),
		Items: resources,
	}, nil
}

// ListDisks 获取磁盘列表
func (e *treeEcsService) ListDisks(ctx context.Context, provider model.CloudProvider, region string, pageSize int, pageNumber int) (model.ListResp[*model.ResourceDisk], error) {
	cloudProvider, err := e.providerFactory.GetProvider(provider)
	if err != nil {
		return model.ListResp[*model.ResourceDisk]{
			Total: 0,
			Items: []*model.ResourceDisk{},
		}, fmt.Errorf("[ListDisks] 获取云提供商失败: %w", err)
	}

	result, err := cloudProvider.ListDisks(ctx, region, pageNumber, pageSize)
	if err != nil {
		e.logger.Error("[ListDisks] 获取磁盘列表失败",
			zap.String("provider", string(provider)),
			zap.String("region", region),
			zap.Error(err))
		return model.ListResp[*model.ResourceDisk]{
			Total: 0,
			Items: []*model.ResourceDisk{},
		}, fmt.Errorf("[ListDisks] 获取磁盘列表失败: %w", err)
	}

	return model.ListResp[*model.ResourceDisk]{
		Total: int64(len(result)),
		Items: result,
	}, nil
}

// CreateDisk 创建磁盘
func (e *treeEcsService) CreateDisk(ctx context.Context, provider model.CloudProvider, region string, params *model.DiskCreationParams) error {
	cloudProvider, err := e.providerFactory.GetProvider(provider)
	if err != nil {
		return fmt.Errorf("[CreateDisk] 获取云提供商失败: %w", err)
	}

	err = cloudProvider.CreateDisk(ctx, region, &model.CreateDiskReq{
		DiskName:     params.DiskName,
		DiskCategory: params.DiskCategory,
		Size:         params.Size,
		Description:  params.Description,
	})
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
func (e *treeEcsService) DeleteDisk(ctx context.Context, provider model.CloudProvider, region string, diskID string) error {
	cloudProvider, err := e.providerFactory.GetProvider(provider)
	if err != nil {
		return fmt.Errorf("[DeleteDisk] 获取云提供商失败: %w", err)
	}

	err = cloudProvider.DeleteDisk(ctx, region, diskID)
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
func (e *treeEcsService) AttachDisk(ctx context.Context, provider model.CloudProvider, region string, diskID string, instanceID string) error {
	cloudProvider, err := e.providerFactory.GetProvider(provider)
	if err != nil {
		return fmt.Errorf("[AttachDisk] 获取云提供商失败: %w", err)
	}

	err = cloudProvider.AttachDisk(ctx, region, diskID, instanceID)
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
func (e *treeEcsService) DetachDisk(ctx context.Context, provider model.CloudProvider, region string, diskID string, instanceID string) error {
	cloudProvider, err := e.providerFactory.GetProvider(provider)
	if err != nil {
		return fmt.Errorf("[DetachDisk] 获取云提供商失败: %w", err)
	}

	err = cloudProvider.DetachDisk(ctx, region, diskID, instanceID)
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

// ListEcsResourceOptions 获取ECS资源选项列表
func (e *treeEcsService) ListEcsResourceOptions(ctx context.Context, req *model.ListEcsResourceOptionsReq) (model.ListResp[*model.ListEcsResourceOptionsResp], error) {
	cloudProvider, err := e.providerFactory.GetProvider(req.Provider)
	if err != nil {
		return model.ListResp[*model.ListEcsResourceOptionsResp]{
			Total: 0,
			Items: []*model.ListEcsResourceOptionsResp{},
		}, fmt.Errorf("[ListEcsResourceOptions] 获取云提供商失败: %w", err)
	}

	switch req.ResourceType {
	case "region":
		result, err := cloudProvider.ListRegionOptions(ctx)
		if err != nil {
			return model.ListResp[*model.ListEcsResourceOptionsResp]{
				Total: 0,
				Items: []*model.ListEcsResourceOptionsResp{},
			}, fmt.Errorf("[ListEcsResourceOptions] 获取区域列表失败: %w", err)
		}
		return model.ListResp[*model.ListEcsResourceOptionsResp]{
			Total: int64(len(result)),
			Items: result,
		}, nil
	case "zone":
		result, err := cloudProvider.ListRegionZones(ctx, req.Region)
		if err != nil {
			return model.ListResp[*model.ListEcsResourceOptionsResp]{
				Total: 0,
				Items: []*model.ListEcsResourceOptionsResp{},
			}, fmt.Errorf("[ListEcsResourceOptions] 获取可用区列表失败: %w", err)
		}
		return model.ListResp[*model.ListEcsResourceOptionsResp]{
			Total: int64(len(result)),
			Items: result,
		}, nil
	case "instanceType":
		result, err := cloudProvider.ListRegionInstanceTypes(ctx, req.Region)
		if err != nil {
			return model.ListResp[*model.ListEcsResourceOptionsResp]{
				Total: 0,
				Items: []*model.ListEcsResourceOptionsResp{},
			}, fmt.Errorf("[ListEcsResourceOptions] 获取实例类型列表失败: %w", err)
		}
		return model.ListResp[*model.ListEcsResourceOptionsResp]{
			Total: int64(len(result)),
			Items: result,
		}, nil
	case "imageId":
		result, err := cloudProvider.ListRegionImages(ctx, req.Region)
		if err != nil {
			return model.ListResp[*model.ListEcsResourceOptionsResp]{
				Total: 0,
				Items: []*model.ListEcsResourceOptionsResp{},
			}, fmt.Errorf("[ListEcsResourceOptions] 获取镜像列表失败: %w", err)
		}
		return model.ListResp[*model.ListEcsResourceOptionsResp]{
			Total: int64(len(result)),
			Items: result,
		}, nil
	case "systemDiskCategory":
		result, err := cloudProvider.ListRegionSystemDiskCategories(ctx, req.Region)
		if err != nil {
			return model.ListResp[*model.ListEcsResourceOptionsResp]{
				Total: 0,
				Items: []*model.ListEcsResourceOptionsResp{},
			}, fmt.Errorf("[ListEcsResourceOptions] 获取系统盘类型列表失败: %w", err)
		}
		return model.ListResp[*model.ListEcsResourceOptionsResp]{
			Total: int64(len(result)),
			Items: result,
		}, nil
	case "dataDiskCategory":
		result, err := cloudProvider.ListRegionDataDiskCategories(ctx, req.Region)
		if err != nil {
			return model.ListResp[*model.ListEcsResourceOptionsResp]{
				Total: 0,
				Items: []*model.ListEcsResourceOptionsResp{},
			}, fmt.Errorf("[ListEcsResourceOptions] 获取数据盘类型列表失败: %w", err)
		}
		return model.ListResp[*model.ListEcsResourceOptionsResp]{
			Total: int64(len(result)),
			Items: result,
		}, nil
	}

	return model.ListResp[*model.ListEcsResourceOptionsResp]{
		Total: 0,
		Items: []*model.ListEcsResourceOptionsResp{},
	}, fmt.Errorf("[ListEcsResourceOptions] 未知的资源类型: %s", req.ResourceType)
}
