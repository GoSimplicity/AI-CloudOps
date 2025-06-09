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
	"errors"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/provider"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"go.uber.org/zap"
)

type TreeEcsService interface {
	// 资源管理
	ListEcsResources(ctx context.Context, req *model.ListEcsResourcesReq) (model.ListResp[*model.ResourceEcs], error)
	GetEcsDetail(ctx context.Context, req *model.GetEcsDetailReq) (*model.ResourceEcs, error)
	CreateEcsResource(ctx context.Context, req *model.CreateEcsResourceReq) error
	UpdateEcs(ctx context.Context, req *model.UpdateEcsReq) error
	DeleteEcs(ctx context.Context, req *model.DeleteEcsReq) error
	StartEcs(ctx context.Context, req *model.StartEcsReq) error
	StopEcs(ctx context.Context, req *model.StopEcsReq) error
	RestartEcs(ctx context.Context, req *model.RestartEcsReq) error
	ResizeEcs(ctx context.Context, req *model.ResizeEcsReq) error
	ResetEcsPassword(ctx context.Context, req *model.ResetEcsPasswordReq) error
	RenewEcs(ctx context.Context, req *model.RenewEcsReq) error
	ListEcsResourceOptions(ctx context.Context, req *model.ListEcsResourceOptionsReq) (model.ListResp[*model.ListEcsResourceOptionsResp], error)
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

// CreateEcsResource 创建ECS实例
func (t *treeEcsService) CreateEcsResource(ctx context.Context, req *model.CreateEcsResourceReq) error {
	// 验证req参数是否合法
	if err := validateCreateEcsResourceReq(req); err != nil {
		t.logger.Error("创建ECS实例参数验证失败", zap.Error(err))
		return err
	}

	// 判断是否是云资源
	if req.Provider != model.CloudProviderLocal {
		provider, err := t.providerFactory.GetProvider(req.Provider)
		if err != nil {
			t.logger.Error("获取云提供商失败", zap.Error(err))
			return err
		}

		err = provider.CreateInstance(ctx, req.Region, req)
		if err != nil {
			t.logger.Error("创建ECS实例失败", zap.Error(err))
			return err
		}
	}

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
	if req.Provider != model.CloudProviderLocal {
		provider, err := t.providerFactory.GetProvider(req.Provider)
		if err != nil {
			t.logger.Error("获取云提供商失败", zap.Error(err))
			return err
		}

		err = provider.DeleteInstance(ctx, req.Region, req.InstanceId)
		if err != nil {
			t.logger.Error("删除ECS实例失败", zap.Error(err))
			return err
		}
	}

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

// ListEcsResourceOptions 获取ECS实例选项
func (t *treeEcsService) ListEcsResourceOptions(ctx context.Context, req *model.ListEcsResourceOptionsReq) (model.ListResp[*model.ListEcsResourceOptionsResp], error) {
	panic("unimplemented")
}

// RenewEcs 续费ECS实例
func (t *treeEcsService) RenewEcs(ctx context.Context, req *model.RenewEcsReq) error {
	panic("unimplemented")
}

// ResetEcsPassword 重置ECS实例密码
func (t *treeEcsService) ResetEcsPassword(ctx context.Context, req *model.ResetEcsPasswordReq) error {
	panic("unimplemented")
}

// ResizeEcs 调整ECS实例规格
func (t *treeEcsService) ResizeEcs(ctx context.Context, req *model.ResizeEcsReq) error {
	panic("unimplemented")
}

// RestartEcs 重启ECS实例
func (t *treeEcsService) RestartEcs(ctx context.Context, req *model.RestartEcsReq) error {
	panic("unimplemented")
}

// StartEcs 启动ECS实例
func (t *treeEcsService) StartEcs(ctx context.Context, req *model.StartEcsReq) error {
	panic("unimplemented")
}

// StopEcs 停止ECS实例
func (t *treeEcsService) StopEcs(ctx context.Context, req *model.StopEcsReq) error {
	panic("unimplemented")
}

// UpdateEcs 更新ECS实例
func (t *treeEcsService) UpdateEcs(ctx context.Context, req *model.UpdateEcsReq) error {
	panic("unimplemented")
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

func validateCreateEcsResourceReq(req *model.CreateEcsResourceReq) error {
	if req.Provider == "" {
		return errors.New("云提供商不能为空")
	}

	if req.InstanceType == "" {
		return errors.New("实例类型不能为空")
	}

	if req.Hostname == "" {
		return errors.New("主机名不能为空")
	}

	if req.AuthMode == "password" && req.Password == "" {
		return errors.New("密码不能为空")
	}

	if req.AuthMode == "key" && req.Key == "" {
		return errors.New("密钥不能为空")
	}

	if req.OsType == "" {
		return errors.New("操作系统类型不能为空")
	}

	return nil
}
