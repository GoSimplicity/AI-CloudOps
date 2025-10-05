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
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/yaml"
)

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

func BuildStatefulSetFromRequest(req *model.CreateStatefulSetReq) (*appsv1.StatefulSet, error) {
	if req == nil {
		return nil, fmt.Errorf("请求不能为空")
	}

	// 如果提供了YAML，直接解析
	if req.YAML != "" {
		return YAMLToStatefulSet(req.YAML)
	}

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

// StatefulSetToYAML 将StatefulSet转换为YAML
func StatefulSetToYAML(statefulSet *appsv1.StatefulSet) (string, error) {
	if statefulSet == nil {
		return "", fmt.Errorf("statefulSet不能为空")
	}

	// 清理不需要的字段
	cleanStatefulSet := statefulSet.DeepCopy()
	cleanStatefulSet.Status = appsv1.StatefulSetStatus{}
	cleanStatefulSet.ManagedFields = nil
	cleanStatefulSet.ResourceVersion = ""
	cleanStatefulSet.UID = ""
	cleanStatefulSet.CreationTimestamp = metav1.Time{}
	cleanStatefulSet.Generation = 0

	yamlBytes, err := yaml.Marshal(cleanStatefulSet)
	if err != nil {
		return "", fmt.Errorf("转换为YAML失败: %w", err)
	}

	return string(yamlBytes), nil
}

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

func BuildStatefulSetListOptions(req *model.GetStatefulSetListReq) metav1.ListOptions {
	options := metav1.ListOptions{}

	var labelSelectors []string
	for key, value := range req.Labels {
		labelSelectors = append(labelSelectors, fmt.Sprintf("%s=%s", key, value))
	}
	if len(labelSelectors) > 0 {
		options.LabelSelector = strings.Join(labelSelectors, ",")
	}

	return options
}

func GetStatefulSetPods(ctx context.Context, kubeClient *kubernetes.Clientset, namespace, statefulSetName string) ([]*model.K8sPod, int64, error) {
	// 首先获取StatefulSet的标签选择器
	statefulSet, err := kubeClient.AppsV1().StatefulSets(namespace).Get(ctx, statefulSetName, metav1.GetOptions{})
	if err != nil {
		return nil, 0, fmt.Errorf("获取StatefulSet信息失败: %w", err)
	}

	var labelSelectors []string
	for key, value := range statefulSet.Spec.Selector.MatchLabels {
		labelSelectors = append(labelSelectors, fmt.Sprintf("%s=%s", key, value))
	}
	labelSelector := strings.Join(labelSelectors, ",")

	// 获取Pod列表
	podList, err := kubeClient.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("获取Pod列表失败: %w", err)
	}

	total := int64(len(podList.Items))
	var pods []*model.K8sPod
	for _, pod := range podList.Items {

		labelsJSON, _ := json.Marshal(pod.Labels)
		annotationsJSON, _ := json.Marshal(pod.Annotations)

		k8sPod := &model.K8sPod{
			Name:        pod.Name,
			Namespace:   pod.Namespace,
			Status:      string(pod.Status.Phase),
			NodeName:    pod.Spec.NodeName,
			Labels:      string(labelsJSON),
			Annotations: string(annotationsJSON),
		}
		pods = append(pods, k8sPod)
	}

	return pods, total, nil
}

// FilterStatefulSetsByStatus 根据StatefulSet状态过滤
func FilterStatefulSetsByStatus(statefulSets []appsv1.StatefulSet, status string) []appsv1.StatefulSet {
	if status == "" {
		return statefulSets
	}

	var filtered []appsv1.StatefulSet
	for _, statefulSet := range statefulSets {
		statefulSetStatus := getStatefulSetStatus(statefulSet)
		// 正确转换状态为字符串
		var statusStr string
		switch statefulSetStatus {
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
		if strings.EqualFold(statusStr, status) {
			filtered = append(filtered, statefulSet)
		}
	}

	return filtered
}

func BuildStatefulSetListPagination(statefulSets []appsv1.StatefulSet, page, size int) ([]appsv1.StatefulSet, int64) {
	total := int64(len(statefulSets))
	if total == 0 {
		return []appsv1.StatefulSet{}, 0
	}

	// 如果没有设置分页参数，返回所有数据
	if page <= 0 || size <= 0 {
		return statefulSets, total
	}

	start := int64((page - 1) * size)
	end := start + int64(size)

	if start >= total {
		return []appsv1.StatefulSet{}, total
	}
	if end > total {
		end = total
	}

	return statefulSets[start:end], total
}

// PaginateK8sStatefulSets 对 K8sStatefulSet 列表进行分页
func PaginateK8sStatefulSets(statefulSets []*model.K8sStatefulSet, page, size int) ([]*model.K8sStatefulSet, int64) {
	total := int64(len(statefulSets))
	if total == 0 {
		return []*model.K8sStatefulSet{}, 0
	}

	// 如果没有设置分页参数，返回所有数据
	if page <= 0 || size <= 0 {
		return statefulSets, total
	}

	start := int64((page - 1) * size)
	end := start + int64(size)

	if start >= total {
		return []*model.K8sStatefulSet{}, total
	}
	if end > total {
		end = total
	}

	return statefulSets[start:end], total
}

// IsStatefulSetReady 判断StatefulSet是否就绪
func IsStatefulSetReady(statefulSet appsv1.StatefulSet) bool {
	replicas := int32(0)
	if statefulSet.Spec.Replicas != nil {
		replicas = *statefulSet.Spec.Replicas
	}
	return statefulSet.Status.ReadyReplicas == replicas &&
		statefulSet.Status.UpdatedReplicas == replicas
}

func GetStatefulSetAge(statefulSet appsv1.StatefulSet) string {
	age := time.Since(statefulSet.CreationTimestamp.Time)
	days := int(age.Hours() / 24)
	if days > 0 {
		return fmt.Sprintf("%dd", days)
	}
	hours := int(age.Hours())
	if hours > 0 {
		return fmt.Sprintf("%dh", hours)
	}
	minutes := int(age.Minutes())
	return fmt.Sprintf("%dm", minutes)
}

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
