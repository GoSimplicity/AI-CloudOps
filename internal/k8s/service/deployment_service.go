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

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/manager"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/utils"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DeploymentService Deployment业务服务接口
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

// deploymentService Deployment业务服务实现
type deploymentService struct {
	deploymentManager manager.DeploymentManager
	logger            *zap.Logger
}

// NewDeploymentService 创建新的Deployment业务服务实例
func NewDeploymentService(deploymentManager manager.DeploymentManager, logger *zap.Logger) DeploymentService {
	return &deploymentService{
		deploymentManager: deploymentManager,
		logger:            logger,
	}
}

// CreateDeployment 创建deployment
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

	// 从请求构建Deployment对象
	deployment, err := utils.BuildDeploymentFromRequest(req)
	if err != nil {
		s.logger.Error("CreateDeployment: 构建Deployment对象失败",
			zap.Error(err),
			zap.String("name", req.Name))
		return fmt.Errorf("构建Deployment对象失败: %w", err)
	}

	// 验证Deployment配置
	if err := utils.ValidateDeployment(deployment); err != nil {
		s.logger.Error("CreateDeployment: Deployment配置验证失败",
			zap.Error(err),
			zap.String("name", req.Name))
		return fmt.Errorf("deployment配置验证失败: %w", err)
	}

	err = s.deploymentManager.CreateDeployment(ctx, req.ClusterID, req.Namespace, deployment)
	if err != nil {
		s.logger.Error("CreateDeployment: 创建Deployment失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("创建Deployment失败: %w", err)
	}

	return nil
}

// DeleteDeployment 删除deployment
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
		s.logger.Error("DeleteDeployment: 删除Deployment失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("删除Deployment失败: %w", err)
	}

	return nil
}

// GetDeploymentDetails 获取deployment详情
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
		s.logger.Error("GetDeploymentDetails: 获取Deployment失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return nil, fmt.Errorf("获取Deployment失败: %w", err)
	}

	// 构建详细信息
	k8sDeployment, err := utils.BuildK8sDeployment(ctx, req.ClusterID, *deployment)
	if err != nil {
		s.logger.Error("GetDeploymentDetails: 构建Deployment详细信息失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return nil, fmt.Errorf("构建Deployment详细信息失败: %w", err)
	}

	return k8sDeployment, nil
}

// GetDeploymentHistory 获取deployment历史
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
		s.logger.Error("GetDeploymentHistory: 获取部署历史失败",
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

// GetDeploymentList 获取deployment列表
func (s *deploymentService) GetDeploymentList(ctx context.Context, req *model.GetDeploymentListReq) (model.ListResp[*model.K8sDeployment], error) {
	if req == nil {
		return model.ListResp[*model.K8sDeployment]{}, fmt.Errorf("获取Deployment列表请求不能为空")
	}

	if req.ClusterID <= 0 {
		return model.ListResp[*model.K8sDeployment]{}, fmt.Errorf("集群ID不能为空")
	}

	// 构建查询选项
	listOptions := utils.BuildDeploymentListOptions(req)

	k8sDeployments, err := s.deploymentManager.GetDeploymentList(ctx, req.ClusterID, req.Namespace, listOptions)
	if err != nil {
		s.logger.Error("GetDeploymentList: 获取Deployment列表失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace))
		return model.ListResp[*model.K8sDeployment]{}, fmt.Errorf("获取Deployment列表失败: %w", err)
	}

	// 根据状态过滤
	var filteredDeployments []*model.K8sDeployment
	if req.Status != "" {
		// 根据状态过滤
		for _, k8sDeployment := range k8sDeployments {
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
			if strings.EqualFold(statusStr, req.Status) {
				filteredDeployments = append(filteredDeployments, k8sDeployment)
			}
		}
	} else {
		filteredDeployments = k8sDeployments
	}

	// 分页处理
	page := req.Page
	size := req.Size
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 10 // 默认每页显示10条
	}

	pagedItems, total := utils.PaginateK8sDeployments(filteredDeployments, page, size)

	return model.ListResp[*model.K8sDeployment]{
		Total: total,
		Items: pagedItems,
	}, nil
}

// GetDeploymentPods 获取deployment的pod列表
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
		s.logger.Error("GetDeploymentPods: 获取部署Pod失败",
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

// GetDeploymentYaml 获取deployment YAML
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
		s.logger.Error("GetDeploymentYaml: 获取Deployment失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return nil, fmt.Errorf("获取Deployment失败: %w", err)
	}

	// 转换为YAML
	yamlContent, err := utils.DeploymentToYAML(deployment)
	if err != nil {
		s.logger.Error("GetDeploymentYaml: 转换为YAML失败",
			zap.Error(err),
			zap.String("deploymentName", deployment.Name))
		return nil, fmt.Errorf("转换为YAML失败: %w", err)
	}

	return &model.K8sYaml{
		YAML: yamlContent,
	}, nil
}

// RestartDeployment 重启deployment
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
		s.logger.Error("RestartDeployment: 重启Deployment失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("重启Deployment失败: %w", err)
	}

	return nil
}

// RollbackDeployment 回滚Deployment
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
		s.logger.Error("RollbackDeployment: 回滚Deployment失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name),
			zap.Int64("revision", req.Revision))
		return fmt.Errorf("回滚Deployment失败: %w", err)
	}

	return nil
}

// ScaleDeployment 扩缩容deployment
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
		s.logger.Error("ScaleDeployment: 扩缩容Deployment失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name),
			zap.Int32("replicas", req.Replicas))
		return fmt.Errorf("扩缩容Deployment失败: %w", err)
	}

	return nil
}

// UpdateDeployment 更新deployment
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
		s.logger.Error("UpdateDeployment: 获取现有Deployment失败",
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

		// 更新 template labels，确保包含 selector 中的所有必需标签
		if updatedDeployment.Spec.Template.Labels == nil {
			updatedDeployment.Spec.Template.Labels = make(map[string]string)
		}

		// 先添加用户指定的标签
		for k, v := range req.Labels {
			updatedDeployment.Spec.Template.Labels[k] = v
		}

		// 然后强制保留 selector 的标签（selector 是不可变的，必须匹配）
		if updatedDeployment.Spec.Selector != nil && updatedDeployment.Spec.Selector.MatchLabels != nil {
			for k, v := range updatedDeployment.Spec.Selector.MatchLabels {
				updatedDeployment.Spec.Template.Labels[k] = v
			}
		}
	}
	if req.Annotations != nil {
		updatedDeployment.Annotations = req.Annotations
	}

	// 验证更新后的Deployment配置
	if err := utils.ValidateDeployment(updatedDeployment); err != nil {
		s.logger.Error("UpdateDeployment: Deployment配置验证失败",
			zap.Error(err),
			zap.String("name", req.Name))
		return fmt.Errorf("deployment配置验证失败: %w", err)
	}

	err = s.deploymentManager.UpdateDeployment(ctx, req.ClusterID, req.Namespace, updatedDeployment)
	if err != nil {
		s.logger.Error("UpdateDeployment: 更新Deployment失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("更新Deployment失败: %w", err)
	}

	return nil
}

// PauseDeployment 暂停Deployment
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
		s.logger.Error("PauseDeployment: 暂停Deployment失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("暂停Deployment失败: %w", err)
	}

	return nil
}

// ResumeDeployment 恢复Deployment
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
		s.logger.Error("ResumeDeployment: 恢复Deployment失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("恢复Deployment失败: %w", err)
	}

	return nil
}

// CreateDeploymentByYaml 通过YAML创建deployment
func (s *deploymentService) CreateDeploymentByYaml(ctx context.Context, req *model.CreateDeploymentByYamlReq) error {
	if req == nil {
		return fmt.Errorf("通过YAML创建Deployment请求不能为空")
	}

	if req.YAML == "" {
		return fmt.Errorf("YAML内容不能为空")
	}

	s.logger.Info("开始通过YAML创建Deployment",
		zap.Int("cluster_id", req.ClusterID))

	// 从YAML构建Deployment对象
	deployment, err := utils.BuildDeploymentFromYaml(req)
	if err != nil {
		s.logger.Error("从YAML构建Deployment失败",
			zap.Int("cluster_id", req.ClusterID),
			zap.Error(err))
		return fmt.Errorf("从YAML构建Deployment失败: %w", err)
	}

	// 使用现有的创建方法
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

// UpdateDeploymentByYaml 通过YAML更新deployment
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

	// 从YAML构建Deployment对象
	deployment, err := utils.BuildDeploymentFromYamlForUpdate(req)
	if err != nil {
		s.logger.Error("从YAML构建Deployment失败",
			zap.Int("cluster_id", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name),
			zap.Error(err))
		return fmt.Errorf("从YAML构建Deployment失败: %w", err)
	}

	// 使用现有的更新方法
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
