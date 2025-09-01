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

package utils

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

// BuildK8sDaemonSet 构建详细的 K8sDaemonSet 模型
func BuildK8sDaemonSet(ctx context.Context, clusterID int, daemonSet appsv1.DaemonSet) (*model.K8sDaemonSet, error) {
	if clusterID <= 0 {
		return nil, fmt.Errorf("无效的集群ID: %d", clusterID)
	}

	// 获取DaemonSet状态
	status := getDaemonSetStatus(daemonSet)

	// 获取更新策略信息
	updateStrategy := "RollingUpdate"
	if daemonSet.Spec.UpdateStrategy.Type == appsv1.OnDeleteDaemonSetStrategyType {
		updateStrategy = "OnDelete"
	}

	// 获取容器镜像列表
	var images []string
	for _, container := range daemonSet.Spec.Template.Spec.Containers {
		images = append(images, container.Image)
	}

	// 构建标签选择器
	selector := make(map[string]string)
	if daemonSet.Spec.Selector != nil && daemonSet.Spec.Selector.MatchLabels != nil {
		selector = daemonSet.Spec.Selector.MatchLabels
	}

	// 构建基础DaemonSet信息
	k8sDaemonSet := &model.K8sDaemonSet{
		Name:                   daemonSet.Name,
		Namespace:              daemonSet.Namespace,
		ClusterID:              clusterID,
		UID:                    string(daemonSet.UID),
		Labels:                 daemonSet.Labels,
		Annotations:            daemonSet.Annotations,
		CreatedAt:              daemonSet.CreationTimestamp.Time,
		Status:                 status,
		DesiredNumberScheduled: daemonSet.Status.DesiredNumberScheduled,
		CurrentNumberScheduled: daemonSet.Status.CurrentNumberScheduled,
		NumberReady:            daemonSet.Status.NumberReady,
		NumberAvailable:        daemonSet.Status.NumberAvailable,
		NumberUnavailable:      daemonSet.Status.NumberUnavailable,
		UpdatedNumberScheduled: daemonSet.Status.UpdatedNumberScheduled,
		NumberMisscheduled:     daemonSet.Status.NumberMisscheduled,
		Images:                 images,
		Selector:               selector,
		UpdateStrategy:         updateStrategy,
	}

	return k8sDaemonSet, nil
}

// getDaemonSetStatus 获取DaemonSet状态
func getDaemonSetStatus(daemonSet appsv1.DaemonSet) model.K8sDaemonSetStatus {
	desired := daemonSet.Status.DesiredNumberScheduled
	ready := daemonSet.Status.NumberReady
	available := daemonSet.Status.NumberAvailable
	unavailable := daemonSet.Status.NumberUnavailable

	if unavailable > 0 {
		return model.K8sDaemonSetStatusUpdating
	}

	if ready == desired && available == desired {
		return model.K8sDaemonSetStatusRunning
	}

	if ready == 0 {
		return model.K8sDaemonSetStatusError
	}

	return model.K8sDaemonSetStatusError
}

// BuildDaemonSetFromRequest 从请求构建DaemonSet对象
func BuildDaemonSetFromRequest(req *model.CreateDaemonSetReq) (*appsv1.DaemonSet, error) {
	if req == nil {
		return nil, fmt.Errorf("请求不能为空")
	}

	// 如果提供了YAML，直接解析
	if req.YAML != "" {
		return YAMLToDaemonSet(req.YAML)
	}

	// 构建基础DaemonSet
	daemonSet := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:        req.Name,
			Namespace:   req.Namespace,
			Labels:      req.Labels,
			Annotations: req.Annotations,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: req.Labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: req.Labels,
				},
				Spec: corev1.PodSpec{},
			},
		},
	}

	// 构建容器
	var containers []corev1.Container
	for i, image := range req.Images {
		containerName := fmt.Sprintf("container-%d", i)

		container := corev1.Container{
			Name:  containerName,
			Image: image,
		}

		containers = append(containers, container)
	}

	daemonSet.Spec.Template.Spec.Containers = containers

	// 如果提供了Spec，使用自定义配置
	if req.Spec.Selector != nil {
		daemonSet.Spec.Selector = req.Spec.Selector
	}
	if req.Spec.Template != nil {
		daemonSet.Spec.Template = *req.Spec.Template
	}
	if req.Spec.UpdateStrategy != nil {
		daemonSet.Spec.UpdateStrategy = *req.Spec.UpdateStrategy
	}

	return daemonSet, nil
}

// YAMLToDaemonSet 将YAML转换为DaemonSet对象
func YAMLToDaemonSet(yamlContent string) (*appsv1.DaemonSet, error) {
	var daemonSet appsv1.DaemonSet
	err := yaml.Unmarshal([]byte(yamlContent), &daemonSet)
	if err != nil {
		return nil, fmt.Errorf("YAML解析失败: %w", err)
	}
	return &daemonSet, nil
}

// DaemonSetToYAML 将DaemonSet对象转换为YAML
func DaemonSetToYAML(daemonSet *appsv1.DaemonSet) (string, error) {
	if daemonSet == nil {
		return "", fmt.Errorf("DaemonSet对象不能为空")
	}

	// 清理不需要的字段
	cleanDaemonSet := daemonSet.DeepCopy()
	cleanDaemonSet.ManagedFields = nil
	cleanDaemonSet.Status = appsv1.DaemonSetStatus{}

	yamlBytes, err := yaml.Marshal(cleanDaemonSet)
	if err != nil {
		return "", fmt.Errorf("YAML序列化失败: %w", err)
	}

	return string(yamlBytes), nil
}

// ValidateDaemonSet 验证DaemonSet配置
func ValidateDaemonSet(daemonSet *appsv1.DaemonSet) error {
	if daemonSet == nil {
		return fmt.Errorf("DaemonSet对象不能为空")
	}

	if daemonSet.Name == "" {
		return fmt.Errorf("DaemonSet名称不能为空")
	}

	if daemonSet.Namespace == "" {
		return fmt.Errorf("DaemonSet命名空间不能为空")
	}

	if len(daemonSet.Spec.Template.Spec.Containers) == 0 {
		return fmt.Errorf("DaemonSet必须包含至少一个容器")
	}

	for i, container := range daemonSet.Spec.Template.Spec.Containers {
		if container.Name == "" {
			return fmt.Errorf("第%d个容器名称不能为空", i+1)
		}
		if container.Image == "" {
			return fmt.Errorf("第%d个容器镜像不能为空", i+1)
		}
	}

	return nil
}

// BuildDaemonSetListOptions 构建DaemonSet列表查询选项
func BuildDaemonSetListOptions(req *model.GetDaemonSetListReq) metav1.ListOptions {
	listOptions := metav1.ListOptions{}

	// 基础查询选项，可以根据需要扩展
	return listOptions
}

// PaginateK8sDaemonSets 对DaemonSet列表进行分页
func PaginateK8sDaemonSets(daemonSets []*model.K8sDaemonSet, page, size int) ([]*model.K8sDaemonSet, int64) {
	total := int64(len(daemonSets))

	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 10
	}

	start := (page - 1) * size
	end := start + size

	if start >= len(daemonSets) {
		return []*model.K8sDaemonSet{}, total
	}

	if end > len(daemonSets) {
		end = len(daemonSets)
	}

	return daemonSets[start:end], total
}

// BuildK8sDaemonSetEvent 构建DaemonSet事件模型
func BuildK8sDaemonSetEvent(event corev1.Event) (*model.K8sDaemonSetEvent, error) {
	return &model.K8sDaemonSetEvent{
		Type:      event.Type,
		Reason:    event.Reason,
		Message:   event.Message,
		Source:    event.Source.Component,
		Count:     event.Count,
		FirstTime: event.FirstTimestamp.Time,
		LastTime:  event.LastTimestamp.Time,
	}, nil
}

// BuildK8sDaemonSetHistory 构建DaemonSet历史版本模型
func BuildK8sDaemonSetHistory(revision appsv1.ControllerRevision) (*model.K8sDaemonSetHistory, error) {
	return &model.K8sDaemonSetHistory{
		Revision: revision.Revision,
		Date:     revision.CreationTimestamp.Time,
		Message:  getChangeReason(revision.Annotations),
	}, nil
}

// ExtractDaemonSetFromRevision 从ControllerRevision中提取DaemonSet模板
func ExtractDaemonSetFromRevision(revision *appsv1.ControllerRevision, daemonSet *appsv1.DaemonSet) error {
	if revision == nil {
		return fmt.Errorf("ControllerRevision不能为空")
	}

	if daemonSet == nil {
		return fmt.Errorf("DaemonSet对象不能为空")
	}

	// 简化实现，实际上ControllerRevision的Data包含序列化的对象数据
	// 这里可以根据需要实现具体的反序列化逻辑
	if revision.Data.Raw == nil {
		return fmt.Errorf("ControllerRevision数据为空")
	}

	// 这里可以添加具体的反序列化逻辑
	// 暂时返回成功，实际使用中需要实现具体的反序列化

	return nil
}

// getChangeReason 获取变更原因
func getChangeReason(annotations map[string]string) string {
	if annotations == nil {
		return ""
	}

	// 常见的变更原因注解
	changeReasonKeys := []string{
		"deployment.kubernetes.io/revision-change-cause",
		"kubernetes.io/change-cause",
	}

	for _, key := range changeReasonKeys {
		if reason, exists := annotations[key]; exists {
			return reason
		}
	}

	return ""
}

// SortDaemonSetsByCreationTime 按创建时间排序DaemonSet列表
func SortDaemonSetsByCreationTime(daemonSets []*model.K8sDaemonSet, desc bool) {
	sort.Slice(daemonSets, func(i, j int) bool {
		if desc {
			return daemonSets[i].CreatedAt.After(daemonSets[j].CreatedAt)
		}
		return daemonSets[i].CreatedAt.Before(daemonSets[j].CreatedAt)
	})
}

// FilterDaemonSetsByStatus 按状态过滤DaemonSet列表
func FilterDaemonSetsByStatus(daemonSets []*model.K8sDaemonSet, status string) []*model.K8sDaemonSet {
	if status == "" {
		return daemonSets
	}

	var filtered []*model.K8sDaemonSet
	for _, ds := range daemonSets {
		statusStr := getDaemonSetStatusString(ds.Status)
		if strings.EqualFold(statusStr, status) {
			filtered = append(filtered, ds)
		}
	}

	return filtered
}

// getDaemonSetStatusString 获取DaemonSet状态字符串
func getDaemonSetStatusString(status model.K8sDaemonSetStatus) string {
	switch status {
	case model.K8sDaemonSetStatusRunning:
		return "running"
	case model.K8sDaemonSetStatusUpdating:
		return "updating"
	case model.K8sDaemonSetStatusError:
		return "error"
	default:
		return "unknown"
	}
}

// GetDaemonSetResourceUsage 计算DaemonSet资源使用情况
func GetDaemonSetResourceUsage(daemonSet *model.K8sDaemonSet) *model.K8sDaemonSetMetrics {
	if daemonSet == nil {
		return &model.K8sDaemonSetMetrics{}
	}

	// 基础指标（实际的CPU和内存使用需要从metrics API获取）
	return &model.K8sDaemonSetMetrics{
		CPUUsage:    0.0, // 这里需要从metrics API获取实际数据
		MemoryUsage: 0.0, // 这里需要从metrics API获取实际数据
	}
}
