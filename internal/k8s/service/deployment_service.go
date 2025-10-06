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
	"strings"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/manager"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/utils"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type DeploymentService interface {
	CreateDeployment(ctx context.Context, req *model.CreateDeploymentReq) error
	GetDeploymentList(ctx context.Context, req *model.GetDeploymentListReq) (model.ListResp[*model.K8sDeployment], error)
	GetDeploymentDetails(ctx context.Context, req *model.GetDeploymentDetailsReq) (*model.K8sDeployment, error)
	GetDeploymentYaml(ctx context.Context, req *model.GetDeploymentYamlReq) (*model.K8sYaml, error)
	UpdateDeployment(ctx context.Context, req *model.UpdateDeploymentReq) error
	DeleteDeployment(ctx context.Context, req *model.DeleteDeploymentReq) error
	CreateDeploymentByYaml(ctx context.Context, req *model.CreateDeploymentByYamlReq) error
	UpdateDeploymentByYaml(ctx context.Context, req *model.UpdateDeploymentByYamlReq) error
	RestartDeployment(ctx context.Context, req *model.RestartDeploymentReq) error
	ScaleDeployment(ctx context.Context, req *model.ScaleDeploymentReq) error
	RollbackDeployment(ctx context.Context, req *model.RollbackDeploymentReq) error
	PauseDeployment(ctx context.Context, req *model.PauseDeploymentReq) error
	ResumeDeployment(ctx context.Context, req *model.ResumeDeploymentReq) error
	GetDeploymentPods(ctx context.Context, req *model.GetDeploymentPodsReq) (model.ListResp[*model.K8sPod], error)
	GetDeploymentHistory(ctx context.Context, req *model.GetDeploymentHistoryReq) (model.ListResp[*model.K8sDeploymentHistory], error)
}

type deploymentService struct {
	deploymentManager manager.DeploymentManager
	logger            *zap.Logger
}

func NewDeploymentService(deploymentManager manager.DeploymentManager, logger *zap.Logger) DeploymentService {
	return &deploymentService{
		deploymentManager: deploymentManager,
		logger:            logger,
	}
}

func (s *deploymentService) CreateDeployment(ctx context.Context, req *model.CreateDeploymentReq) error {
	if req == nil {
		return fmt.Errorf("创建Deployment请求不能为空")
	}

	if req.ClusterID <= 0 {
		return fmt.Errorf("集群ID不能为空")
	}

	if req.Name == "" {
		return fmt.Errorf("Deployment名称不能为空")
	}

	if req.Namespace == "" {
		return fmt.Errorf("命名空间不能为空")
	}

	deployment, err := utils.BuildDeploymentFromRequest(req)
	if err != nil {
		s.logger.Error("构建Deployment对象失败",
			zap.Error(err),
			zap.String("name", req.Name))
		return fmt.Errorf("构建Deployment对象失败: %w", err)
	}

	if err := utils.ValidateDeployment(deployment); err != nil {
		s.logger.Error("Deployment配置验证失败",
			zap.Error(err),
			zap.String("name", req.Name))
		return fmt.Errorf("deployment配置验证失败: %w", err)
	}

	err = s.deploymentManager.CreateDeployment(ctx, req.ClusterID, req.Namespace, deployment)
	if err != nil {
		s.logger.Error("创建Deployment失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("创建Deployment失败: %w", err)
	}

	return nil
}

func (s *deploymentService) DeleteDeployment(ctx context.Context, req *model.DeleteDeploymentReq) error {
	if req == nil {
		return fmt.Errorf("删除Deployment请求不能为空")
	}

	if req.ClusterID <= 0 {
		return fmt.Errorf("集群ID不能为空")
	}

	if req.Namespace == "" {
		return fmt.Errorf("命名空间不能为空")
	}

	if req.Name == "" {
		return fmt.Errorf("Deployment名称不能为空")
	}

	err := s.deploymentManager.DeleteDeployment(ctx, req.ClusterID, req.Namespace, req.Name, metav1.DeleteOptions{})
	if err != nil {
		s.logger.Error("删除Deployment失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("删除Deployment失败: %w", err)
	}

	return nil
}

func (s *deploymentService) GetDeploymentDetails(ctx context.Context, req *model.GetDeploymentDetailsReq) (*model.K8sDeployment, error) {
	if req == nil {
		return nil, fmt.Errorf("获取Deployment详情请求不能为空")
	}

	if req.ClusterID <= 0 {
		return nil, fmt.Errorf("集群ID不能为空")
	}

	if req.Namespace == "" {
		return nil, fmt.Errorf("命名空间不能为空")
	}

	if req.Name == "" {
		return nil, fmt.Errorf("Deployment名称不能为空")
	}

	deployment, err := s.deploymentManager.GetDeployment(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		s.logger.Error("获取Deployment失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return nil, fmt.Errorf("获取Deployment失败: %w", err)
	}

	k8sDeployment, err := utils.BuildK8sDeployment(ctx, req.ClusterID, *deployment)
	if err != nil {
		s.logger.Error("构建Deployment详细信息失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return nil, fmt.Errorf("构建Deployment详细信息失败: %w", err)
	}

	return k8sDeployment, nil
}

func (s *deploymentService) GetDeploymentHistory(ctx context.Context, req *model.GetDeploymentHistoryReq) (model.ListResp[*model.K8sDeploymentHistory], error) {
	if req == nil {
		return model.ListResp[*model.K8sDeploymentHistory]{}, fmt.Errorf("获取Deployment历史请求不能为空")
	}

	if req.ClusterID <= 0 {
		return model.ListResp[*model.K8sDeploymentHistory]{}, fmt.Errorf("集群ID不能为空")
	}

	if req.Namespace == "" {
		return model.ListResp[*model.K8sDeploymentHistory]{}, fmt.Errorf("命名空间不能为空")
	}

	if req.Name == "" {
		return model.ListResp[*model.K8sDeploymentHistory]{}, fmt.Errorf("Deployment名称不能为空")
	}

	history, total, err := s.deploymentManager.GetDeploymentHistory(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		s.logger.Error("获取部署历史失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return model.ListResp[*model.K8sDeploymentHistory]{}, fmt.Errorf("获取部署历史失败: %w", err)
	}

	return model.ListResp[*model.K8sDeploymentHistory]{
		Total: total,
		Items: history,
	}, nil
}

func (s *deploymentService) GetDeploymentList(ctx context.Context, req *model.GetDeploymentListReq) (model.ListResp[*model.K8sDeployment], error) {
	if req == nil {
		return model.ListResp[*model.K8sDeployment]{}, fmt.Errorf("获取Deployment列表请求不能为空")
	}

	if req.ClusterID <= 0 {
		return model.ListResp[*model.K8sDeployment]{}, fmt.Errorf("集群ID不能为空")
	}

	listOptions := utils.BuildDeploymentListOptions(req)

	k8sDeployments, err := s.deploymentManager.GetDeploymentList(ctx, req.ClusterID, req.Namespace, listOptions)
	if err != nil {
		s.logger.Error("获取Deployment列表失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace))
		return model.ListResp[*model.K8sDeployment]{}, fmt.Errorf("获取Deployment列表失败: %w", err)
	}

	// 应用过滤条件
	var filteredDeployments []*model.K8sDeployment
	for _, k8sDeployment := range k8sDeployments {
		// 状态过滤
		if req.Status != "" {
			var statusStr string
			switch k8sDeployment.Status {
			case model.K8sDeploymentStatusRunning:
				statusStr = "running"
			case model.K8sDeploymentStatusStopped:
				statusStr = "stopped"
			case model.K8sDeploymentStatusPaused:
				statusStr = "paused"
			case model.K8sDeploymentStatusError:
				statusStr = "error"
			default:
				statusStr = "unknown"
			}
			if !strings.EqualFold(statusStr, req.Status) {
				continue
			}
		}
		// 名称过滤（使用通用的Search字段，支持不区分大小写）
		if !utils.FilterByName(k8sDeployment.Name, req.Search) {
			continue
		}
		filteredDeployments = append(filteredDeployments, k8sDeployment)
	}

	// 按创建时间排序（最新的在前）
	utils.SortByCreationTime(filteredDeployments, func(deployment *model.K8sDeployment) time.Time {
		return deployment.CreatedAt
	})

	// 分页处理
	pagedItems, total := utils.Paginate(filteredDeployments, req.Page, req.Size)

	return model.ListResp[*model.K8sDeployment]{
		Total: total,
		Items: pagedItems,
	}, nil
}

func (s *deploymentService) GetDeploymentPods(ctx context.Context, req *model.GetDeploymentPodsReq) (model.ListResp[*model.K8sPod], error) {
	if req == nil {
		return model.ListResp[*model.K8sPod]{}, fmt.Errorf("获取Deployment Pods请求不能为空")
	}

	if req.ClusterID <= 0 {
		return model.ListResp[*model.K8sPod]{}, fmt.Errorf("集群ID不能为空")
	}

	if req.Namespace == "" {
		return model.ListResp[*model.K8sPod]{}, fmt.Errorf("命名空间不能为空")
	}

	if req.Name == "" {
		return model.ListResp[*model.K8sPod]{}, fmt.Errorf("Deployment名称不能为空")
	}

	pods, total, err := s.deploymentManager.GetDeploymentPods(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		s.logger.Error("获取部署Pod失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return model.ListResp[*model.K8sPod]{}, fmt.Errorf("获取部署Pod失败: %w", err)
	}

	return model.ListResp[*model.K8sPod]{
		Total: total,
		Items: pods,
	}, nil
}

func (s *deploymentService) GetDeploymentYaml(ctx context.Context, req *model.GetDeploymentYamlReq) (*model.K8sYaml, error) {
	if req == nil {
		return nil, fmt.Errorf("获取Deployment YAML请求不能为空")
	}

	if req.ClusterID <= 0 {
		return nil, fmt.Errorf("集群ID不能为空")
	}

	if req.Namespace == "" {
		return nil, fmt.Errorf("命名空间不能为空")
	}

	if req.Name == "" {
		return nil, fmt.Errorf("Deployment名称不能为空")
	}

	deployment, err := s.deploymentManager.GetDeployment(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		s.logger.Error("获取Deployment失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return nil, fmt.Errorf("获取Deployment失败: %w", err)
	}

	yamlContent, err := utils.DeploymentToYAML(deployment)
	if err != nil {
		s.logger.Error("转换为YAML失败",
			zap.Error(err),
			zap.String("deploymentName", deployment.Name))
		return nil, fmt.Errorf("转换为YAML失败: %w", err)
	}

	return &model.K8sYaml{
		YAML: yamlContent,
	}, nil
}

func (s *deploymentService) RestartDeployment(ctx context.Context, req *model.RestartDeploymentReq) error {
	if req == nil {
		return fmt.Errorf("重启Deployment请求不能为空")
	}

	if req.ClusterID <= 0 {
		return fmt.Errorf("集群ID不能为空")
	}

	if req.Namespace == "" {
		return fmt.Errorf("命名空间不能为空")
	}

	if req.Name == "" {
		return fmt.Errorf("Deployment名称不能为空")
	}

	err := s.deploymentManager.RestartDeployment(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		s.logger.Error("重启Deployment失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("重启Deployment失败: %w", err)
	}

	return nil
}

func (s *deploymentService) RollbackDeployment(ctx context.Context, req *model.RollbackDeploymentReq) error {
	if req == nil {
		return fmt.Errorf("回滚Deployment请求不能为空")
	}

	if req.ClusterID <= 0 {
		return fmt.Errorf("集群ID不能为空")
	}

	if req.Namespace == "" {
		return fmt.Errorf("命名空间不能为空")
	}

	if req.Name == "" {
		return fmt.Errorf("Deployment名称不能为空")
	}

	if req.Revision <= 0 {
		return fmt.Errorf("回滚版本号必须大于0")
	}

	err := s.deploymentManager.RollbackDeployment(ctx, req.ClusterID, req.Namespace, req.Name, req.Revision)
	if err != nil {
		s.logger.Error("回滚Deployment失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name),
			zap.Int64("revision", req.Revision))
		return fmt.Errorf("回滚Deployment失败: %w", err)
	}

	return nil
}

func (s *deploymentService) ScaleDeployment(ctx context.Context, req *model.ScaleDeploymentReq) error {
	if req == nil {
		return fmt.Errorf("扩缩容Deployment请求不能为空")
	}

	if req.ClusterID <= 0 {
		return fmt.Errorf("集群ID不能为空")
	}

	if req.Namespace == "" {
		return fmt.Errorf("命名空间不能为空")
	}

	if req.Name == "" {
		return fmt.Errorf("Deployment名称不能为空")
	}

	if req.Replicas < 0 {
		return fmt.Errorf("副本数不能为负数")
	}

	err := s.deploymentManager.ScaleDeployment(ctx, req.ClusterID, req.Namespace, req.Name, req.Replicas)
	if err != nil {
		s.logger.Error("扩缩容Deployment失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name),
			zap.Int32("replicas", req.Replicas))
		return fmt.Errorf("扩缩容Deployment失败: %w", err)
	}

	return nil
}

func (s *deploymentService) UpdateDeployment(ctx context.Context, req *model.UpdateDeploymentReq) error {
	if req == nil {
		return fmt.Errorf("更新Deployment请求不能为空")
	}

	if req.ClusterID <= 0 {
		return fmt.Errorf("集群ID不能为空")
	}

	if req.Name == "" {
		return fmt.Errorf("Deployment名称不能为空")
	}

	if req.Namespace == "" {
		return fmt.Errorf("命名空间不能为空")
	}

	existingDeployment, err := s.deploymentManager.GetDeployment(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		s.logger.Error("获取现有Deployment失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("获取现有Deployment失败: %w", err)
	}

	updatedDeployment := existingDeployment.DeepCopy()

	// 更新基本字段
	if req.Replicas != nil {
		updatedDeployment.Spec.Replicas = req.Replicas
	}
	if len(req.Images) > 0 {
		for i, image := range req.Images {
			if i < len(updatedDeployment.Spec.Template.Spec.Containers) {
				updatedDeployment.Spec.Template.Spec.Containers[i].Image = image
			}
		}
	}
	if req.Labels != nil {
		// 合并标签到对象级别
		if updatedDeployment.Labels == nil {
			updatedDeployment.Labels = make(map[string]string)
		}
		for k, v := range req.Labels {
			updatedDeployment.Labels[k] = v
		}

		// 关键：更新 template labels 时必须保留 selector 的标签
		// K8s要求：Pod template的labels必须包含selector中的所有标签
		// 否则会导致"selector does not match template labels"错误
		if updatedDeployment.Spec.Template.Labels == nil {
			updatedDeployment.Spec.Template.Labels = make(map[string]string)
		}

		// 先添加用户指定的标签
		for k, v := range req.Labels {
			updatedDeployment.Spec.Template.Labels[k] = v
		}

		// 强制保留 selector 的标签（selector 是不可变的，必须匹配）
		if updatedDeployment.Spec.Selector != nil && updatedDeployment.Spec.Selector.MatchLabels != nil {
			for k, v := range updatedDeployment.Spec.Selector.MatchLabels {
				updatedDeployment.Spec.Template.Labels[k] = v
			}
		}
	}
	if req.Annotations != nil {
		updatedDeployment.Annotations = req.Annotations
	}

	if err := utils.ValidateDeployment(updatedDeployment); err != nil {
		s.logger.Error("Deployment配置验证失败",
			zap.Error(err),
			zap.String("name", req.Name))
		return fmt.Errorf("deployment配置验证失败: %w", err)
	}

	err = s.deploymentManager.UpdateDeployment(ctx, req.ClusterID, req.Namespace, updatedDeployment)
	if err != nil {
		s.logger.Error("更新Deployment失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("更新Deployment失败: %w", err)
	}

	return nil
}

func (s *deploymentService) PauseDeployment(ctx context.Context, req *model.PauseDeploymentReq) error {
	if req == nil {
		return fmt.Errorf("暂停Deployment请求不能为空")
	}

	if req.ClusterID <= 0 {
		return fmt.Errorf("集群ID不能为空")
	}

	if req.Namespace == "" {
		return fmt.Errorf("命名空间不能为空")
	}

	if req.Name == "" {
		return fmt.Errorf("Deployment名称不能为空")
	}

	err := s.deploymentManager.PauseDeployment(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		s.logger.Error("暂停Deployment失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("暂停Deployment失败: %w", err)
	}

	return nil
}

func (s *deploymentService) ResumeDeployment(ctx context.Context, req *model.ResumeDeploymentReq) error {
	if req == nil {
		return fmt.Errorf("恢复Deployment请求不能为空")
	}

	if req.ClusterID <= 0 {
		return fmt.Errorf("集群ID不能为空")
	}

	if req.Namespace == "" {
		return fmt.Errorf("命名空间不能为空")
	}

	if req.Name == "" {
		return fmt.Errorf("Deployment名称不能为空")
	}

	err := s.deploymentManager.ResumeDeployment(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		s.logger.Error("恢复Deployment失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("恢复Deployment失败: %w", err)
	}

	return nil
}

func (s *deploymentService) CreateDeploymentByYaml(ctx context.Context, req *model.CreateDeploymentByYamlReq) error {
	if req == nil {
		return fmt.Errorf("通过YAML创建Deployment请求不能为空")
	}

	if req.YAML == "" {
		return fmt.Errorf("YAML内容不能为空")
	}

	s.logger.Info("开始通过YAML创建Deployment",
		zap.Int("cluster_id", req.ClusterID))

	deployment, err := utils.BuildDeploymentFromYaml(req)
	if err != nil {
		s.logger.Error("从YAML构建Deployment失败",
			zap.Int("cluster_id", req.ClusterID),
			zap.Error(err))
		return fmt.Errorf("从YAML构建Deployment失败: %w", err)
	}

	err = s.deploymentManager.CreateDeployment(ctx, req.ClusterID, deployment.Namespace, deployment)
	if err != nil {
		s.logger.Error("通过YAML创建Deployment失败",
			zap.Int("cluster_id", req.ClusterID),
			zap.String("namespace", deployment.Namespace),
			zap.String("name", deployment.Name),
			zap.Error(err))
		return fmt.Errorf("通过YAML创建Deployment失败: %w", err)
	}

	s.logger.Info("通过YAML创建Deployment成功",
		zap.Int("cluster_id", req.ClusterID))

	return nil
}

func (s *deploymentService) UpdateDeploymentByYaml(ctx context.Context, req *model.UpdateDeploymentByYamlReq) error {
	if req == nil {
		return fmt.Errorf("通过YAML更新Deployment请求不能为空")
	}

	if req.YAML == "" {
		return fmt.Errorf("YAML内容不能为空")
	}

	s.logger.Info("开始通过YAML更新Deployment",
		zap.Int("cluster_id", req.ClusterID),
		zap.String("namespace", req.Namespace),
		zap.String("name", req.Name))

	deployment, err := utils.BuildDeploymentFromYamlForUpdate(req)
	if err != nil {
		s.logger.Error("从YAML构建Deployment失败",
			zap.Int("cluster_id", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name),
			zap.Error(err))
		return fmt.Errorf("从YAML构建Deployment失败: %w", err)
	}

	err = s.deploymentManager.UpdateDeployment(ctx, req.ClusterID, req.Namespace, deployment)
	if err != nil {
		s.logger.Error("通过YAML更新Deployment失败",
			zap.Int("cluster_id", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name),
			zap.Error(err))
		return fmt.Errorf("通过YAML更新Deployment失败: %w", err)
	}

	s.logger.Info("通过YAML更新Deployment成功",
		zap.Int("cluster_id", req.ClusterID),
		zap.String("namespace", req.Namespace),
		zap.String("name", req.Name))

	return nil
}
