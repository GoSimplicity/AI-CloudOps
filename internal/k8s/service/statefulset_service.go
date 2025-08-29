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
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/manager"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/utils"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/yaml"
)

type StatefulSetService interface {
	GetStatefulSetList(ctx context.Context, req *model.K8sListReq) ([]*model.K8sStatefulSet, error)
	GetStatefulSet(ctx context.Context, req *model.K8sResourceIdentifierReq) (*model.K8sStatefulSet, error)
	CreateStatefulSet(ctx context.Context, req *model.StatefulSetCreateReq) error
	UpdateStatefulSet(ctx context.Context, req *model.StatefulSetUpdateReq) error
	ScaleStatefulSet(ctx context.Context, req *model.StatefulSetScaleReq) error
	DeleteStatefulSet(ctx context.Context, req *model.K8sResourceIdentifierReq) error
	BatchDeleteStatefulSets(ctx context.Context, req *model.K8sBatchDeleteReq) error
	GetStatefulSetYAML(ctx context.Context, req *model.K8sResourceIdentifierReq) (string, error)
}

type statefulSetService struct {
	k8sClient          client.K8sClient           // 保持向后兼容
	statefulSetManager manager.StatefulSetManager // 新的依赖注入
	logger             *zap.Logger
}

func NewStatefulSetService(k8sClient client.K8sClient, statefulSetManager manager.StatefulSetManager, logger *zap.Logger) StatefulSetService {
	return &statefulSetService{
		k8sClient:          k8sClient,
		statefulSetManager: statefulSetManager,
		logger:             logger,
	}
}

// GetStatefulSetList 获取StatefulSet列表
func (s *statefulSetService) GetStatefulSetList(ctx context.Context, req *model.K8sListReq) ([]*model.K8sStatefulSet, error) {
	// 使用 StatefulSetManager 获取列表
	listOptions := utils.ConvertK8sListReqToMetaV1ListOptions(req)
	statefulSetList, err := s.statefulSetManager.GetStatefulSetList(ctx, req.ClusterID, req.Namespace, listOptions)
	if err != nil {
		s.logger.Error("获取StatefulSet列表失败", zap.Error(err),
			zap.Int("cluster_id", req.ClusterID), zap.String("namespace", req.Namespace))
		return nil, fmt.Errorf("获取StatefulSet列表失败: %w", err)
	}

	result := make([]*model.K8sStatefulSet, 0, len(statefulSetList.Items))
	for _, sts := range statefulSetList.Items {
		k8sStatefulSet := s.convertToK8sStatefulSet(&sts)
		result = append(result, k8sStatefulSet)
	}

	s.logger.Info("成功获取StatefulSet列表",
		zap.Int("cluster_id", req.ClusterID),
		zap.String("namespace", req.Namespace),
		zap.Int("count", len(result)))

	return result, nil
}

// GetStatefulSet 获取单个StatefulSet详情
func (s *statefulSetService) GetStatefulSet(ctx context.Context, req *model.K8sResourceIdentifierReq) (*model.K8sStatefulSet, error) {
	clientset, err := s.k8sClient.GetKubeClient(req.ClusterID)
	if err != nil {
		s.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("cluster_id", req.ClusterID))
		return nil, fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	sts, err := clientset.AppsV1().StatefulSets(req.Namespace).Get(ctx, req.ResourceName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, fmt.Errorf("StatefulSet不存在: %s/%s", req.Namespace, req.ResourceName)
		}
		s.logger.Error("获取StatefulSet失败", zap.Error(err),
			zap.Int("cluster_id", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.ResourceName))
		return nil, fmt.Errorf("获取StatefulSet失败: %w", err)
	}

	result := s.convertToK8sStatefulSet(sts)
	return result, nil
}

// CreateStatefulSet 创建StatefulSet
func (s *statefulSetService) CreateStatefulSet(ctx context.Context, req *model.StatefulSetCreateReq) error {
	clientset, err := s.k8sClient.GetKubeClient(req.ClusterID)
	if err != nil {
		s.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("cluster_id", req.ClusterID))
		return fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	// 构造StatefulSet对象
	sts := s.buildStatefulSetFromCreateRequest(req)

	_, err = clientset.AppsV1().StatefulSets(req.Namespace).Create(ctx, sts, metav1.CreateOptions{})
	if err != nil {
		if errors.IsAlreadyExists(err) {
			return fmt.Errorf("StatefulSet已存在: %s/%s", req.Namespace, req.Name)
		}
		s.logger.Error("创建StatefulSet失败", zap.Error(err),
			zap.Int("cluster_id", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("创建StatefulSet失败: %w", err)
	}

	s.logger.Info("成功创建StatefulSet",
		zap.Int("cluster_id", req.ClusterID),
		zap.String("namespace", req.Namespace),
		zap.String("name", req.Name))

	return nil
}

// UpdateStatefulSet 更新StatefulSet
func (s *statefulSetService) UpdateStatefulSet(ctx context.Context, req *model.StatefulSetUpdateReq) error {
	clientset, err := s.k8sClient.GetKubeClient(req.ClusterID)
	if err != nil {
		s.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("cluster_id", req.ClusterID))
		return fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	// 先获取现有的StatefulSet
	existingSts, err := clientset.AppsV1().StatefulSets(req.Namespace).Get(ctx, req.ResourceName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return fmt.Errorf("StatefulSet不存在: %s/%s", req.Namespace, req.ResourceName)
		}
		return fmt.Errorf("获取StatefulSet失败: %w", err)
	}

	// 更新StatefulSet
	s.updateStatefulSetFromUpdateRequest(existingSts, req)

	_, err = clientset.AppsV1().StatefulSets(req.Namespace).Update(ctx, existingSts, metav1.UpdateOptions{})
	if err != nil {
		s.logger.Error("更新StatefulSet失败", zap.Error(err),
			zap.Int("cluster_id", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.ResourceName))
		return fmt.Errorf("更新StatefulSet失败: %w", err)
	}

	s.logger.Info("成功更新StatefulSet",
		zap.Int("cluster_id", req.ClusterID),
		zap.String("namespace", req.Namespace),
		zap.String("name", req.ResourceName))

	return nil
}

// ScaleStatefulSet 扩缩容StatefulSet
func (s *statefulSetService) ScaleStatefulSet(ctx context.Context, req *model.StatefulSetScaleReq) error {
	clientset, err := s.k8sClient.GetKubeClient(req.ClusterID)
	if err != nil {
		s.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("cluster_id", req.ClusterID))
		return fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	scale, err := clientset.AppsV1().StatefulSets(req.Namespace).GetScale(ctx, req.ResourceName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return fmt.Errorf("StatefulSet不存在: %s/%s", req.Namespace, req.ResourceName)
		}
		return fmt.Errorf("获取StatefulSet Scale失败: %w", err)
	}

	scale.Spec.Replicas = req.Replicas

	_, err = clientset.AppsV1().StatefulSets(req.Namespace).UpdateScale(ctx, req.ResourceName, scale, metav1.UpdateOptions{})
	if err != nil {
		s.logger.Error("扩缩容StatefulSet失败", zap.Error(err),
			zap.Int("cluster_id", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.ResourceName),
			zap.Int32("replicas", req.Replicas))
		return fmt.Errorf("扩缩容StatefulSet失败: %w", err)
	}

	s.logger.Info("成功扩缩容StatefulSet",
		zap.Int("cluster_id", req.ClusterID),
		zap.String("namespace", req.Namespace),
		zap.String("name", req.ResourceName),
		zap.Int32("replicas", req.Replicas))

	return nil
}

// DeleteStatefulSet 删除StatefulSet
func (s *statefulSetService) DeleteStatefulSet(ctx context.Context, req *model.K8sResourceIdentifierReq) error {
	clientset, err := s.k8sClient.GetKubeClient(req.ClusterID)
	if err != nil {
		s.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("cluster_id", req.ClusterID))
		return fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	err = clientset.AppsV1().StatefulSets(req.Namespace).Delete(ctx, req.ResourceName, metav1.DeleteOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return fmt.Errorf("StatefulSet不存在: %s/%s", req.Namespace, req.ResourceName)
		}
		s.logger.Error("删除StatefulSet失败", zap.Error(err),
			zap.Int("cluster_id", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.ResourceName))
		return fmt.Errorf("删除StatefulSet失败: %w", err)
	}

	s.logger.Info("成功删除StatefulSet",
		zap.Int("cluster_id", req.ClusterID),
		zap.String("namespace", req.Namespace),
		zap.String("name", req.ResourceName))

	return nil
}

// BatchDeleteStatefulSets 批量删除StatefulSet
func (s *statefulSetService) BatchDeleteStatefulSets(ctx context.Context, req *model.K8sBatchDeleteReq) error {
	clientset, err := s.k8sClient.GetKubeClient(req.ClusterID)
	if err != nil {
		s.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("cluster_id", req.ClusterID))
		return fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	var failedDeletes []string
	successCount := 0

	for _, name := range req.ResourceNames {
		err = clientset.AppsV1().StatefulSets(req.Namespace).Delete(ctx, name, metav1.DeleteOptions{})
		if err != nil {
			if !errors.IsNotFound(err) {
				failedDeletes = append(failedDeletes, fmt.Sprintf("%s: %v", name, err))
			}
		} else {
			successCount++
		}
	}

	if len(failedDeletes) > 0 {
		return fmt.Errorf("部分StatefulSet删除失败: %v", failedDeletes)
	}

	s.logger.Info("批量删除StatefulSet完成",
		zap.Int("cluster_id", req.ClusterID),
		zap.String("namespace", req.Namespace),
		zap.Int("success_count", successCount))

	return nil
}

// GetStatefulSetYAML 获取StatefulSet的YAML配置
func (s *statefulSetService) GetStatefulSetYAML(ctx context.Context, req *model.K8sResourceIdentifierReq) (string, error) {
	clientset, err := s.k8sClient.GetKubeClient(req.ClusterID)
	if err != nil {
		s.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("cluster_id", req.ClusterID))
		return "", fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	sts, err := clientset.AppsV1().StatefulSets(req.Namespace).Get(ctx, req.ResourceName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return "", fmt.Errorf("StatefulSet不存在: %s/%s", req.Namespace, req.ResourceName)
		}
		return "", fmt.Errorf("获取StatefulSet失败: %w", err)
	}

	// 清除不需要的字段
	sts.ManagedFields = nil
	sts.Status = appsv1.StatefulSetStatus{}

	yamlData, err := yaml.Marshal(sts)
	if err != nil {
		s.logger.Error("转换StatefulSet为YAML失败", zap.Error(err))
		return "", fmt.Errorf("转换StatefulSet为YAML失败: %w", err)
	}

	return string(yamlData), nil
}

// convertToK8sStatefulSet 将Kubernetes StatefulSet转换为模型对象
func (s *statefulSetService) convertToK8sStatefulSet(sts *appsv1.StatefulSet) *model.K8sStatefulSet {
	// 提取镜像信息
	images := make([]string, 0)
	for _, container := range sts.Spec.Template.Spec.Containers {
		images = append(images, container.Image)
	}

	return &model.K8sStatefulSet{
		Name:              sts.Name,
		UID:               string(sts.UID),
		Namespace:         sts.Namespace,
		Replicas:          *sts.Spec.Replicas,
		ReadyReplicas:     sts.Status.ReadyReplicas,
		CurrentReplicas:   sts.Status.CurrentReplicas,
		UpdatedReplicas:   sts.Status.UpdatedReplicas,
		ServiceName:       sts.Spec.ServiceName,
		UpdateStrategy:    string(sts.Spec.UpdateStrategy.Type),
		Labels:            sts.Labels,
		Annotations:       sts.Annotations,
		CreationTimestamp: sts.CreationTimestamp.Time,
		Images:            images,
		Age:               time.Since(sts.CreationTimestamp.Time).String(),
	}
}

// buildStatefulSetFromCreateRequest 从创建请求构建StatefulSet对象
func (s *statefulSetService) buildStatefulSetFromCreateRequest(req *model.StatefulSetCreateReq) *appsv1.StatefulSet {
	// 构建容器端口
	ports := make([]corev1.ContainerPort, 0, len(req.Ports))
	for _, port := range req.Ports {
		ports = append(ports, corev1.ContainerPort{
			Name:          port.Name,
			ContainerPort: port.ContainerPort,
			Protocol:      corev1.Protocol(port.Protocol),
		})
	}

	// 构建环境变量
	envVars := make([]corev1.EnvVar, 0, len(req.Env))
	for _, env := range req.Env {
		envVar := corev1.EnvVar{
			Name:  env.Name,
			Value: env.Value,
		}
		envVars = append(envVars, envVar)
	}

	// 构建资源需求
	resources := corev1.ResourceRequirements{}
	if req.Resources.Requests.CPU != "" || req.Resources.Requests.Memory != "" {
		resources.Requests = corev1.ResourceList{}
		if req.Resources.Requests.CPU != "" {
			if cpu, err := resource.ParseQuantity(req.Resources.Requests.CPU); err == nil {
				resources.Requests[corev1.ResourceCPU] = cpu
			}
		}
		if req.Resources.Requests.Memory != "" {
			if memory, err := resource.ParseQuantity(req.Resources.Requests.Memory); err == nil {
				resources.Requests[corev1.ResourceMemory] = memory
			}
		}
	}

	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:        req.Name,
			Namespace:   req.Namespace,
			Labels:      req.Labels,
			Annotations: req.Annotations,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas:    ptr.To(req.Replicas),
			ServiceName: req.ServiceName,
			Selector: &metav1.LabelSelector{
				MatchLabels: req.Labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: req.Labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:      req.Name,
							Image:     req.Image,
							Ports:     ports,
							Env:       envVars,
							Resources: resources,
						},
					},
				},
			},
		},
	}
}

// updateStatefulSetFromUpdateRequest 从更新请求更新StatefulSet对象
func (s *statefulSetService) updateStatefulSetFromUpdateRequest(sts *appsv1.StatefulSet, req *model.StatefulSetUpdateReq) {
	if req.Replicas != nil {
		sts.Spec.Replicas = req.Replicas
	}

	if req.Image != "" {
		for i := range sts.Spec.Template.Spec.Containers {
			sts.Spec.Template.Spec.Containers[i].Image = req.Image
		}
	}

	if req.Labels != nil {
		sts.Labels = req.Labels
		sts.Spec.Template.Labels = req.Labels
	}

	if req.Annotations != nil {
		sts.Annotations = req.Annotations
	}
}
