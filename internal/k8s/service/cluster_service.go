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

	"gorm.io/gorm"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/manager"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/utils"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"

	"go.uber.org/zap"
)

type ClusterService interface {
	ListClusters(ctx context.Context, req *model.ListClustersReq) (model.ListResp[*model.K8sCluster], error)
	CreateCluster(ctx context.Context, req *model.CreateClusterReq) error
	UpdateCluster(ctx context.Context, req *model.UpdateClusterReq) error
	DeleteCluster(ctx context.Context, req *model.DeleteClusterReq) error
	GetClusterByID(ctx context.Context, req *model.GetClusterReq) (*model.K8sCluster, error)
	RefreshClusterStatus(ctx context.Context, req *model.RefreshClusterReq) error
}

type clusterService struct {
	dao        dao.ClusterDAO
	client     client.K8sClient
	clusterMgr manager.ClusterManager
	logger     *zap.Logger
}

func NewClusterService(dao dao.ClusterDAO, client client.K8sClient, clusterMgr manager.ClusterManager, logger *zap.Logger) ClusterService {
	return &clusterService{
		dao:        dao,
		client:     client,
		clusterMgr: clusterMgr,
		logger:     logger,
	}
}

// ListClusters 获取集群列表
func (s *clusterService) ListClusters(ctx context.Context, req *model.ListClustersReq) (model.ListResp[*model.K8sCluster], error) {
	if req == nil {
		return model.ListResp[*model.K8sCluster]{}, fmt.Errorf("获取集群列表请求参数不能为空")
	}

	list, total, err := s.dao.GetClusterList(ctx, req)
	if err != nil {
		s.logger.Error("ListClusters: 查询所有集群失败", zap.Error(err))
		return model.ListResp[*model.K8sCluster]{}, fmt.Errorf("查询所有集群失败: %w", err)
	}

	// 清理敏感信息
	utils.CleanClusterSensitiveInfoList(list)

	return model.ListResp[*model.K8sCluster]{
		Total: total,
		Items: list,
	}, nil
}

// GetClusterByID 根据ID获取集群
func (s *clusterService) GetClusterByID(ctx context.Context, req *model.GetClusterReq) (*model.K8sCluster, error) {
	if req == nil {
		return nil, fmt.Errorf("获取集群请求参数不能为空")
	}

	// 验证集群ID
	if err := utils.ValidateClusterID(req.ID); err != nil {
		return nil, err
	}

	cluster, err := s.dao.GetClusterByID(ctx, req.ID)
	if err != nil {
		s.logger.Error("GetClusterByID: 查询集群失败", zap.Error(err), zap.Int("clusterID", req.ID))
		return nil, fmt.Errorf("查询集群失败: %w", err)
	}

	// 验证集群是否存在
	if err := utils.ValidateClusterExists(cluster, req.ID); err != nil {
		return nil, err
	}

	return cluster, nil
}

// CreateCluster 创建集群
func (s *clusterService) CreateCluster(ctx context.Context, req *model.CreateClusterReq) error {
	if req == nil {
		return fmt.Errorf("创建集群请求参数不能为空")
	}

	// 验证创建集群参数
	if err := utils.ValidateClusterCreateParams(req.Name, req.ApiServerAddr, req.KubeConfigContent); err != nil {
		return err
	}

	// 检查集群名称是否已存在
	existingCluster, err := s.dao.GetClusterByName(ctx, req.Name)
	if err != nil && err != gorm.ErrRecordNotFound {
		s.logger.Error("CreateCluster: 查询集群失败", zap.Error(err))
		return fmt.Errorf("查询集群失败: %w", err)
	}

	if existingCluster != nil {
		return fmt.Errorf("集群名称 %s 已存在", req.Name)
	}

	cluster := &model.K8sCluster{
		Name:                 req.Name,
		CpuRequest:           req.CpuRequest,
		CpuLimit:             req.CpuLimit,
		MemoryRequest:        req.MemoryRequest,
		MemoryLimit:          req.MemoryLimit,
		RestrictNamespace:    req.RestrictNamespace,
		Status:               model.StatusRunning,
		Env:                  req.Env,
		Version:              req.Version,
		ApiServerAddr:        req.ApiServerAddr,
		KubeConfigContent:    req.KubeConfigContent,
		ActionTimeoutSeconds: req.ActionTimeoutSeconds,
		CreateUserName:       req.CreateUserName,
		CreateUserID:         req.CreateUserID,
		Tags:                 req.Tags,
	}

	// 验证资源配置
	if err := utils.ValidateResourceQuantities(cluster); err != nil {
		return fmt.Errorf("资源配置验证失败: %w", err)
	}

	// 创建集群记录
	if err := s.dao.CreateCluster(ctx, cluster); err != nil {
		s.logger.Error("CreateCluster: 创建集群记录失败", zap.Error(err))
		return fmt.Errorf("创建集群记录失败: %w", err)
	}

	// 使用集群管理器创建集群
	if err := s.clusterMgr.CreateCluster(ctx, cluster); err != nil {
		s.logger.Error("CreateCluster: 创建集群失败", zap.Error(err))
		// 回滚数据库记录
		if rollbackErr := s.dao.DeleteCluster(ctx, cluster.ID); rollbackErr != nil {
			s.logger.Error("CreateCluster: 回滚失败", zap.Error(rollbackErr))
		}
		return fmt.Errorf("创建集群失败: %w", err)
	}

	return nil
}

// UpdateCluster 更新集群
func (s *clusterService) UpdateCluster(ctx context.Context, req *model.UpdateClusterReq) error {
	if req == nil {
		return fmt.Errorf("更新集群请求参数不能为空")
	}

	// 验证更新集群参数
	if err := utils.ValidateClusterUpdateParams(req.ID); err != nil {
		return err
	}

	// 检查集群是否存在
	existingCluster, err := s.dao.GetClusterByID(ctx, req.ID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("集群不存在，ID: %d", req.ID)
		}

		s.logger.Error("UpdateCluster: 查询集群失败", zap.Error(err), zap.Int("clusterID", req.ID))
		return fmt.Errorf("查询集群失败: %w", err)
	}

	// 检查集群名称是否冲突
	if req.Name != "" && req.Name != existingCluster.Name {
		conflictCluster, err := s.dao.GetClusterByName(ctx, req.Name)
		if err != nil && err != gorm.ErrRecordNotFound {
			s.logger.Error("UpdateCluster: 查询集群名称冲突失败", zap.Error(err))
			return fmt.Errorf("查询集群名称冲突失败: %w", err)
		}
		if conflictCluster != nil {
			return fmt.Errorf("集群名称 %s 已存在", req.Name)
		}
	}

	// 构建更新的集群对象
	cluster := &model.K8sCluster{
		Model:                model.Model{ID: req.ID},
		Name:                 req.Name,
		ApiServerAddr:        req.ApiServerAddr,
		KubeConfigContent:    req.KubeConfigContent,
		ActionTimeoutSeconds: req.ActionTimeoutSeconds,
		Tags:                 req.Tags,
		RestrictNamespace:    req.RestrictNamespace,
		Status:               model.StatusRunning,
		Env:                  req.Env,
		Version:              req.Version,
		CpuRequest:           req.CpuRequest,
		CpuLimit:             req.CpuLimit,
		MemoryRequest:        req.MemoryRequest,
		MemoryLimit:          req.MemoryLimit,
	}

	// 验证资源配置
	if err := utils.ValidateResourceQuantities(cluster); err != nil {
		return fmt.Errorf("资源配置验证失败: %w", err)
	}

	// 更新集群记录
	if err := s.dao.UpdateCluster(ctx, cluster); err != nil {
		s.logger.Error("UpdateCluster: 更新集群失败", zap.Error(err), zap.Int("clusterID", cluster.ID))
		return fmt.Errorf("更新集群失败: %w", err)
	}

	// 使用集群管理器更新集群
	if err := s.clusterMgr.UpdateCluster(ctx, cluster); err != nil {
		s.logger.Error("UpdateCluster: 更新集群客户端失败", zap.Error(err))
		return fmt.Errorf("更新集群客户端失败: %w", err)
	}

	return nil
}

// DeleteCluster 删除集群
func (s *clusterService) DeleteCluster(ctx context.Context, req *model.DeleteClusterReq) error {
	if req == nil {
		return fmt.Errorf("删除集群请求参数不能为空")
	}

	// 验证集群ID
	if err := utils.ValidateClusterID(req.ID); err != nil {
		return err
	}

	// 检查集群是否存在
	_, err := s.dao.GetClusterByID(ctx, req.ID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("集群不存在，ID: %d", req.ID)
		}

		s.logger.Error("DeleteCluster: 查询集群失败", zap.Error(err), zap.Int("clusterID", req.ID))
		return fmt.Errorf("查询集群失败: %w", err)
	}

	// 删除集群客户端
	s.client.RemoveCluster(req.ID)

	// 删除集群记录
	if err := s.dao.DeleteCluster(ctx, req.ID); err != nil {
		s.logger.Error("DeleteCluster: 删除集群失败", zap.Error(err), zap.Int("clusterID", req.ID))
		return fmt.Errorf("删除集群失败: %w", err)
	}

	return nil
}

// RefreshClusterStatus 刷新集群状态
func (s *clusterService) RefreshClusterStatus(ctx context.Context, req *model.RefreshClusterReq) error {
	if req == nil {
		return fmt.Errorf("刷新集群状态请求参数不能为空")
	}

	// 验证集群ID
	if err := utils.ValidateClusterID(req.ID); err != nil {
		return err
	}

	// 检查集群是否存在
	_, err := s.dao.GetClusterByID(ctx, req.ID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("集群不存在，ID: %d", req.ID)
		}

		s.logger.Error("RefreshClusterStatus: 查询集群失败", zap.Error(err), zap.Int("clusterID", req.ID))
		return fmt.Errorf("查询集群失败: %w", err)
	}

	// 使用集群管理器刷新集群状态
	if err := s.clusterMgr.RefreshCluster(ctx, req.ID); err != nil {
		s.logger.Error("RefreshClusterStatus: 刷新集群状态失败", zap.Error(err))
		return fmt.Errorf("刷新集群状态失败: %w", err)
	}

	return nil
}
