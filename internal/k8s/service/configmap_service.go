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
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

type ConfigMapService interface {
	GetConfigMapList(ctx context.Context, req *model.K8sListRequest) ([]*model.K8sConfigMap, error)
	GetConfigMap(ctx context.Context, req *model.K8sResourceIdentifier) (*model.K8sConfigMap, error)
	CreateConfigMap(ctx context.Context, req *model.ConfigMapCreateRequest) error
	UpdateConfigMap(ctx context.Context, req *model.ConfigMapUpdateRequest) error
	DeleteConfigMap(ctx context.Context, req *model.K8sResourceIdentifier) error
	BatchDeleteConfigMaps(ctx context.Context, req *model.K8sBatchDeleteRequest) error
	GetConfigMapYAML(ctx context.Context, req *model.K8sResourceIdentifier) (string, error)
}

type configMapService struct {
	k8sClient client.K8sClient
	logger    *zap.Logger
}

func NewConfigMapService(k8sClient client.K8sClient, logger *zap.Logger) ConfigMapService {
	return &configMapService{
		k8sClient: k8sClient,
		logger:    logger,
	}
}

// GetConfigMapList 获取ConfigMap列表
func (s *configMapService) GetConfigMapList(ctx context.Context, req *model.K8sListRequest) ([]*model.K8sConfigMap, error) {
	clientset, err := s.k8sClient.GetKubeClient(req.ClusterID)
	if err != nil {
		s.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("cluster_id", req.ClusterID))
		return nil, fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	listOptions := req.ToMetaV1ListOptions()
	configMapList, err := clientset.CoreV1().ConfigMaps(req.Namespace).List(ctx, listOptions)
	if err != nil {
		s.logger.Error("获取ConfigMap列表失败", zap.Error(err), 
			zap.Int("cluster_id", req.ClusterID), zap.String("namespace", req.Namespace))
		return nil, fmt.Errorf("获取ConfigMap列表失败: %w", err)
	}

	result := make([]*model.K8sConfigMap, 0, len(configMapList.Items))
	for _, configMap := range configMapList.Items {
		k8sConfigMap := s.convertToK8sConfigMap(&configMap)
		result = append(result, k8sConfigMap)
	}

	s.logger.Info("成功获取ConfigMap列表", 
		zap.Int("cluster_id", req.ClusterID), 
		zap.String("namespace", req.Namespace),
		zap.Int("count", len(result)))

	return result, nil
}

// GetConfigMap 获取单个ConfigMap详情
func (s *configMapService) GetConfigMap(ctx context.Context, req *model.K8sResourceIdentifier) (*model.K8sConfigMap, error) {
	clientset, err := s.k8sClient.GetKubeClient(req.ClusterID)
	if err != nil {
		s.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("cluster_id", req.ClusterID))
		return nil, fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	configMap, err := clientset.CoreV1().ConfigMaps(req.Namespace).Get(ctx, req.ResourceName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, fmt.Errorf("ConfigMap不存在: %s/%s", req.Namespace, req.ResourceName)
		}
		s.logger.Error("获取ConfigMap失败", zap.Error(err),
			zap.Int("cluster_id", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.ResourceName))
		return nil, fmt.Errorf("获取ConfigMap失败: %w", err)
	}

	result := s.convertToK8sConfigMap(configMap)

	s.logger.Info("成功获取ConfigMap详情",
		zap.Int("cluster_id", req.ClusterID),
		zap.String("namespace", req.Namespace),
		zap.String("name", req.ResourceName))

	return result, nil
}

// CreateConfigMap 创建ConfigMap
func (s *configMapService) CreateConfigMap(ctx context.Context, req *model.ConfigMapCreateRequest) error {
	clientset, err := s.k8sClient.GetKubeClient(req.ClusterID)
	if err != nil {
		s.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("cluster_id", req.ClusterID))
		return fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	// 构造ConfigMap对象
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:        req.Name,
			Namespace:   req.Namespace,
			Labels:      req.Labels,
			Annotations: req.Annotations,
		},
		Data:       req.Data,
		BinaryData: req.BinaryData,
	}

	_, err = clientset.CoreV1().ConfigMaps(req.Namespace).Create(ctx, configMap, metav1.CreateOptions{})
	if err != nil {
		if errors.IsAlreadyExists(err) {
			return fmt.Errorf("ConfigMap已存在: %s/%s", req.Namespace, req.Name)
		}
		s.logger.Error("创建ConfigMap失败", zap.Error(err),
			zap.Int("cluster_id", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("创建ConfigMap失败: %w", err)
	}

	s.logger.Info("成功创建ConfigMap",
		zap.Int("cluster_id", req.ClusterID),
		zap.String("namespace", req.Namespace),
		zap.String("name", req.Name))

	return nil
}

// UpdateConfigMap 更新ConfigMap
func (s *configMapService) UpdateConfigMap(ctx context.Context, req *model.ConfigMapUpdateRequest) error {
	clientset, err := s.k8sClient.GetKubeClient(req.ClusterID)
	if err != nil {
		s.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("cluster_id", req.ClusterID))
		return fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	// 先获取现有的ConfigMap
	existingConfigMap, err := clientset.CoreV1().ConfigMaps(req.Namespace).Get(ctx, req.ResourceName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return fmt.Errorf("ConfigMap不存在: %s/%s", req.Namespace, req.ResourceName)
		}
		s.logger.Error("获取ConfigMap失败", zap.Error(err),
			zap.Int("cluster_id", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.ResourceName))
		return fmt.Errorf("获取ConfigMap失败: %w", err)
	}

	// 更新ConfigMap数据
	existingConfigMap.Data = req.Data
	existingConfigMap.BinaryData = req.BinaryData
	if req.Labels != nil {
		existingConfigMap.Labels = req.Labels
	}
	if req.Annotations != nil {
		existingConfigMap.Annotations = req.Annotations
	}

	_, err = clientset.CoreV1().ConfigMaps(req.Namespace).Update(ctx, existingConfigMap, metav1.UpdateOptions{})
	if err != nil {
		s.logger.Error("更新ConfigMap失败", zap.Error(err),
			zap.Int("cluster_id", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.ResourceName))
		return fmt.Errorf("更新ConfigMap失败: %w", err)
	}

	s.logger.Info("成功更新ConfigMap",
		zap.Int("cluster_id", req.ClusterID),
		zap.String("namespace", req.Namespace),
		zap.String("name", req.ResourceName))

	return nil
}

// DeleteConfigMap 删除ConfigMap
func (s *configMapService) DeleteConfigMap(ctx context.Context, req *model.K8sResourceIdentifier) error {
	clientset, err := s.k8sClient.GetKubeClient(req.ClusterID)
	if err != nil {
		s.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("cluster_id", req.ClusterID))
		return fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	err = clientset.CoreV1().ConfigMaps(req.Namespace).Delete(ctx, req.ResourceName, metav1.DeleteOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return fmt.Errorf("ConfigMap不存在: %s/%s", req.Namespace, req.ResourceName)
		}
		s.logger.Error("删除ConfigMap失败", zap.Error(err),
			zap.Int("cluster_id", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.ResourceName))
		return fmt.Errorf("删除ConfigMap失败: %w", err)
	}

	s.logger.Info("成功删除ConfigMap",
		zap.Int("cluster_id", req.ClusterID),
		zap.String("namespace", req.Namespace),
		zap.String("name", req.ResourceName))

	return nil
}

// BatchDeleteConfigMaps 批量删除ConfigMap
func (s *configMapService) BatchDeleteConfigMaps(ctx context.Context, req *model.K8sBatchDeleteRequest) error {
	clientset, err := s.k8sClient.GetKubeClient(req.ClusterID)
	if err != nil {
		s.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("cluster_id", req.ClusterID))
		return fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	var failedDeletes []string
	successCount := 0

	for _, name := range req.ResourceNames {
		err = clientset.CoreV1().ConfigMaps(req.Namespace).Delete(ctx, name, metav1.DeleteOptions{})
		if err != nil {
			if !errors.IsNotFound(err) {
				failedDeletes = append(failedDeletes, fmt.Sprintf("%s: %v", name, err))
				s.logger.Error("删除ConfigMap失败", zap.Error(err),
					zap.Int("cluster_id", req.ClusterID),
					zap.String("namespace", req.Namespace),
					zap.String("name", name))
			}
		} else {
			successCount++
			s.logger.Info("成功删除ConfigMap",
				zap.Int("cluster_id", req.ClusterID),
				zap.String("namespace", req.Namespace),
				zap.String("name", name))
		}
	}

	if len(failedDeletes) > 0 {
		return fmt.Errorf("部分ConfigMap删除失败: %v", failedDeletes)
	}

	s.logger.Info("批量删除ConfigMap完成",
		zap.Int("cluster_id", req.ClusterID),
		zap.String("namespace", req.Namespace),
		zap.Int("success_count", successCount),
		zap.Int("total_count", len(req.ResourceNames)))

	return nil
}

// GetConfigMapYAML 获取ConfigMap的YAML配置
func (s *configMapService) GetConfigMapYAML(ctx context.Context, req *model.K8sResourceIdentifier) (string, error) {
	clientset, err := s.k8sClient.GetKubeClient(req.ClusterID)
	if err != nil {
		s.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("cluster_id", req.ClusterID))
		return "", fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	configMap, err := clientset.CoreV1().ConfigMaps(req.Namespace).Get(ctx, req.ResourceName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return "", fmt.Errorf("ConfigMap不存在: %s/%s", req.Namespace, req.ResourceName)
		}
		s.logger.Error("获取ConfigMap失败", zap.Error(err),
			zap.Int("cluster_id", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.ResourceName))
		return "", fmt.Errorf("获取ConfigMap失败: %w", err)
	}

	// 清除不需要的字段
	configMap.ManagedFields = nil

	yamlData, err := yaml.Marshal(configMap)
	if err != nil {
		s.logger.Error("转换ConfigMap为YAML失败", zap.Error(err))
		return "", fmt.Errorf("转换ConfigMap为YAML失败: %w", err)
	}

	s.logger.Info("成功获取ConfigMap YAML",
		zap.Int("cluster_id", req.ClusterID),
		zap.String("namespace", req.Namespace),
		zap.String("name", req.ResourceName))

	return string(yamlData), nil
}

// convertToK8sConfigMap 将Kubernetes ConfigMap转换为模型对象
func (s *configMapService) convertToK8sConfigMap(configMap *corev1.ConfigMap) *model.K8sConfigMap {
	return &model.K8sConfigMap{
		Name:              configMap.Name,
		UID:               string(configMap.UID),
		Namespace:         configMap.Namespace,
		Data:              configMap.Data,
		BinaryData:        configMap.BinaryData,
		Labels:            configMap.Labels,
		Annotations:       configMap.Annotations,
		CreationTimestamp: configMap.CreationTimestamp.Time,
		Age:               time.Since(configMap.CreationTimestamp.Time).String(),
	}
}