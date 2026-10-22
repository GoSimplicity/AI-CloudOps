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
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/connection"
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

func (s *clusterService) ListClusters(ctx context.Context, req *model.ListClustersReq) (model.ListResp[*model.K8sCluster], error) {
	if req == nil {
		return model.ListResp[*model.K8sCluster]{}, fmt.Errorf("获取集群列表请求参数不能为空")
	}

	list, total, err := s.dao.GetClusterList(ctx, req)
	if err != nil {
		s.logger.Error("查询所有集群失败", zap.Error(err))
		return model.ListResp[*model.K8sCluster]{}, fmt.Errorf("查询所有集群失败: %w", err)
	}

	utils.CleanClusterSensitiveInfoList(list)

	return model.ListResp[*model.K8sCluster]{
		Total: total,
		Items: list,
	}, nil
}

func (s *clusterService) GetClusterByID(ctx context.Context, req *model.GetClusterReq) (*model.K8sCluster, error) {
	if req == nil {
		return nil, fmt.Errorf("获取集群请求参数不能为空")
	}

	if err := utils.ValidateClusterID(req.ID); err != nil {
		return nil, err
	}

	cluster, err := s.dao.GetClusterByID(ctx, req.ID)
	if err != nil {
		s.logger.Error("查询集群失败", zap.Error(err), zap.Int("clusterID", req.ID))
		return nil, fmt.Errorf("查询集群失败: %w", err)
	}

	if err := utils.ValidateClusterExists(cluster, req.ID); err != nil {
		return nil, err
	}

	return cluster, nil
}

func (s *clusterService) CreateCluster(ctx context.Context, req *model.CreateClusterReq) error {
	if req == nil {
		return fmt.Errorf("创建集群请求参数不能为空")
	}

	if err := utils.ValidateClusterCreateParams(req.Name, req.ApiServerAddr, req.KubeConfigContent); err != nil {
		return err
	}

	existingCluster, err := s.dao.GetClusterByName(ctx, req.Name)
	if err != nil && err != gorm.ErrRecordNotFound {
		s.logger.Error("查询集群失败", zap.Error(err))
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

	if _, err := connection.Parse(cluster.KubeConfigContent); err != nil {
		return err
	}
	if err := utils.ValidateResourceQuantities(cluster); err != nil {
		return fmt.Errorf("资源配置验证失败: %w", err)
	}

	if err := s.dao.CreateCluster(ctx, cluster); err != nil {
		s.logger.Error("创建集群记录失败", zap.Error(err))
		return fmt.Errorf("创建集群记录失败: %w", err)
	}

	if err := s.clusterMgr.CreateCluster(ctx, cluster); err != nil {
		s.logger.Error("创建集群失败", zap.Error(err))
		if len(cluster.RestrictNamespace) > 0 && (cluster.CpuRequest != "" || cluster.CpuLimit != "" || cluster.MemoryRequest != "" || cluster.MemoryLimit != "") {
			_ = s.dao.UpdateClusterStatus(ctx, cluster.ID, model.StatusError)
			return fmt.Errorf("配额配置未完成，集群登记 %d 已保留为异常状态，请修正后更新或删除: %w", cluster.ID, err)
		}
		if rollbackErr := s.dao.DeleteCluster(ctx, cluster.ID); rollbackErr != nil {
			s.logger.Error("回滚失败", zap.Error(rollbackErr))
		}
		return fmt.Errorf("创建集群失败: %w", err)
	}

	return nil
}

func (s *clusterService) UpdateCluster(ctx context.Context, req *model.UpdateClusterReq) error {
	if req == nil {
		return fmt.Errorf("更新集群请求参数不能为空")
	}

	if err := utils.ValidateClusterUpdateParams(req.ID); err != nil {
		return err
	}

	existingCluster, err := s.dao.GetClusterByID(ctx, req.ID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("集群不存在，ID: %d", req.ID)
		}

		s.logger.Error("查询集群失败", zap.Error(err), zap.Int("clusterID", req.ID))
		return fmt.Errorf("查询集群失败: %w", err)
	}

	if req.Name != "" && req.Name != existingCluster.Name {
		conflictCluster, err := s.dao.GetClusterByName(ctx, req.Name)
		if err != nil && err != gorm.ErrRecordNotFound {
			s.logger.Error("查询集群名称冲突失败", zap.Error(err))
			return fmt.Errorf("查询集群名称冲突失败: %w", err)
		}
		if conflictCluster != nil {
			return fmt.Errorf("集群名称 %s 已存在", req.Name)
		}
	}

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

	if cluster.Name == "" {
		cluster.Name = existingCluster.Name
	}
	if cluster.ApiServerAddr == "" {
		cluster.ApiServerAddr = existingCluster.ApiServerAddr
	}
	if cluster.KubeConfigContent == "" {
		cluster.KubeConfigContent = existingCluster.KubeConfigContent
	}
	if cluster.Env == 0 {
		cluster.Env = existingCluster.Env
	}
	if cluster.Version == "" {
		cluster.Version = existingCluster.Version
	}
	if cluster.ActionTimeoutSeconds == 0 {
		cluster.ActionTimeoutSeconds = existingCluster.ActionTimeoutSeconds
	}
	if cluster.Tags == nil {
		cluster.Tags = existingCluster.Tags
	}
	if cluster.RestrictNamespace == nil {
		cluster.RestrictNamespace = existingCluster.RestrictNamespace
	}

	if req.RestrictNamespace == nil && req.CpuRequest == "" && req.CpuLimit == "" && req.MemoryRequest == "" && req.MemoryLimit == "" {
		cluster.RestrictNamespace = existingCluster.RestrictNamespace
		cluster.CpuRequest = existingCluster.CpuRequest
		cluster.CpuLimit = existingCluster.CpuLimit
		cluster.MemoryRequest = existingCluster.MemoryRequest
		cluster.MemoryLimit = existingCluster.MemoryLimit
	}
	oldPolicy := len(existingCluster.RestrictNamespace) > 0 && (existingCluster.CpuRequest != "" || existingCluster.CpuLimit != "" || existingCluster.MemoryRequest != "" || existingCluster.MemoryLimit != "")
	if oldPolicy {
		same, err := connection.SameCluster(existingCluster.KubeConfigContent, cluster.KubeConfigContent)
		if err != nil {
			return err
		}
		if !same {
			return fmt.Errorf("更换集群连接前请先清除原命名空间配额配置")
		}
	}

	if _, err := connection.Parse(cluster.KubeConfigContent); err != nil {
		return err
	}
	if err := utils.ValidateResourceQuantities(cluster); err != nil {
		return fmt.Errorf("资源配置验证失败: %w", err)
	}

	if err := s.dao.UpdateCluster(ctx, cluster); err != nil {
		s.logger.Error("更新集群失败", zap.Error(err), zap.Int("clusterID", cluster.ID))
		return fmt.Errorf("更新集群失败: %w", err)
	}

	if err := s.clusterMgr.UpdateCluster(ctx, cluster); err != nil {
		s.logger.Error("更新集群客户端失败", zap.Error(err))
		rollbackErr := s.dao.UpdateCluster(ctx, existingCluster)
		if rollbackErr != nil {
			return fmt.Errorf("更新集群失败: %v; 配置回滚失败: %w", err, rollbackErr)
		}
		s.client.RemoveCluster(cluster.ID)
		return fmt.Errorf("更新失败，已恢复登记配置；部分远端配额可能已生效，请重试: %w", err)
	}
	if oldPolicy {
		var removed []string
		active := cluster.CpuRequest != "" || cluster.CpuLimit != "" || cluster.MemoryRequest != "" || cluster.MemoryLimit != ""
		targets := map[string]bool{}
		if active {
			for _, ns := range cluster.RestrictNamespace {
				targets[ns] = true
			}
		}
		for _, ns := range existingCluster.RestrictNamespace {
			if !targets[ns] {
				removed = append(removed, ns)
			}
		}
		if len(removed) > 0 {
			client, err := s.client.GetKubeClient(cluster.ID)
			if err == nil {
				err = utils.RemoveClusterResourceLimits(ctx, client, cluster.ID, removed)
			}
			if err != nil {
				rollbackErr := s.dao.UpdateCluster(ctx, existingCluster)
				if rollbackErr != nil {
					return fmt.Errorf("清理配额失败: %v; 配置回滚失败: %w", err, rollbackErr)
				}
				s.client.RemoveCluster(cluster.ID)
				return fmt.Errorf("清理配额失败，已保留原登记配置；请重试完成剩余清理: %w", err)
			}
		}
	}
	return nil
}

func (s *clusterService) DeleteCluster(ctx context.Context, req *model.DeleteClusterReq) error {
	if req == nil {
		return fmt.Errorf("删除集群请求参数不能为空")
	}

	if err := utils.ValidateClusterID(req.ID); err != nil {
		return err
	}

	cluster, err := s.dao.GetClusterByID(ctx, req.ID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("集群不存在，ID: %d", req.ID)
		}

		s.logger.Error("查询集群失败", zap.Error(err), zap.Int("clusterID", req.ID))
		return fmt.Errorf("查询集群失败: %w", err)
	}

	if len(cluster.RestrictNamespace) > 0 && (cluster.CpuRequest != "" || cluster.CpuLimit != "" || cluster.MemoryRequest != "" || cluster.MemoryLimit != "") {
		client, err := s.client.GetKubeClient(req.ID)
		if err == nil {
			err = utils.RemoveClusterResourceLimits(ctx, client, req.ID, cluster.RestrictNamespace)
		}
		if err != nil {
			return fmt.Errorf("清理平台配额失败，集群登记仍保留: %w", err)
		}
	}
	s.client.RemoveCluster(req.ID)

	if err := s.dao.DeleteCluster(ctx, req.ID); err != nil {
		s.logger.Error("删除集群失败", zap.Error(err), zap.Int("clusterID", req.ID))
		return fmt.Errorf("删除集群失败: %w", err)
	}

	return nil
}

func (s *clusterService) RefreshClusterStatus(ctx context.Context, req *model.RefreshClusterReq) error {
	if req == nil {
		return fmt.Errorf("刷新集群状态请求参数不能为空")
	}

	if err := utils.ValidateClusterID(req.ID); err != nil {
		return err
	}

	_, err := s.dao.GetClusterByID(ctx, req.ID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("集群不存在，ID: %d", req.ID)
		}

		s.logger.Error("查询集群失败", zap.Error(err), zap.Int("clusterID", req.ID))
		return fmt.Errorf("查询集群失败: %w", err)
	}

	if err := s.clusterMgr.RefreshCluster(ctx, req.ID); err != nil {
		s.logger.Error("刷新集群状态失败", zap.Error(err))
		return fmt.Errorf("刷新集群状态失败: %w", err)
	}

	return nil
}
