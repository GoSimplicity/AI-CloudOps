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
	UpdateDaemonSet(ctx context.Context, req *model.UpdateDaemonSetReq) error
	DeleteDaemonSet(ctx context.Context, req *model.DeleteDaemonSetReq) error
	RestartDaemonSet(ctx context.Context, req *model.RestartDaemonSetReq) error

	GetDaemonSetEvents(ctx context.Context, req *model.GetDaemonSetEventsReq) (model.ListResp[*model.K8sDaemonSetEvent], error)
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
func (d *daemonSetService) CreateDaemonSet(ctx context.Context, req *model.CreateDaemonSetReq) error {
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
		d.logger.Error("CreateDaemonSet: 构建DaemonSet对象失败",
			zap.Error(err),
			zap.String("name", req.Name))
		return fmt.Errorf("构建DaemonSet对象失败: %w", err)
	}

	// 验证DaemonSet配置
	if err := utils.ValidateDaemonSet(daemonSet); err != nil {
		d.logger.Error("CreateDaemonSet: DaemonSet配置验证失败",
			zap.Error(err),
			zap.String("name", req.Name))
		return fmt.Errorf("daemonSet配置验证失败: %w", err)
	}

	err = d.daemonSetManager.CreateDaemonSet(ctx, req.ClusterID, req.Namespace, daemonSet)
	if err != nil {
		d.logger.Error("CreateDaemonSet: 创建DaemonSet失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("创建DaemonSet失败: %w", err)
	}

	return nil
}

// DeleteDaemonSet 删除DaemonSet
func (d *daemonSetService) DeleteDaemonSet(ctx context.Context, req *model.DeleteDaemonSetReq) error {
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

	err := d.daemonSetManager.DeleteDaemonSet(ctx, req.ClusterID, req.Namespace, req.Name, metav1.DeleteOptions{})
	if err != nil {
		d.logger.Error("DeleteDaemonSet: 删除DaemonSet失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("删除DaemonSet失败: %w", err)
	}

	return nil
}

// GetDaemonSetDetails 获取DaemonSet详情
func (d *daemonSetService) GetDaemonSetDetails(ctx context.Context, req *model.GetDaemonSetDetailsReq) (*model.K8sDaemonSet, error) {
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

	daemonSet, err := d.daemonSetManager.GetDaemonSet(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		d.logger.Error("GetDaemonSetDetails: 获取DaemonSet失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return nil, fmt.Errorf("获取DaemonSet失败: %w", err)
	}

	// 构建详细信息
	k8sDaemonSet, err := utils.BuildK8sDaemonSet(ctx, req.ClusterID, *daemonSet)
	if err != nil {
		d.logger.Error("GetDaemonSetDetails: 构建DaemonSet详细信息失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return nil, fmt.Errorf("构建DaemonSet详细信息失败: %w", err)
	}

	return k8sDaemonSet, nil
}

// GetDaemonSetEvents 获取DaemonSet事件
func (d *daemonSetService) GetDaemonSetEvents(ctx context.Context, req *model.GetDaemonSetEventsReq) (model.ListResp[*model.K8sDaemonSetEvent], error) {
	if req == nil {
		return model.ListResp[*model.K8sDaemonSetEvent]{}, fmt.Errorf("获取DaemonSet事件请求不能为空")
	}

	if req.ClusterID <= 0 {
		return model.ListResp[*model.K8sDaemonSetEvent]{}, fmt.Errorf("集群ID不能为空")
	}

	if req.Namespace == "" {
		return model.ListResp[*model.K8sDaemonSetEvent]{}, fmt.Errorf("命名空间不能为空")
	}

	if req.Name == "" {
		return model.ListResp[*model.K8sDaemonSetEvent]{}, fmt.Errorf("DaemonSet名称不能为空")
	}

	// 设置默认限制数量
	limit := req.Limit
	if limit <= 0 {
		limit = 100 // 默认获取100个事件
	}

	events, total, err := d.daemonSetManager.GetDaemonSetEvents(ctx, req.ClusterID, req.Namespace, req.Name, limit)
	if err != nil {
		d.logger.Error("GetDaemonSetEvents: 获取DaemonSet事件失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return model.ListResp[*model.K8sDaemonSetEvent]{}, fmt.Errorf("获取DaemonSet事件失败: %w", err)
	}

	return model.ListResp[*model.K8sDaemonSetEvent]{
		Total: total,
		Items: events,
	}, nil
}

// GetDaemonSetHistory 获取DaemonSet版本历史
func (d *daemonSetService) GetDaemonSetHistory(ctx context.Context, req *model.GetDaemonSetHistoryReq) (model.ListResp[*model.K8sDaemonSetHistory], error) {
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

	history, total, err := d.daemonSetManager.GetDaemonSetHistory(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		d.logger.Error("GetDaemonSetHistory: 获取DaemonSet历史失败",
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
func (d *daemonSetService) GetDaemonSetList(ctx context.Context, req *model.GetDaemonSetListReq) (model.ListResp[*model.K8sDaemonSet], error) {
	if req == nil {
		return model.ListResp[*model.K8sDaemonSet]{}, fmt.Errorf("获取DaemonSet列表请求不能为空")
	}

	if req.ClusterID <= 0 {
		return model.ListResp[*model.K8sDaemonSet]{}, fmt.Errorf("集群ID不能为空")
	}

	// 构建查询选项
	listOptions := utils.BuildDaemonSetListOptions(req)

	k8sDaemonSets, err := d.daemonSetManager.GetDaemonSetList(ctx, req.ClusterID, req.Namespace, listOptions)
	if err != nil {
		d.logger.Error("GetDaemonSetList: 获取DaemonSet列表失败",
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
func (d *daemonSetService) GetDaemonSetPods(ctx context.Context, req *model.GetDaemonSetPodsReq) (model.ListResp[*model.K8sPod], error) {
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

	pods, total, err := d.daemonSetManager.GetDaemonSetPods(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		d.logger.Error("GetDaemonSetPods: 获取DaemonSet Pod失败",
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
func (d *daemonSetService) GetDaemonSetYaml(ctx context.Context, req *model.GetDaemonSetYamlReq) (*model.K8sYaml, error) {
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

	daemonSet, err := d.daemonSetManager.GetDaemonSet(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		d.logger.Error("GetDaemonSetYaml: 获取DaemonSet失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return nil, fmt.Errorf("获取DaemonSet失败: %w", err)
	}

	// 转换为YAML
	yamlContent, err := utils.DaemonSetToYAML(daemonSet)
	if err != nil {
		d.logger.Error("GetDaemonSetYaml: 转换为YAML失败",
			zap.Error(err),
			zap.String("daemonSetName", daemonSet.Name))
		return nil, fmt.Errorf("转换为YAML失败: %w", err)
	}

	return &model.K8sYaml{
		YAML: yamlContent,
	}, nil
}

// RestartDaemonSet 重启DaemonSet
func (d *daemonSetService) RestartDaemonSet(ctx context.Context, req *model.RestartDaemonSetReq) error {
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

	err := d.daemonSetManager.RestartDaemonSet(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		d.logger.Error("RestartDaemonSet: 重启DaemonSet失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("重启DaemonSet失败: %w", err)
	}

	return nil
}

// RollbackDaemonSet 回滚DaemonSet
func (d *daemonSetService) RollbackDaemonSet(ctx context.Context, req *model.RollbackDaemonSetReq) error {
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

	err := d.daemonSetManager.RollbackDaemonSet(ctx, req.ClusterID, req.Namespace, req.Name, req.Revision)
	if err != nil {
		d.logger.Error("RollbackDaemonSet: 回滚DaemonSet失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name),
			zap.Int64("revision", req.Revision))
		return fmt.Errorf("回滚DaemonSet失败: %w", err)
	}

	return nil
}

// UpdateDaemonSet 更新DaemonSet
func (d *daemonSetService) UpdateDaemonSet(ctx context.Context, req *model.UpdateDaemonSetReq) error {
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

	existingDaemonSet, err := d.daemonSetManager.GetDaemonSet(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		d.logger.Error("UpdateDaemonSet: 获取现有DaemonSet失败",
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
			d.logger.Error("UpdateDaemonSet: 解析YAML失败",
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
			updatedDaemonSet.Labels = req.Labels
			updatedDaemonSet.Spec.Template.Labels = req.Labels
		}
		if req.Annotations != nil {
			updatedDaemonSet.Annotations = req.Annotations
		}
	}

	// 验证更新后的DaemonSet配置
	if err := utils.ValidateDaemonSet(updatedDaemonSet); err != nil {
		d.logger.Error("UpdateDaemonSet: DaemonSet配置验证失败",
			zap.Error(err),
			zap.String("name", req.Name))
		return fmt.Errorf("daemonSet配置验证失败: %w", err)
	}

	err = d.daemonSetManager.UpdateDaemonSet(ctx, req.ClusterID, req.Namespace, updatedDaemonSet)
	if err != nil {
		d.logger.Error("UpdateDaemonSet: 更新DaemonSet失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("更新DaemonSet失败: %w", err)
	}

	return nil
}
