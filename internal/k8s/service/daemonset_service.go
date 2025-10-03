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

type DaemonSetService interface {
	GetDaemonSetList(ctx context.Context, req *model.GetDaemonSetListReq) (model.ListResp[*model.K8sDaemonSet], error)
	GetDaemonSetDetails(ctx context.Context, req *model.GetDaemonSetDetailsReq) (*model.K8sDaemonSet, error)
	GetDaemonSetYaml(ctx context.Context, req *model.GetDaemonSetYamlReq) (*model.K8sYaml, error)
	CreateDaemonSet(ctx context.Context, req *model.CreateDaemonSetReq) error
	CreateDaemonSetByYaml(ctx context.Context, req *model.CreateDaemonSetByYamlReq) error
	UpdateDaemonSet(ctx context.Context, req *model.UpdateDaemonSetReq) error
	UpdateDaemonSetByYaml(ctx context.Context, req *model.UpdateDaemonSetByYamlReq) error
	DeleteDaemonSet(ctx context.Context, req *model.DeleteDaemonSetReq) error
	RestartDaemonSet(ctx context.Context, req *model.RestartDaemonSetReq) error
	GetDaemonSetPods(ctx context.Context, req *model.GetDaemonSetPodsReq) (model.ListResp[*model.K8sPod], error)
	GetDaemonSetHistory(ctx context.Context, req *model.GetDaemonSetHistoryReq) (model.ListResp[*model.K8sDaemonSetHistory], error)
	RollbackDaemonSet(ctx context.Context, req *model.RollbackDaemonSetReq) error
}

type daemonSetService struct {
	daemonSetManager manager.DaemonSetManager
	logger           *zap.Logger
}

func NewDaemonSetService(daemonSetManager manager.DaemonSetManager, logger *zap.Logger) DaemonSetService {
	return &daemonSetService{
		daemonSetManager: daemonSetManager,
		logger:           logger,
	}
}

// CreateDaemonSet 创建DaemonSet
func (s *daemonSetService) CreateDaemonSet(ctx context.Context, req *model.CreateDaemonSetReq) error {
	if req == nil {
		return fmt.Errorf("创建DaemonSet请求不能为空")
	}

	if req.ClusterID <= 0 {
		return fmt.Errorf("集群ID不能为空")
	}

	if req.Name == "" {
		return fmt.Errorf("DaemonSet名称不能为空")
	}

	if req.Namespace == "" {
		return fmt.Errorf("命名空间不能为空")
	}

	// 从请求构建DaemonSet对象
	daemonSet, err := utils.BuildDaemonSetFromRequest(req)
	if err != nil {
		s.logger.Error("CreateDaemonSet: 构建DaemonSet对象失败",
			zap.Error(err),
			zap.String("name", req.Name))
		return fmt.Errorf("构建DaemonSet对象失败: %w", err)
	}

	// 验证DaemonSet配置
	if err := utils.ValidateDaemonSet(daemonSet); err != nil {
		s.logger.Error("CreateDaemonSet: DaemonSet配置验证失败",
			zap.Error(err),
			zap.String("name", req.Name))
		return fmt.Errorf("daemonSet配置验证失败: %w", err)
	}

	err = s.daemonSetManager.CreateDaemonSet(ctx, req.ClusterID, req.Namespace, daemonSet)
	if err != nil {
		s.logger.Error("CreateDaemonSet: 创建DaemonSet失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("创建DaemonSet失败: %w", err)
	}

	return nil
}

// CreateDaemonSetByYaml 通过YAML创建DaemonSet
func (s *daemonSetService) CreateDaemonSetByYaml(ctx context.Context, req *model.CreateDaemonSetByYamlReq) error {
	if req == nil {
		return fmt.Errorf("通过YAML创建DaemonSet请求不能为空")
	}

	if req.ClusterID <= 0 {
		return fmt.Errorf("集群ID不能为空")
	}

	if req.YAML == "" {
		return fmt.Errorf("YAML内容不能为空")
	}

	s.logger.Info("开始通过YAML创建DaemonSet",
		zap.Int("clusterID", req.ClusterID))

	// 从YAML构建DaemonSet对象
	daemonSet, err := utils.BuildDaemonSetFromYaml(req)
	if err != nil {
		s.logger.Error("从YAML构建DaemonSet失败",
			zap.Int("clusterID", req.ClusterID),
			zap.Error(err))
		return fmt.Errorf("从YAML构建DaemonSet失败: %w", err)
	}

	// 验证DaemonSet配置
	if err := utils.ValidateDaemonSet(daemonSet); err != nil {
		s.logger.Error("CreateDaemonSetByYaml: DaemonSet配置验证失败",
			zap.Error(err),
			zap.String("name", daemonSet.Name))
		return fmt.Errorf("daemonSet配置验证失败: %w", err)
	}

	// 使用现有的创建方法
	err = s.daemonSetManager.CreateDaemonSet(ctx, req.ClusterID, daemonSet.Namespace, daemonSet)
	if err != nil {
		s.logger.Error("通过YAML创建DaemonSet失败",
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", daemonSet.Namespace),
			zap.String("name", daemonSet.Name),
			zap.Error(err))
		return fmt.Errorf("通过YAML创建DaemonSet失败: %w", err)
	}

	s.logger.Info("通过YAML创建DaemonSet成功",
		zap.Int("clusterID", req.ClusterID),
		zap.String("namespace", daemonSet.Namespace),
		zap.String("name", daemonSet.Name))

	return nil
}

// DeleteDaemonSet 删除DaemonSet
func (s *daemonSetService) DeleteDaemonSet(ctx context.Context, req *model.DeleteDaemonSetReq) error {
	if req == nil {
		return fmt.Errorf("删除DaemonSet请求不能为空")
	}

	if req.ClusterID <= 0 {
		return fmt.Errorf("集群ID不能为空")
	}

	if req.Namespace == "" {
		return fmt.Errorf("命名空间不能为空")
	}

	if req.Name == "" {
		return fmt.Errorf("DaemonSet名称不能为空")
	}

	err := s.daemonSetManager.DeleteDaemonSet(ctx, req.ClusterID, req.Namespace, req.Name, metav1.DeleteOptions{})
	if err != nil {
		s.logger.Error("DeleteDaemonSet: 删除DaemonSet失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("删除DaemonSet失败: %w", err)
	}

	return nil
}

// GetDaemonSetDetails 获取DaemonSet详情
func (s *daemonSetService) GetDaemonSetDetails(ctx context.Context, req *model.GetDaemonSetDetailsReq) (*model.K8sDaemonSet, error) {
	if req == nil {
		return nil, fmt.Errorf("获取DaemonSet详情请求不能为空")
	}

	if req.ClusterID <= 0 {
		return nil, fmt.Errorf("集群ID不能为空")
	}

	if req.Namespace == "" {
		return nil, fmt.Errorf("命名空间不能为空")
	}

	if req.Name == "" {
		return nil, fmt.Errorf("DaemonSet名称不能为空")
	}

	daemonSet, err := s.daemonSetManager.GetDaemonSet(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		s.logger.Error("GetDaemonSetDetails: 获取DaemonSet失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return nil, fmt.Errorf("获取DaemonSet失败: %w", err)
	}

	// 构建详细信息
	k8sDaemonSet, err := utils.BuildK8sDaemonSet(ctx, req.ClusterID, *daemonSet)
	if err != nil {
		s.logger.Error("GetDaemonSetDetails: 构建DaemonSet详细信息失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return nil, fmt.Errorf("构建DaemonSet详细信息失败: %w", err)
	}

	return k8sDaemonSet, nil
}

// GetDaemonSetHistory 获取DaemonSet版本历史
func (s *daemonSetService) GetDaemonSetHistory(ctx context.Context, req *model.GetDaemonSetHistoryReq) (model.ListResp[*model.K8sDaemonSetHistory], error) {
	if req == nil {
		return model.ListResp[*model.K8sDaemonSetHistory]{}, fmt.Errorf("获取DaemonSet历史请求不能为空")
	}

	if req.ClusterID <= 0 {
		return model.ListResp[*model.K8sDaemonSetHistory]{}, fmt.Errorf("集群ID不能为空")
	}

	if req.Namespace == "" {
		return model.ListResp[*model.K8sDaemonSetHistory]{}, fmt.Errorf("命名空间不能为空")
	}

	if req.Name == "" {
		return model.ListResp[*model.K8sDaemonSetHistory]{}, fmt.Errorf("DaemonSet名称不能为空")
	}

	history, total, err := s.daemonSetManager.GetDaemonSetHistory(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		s.logger.Error("GetDaemonSetHistory: 获取DaemonSet历史失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return model.ListResp[*model.K8sDaemonSetHistory]{}, fmt.Errorf("获取DaemonSet历史失败: %w", err)
	}

	return model.ListResp[*model.K8sDaemonSetHistory]{
		Total: total,
		Items: history,
	}, nil
}

// GetDaemonSetList 获取DaemonSet列表
func (s *daemonSetService) GetDaemonSetList(ctx context.Context, req *model.GetDaemonSetListReq) (model.ListResp[*model.K8sDaemonSet], error) {
	if req == nil {
		return model.ListResp[*model.K8sDaemonSet]{}, fmt.Errorf("获取DaemonSet列表请求不能为空")
	}

	if req.ClusterID <= 0 {
		return model.ListResp[*model.K8sDaemonSet]{}, fmt.Errorf("集群ID不能为空")
	}

	// 构建查询选项
	listOptions := utils.BuildDaemonSetListOptions(req)

	k8sDaemonSets, err := s.daemonSetManager.GetDaemonSetList(ctx, req.ClusterID, req.Namespace, listOptions)
	if err != nil {
		s.logger.Error("GetDaemonSetList: 获取DaemonSet列表失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace))
		return model.ListResp[*model.K8sDaemonSet]{}, fmt.Errorf("获取DaemonSet列表失败: %w", err)
	}

	// 根据状态过滤
	var filteredDaemonSets []*model.K8sDaemonSet
	if req.Status != "" {
		// 根据状态过滤
		for _, k8sDaemonSet := range k8sDaemonSets {
			var statusStr string
			switch k8sDaemonSet.Status {
			case model.K8sDaemonSetStatusRunning:
				statusStr = "running"
			case model.K8sDaemonSetStatusError:
				statusStr = "error"
			case model.K8sDaemonSetStatusUpdating:
				statusStr = "updating"
			default:
				statusStr = "unknown"
			}
			if strings.EqualFold(statusStr, req.Status) {
				filteredDaemonSets = append(filteredDaemonSets, k8sDaemonSet)
			}
		}
	} else {
		filteredDaemonSets = k8sDaemonSets
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

	pagedItems, total := utils.PaginateK8sDaemonSets(filteredDaemonSets, page, size)

	return model.ListResp[*model.K8sDaemonSet]{
		Total: total,
		Items: pagedItems,
	}, nil
}

// GetDaemonSetPods 获取DaemonSet下的Pod列表
func (s *daemonSetService) GetDaemonSetPods(ctx context.Context, req *model.GetDaemonSetPodsReq) (model.ListResp[*model.K8sPod], error) {
	if req == nil {
		return model.ListResp[*model.K8sPod]{}, fmt.Errorf("获取DaemonSet Pods请求不能为空")
	}

	if req.ClusterID <= 0 {
		return model.ListResp[*model.K8sPod]{}, fmt.Errorf("集群ID不能为空")
	}

	if req.Namespace == "" {
		return model.ListResp[*model.K8sPod]{}, fmt.Errorf("命名空间不能为空")
	}

	if req.Name == "" {
		return model.ListResp[*model.K8sPod]{}, fmt.Errorf("DaemonSet名称不能为空")
	}

	pods, total, err := s.daemonSetManager.GetDaemonSetPods(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		s.logger.Error("GetDaemonSetPods: 获取DaemonSet Pod失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return model.ListResp[*model.K8sPod]{}, fmt.Errorf("获取DaemonSet Pod失败: %w", err)
	}

	return model.ListResp[*model.K8sPod]{
		Total: total,
		Items: pods,
	}, nil
}

// GetDaemonSetYaml 获取DaemonSet YAML
func (s *daemonSetService) GetDaemonSetYaml(ctx context.Context, req *model.GetDaemonSetYamlReq) (*model.K8sYaml, error) {
	if req == nil {
		return nil, fmt.Errorf("获取DaemonSet YAML请求不能为空")
	}

	if req.ClusterID <= 0 {
		return nil, fmt.Errorf("集群ID不能为空")
	}

	if req.Namespace == "" {
		return nil, fmt.Errorf("命名空间不能为空")
	}

	if req.Name == "" {
		return nil, fmt.Errorf("DaemonSet名称不能为空")
	}

	daemonSet, err := s.daemonSetManager.GetDaemonSet(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		s.logger.Error("GetDaemonSetYaml: 获取DaemonSet失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return nil, fmt.Errorf("获取DaemonSet失败: %w", err)
	}

	// 转换为YAML
	yamlContent, err := utils.DaemonSetToYAML(daemonSet)
	if err != nil {
		s.logger.Error("GetDaemonSetYaml: 转换为YAML失败",
			zap.Error(err),
			zap.String("daemonSetName", daemonSet.Name))
		return nil, fmt.Errorf("转换为YAML失败: %w", err)
	}

	return &model.K8sYaml{
		YAML: yamlContent,
	}, nil
}

// RestartDaemonSet 重启DaemonSet
func (s *daemonSetService) RestartDaemonSet(ctx context.Context, req *model.RestartDaemonSetReq) error {
	if req == nil {
		return fmt.Errorf("重启DaemonSet请求不能为空")
	}

	if req.ClusterID <= 0 {
		return fmt.Errorf("集群ID不能为空")
	}

	if req.Namespace == "" {
		return fmt.Errorf("命名空间不能为空")
	}

	if req.Name == "" {
		return fmt.Errorf("DaemonSet名称不能为空")
	}

	err := s.daemonSetManager.RestartDaemonSet(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		s.logger.Error("RestartDaemonSet: 重启DaemonSet失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("重启DaemonSet失败: %w", err)
	}

	return nil
}

// RollbackDaemonSet 回滚DaemonSet
func (s *daemonSetService) RollbackDaemonSet(ctx context.Context, req *model.RollbackDaemonSetReq) error {
	if req == nil {
		return fmt.Errorf("回滚DaemonSet请求不能为空")
	}

	if req.ClusterID <= 0 {
		return fmt.Errorf("集群ID不能为空")
	}

	if req.Namespace == "" {
		return fmt.Errorf("命名空间不能为空")
	}

	if req.Name == "" {
		return fmt.Errorf("DaemonSet名称不能为空")
	}

	if req.Revision <= 0 {
		return fmt.Errorf("回滚版本号必须大于0")
	}

	err := s.daemonSetManager.RollbackDaemonSet(ctx, req.ClusterID, req.Namespace, req.Name, req.Revision)
	if err != nil {
		s.logger.Error("RollbackDaemonSet: 回滚DaemonSet失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name),
			zap.Int64("revision", req.Revision))
		return fmt.Errorf("回滚DaemonSet失败: %w", err)
	}

	return nil
}

// UpdateDaemonSetByYaml 通过YAML更新DaemonSet
func (s *daemonSetService) UpdateDaemonSetByYaml(ctx context.Context, req *model.UpdateDaemonSetByYamlReq) error {
	if req == nil {
		return fmt.Errorf("通过YAML更新DaemonSet请求不能为空")
	}

	if req.ClusterID <= 0 {
		return fmt.Errorf("集群ID不能为空")
	}

	if req.Namespace == "" {
		return fmt.Errorf("命名空间不能为空")
	}

	if req.Name == "" {
		return fmt.Errorf("DaemonSet名称不能为空")
	}

	if req.YAML == "" {
		return fmt.Errorf("YAML内容不能为空")
	}

	s.logger.Info("开始通过YAML更新DaemonSet",
		zap.Int("clusterID", req.ClusterID),
		zap.String("namespace", req.Namespace),
		zap.String("name", req.Name))

	// 从YAML构建DaemonSet对象
	daemonSet, err := utils.BuildDaemonSetFromYamlForUpdate(req)
	if err != nil {
		s.logger.Error("从YAML构建DaemonSet失败",
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name),
			zap.Error(err))
		return fmt.Errorf("从YAML构建DaemonSet失败: %w", err)
	}

	// 验证更新后的DaemonSet配置
	if err := utils.ValidateDaemonSet(daemonSet); err != nil {
		s.logger.Error("UpdateDaemonSetByYaml: DaemonSet配置验证失败",
			zap.Error(err),
			zap.String("name", req.Name))
		return fmt.Errorf("daemonSet配置验证失败: %w", err)
	}

	// 使用现有的更新方法
	err = s.daemonSetManager.UpdateDaemonSet(ctx, req.ClusterID, req.Namespace, daemonSet)
	if err != nil {
		s.logger.Error("通过YAML更新DaemonSet失败",
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name),
			zap.Error(err))
		return fmt.Errorf("通过YAML更新DaemonSet失败: %w", err)
	}

	s.logger.Info("通过YAML更新DaemonSet成功",
		zap.Int("clusterID", req.ClusterID),
		zap.String("namespace", req.Namespace),
		zap.String("name", req.Name))

	return nil
}

// UpdateDaemonSet 更新DaemonSet
func (s *daemonSetService) UpdateDaemonSet(ctx context.Context, req *model.UpdateDaemonSetReq) error {
	if req == nil {
		return fmt.Errorf("更新DaemonSet请求不能为空")
	}

	if req.ClusterID <= 0 {
		return fmt.Errorf("集群ID不能为空")
	}

	if req.Name == "" {
		return fmt.Errorf("DaemonSet名称不能为空")
	}

	if req.Namespace == "" {
		return fmt.Errorf("命名空间不能为空")
	}

	existingDaemonSet, err := s.daemonSetManager.GetDaemonSet(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		s.logger.Error("UpdateDaemonSet: 获取现有DaemonSet失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("获取现有DaemonSet失败: %w", err)
	}

	updatedDaemonSet := existingDaemonSet.DeepCopy()

	// 如果提供了YAML，使用YAML内容
	if req.YAML != "" {
		yamlDaemonSet, err := utils.YAMLToDaemonSet(req.YAML)
		if err != nil {
			s.logger.Error("UpdateDaemonSet: 解析YAML失败",
				zap.Error(err),
				zap.String("name", req.Name))
			return fmt.Errorf("解析YAML失败: %w", err)
		}
		updatedDaemonSet.Spec = yamlDaemonSet.Spec
		updatedDaemonSet.Labels = yamlDaemonSet.Labels
		updatedDaemonSet.Annotations = yamlDaemonSet.Annotations
	} else {
		// 更新基本字段
		if len(req.Images) > 0 {
			for i, image := range req.Images {
				if i < len(updatedDaemonSet.Spec.Template.Spec.Containers) {
					updatedDaemonSet.Spec.Template.Spec.Containers[i].Image = image
				}
			}
		}
		if req.Labels != nil {
			// 合并标签到对象级别
			if updatedDaemonSet.Labels == nil {
				updatedDaemonSet.Labels = make(map[string]string)
			}
			for k, v := range req.Labels {
				updatedDaemonSet.Labels[k] = v
			}

			// 更新 template labels，确保包含 selector 中的所有必需标签
			if updatedDaemonSet.Spec.Template.Labels == nil {
				updatedDaemonSet.Spec.Template.Labels = make(map[string]string)
			}

			// 先添加用户指定的标签
			for k, v := range req.Labels {
				updatedDaemonSet.Spec.Template.Labels[k] = v
			}

			// 然后强制保留 selector 的标签（selector 是不可变的，必须匹配）
			if updatedDaemonSet.Spec.Selector != nil && updatedDaemonSet.Spec.Selector.MatchLabels != nil {
				for k, v := range updatedDaemonSet.Spec.Selector.MatchLabels {
					updatedDaemonSet.Spec.Template.Labels[k] = v
				}
			}
		}
		if req.Annotations != nil {
			updatedDaemonSet.Annotations = req.Annotations
		}
	}

	// 验证更新后的DaemonSet配置
	if err := utils.ValidateDaemonSet(updatedDaemonSet); err != nil {
		s.logger.Error("UpdateDaemonSet: DaemonSet配置验证失败",
			zap.Error(err),
			zap.String("name", req.Name))
		return fmt.Errorf("daemonSet配置验证失败: %w", err)
	}

	err = s.daemonSetManager.UpdateDaemonSet(ctx, req.ClusterID, req.Namespace, updatedDaemonSet)
	if err != nil {
		s.logger.Error("UpdateDaemonSet: 更新DaemonSet失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("更新DaemonSet失败: %w", err)
	}

	return nil
}
