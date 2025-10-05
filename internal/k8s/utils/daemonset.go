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

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

func BuildK8sDaemonSet(ctx context.Context, clusterID int, daemonSet appsv1.DaemonSet) (*model.K8sDaemonSet, error) {
	if clusterID <= 0 {
		return nil, fmt.Errorf("无效的集群ID: %d", clusterID)
	}

	status := getDaemonSetStatus(daemonSet)

	updateStrategy := "RollingUpdate"
	if daemonSet.Spec.UpdateStrategy.Type == appsv1.OnDeleteDaemonSetStrategyType {
		updateStrategy = "OnDelete"
	}

	var images []string
	for _, container := range daemonSet.Spec.Template.Spec.Containers {
		images = append(images, container.Image)
	}

	selector := make(map[string]string)
	if daemonSet.Spec.Selector != nil && daemonSet.Spec.Selector.MatchLabels != nil {
		selector = daemonSet.Spec.Selector.MatchLabels
	}

	var conditions []model.DaemonSetCondition
	for _, condition := range daemonSet.Status.Conditions {
		dsCondition := model.DaemonSetCondition{
			Type:               string(condition.Type),
			Status:             string(condition.Status),
			LastUpdateTime:     condition.LastTransitionTime.Time,
			LastTransitionTime: condition.LastTransitionTime.Time,
			Reason:             condition.Reason,
			Message:            condition.Message,
		}
		conditions = append(conditions, dsCondition)
	}

	revisionHistoryLimit := int32(10)
	if daemonSet.Spec.RevisionHistoryLimit != nil {
		revisionHistoryLimit = *daemonSet.Spec.RevisionHistoryLimit
	}

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
		RevisionHistoryLimit:   revisionHistoryLimit,
		Conditions:             conditions,
		RawDaemonSet:           &daemonSet,
	}

	return k8sDaemonSet, nil
}

// getDaemonSetStatus 获取DaemonSet状态
func getDaemonSetStatus(daemonSet appsv1.DaemonSet) model.K8sDaemonSetStatus {
	desired := daemonSet.Status.DesiredNumberScheduled
	ready := daemonSet.Status.NumberReady
	available := daemonSet.Status.NumberAvailable
	unavailable := daemonSet.Status.NumberUnavailable

	if ready == desired && available == desired && desired > 0 {
		return model.K8sDaemonSetStatusRunning
	}

	if unavailable > 0 || ready < desired {
		return model.K8sDaemonSetStatusUpdating
	}

	if ready == 0 && desired == 0 {
		return model.K8sDaemonSetStatusRunning
	}

	return model.K8sDaemonSetStatusError
}

func BuildDaemonSetFromRequest(req *model.CreateDaemonSetReq) (*appsv1.DaemonSet, error) {
	if req == nil {
		return nil, fmt.Errorf("请求不能为空")
	}

	// 如果提供了YAML，直接解析
	if req.YAML != "" {
		return YAMLToDaemonSet(req.YAML)
	}

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

func BuildDaemonSetListOptions(req *model.GetDaemonSetListReq) metav1.ListOptions {
	return metav1.ListOptions{}
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

func BuildK8sDaemonSetHistory(revision appsv1.ControllerRevision) (*model.K8sDaemonSetHistory, error) {
	return &model.K8sDaemonSetHistory{
		Revision: revision.Revision,
		Date:     revision.CreationTimestamp.Time,
		Message:  GetChangeReason(revision.Annotations),
	}, nil
}

// ExtractDaemonSetFromRevision 从ControllerRevision提取DaemonSet配置用于回滚
func ExtractDaemonSetFromRevision(revision *appsv1.ControllerRevision, daemonSet *appsv1.DaemonSet) error {
	if revision == nil {
		return fmt.Errorf("ControllerRevision不能为空")
	}

	if daemonSet == nil {
		return fmt.Errorf("DaemonSet对象不能为空")
	}

	if len(revision.Data.Raw) == 0 {
		return fmt.Errorf("ControllerRevision数据为空")
	}

	var revisionDaemonSet appsv1.DaemonSet
	if err := json.Unmarshal(revision.Data.Raw, &revisionDaemonSet); err != nil {
		var patchData map[string]interface{}
		if err := json.Unmarshal(revision.Data.Raw, &patchData); err != nil {
			return fmt.Errorf("反序列化数据失败: %w", err)
		}

		if spec, ok := patchData["spec"]; ok {
			specBytes, err := json.Marshal(spec)
			if err != nil {
				return fmt.Errorf("序列化spec失败: %w", err)
			}

			var daemonSetSpec appsv1.DaemonSetSpec
			if err := json.Unmarshal(specBytes, &daemonSetSpec); err != nil {
				return fmt.Errorf("反序列化spec失败: %w", err)
			}

			daemonSet.Spec = daemonSetSpec
			return nil
		}

		return fmt.Errorf("无法提取DaemonSet配置")
	}

	daemonSet.Spec = revisionDaemonSet.Spec
	if revisionDaemonSet.Labels != nil {
		daemonSet.Labels = revisionDaemonSet.Labels
	}
	if revisionDaemonSet.Annotations != nil {
		daemonSet.Annotations = revisionDaemonSet.Annotations
	}

	return nil
}

func GetChangeReason(annotations map[string]string) string {
	if annotations == nil {
		return ""
	}

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

func BuildDaemonSetFromYaml(req *model.CreateDaemonSetByYamlReq) (*appsv1.DaemonSet, error) {
	if req == nil {
		return nil, fmt.Errorf("请求不能为空")
	}

	if req.YAML == "" {
		return nil, fmt.Errorf("YAML内容不能为空")
	}

	daemonSet, err := YAMLToDaemonSet(req.YAML)
	if err != nil {
		return nil, err
	}

	if daemonSet.Namespace == "" {
		daemonSet.Namespace = "default"
	}

	if daemonSet.Name == "" {
		return nil, fmt.Errorf("YAML中必须指定name")
	}

	return daemonSet, nil
}

func BuildDaemonSetFromYamlForUpdate(req *model.UpdateDaemonSetByYamlReq) (*appsv1.DaemonSet, error) {
	if req == nil {
		return nil, fmt.Errorf("请求不能为空")
	}

	if req.YAML == "" {
		return nil, fmt.Errorf("YAML内容不能为空")
	}

	daemonSet, err := YAMLToDaemonSet(req.YAML)
	if err != nil {
		return nil, err
	}

	if daemonSet.Namespace != "" && daemonSet.Namespace != req.Namespace {
		return nil, fmt.Errorf("YAML中的namespace与请求参数不一致")
	}

	if daemonSet.Name != "" && daemonSet.Name != req.Name {
		return nil, fmt.Errorf("YAML中的name与请求参数不一致")
	}

	if daemonSet.Namespace == "" {
		daemonSet.Namespace = req.Namespace
	}

	if daemonSet.Name == "" {
		daemonSet.Name = req.Name
	}

	return daemonSet, nil
}

func ConvertToK8sDaemonSet(daemonSet *appsv1.DaemonSet) *model.K8sDaemonSet {
	if daemonSet == nil {
		return nil
	}

	status := getDaemonSetStatus(*daemonSet)

	updateStrategy := "RollingUpdate"
	if daemonSet.Spec.UpdateStrategy.Type == appsv1.OnDeleteDaemonSetStrategyType {
		updateStrategy = "OnDelete"
	}

	var images []string
	for _, container := range daemonSet.Spec.Template.Spec.Containers {
		images = append(images, container.Image)
	}

	selector := make(map[string]string)
	if daemonSet.Spec.Selector != nil && daemonSet.Spec.Selector.MatchLabels != nil {
		selector = daemonSet.Spec.Selector.MatchLabels
	}

	var conditions []model.DaemonSetCondition
	for _, condition := range daemonSet.Status.Conditions {
		dsCondition := model.DaemonSetCondition{
			Type:               string(condition.Type),
			Status:             string(condition.Status),
			LastUpdateTime:     condition.LastTransitionTime.Time,
			LastTransitionTime: condition.LastTransitionTime.Time,
			Reason:             condition.Reason,
			Message:            condition.Message,
		}
		conditions = append(conditions, dsCondition)
	}

	revisionHistoryLimit := int32(10)
	if daemonSet.Spec.RevisionHistoryLimit != nil {
		revisionHistoryLimit = *daemonSet.Spec.RevisionHistoryLimit
	}

	return &model.K8sDaemonSet{
		Name:                   daemonSet.Name,
		Namespace:              daemonSet.Namespace,
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
		RevisionHistoryLimit:   revisionHistoryLimit,
		Conditions:             conditions,
		RawDaemonSet:           daemonSet,
	}
}
