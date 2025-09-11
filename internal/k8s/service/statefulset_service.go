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

type StatefulSetService interface {
	GetStatefulSetList(ctx context.Context, req *model.GetStatefulSetListReq) (model.ListResp[*model.K8sStatefulSet], error)
	GetStatefulSetDetails(ctx context.Context, req *model.GetStatefulSetDetailsReq) (*model.K8sStatefulSet, error)
	GetStatefulSetYaml(ctx context.Context, req *model.GetStatefulSetYamlReq) (*model.K8sYaml, error)
	CreateStatefulSet(ctx context.Context, req *model.CreateStatefulSetReq) error
	CreateStatefulSetByYaml(ctx context.Context, req *model.CreateStatefulSetByYamlReq) error
	UpdateStatefulSet(ctx context.Context, req *model.UpdateStatefulSetReq) error
	UpdateStatefulSetByYaml(ctx context.Context, req *model.UpdateStatefulSetByYamlReq) error
	DeleteStatefulSet(ctx context.Context, req *model.DeleteStatefulSetReq) error
	RestartStatefulSet(ctx context.Context, req *model.RestartStatefulSetReq) error
	ScaleStatefulSet(ctx context.Context, req *model.ScaleStatefulSetReq) error
	GetStatefulSetPods(ctx context.Context, req *model.GetStatefulSetPodsReq) (model.ListResp[*model.K8sPod], error)
	GetStatefulSetHistory(ctx context.Context, req *model.GetStatefulSetHistoryReq) (model.ListResp[*model.K8sStatefulSetHistory], error)
	RollbackStatefulSet(ctx context.Context, req *model.RollbackStatefulSetReq) error
}

type statefulSetService struct {
	statefulSetManager manager.StatefulSetManager
	logger             *zap.Logger
}

func NewStatefulSetService(statefulSetManager manager.StatefulSetManager, logger *zap.Logger) StatefulSetService {
	return &statefulSetService{
		statefulSetManager: statefulSetManager,
		logger:             logger,
	}
}

// CreateStatefulSet 创建StatefulSet
func (s *statefulSetService) CreateStatefulSet(ctx context.Context, req *model.CreateStatefulSetReq) error {
	if req == nil {
		return fmt.Errorf("创建StatefulSet请求不能为空")
	}

	if req.ClusterID <= 0 {
		return fmt.Errorf("集群ID不能为空")
	}

	if req.Name == "" {
		return fmt.Errorf("StatefulSet名称不能为空")
	}

	if req.Namespace == "" {
		return fmt.Errorf("命名空间不能为空")
	}

	if req.ServiceName == "" {
		return fmt.Errorf("服务名称不能为空")
	}

	// 从请求构建StatefulSet对象
	statefulSet, err := utils.BuildStatefulSetFromRequest(req)
	if err != nil {
		s.logger.Error("CreateStatefulSet: 构建StatefulSet对象失败",
			zap.Error(err),
			zap.String("name", req.Name))
		return fmt.Errorf("构建StatefulSet对象失败: %w", err)
	}

	// 验证StatefulSet配置
	if err := utils.ValidateStatefulSet(statefulSet); err != nil {
		s.logger.Error("CreateStatefulSet: StatefulSet配置验证失败",
			zap.Error(err),
			zap.String("name", req.Name))
		return fmt.Errorf("statefulSet配置验证失败: %w", err)
	}

	err = s.statefulSetManager.CreateStatefulSet(ctx, req.ClusterID, req.Namespace, statefulSet)
	if err != nil {
		s.logger.Error("CreateStatefulSet: 创建StatefulSet失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("创建StatefulSet失败: %w", err)
	}

	return nil
}

// CreateStatefulSetByYaml 通过YAML创建StatefulSet
func (s *statefulSetService) CreateStatefulSetByYaml(ctx context.Context, req *model.CreateStatefulSetByYamlReq) error {
	if req == nil {
		return fmt.Errorf("创建StatefulSet YAML请求不能为空")
	}

	if req.ClusterID <= 0 {
		return fmt.Errorf("集群ID不能为空")
	}

	if req.YAML == "" {
		return fmt.Errorf("YAML内容不能为空")
	}

	// 解析YAML为StatefulSet对象
	statefulSet, err := utils.YAMLToStatefulSet(req.YAML)
	if err != nil {
		s.logger.Error("CreateStatefulSetByYaml: 解析YAML失败",
			zap.Error(err))
		return fmt.Errorf("解析YAML失败: %w", err)
	}

	// 验证StatefulSet配置
	if err := utils.ValidateStatefulSet(statefulSet); err != nil {
		s.logger.Error("CreateStatefulSetByYaml: StatefulSet配置验证失败",
			zap.Error(err),
			zap.String("name", statefulSet.Name))
		return fmt.Errorf("statefulSet配置验证失败: %w", err)
	}

	err = s.statefulSetManager.CreateStatefulSet(ctx, req.ClusterID, statefulSet.Namespace, statefulSet)
	if err != nil {
		s.logger.Error("CreateStatefulSetByYaml: 创建StatefulSet失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", statefulSet.Namespace),
			zap.String("name", statefulSet.Name))
		return fmt.Errorf("创建StatefulSet失败: %w", err)
	}

	return nil
}

// DeleteStatefulSet 删除StatefulSet
func (s *statefulSetService) DeleteStatefulSet(ctx context.Context, req *model.DeleteStatefulSetReq) error {
	if req == nil {
		return fmt.Errorf("删除StatefulSet请求不能为空")
	}

	if req.ClusterID <= 0 {
		return fmt.Errorf("集群ID不能为空")
	}

	if req.Namespace == "" {
		return fmt.Errorf("命名空间不能为空")
	}

	if req.Name == "" {
		return fmt.Errorf("StatefulSet名称不能为空")
	}

	err := s.statefulSetManager.DeleteStatefulSet(ctx, req.ClusterID, req.Namespace, req.Name, metav1.DeleteOptions{})
	if err != nil {
		s.logger.Error("DeleteStatefulSet: 删除StatefulSet失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("删除StatefulSet失败: %w", err)
	}

	return nil
}

// GetStatefulSetDetails 获取StatefulSet详情
func (s *statefulSetService) GetStatefulSetDetails(ctx context.Context, req *model.GetStatefulSetDetailsReq) (*model.K8sStatefulSet, error) {
	if req == nil {
		return nil, fmt.Errorf("获取StatefulSet详情请求不能为空")
	}

	if req.ClusterID <= 0 {
		return nil, fmt.Errorf("集群ID不能为空")
	}

	if req.Namespace == "" {
		return nil, fmt.Errorf("命名空间不能为空")
	}

	if req.Name == "" {
		return nil, fmt.Errorf("StatefulSet名称不能为空")
	}

	statefulSet, err := s.statefulSetManager.GetStatefulSet(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		s.logger.Error("GetStatefulSetDetails: 获取StatefulSet失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return nil, fmt.Errorf("获取StatefulSet失败: %w", err)
	}

	// 构建详细信息
	k8sStatefulSet, err := utils.BuildK8sStatefulSet(ctx, req.ClusterID, *statefulSet)
	if err != nil {
		s.logger.Error("GetStatefulSetDetails: 构建StatefulSet详细信息失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return nil, fmt.Errorf("构建StatefulSet详细信息失败: %w", err)
	}

	return k8sStatefulSet, nil
}

// GetStatefulSetHistory 获取StatefulSet版本历史
func (s *statefulSetService) GetStatefulSetHistory(ctx context.Context, req *model.GetStatefulSetHistoryReq) (model.ListResp[*model.K8sStatefulSetHistory], error) {
	if req == nil {
		return model.ListResp[*model.K8sStatefulSetHistory]{}, fmt.Errorf("获取StatefulSet历史请求不能为空")
	}

	if req.ClusterID <= 0 {
		return model.ListResp[*model.K8sStatefulSetHistory]{}, fmt.Errorf("集群ID不能为空")
	}

	if req.Namespace == "" {
		return model.ListResp[*model.K8sStatefulSetHistory]{}, fmt.Errorf("命名空间不能为空")
	}

	if req.Name == "" {
		return model.ListResp[*model.K8sStatefulSetHistory]{}, fmt.Errorf("StatefulSet名称不能为空")
	}

	history, total, err := s.statefulSetManager.GetStatefulSetHistory(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		s.logger.Error("GetStatefulSetHistory: 获取StatefulSet历史失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return model.ListResp[*model.K8sStatefulSetHistory]{}, fmt.Errorf("获取StatefulSet历史失败: %w", err)
	}

	return model.ListResp[*model.K8sStatefulSetHistory]{
		Total: total,
		Items: history,
	}, nil
}

// GetStatefulSetList 获取StatefulSet列表
func (s *statefulSetService) GetStatefulSetList(ctx context.Context, req *model.GetStatefulSetListReq) (model.ListResp[*model.K8sStatefulSet], error) {
	if req == nil {
		return model.ListResp[*model.K8sStatefulSet]{}, fmt.Errorf("获取StatefulSet列表请求不能为空")
	}

	if req.ClusterID <= 0 {
		return model.ListResp[*model.K8sStatefulSet]{}, fmt.Errorf("集群ID不能为空")
	}

	// 构建查询选项
	listOptions := utils.BuildStatefulSetListOptions(req)

	k8sStatefulSets, err := s.statefulSetManager.GetStatefulSetList(ctx, req.ClusterID, req.Namespace, listOptions)
	if err != nil {
		s.logger.Error("GetStatefulSetList: 获取StatefulSet列表失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace))
		return model.ListResp[*model.K8sStatefulSet]{}, fmt.Errorf("获取StatefulSet列表失败: %w", err)
	}

	// 根据状态过滤
	var filteredStatefulSets []*model.K8sStatefulSet
	if req.Status != "" {
		// 根据状态过滤
		for _, k8sStatefulSet := range k8sStatefulSets {
			var statusStr string
			switch k8sStatefulSet.Status {
			case model.K8sStatefulSetStatusRunning:
				statusStr = "running"
			case model.K8sStatefulSetStatusStopped:
				statusStr = "stopped"
			case model.K8sStatefulSetStatusUpdating:
				statusStr = "updating"
			case model.K8sStatefulSetStatusError:
				statusStr = "error"
			default:
				statusStr = "unknown"
			}
			if strings.EqualFold(statusStr, req.Status) {
				filteredStatefulSets = append(filteredStatefulSets, k8sStatefulSet)
			}
		}
	} else {
		filteredStatefulSets = k8sStatefulSets
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

	pagedItems, total := utils.PaginateK8sStatefulSets(filteredStatefulSets, page, size)

	return model.ListResp[*model.K8sStatefulSet]{
		Total: total,
		Items: pagedItems,
	}, nil
}

// GetStatefulSetPods 获取StatefulSet下的Pod列表
func (s *statefulSetService) GetStatefulSetPods(ctx context.Context, req *model.GetStatefulSetPodsReq) (model.ListResp[*model.K8sPod], error) {
	if req == nil {
		return model.ListResp[*model.K8sPod]{}, fmt.Errorf("获取StatefulSet Pods请求不能为空")
	}

	if req.ClusterID <= 0 {
		return model.ListResp[*model.K8sPod]{}, fmt.Errorf("集群ID不能为空")
	}

	if req.Namespace == "" {
		return model.ListResp[*model.K8sPod]{}, fmt.Errorf("命名空间不能为空")
	}

	if req.Name == "" {
		return model.ListResp[*model.K8sPod]{}, fmt.Errorf("StatefulSet名称不能为空")
	}

	pods, total, err := s.statefulSetManager.GetStatefulSetPods(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		s.logger.Error("GetStatefulSetPods: 获取StatefulSet Pod失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return model.ListResp[*model.K8sPod]{}, fmt.Errorf("获取StatefulSet Pod失败: %w", err)
	}

	return model.ListResp[*model.K8sPod]{
		Total: total,
		Items: pods,
	}, nil
}

// GetStatefulSetYaml 获取StatefulSet YAML
func (s *statefulSetService) GetStatefulSetYaml(ctx context.Context, req *model.GetStatefulSetYamlReq) (*model.K8sYaml, error) {
	if req == nil {
		return nil, fmt.Errorf("获取StatefulSet YAML请求不能为空")
	}

	if req.ClusterID <= 0 {
		return nil, fmt.Errorf("集群ID不能为空")
	}

	if req.Namespace == "" {
		return nil, fmt.Errorf("命名空间不能为空")
	}

	if req.Name == "" {
		return nil, fmt.Errorf("StatefulSet名称不能为空")
	}

	statefulSet, err := s.statefulSetManager.GetStatefulSet(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		s.logger.Error("GetStatefulSetYaml: 获取StatefulSet失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return nil, fmt.Errorf("获取StatefulSet失败: %w", err)
	}

	// 转换为YAML
	yamlContent, err := utils.StatefulSetToYAML(statefulSet)
	if err != nil {
		s.logger.Error("GetStatefulSetYaml: 转换为YAML失败",
			zap.Error(err),
			zap.String("statefulSetName", statefulSet.Name))
		return nil, fmt.Errorf("转换为YAML失败: %w", err)
	}

	return &model.K8sYaml{
		YAML: yamlContent,
	}, nil
}

// RestartStatefulSet 重启StatefulSet
func (s *statefulSetService) RestartStatefulSet(ctx context.Context, req *model.RestartStatefulSetReq) error {
	if req == nil {
		return fmt.Errorf("重启StatefulSet请求不能为空")
	}

	if req.ClusterID <= 0 {
		return fmt.Errorf("集群ID不能为空")
	}

	if req.Namespace == "" {
		return fmt.Errorf("命名空间不能为空")
	}

	if req.Name == "" {
		return fmt.Errorf("StatefulSet名称不能为空")
	}

	err := s.statefulSetManager.RestartStatefulSet(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		s.logger.Error("RestartStatefulSet: 重启StatefulSet失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("重启StatefulSet失败: %w", err)
	}

	return nil
}

// RollbackStatefulSet 回滚StatefulSet
func (s *statefulSetService) RollbackStatefulSet(ctx context.Context, req *model.RollbackStatefulSetReq) error {
	if req == nil {
		return fmt.Errorf("回滚StatefulSet请求不能为空")
	}

	if req.ClusterID <= 0 {
		return fmt.Errorf("集群ID不能为空")
	}

	if req.Namespace == "" {
		return fmt.Errorf("命名空间不能为空")
	}

	if req.Name == "" {
		return fmt.Errorf("StatefulSet名称不能为空")
	}

	if req.Revision <= 0 {
		return fmt.Errorf("回滚版本号必须大于0")
	}

	err := s.statefulSetManager.RollbackStatefulSet(ctx, req.ClusterID, req.Namespace, req.Name, req.Revision)
	if err != nil {
		s.logger.Error("RollbackStatefulSet: 回滚StatefulSet失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name),
			zap.Int64("revision", req.Revision))
		return fmt.Errorf("回滚StatefulSet失败: %w", err)
	}

	return nil
}

// ScaleStatefulSet 扩缩容StatefulSet
func (s *statefulSetService) ScaleStatefulSet(ctx context.Context, req *model.ScaleStatefulSetReq) error {
	if req == nil {
		return fmt.Errorf("扩缩容StatefulSet请求不能为空")
	}

	if req.ClusterID <= 0 {
		return fmt.Errorf("集群ID不能为空")
	}

	if req.Namespace == "" {
		return fmt.Errorf("命名空间不能为空")
	}

	if req.Name == "" {
		return fmt.Errorf("StatefulSet名称不能为空")
	}

	if req.Replicas < 0 {
		return fmt.Errorf("副本数不能为负数")
	}

	err := s.statefulSetManager.ScaleStatefulSet(ctx, req.ClusterID, req.Namespace, req.Name, req.Replicas)
	if err != nil {
		s.logger.Error("ScaleStatefulSet: 扩缩容StatefulSet失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name),
			zap.Int32("replicas", req.Replicas))
		return fmt.Errorf("扩缩容StatefulSet失败: %w", err)
	}

	return nil
}

// UpdateStatefulSetByYaml 通过YAML更新StatefulSet
func (s *statefulSetService) UpdateStatefulSetByYaml(ctx context.Context, req *model.UpdateStatefulSetByYamlReq) error {
	if req == nil {
		return fmt.Errorf("更新StatefulSet YAML请求不能为空")
	}

	if req.ClusterID <= 0 {
		return fmt.Errorf("集群ID不能为空")
	}

	if req.Name == "" {
		return fmt.Errorf("StatefulSet名称不能为空")
	}

	if req.Namespace == "" {
		return fmt.Errorf("命名空间不能为空")
	}

	if req.YAML == "" {
		return fmt.Errorf("YAML内容不能为空")
	}

	// 解析YAML为StatefulSet对象
	yamlStatefulSet, err := utils.YAMLToStatefulSet(req.YAML)
	if err != nil {
		s.logger.Error("UpdateStatefulSetByYaml: 解析YAML失败",
			zap.Error(err),
			zap.String("name", req.Name))
		return fmt.Errorf("解析YAML失败: %w", err)
	}

	// 验证StatefulSet配置
	if err := utils.ValidateStatefulSet(yamlStatefulSet); err != nil {
		s.logger.Error("UpdateStatefulSetByYaml: StatefulSet配置验证失败",
			zap.Error(err),
			zap.String("name", req.Name))
		return fmt.Errorf("statefulSet配置验证失败: %w", err)
	}

	// 获取现有StatefulSet以保持资源版本等元数据
	existingStatefulSet, err := s.statefulSetManager.GetStatefulSet(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		s.logger.Error("UpdateStatefulSetByYaml: 获取现有StatefulSet失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("获取现有StatefulSet失败: %w", err)
	}

	// 保留必要的元数据并更新spec
	yamlStatefulSet.ObjectMeta.ResourceVersion = existingStatefulSet.ObjectMeta.ResourceVersion
	yamlStatefulSet.ObjectMeta.UID = existingStatefulSet.ObjectMeta.UID

	err = s.statefulSetManager.UpdateStatefulSet(ctx, req.ClusterID, req.Namespace, yamlStatefulSet)
	if err != nil {
		s.logger.Error("UpdateStatefulSetByYaml: 更新StatefulSet失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("更新StatefulSet失败: %w", err)
	}

	return nil
}

// UpdateStatefulSet 更新StatefulSet
func (s *statefulSetService) UpdateStatefulSet(ctx context.Context, req *model.UpdateStatefulSetReq) error {
	if req == nil {
		return fmt.Errorf("更新StatefulSet请求不能为空")
	}

	if req.ClusterID <= 0 {
		return fmt.Errorf("集群ID不能为空")
	}

	if req.Name == "" {
		return fmt.Errorf("StatefulSet名称不能为空")
	}

	if req.Namespace == "" {
		return fmt.Errorf("命名空间不能为空")
	}

	existingStatefulSet, err := s.statefulSetManager.GetStatefulSet(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		s.logger.Error("UpdateStatefulSet: 获取现有StatefulSet失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("获取现有StatefulSet失败: %w", err)
	}

	updatedStatefulSet := existingStatefulSet.DeepCopy()

	// 如果提供了YAML，使用YAML内容
	if req.YAML != "" {
		yamlStatefulSet, err := utils.YAMLToStatefulSet(req.YAML)
		if err != nil {
			s.logger.Error("UpdateStatefulSet: 解析YAML失败",
				zap.Error(err),
				zap.String("name", req.Name))
			return fmt.Errorf("解析YAML失败: %w", err)
		}
		updatedStatefulSet.Spec = yamlStatefulSet.Spec
		updatedStatefulSet.Labels = yamlStatefulSet.Labels
		updatedStatefulSet.Annotations = yamlStatefulSet.Annotations
	} else {
		// 更新基本字段
		if req.Replicas > 0 {
			updatedStatefulSet.Spec.Replicas = &req.Replicas
		}
		if len(req.Images) > 0 {
			for i, image := range req.Images {
				if i < len(updatedStatefulSet.Spec.Template.Spec.Containers) {
					updatedStatefulSet.Spec.Template.Spec.Containers[i].Image = image
				}
			}
		}
		if req.Labels != nil {
			updatedStatefulSet.Labels = req.Labels
			updatedStatefulSet.Spec.Template.Labels = req.Labels
		}
		if req.Annotations != nil {
			updatedStatefulSet.Annotations = req.Annotations
		}
		if req.ServiceName != "" {
			updatedStatefulSet.Spec.ServiceName = req.ServiceName
		}
	}

	// 验证更新后的StatefulSet配置
	if err := utils.ValidateStatefulSet(updatedStatefulSet); err != nil {
		s.logger.Error("UpdateStatefulSet: StatefulSet配置验证失败",
			zap.Error(err),
			zap.String("name", req.Name))
		return fmt.Errorf("statefulSet配置验证失败: %w", err)
	}

	err = s.statefulSetManager.UpdateStatefulSet(ctx, req.ClusterID, req.Namespace, updatedStatefulSet)
	if err != nil {
		s.logger.Error("UpdateStatefulSet: 更新StatefulSet失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("更新StatefulSet失败: %w", err)
	}

	return nil
}
