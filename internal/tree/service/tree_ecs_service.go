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

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/dao"

	// "github.com/GoSimplicity/AI-CloudOps/internal/tree/provider" // 云资源提供商，仅支持本地资源时暂不需要
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"go.uber.org/zap"
)

type TreeEcsService interface {
	ListEcsResources(ctx context.Context, req *model.ListEcsResourcesReq) (model.ListResp[*model.ResourceEcs], error)
	GetEcsDetail(ctx context.Context, req *model.GetEcsDetailReq) (*model.ResourceEcs, error)
	CreateEcsResource(ctx context.Context, req *model.CreateEcsResourceReq) error
	UpdateEcs(ctx context.Context, req *model.UpdateEcsReq) error
	DeleteEcs(ctx context.Context, req *model.DeleteEcsReq) error
	StartEcs(ctx context.Context, req *model.StartEcsReq) error
	StopEcs(ctx context.Context, req *model.StopEcsReq) error
	RestartEcs(ctx context.Context, req *model.RestartEcsReq) error
	// ResizeEcs(ctx context.Context, req *model.ResizeEcsReq) error
	// ResetEcsPassword(ctx context.Context, req *model.ResetEcsPasswordReq) error
	// RenewEcs(ctx context.Context, req *model.RenewEcsReq) error
	// ListEcsResourceOptions(ctx context.Context, req *model.ListEcsResourceOptionsReq) (model.ListResp[*model.ListEcsResourceOptionsResp], error)
}

type treeEcsService struct {
	// providerFactory *provider.ProviderFactory
	logger *zap.Logger
	dao    dao.TreeEcsDAO
	// cloudDao dao.TreeCloudDAO
}

func NewTreeEcsService(logger *zap.Logger, dao dao.TreeEcsDAO) TreeEcsService {
	return &treeEcsService{
		logger: logger,
		dao:    dao,
		// providerFactory: providerFactory,
		// cloudDao:        cloudDao,
	}
}

// CreateEcsResource 创建ECS实例
func (t *treeEcsService) CreateEcsResource(ctx context.Context, req *model.CreateEcsResourceReq) error {
	/*
		// 判断是否是云资源
		if req.Provider != model.CloudProviderLocal {
			account, err := t.cloudDao.GetCloudAccount(ctx, req.AccountId)
			if err != nil {
				t.logger.Error("获取云账号失败", zap.Error(err))
				return err
			}
			provider, err := t.providerFactory.GetProvider(account.Provider)
			if err != nil {
				t.logger.Error("创建云Provider失败", zap.Error(err))
				return err
			}

			err = provider.CreateInstance(ctx, req.Region, req)
			if err != nil {
				t.logger.Error("创建ECS实例失败", zap.Error(err))
				return err
			}
		} else {
			req.InstanceType = "ecs-local"
		}
	*/

	// 强制设置为本地资源
	req.Provider = model.CloudProviderLocal
	req.InstanceType = "ecs-local"

	// 加密密码
	req.Password = utils.Base64EncryptWithMagic(req.Password)

	// 创建本地ECS实例
	err := t.dao.CreateEcsResource(ctx, convertCreateEcsResourceReqToResourceEcs(req))
	if err != nil {
		t.logger.Error("创建本地ECS实例失败", zap.Error(err))
		return err
	}

	return nil
}

// DeleteEcs 删除ECS实例
func (t *treeEcsService) DeleteEcs(ctx context.Context, req *model.DeleteEcsReq) error {
	// 仅支持本地资源，注释云资源逻辑
	/*
		if req.Provider != model.CloudProviderLocal {
			account, err := t.cloudDao.GetCloudAccount(ctx, req.AccountId)
			if err != nil {
				t.logger.Error("获取云账号失败", zap.Error(err))
				return err
			}
			provider, err := t.providerFactory.GetProvider(account.Provider)
			if err != nil {
				t.logger.Error("创建云Provider失败", zap.Error(err))
				return err
			}

			err = provider.DeleteInstance(ctx, req.Region, req.InstanceId)
			if err != nil {
				t.logger.Error("删除ECS实例失败", zap.Error(err))
				return err
			}
		}
	*/

	// 删除本地ECS实例
	err := t.dao.DeleteEcsResource(ctx, req.ID)
	if err != nil {
		t.logger.Error("删除本地ECS实例失败", zap.Error(err))
		return err
	}

	return nil
}

// GetEcsDetail 获取ECS实例详情
func (t *treeEcsService) GetEcsDetail(ctx context.Context, req *model.GetEcsDetailReq) (*model.ResourceEcs, error) {
	resource, err := t.dao.GetEcsResourceById(ctx, req.ID)
	if err != nil {
		t.logger.Error("获取ECS实例详情失败", zap.Error(err), zap.Int("id", req.ID))
		return nil, err
	}

	return resource, nil
}

// ListEcsResources 获取ECS实例列表
func (t *treeEcsService) ListEcsResources(ctx context.Context, req *model.ListEcsResourcesReq) (model.ListResp[*model.ResourceEcs], error) {
	resources, total, err := t.dao.ListEcsResources(ctx, req)
	if err != nil {
		t.logger.Error("获取ECS实例列表失败", zap.Error(err))
		return model.ListResp[*model.ResourceEcs]{}, err
	}

	return model.ListResp[*model.ResourceEcs]{
		Total: total,
		Items: resources,
	}, nil
}

// ListEcsResourceOptions 获取ECS实例选项 - 云资源特有功能，仅支持本地资源时暂不提供
/*
func (t *treeEcsService) ListEcsResourceOptions(ctx context.Context, req *model.ListEcsResourceOptionsReq) (model.ListResp[*model.ListEcsResourceOptionsResp], error) {
	options, total, err := t.dao.GetEcsResourceOptions(ctx, req)
	if err != nil {
		t.logger.Error("获取ECS实例选项失败", zap.Error(err))
		return model.ListResp[*model.ListEcsResourceOptionsResp]{}, err
	}

	return model.ListResp[*model.ListEcsResourceOptionsResp]{
		Items: options,
		Total: total,
	}, nil
}
*/

// RenewEcs 续费ECS实例 - 云资源特有功能，仅支持本地资源时暂不提供
/*
func (t *treeEcsService) RenewEcs(ctx context.Context, req *model.RenewEcsReq) error {
	// 获取ECS实例信息
	resource, err := t.dao.GetEcsResourceById(ctx, req.ID)
	if err != nil {
		t.logger.Error("获取ECS实例失败", zap.Error(err))
		return err
	}

	// 更新续费信息
	if err := t.dao.UpdateEcsRenewalInfo(ctx, resource.InstanceId, req.ExpectedStartTime, req.AutoRenewPeriod); err != nil {
		t.logger.Error("更新ECS续费信息失败", zap.Error(err))
		return err
	}

	return nil
}
*/

// ResetEcsPassword 重置ECS实例密码 - 云资源特有功能，仅支持本地资源时暂不提供
/*
func (t *treeEcsService) ResetEcsPassword(ctx context.Context, req *model.ResetEcsPasswordReq) error {
	// 获取ECS实例信息
	resource, err := t.dao.GetEcsResourceById(ctx, req.ID)
	if err != nil {
		t.logger.Error("获取ECS实例失败", zap.Error(err))
		return err
	}

	// 加密新密码
	encryptedPassword := utils.Base64EncryptWithMagic(req.NewPassword)

	// 更新密码
	if err := t.dao.UpdateEcsPassword(ctx, resource.InstanceId, encryptedPassword); err != nil {
		t.logger.Error("更新ECS密码失败", zap.Error(err))
		return err
	}

	return nil
}
*/

// ResizeEcs 调整ECS实例规格 - 云资源特有功能，仅支持本地资源时暂不提供
/*
func (t *treeEcsService) ResizeEcs(ctx context.Context, req *model.ResizeEcsReq) error {
	// 获取ECS实例信息
	resource, err := t.dao.GetEcsResourceById(ctx, req.ID)
	if err != nil {
		t.logger.Error("获取ECS实例失败", zap.Error(err))
		return err
	}

	// 根据新规格计算CPU和内存
	cpu, memory := parseInstanceType(req.InstanceType)
	diskSize := req.SystemDisk.NewSize
	if diskSize == 0 {
		diskSize = resource.Disk
	}

	// 更新配置信息
	if err := t.dao.UpdateEcsConfiguration(ctx, resource.InstanceId, cpu, memory, diskSize); err != nil {
		t.logger.Error("更新ECS配置失败", zap.Error(err))
		return err
	}

	return nil
}
*/

// RestartEcs 重启ECS实例
func (t *treeEcsService) RestartEcs(ctx context.Context, req *model.RestartEcsReq) error {
	// 获取ECS实例信息
	resource, err := t.dao.GetEcsResourceById(ctx, req.ID)
	if err != nil {
		t.logger.Error("获取ECS实例失败", zap.Error(err))
		return err
	}

	// 更新状态为重启中
	if err := t.dao.UpdateEcsStatus(ctx, resource.InstanceId, "Restarting"); err != nil {
		t.logger.Error("更新ECS状态失败", zap.Error(err))
		return err
	}

	return nil
}

// StartEcs 启动ECS实例
func (t *treeEcsService) StartEcs(ctx context.Context, req *model.StartEcsReq) error {
	// 获取ECS实例信息
	resource, err := t.dao.GetEcsResourceById(ctx, req.ID)
	if err != nil {
		t.logger.Error("获取ECS实例失败", zap.Error(err))
		return err
	}

	// 更新状态为启动中
	if err := t.dao.UpdateEcsStatus(ctx, resource.InstanceId, "Starting"); err != nil {
		t.logger.Error("更新ECS状态失败", zap.Error(err))
		return err
	}

	return nil
}

// StopEcs 停止ECS实例
func (t *treeEcsService) StopEcs(ctx context.Context, req *model.StopEcsReq) error {
	// 获取ECS实例信息
	resource, err := t.dao.GetEcsResourceById(ctx, req.ID)
	if err != nil {
		t.logger.Error("获取ECS实例失败", zap.Error(err))
		return err
	}

	// 更新状态为停止中
	if err := t.dao.UpdateEcsStatus(ctx, resource.InstanceId, "Stopping"); err != nil {
		t.logger.Error("更新ECS状态失败", zap.Error(err))
		return err
	}

	return nil
}

// UpdateEcs 更新ECS实例
func (t *treeEcsService) UpdateEcs(ctx context.Context, req *model.UpdateEcsReq) error {
	// 仅支持本地资源，注释云资源逻辑
	/*
		if req.Provider != model.CloudProviderLocal {
			account, err := t.cloudDao.GetCloudAccount(ctx, req.AccountId)
			if err != nil {
				t.logger.Error("获取云账号失败", zap.Error(err))
				return err
			}
			provider, err := t.providerFactory.GetProvider(account.Provider)
			if err != nil {
				t.logger.Error("创建云Provider失败", zap.Error(err))
				return err
			}

			err = provider.StopInstance(ctx, req.Region, req.InstanceId)
			if err != nil {
				t.logger.Error("停止ECS实例失败", zap.Error(err))
				return err
			}
		}
	*/

	// 更新本地ECS实例
	err := t.dao.UpdateEcsResource(ctx, convertUpdateEcsReqToResourceEcs(req))
	if err != nil {
		t.logger.Error("更新ECS实例失败", zap.Error(err))
		return err
	}

	return nil
}

func convertCreateEcsResourceReqToResourceEcs(req *model.CreateEcsResourceReq) *model.ResourceEcs {
	return &model.ResourceEcs{
		Provider:     req.Provider,
		InstanceName: req.InstanceName,
		InstanceType: req.InstanceType,
		ImageName:    req.ImageName,
		HostName:     req.Hostname,
		TreeNodeID:   req.TreeNodeId,
		Tags:         req.Tags,
		OsType:       req.OsType,
		AuthMode:     req.AuthMode,
		Key:          req.Key,
		IpAddr:       req.IpAddr,
		Port:         req.Port,
		Password:     req.Password,
		Description:  req.Description,
	}
}

func convertUpdateEcsReqToResourceEcs(req *model.UpdateEcsReq) *model.ResourceEcs {
	return &model.ResourceEcs{
		Model: model.Model{
			ID: req.ID,
		},
		InstanceName:     req.InstanceName,
		InstanceId:       req.InstanceId,
		Provider:         req.Provider,
		RegionId:         req.Region,
		Description:      req.Description,
		Tags:             req.Tags,
		SecurityGroupIds: req.SecurityGroupIds,
		HostName:         req.Hostname,
		Password:         req.Password,
		TreeNodeID:       req.TreeNodeId,
		Env:              req.Env,
		IpAddr:           req.IpAddr,
		Port:             req.Port,
		AuthMode:         req.AuthMode,
		Key:              req.Key,
	}
}
