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
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

// BuildK8sStatefulSet 构建详细的 K8sStatefulSet 模型
func BuildK8sStatefulSet(ctx context.Context, clusterID int, statefulSet appsv1.StatefulSet) (*model.K8sStatefulSet, error) {
	if clusterID <= 0 {
		return nil, fmt.Errorf("无效的集群ID: %d", clusterID)
	}

	status := getStatefulSetStatus(statefulSet)

	updateStrategy := "RollingUpdate"
	if statefulSet.Spec.UpdateStrategy.Type == appsv1.OnDeleteStatefulSetStrategyType {
		updateStrategy = "OnDelete"
	}

	podManagementPolicy := string(appsv1.OrderedReadyPodManagement)
	if statefulSet.Spec.PodManagementPolicy != "" {
		podManagementPolicy = string(statefulSet.Spec.PodManagementPolicy)
	}

	var images []string
	for _, container := range statefulSet.Spec.Template.Spec.Containers {
		images = append(images, container.Image)
	}

	selector := make(map[string]string)
	if statefulSet.Spec.Selector != nil && statefulSet.Spec.Selector.MatchLabels != nil {
		selector = statefulSet.Spec.Selector.MatchLabels
	}

	var conditions []model.StatefulSetCondition
	for _, condition := range statefulSet.Status.Conditions {
		stsCondition := model.StatefulSetCondition{
			Type:               string(condition.Type),
			Status:             string(condition.Status),
			LastUpdateTime:     condition.LastTransitionTime.Time,
			LastTransitionTime: condition.LastTransitionTime.Time,
			Reason:             condition.Reason,
			Message:            condition.Message,
		}
		conditions = append(conditions, stsCondition)
	}

	revisionHistoryLimit := int32(10)
	if statefulSet.Spec.RevisionHistoryLimit != nil {
		revisionHistoryLimit = *statefulSet.Spec.RevisionHistoryLimit
	}

	replicas := int32(0)
	if statefulSet.Spec.Replicas != nil {
		replicas = *statefulSet.Spec.Replicas
	}

	k8sStatefulSet := &model.K8sStatefulSet{
		Name:                 statefulSet.Name,
		Namespace:            statefulSet.Namespace,
		ClusterID:            clusterID,
		UID:                  string(statefulSet.UID),
		Labels:               statefulSet.Labels,
		Annotations:          statefulSet.Annotations,
		CreatedAt:            statefulSet.CreationTimestamp.Time,
		UpdatedAt:            time.Now(),
		Status:               status,
		Replicas:             replicas,
		ReadyReplicas:        statefulSet.Status.ReadyReplicas,
		CurrentReplicas:      statefulSet.Status.CurrentReplicas,
		UpdatedReplicas:      statefulSet.Status.UpdatedReplicas,
		Images:               images,
		Selector:             selector,
		ServiceName:          statefulSet.Spec.ServiceName,
		UpdateStrategy:       updateStrategy,
		PodManagementPolicy:  podManagementPolicy,
		RevisionHistoryLimit: revisionHistoryLimit,
		Conditions:           conditions,
		RawStatefulSet:       &statefulSet,
	}

	return k8sStatefulSet, nil
}

// getStatefulSetStatus 获取StatefulSet状态
func getStatefulSetStatus(statefulSet appsv1.StatefulSet) model.K8sStatefulSetStatus {
	replicas := int32(0)
	if statefulSet.Spec.Replicas != nil {
		replicas = *statefulSet.Spec.Replicas
	}

	ready := statefulSet.Status.ReadyReplicas
	updated := statefulSet.Status.UpdatedReplicas

	if updated < replicas {
		return model.K8sStatefulSetStatusUpdating
	}

	if ready == replicas && ready > 0 {
		return model.K8sStatefulSetStatusRunning
	}

	if ready == 0 {
		return model.K8sStatefulSetStatusStopped
	}

	return model.K8sStatefulSetStatusError
}

// BuildStatefulSetFromRequest 从请求构建StatefulSet对象
func BuildStatefulSetFromRequest(req *model.CreateStatefulSetReq) (*appsv1.StatefulSet, error) {
	if req == nil {
		return nil, fmt.Errorf("请求不能为空")
	}

	// 如果提供了YAML，直接解析
	if req.YAML != "" {
		return YAMLToStatefulSet(req.YAML)
	}

	// 构建基础StatefulSet
	statefulSet := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:        req.Name,
			Namespace:   req.Namespace,
			Labels:      req.Labels,
			Annotations: req.Annotations,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas:    &req.Replicas,
			ServiceName: req.ServiceName,
			Selector: &metav1.LabelSelector{
				MatchLabels: req.Labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: req.Labels,
				},
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

	statefulSet.Spec.Template.Spec.Containers = containers

	// 如果提供了Spec，使用自定义配置
	if req.Spec.Selector != nil {
		statefulSet.Spec.Selector = req.Spec.Selector
	}
	if req.Spec.Template != nil {
		statefulSet.Spec.Template = *req.Spec.Template
	}
	if req.Spec.UpdateStrategy != nil {
		statefulSet.Spec.UpdateStrategy = *req.Spec.UpdateStrategy
	}
	if req.Spec.VolumeClaimTemplates != nil {
		statefulSet.Spec.VolumeClaimTemplates = req.Spec.VolumeClaimTemplates
	}

	return statefulSet, nil
}

// YAMLToStatefulSet 将YAML转换为StatefulSet对象
func YAMLToStatefulSet(yamlContent string) (*appsv1.StatefulSet, error) {
	var statefulSet appsv1.StatefulSet
	err := yaml.Unmarshal([]byte(yamlContent), &statefulSet)
	if err != nil {
		return nil, fmt.Errorf("YAML解析失败: %w", err)
	}
	return &statefulSet, nil
}

// StatefulSetToYAML 将StatefulSet对象转换为YAML
func StatefulSetToYAML(statefulSet *appsv1.StatefulSet) (string, error) {
	if statefulSet == nil {
		return "", fmt.Errorf("StatefulSet对象不能为空")
	}

	// 清理不需要的字段
	cleanStatefulSet := statefulSet.DeepCopy()
	cleanStatefulSet.ManagedFields = nil
	cleanStatefulSet.Status = appsv1.StatefulSetStatus{}

	yamlBytes, err := yaml.Marshal(cleanStatefulSet)
	if err != nil {
		return "", fmt.Errorf("YAML序列化失败: %w", err)
	}

	return string(yamlBytes), nil
}

// ValidateStatefulSet 验证StatefulSet配置
func ValidateStatefulSet(statefulSet *appsv1.StatefulSet) error {
	if statefulSet == nil {
		return fmt.Errorf("StatefulSet对象不能为空")
	}

	if statefulSet.Name == "" {
		return fmt.Errorf("StatefulSet名称不能为空")
	}

	if statefulSet.Namespace == "" {
		return fmt.Errorf("StatefulSet命名空间不能为空")
	}

	if statefulSet.Spec.ServiceName == "" {
		return fmt.Errorf("StatefulSet服务名称不能为空")
	}

	if len(statefulSet.Spec.Template.Spec.Containers) == 0 {
		return fmt.Errorf("StatefulSet必须包含至少一个容器")
	}

	for i, container := range statefulSet.Spec.Template.Spec.Containers {
		if container.Name == "" {
			return fmt.Errorf("第%d个容器名称不能为空", i+1)
		}
		if container.Image == "" {
			return fmt.Errorf("第%d个容器镜像不能为空", i+1)
		}
	}

	return nil
}

// BuildStatefulSetListOptions 构建StatefulSet列表查询选项
func BuildStatefulSetListOptions(req *model.GetStatefulSetListReq) metav1.ListOptions {
	return metav1.ListOptions{}
}

// PaginateK8sStatefulSets 对StatefulSet列表进行分页
func PaginateK8sStatefulSets(statefulSets []*model.K8sStatefulSet, page, size int) ([]*model.K8sStatefulSet, int64) {
	total := int64(len(statefulSets))

	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 10
	}

	start := (page - 1) * size
	end := start + size

	if start >= len(statefulSets) {
		return []*model.K8sStatefulSet{}, total
	}

	if end > len(statefulSets) {
		end = len(statefulSets)
	}

	return statefulSets[start:end], total
}

// BuildK8sStatefulSetHistory 构建StatefulSet历史版本模型
func BuildK8sStatefulSetHistory(revision appsv1.ControllerRevision) (*model.K8sStatefulSetHistory, error) {
	return &model.K8sStatefulSetHistory{
		Revision: revision.Revision,
		Date:     revision.CreationTimestamp.Time,
		Message:  GetChangeReason(revision.Annotations),
	}, nil
}

// ExtractStatefulSetFromRevision 从ControllerRevision提取StatefulSet配置用于回滚
func ExtractStatefulSetFromRevision(revision *appsv1.ControllerRevision, statefulSet *appsv1.StatefulSet) error {
	if revision == nil {
		return fmt.Errorf("ControllerRevision不能为空")
	}

	if statefulSet == nil {
		return fmt.Errorf("StatefulSet对象不能为空")
	}

	if len(revision.Data.Raw) == 0 {
		return fmt.Errorf("ControllerRevision数据为空")
	}

	var revisionStatefulSet appsv1.StatefulSet
	if err := json.Unmarshal(revision.Data.Raw, &revisionStatefulSet); err != nil {
		var patchData map[string]interface{}
		if err := json.Unmarshal(revision.Data.Raw, &patchData); err != nil {
			return fmt.Errorf("反序列化数据失败: %w", err)
		}

		if spec, ok := patchData["spec"]; ok {
			specBytes, err := json.Marshal(spec)
			if err != nil {
				return fmt.Errorf("序列化spec失败: %w", err)
			}

			var statefulSetSpec appsv1.StatefulSetSpec
			if err := json.Unmarshal(specBytes, &statefulSetSpec); err != nil {
				return fmt.Errorf("反序列化spec失败: %w", err)
			}

			statefulSet.Spec = statefulSetSpec
			return nil
		}

		return fmt.Errorf("无法提取StatefulSet配置")
	}

	statefulSet.Spec = revisionStatefulSet.Spec
	if revisionStatefulSet.Labels != nil {
		statefulSet.Labels = revisionStatefulSet.Labels
	}
	if revisionStatefulSet.Annotations != nil {
		statefulSet.Annotations = revisionStatefulSet.Annotations
	}

	return nil
}

// SortStatefulSetsByCreationTime 按创建时间排序StatefulSet列表
func SortStatefulSetsByCreationTime(statefulSets []*model.K8sStatefulSet, desc bool) {
	sort.Slice(statefulSets, func(i, j int) bool {
		if desc {
			return statefulSets[i].CreatedAt.After(statefulSets[j].CreatedAt)
		}
		return statefulSets[i].CreatedAt.Before(statefulSets[j].CreatedAt)
	})
}

// FilterStatefulSetsByStatus 按状态过滤StatefulSet列表
func FilterStatefulSetsByStatus(statefulSets []*model.K8sStatefulSet, status string) []*model.K8sStatefulSet {
	if status == "" {
		return statefulSets
	}

	var filtered []*model.K8sStatefulSet
	for _, ss := range statefulSets {
		statusStr := getStatefulSetStatusString(ss.Status)
		if strings.EqualFold(statusStr, status) {
			filtered = append(filtered, ss)
		}
	}

	return filtered
}

// getStatefulSetStatusString 获取StatefulSet状态字符串
func getStatefulSetStatusString(status model.K8sStatefulSetStatus) string {
	switch status {
	case model.K8sStatefulSetStatusRunning:
		return "running"
	case model.K8sStatefulSetStatusStopped:
		return "stopped"
	case model.K8sStatefulSetStatusUpdating:
		return "updating"
	case model.K8sStatefulSetStatusError:
		return "error"
	default:
		return "unknown"
	}
}

// BuildStatefulSetFromYaml 从YAML构建StatefulSet对象
func BuildStatefulSetFromYaml(req *model.CreateStatefulSetByYamlReq) (*appsv1.StatefulSet, error) {
	if req == nil {
		return nil, fmt.Errorf("请求不能为空")
	}

	if req.YAML == "" {
		return nil, fmt.Errorf("YAML内容不能为空")
	}

	statefulSet, err := YAMLToStatefulSet(req.YAML)
	if err != nil {
		return nil, err
	}

	if statefulSet.Namespace == "" {
		statefulSet.Namespace = "default"
	}

	if statefulSet.Name == "" {
		return nil, fmt.Errorf("YAML中必须指定name")
	}

	return statefulSet, nil
}

// BuildStatefulSetFromYamlForUpdate 构建用于更新的StatefulSet对象
func BuildStatefulSetFromYamlForUpdate(req *model.UpdateStatefulSetByYamlReq) (*appsv1.StatefulSet, error) {
	if req == nil {
		return nil, fmt.Errorf("请求不能为空")
	}

	if req.YAML == "" {
		return nil, fmt.Errorf("YAML内容不能为空")
	}

	statefulSet, err := YAMLToStatefulSet(req.YAML)
	if err != nil {
		return nil, err
	}

	if statefulSet.Namespace != "" && statefulSet.Namespace != req.Namespace {
		return nil, fmt.Errorf("YAML中的namespace与请求参数不一致")
	}

	if statefulSet.Name != "" && statefulSet.Name != req.Name {
		return nil, fmt.Errorf("YAML中的name与请求参数不一致")
	}

	if statefulSet.Namespace == "" {
		statefulSet.Namespace = req.Namespace
	}

	if statefulSet.Name == "" {
		statefulSet.Name = req.Name
	}

	return statefulSet, nil
}

// ConvertToK8sStatefulSet 将 appsv1.StatefulSet 转换为 model.K8sStatefulSet
func ConvertToK8sStatefulSet(statefulSet *appsv1.StatefulSet) *model.K8sStatefulSet {
	if statefulSet == nil {
		return nil
	}

	status := getStatefulSetStatus(*statefulSet)

	updateStrategy := "RollingUpdate"
	if statefulSet.Spec.UpdateStrategy.Type == appsv1.OnDeleteStatefulSetStrategyType {
		updateStrategy = "OnDelete"
	}

	podManagementPolicy := string(appsv1.OrderedReadyPodManagement)
	if statefulSet.Spec.PodManagementPolicy != "" {
		podManagementPolicy = string(statefulSet.Spec.PodManagementPolicy)
	}

	var images []string
	for _, container := range statefulSet.Spec.Template.Spec.Containers {
		images = append(images, container.Image)
	}

	selector := make(map[string]string)
	if statefulSet.Spec.Selector != nil && statefulSet.Spec.Selector.MatchLabels != nil {
		selector = statefulSet.Spec.Selector.MatchLabels
	}

	var conditions []model.StatefulSetCondition
	for _, condition := range statefulSet.Status.Conditions {
		stsCondition := model.StatefulSetCondition{
			Type:               string(condition.Type),
			Status:             string(condition.Status),
			LastUpdateTime:     condition.LastTransitionTime.Time,
			LastTransitionTime: condition.LastTransitionTime.Time,
			Reason:             condition.Reason,
			Message:            condition.Message,
		}
		conditions = append(conditions, stsCondition)
	}

	revisionHistoryLimit := int32(10)
	if statefulSet.Spec.RevisionHistoryLimit != nil {
		revisionHistoryLimit = *statefulSet.Spec.RevisionHistoryLimit
	}

	replicas := int32(0)
	if statefulSet.Spec.Replicas != nil {
		replicas = *statefulSet.Spec.Replicas
	}

	return &model.K8sStatefulSet{
		Name:                 statefulSet.Name,
		Namespace:            statefulSet.Namespace,
		UID:                  string(statefulSet.UID),
		Labels:               statefulSet.Labels,
		Annotations:          statefulSet.Annotations,
		CreatedAt:            statefulSet.CreationTimestamp.Time,
		UpdatedAt:            time.Now(),
		Status:               status,
		Replicas:             replicas,
		ReadyReplicas:        statefulSet.Status.ReadyReplicas,
		CurrentReplicas:      statefulSet.Status.CurrentReplicas,
		UpdatedReplicas:      statefulSet.Status.UpdatedReplicas,
		Images:               images,
		Selector:             selector,
		ServiceName:          statefulSet.Spec.ServiceName,
		UpdateStrategy:       updateStrategy,
		PodManagementPolicy:  podManagementPolicy,
		RevisionHistoryLimit: revisionHistoryLimit,
		Conditions:           conditions,
		RawStatefulSet:       statefulSet,
	}
}
