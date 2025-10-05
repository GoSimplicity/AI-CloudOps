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
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/manager"
	k8sutils "github.com/GoSimplicity/AI-CloudOps/internal/k8s/utils"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ConfigMapService interface {
	GetConfigMapList(ctx context.Context, req *model.GetConfigMapListReq) (model.ListResp[*model.K8sConfigMap], error)
	GetConfigMap(ctx context.Context, req *model.GetConfigMapDetailsReq) (*model.K8sConfigMap, error)
	CreateConfigMap(ctx context.Context, req *model.CreateConfigMapReq) error
	UpdateConfigMap(ctx context.Context, req *model.UpdateConfigMapReq) error
	CreateConfigMapByYaml(ctx context.Context, req *model.CreateConfigMapByYamlReq) error
	UpdateConfigMapByYaml(ctx context.Context, req *model.UpdateConfigMapByYamlReq) error
	DeleteConfigMap(ctx context.Context, req *model.DeleteConfigMapReq) error
	GetConfigMapYAML(ctx context.Context, req *model.GetConfigMapYamlReq) (*model.K8sYaml, error)
}

// configMapService ConfigMap服务实现
type configMapService struct {
	k8sClient        client.K8sClient         // 保持向后兼容
	configMapManager manager.ConfigMapManager // 新的依赖注入
	logger           *zap.Logger
}

// NewConfigMapService 创建新的ConfigMap服务实例
func NewConfigMapService(k8sClient client.K8sClient, configMapManager manager.ConfigMapManager, logger *zap.Logger) ConfigMapService {
	return &configMapService{
		k8sClient:        k8sClient,
		configMapManager: configMapManager,
		logger:           logger,
	}
}

func (s *configMapService) GetConfigMapList(ctx context.Context, req *model.GetConfigMapListReq) (model.ListResp[*model.K8sConfigMap], error) {
	if req == nil {
		return model.ListResp[*model.K8sConfigMap]{}, fmt.Errorf("请求参数不能为空")
	}

	var list *corev1.ConfigMapList
	var err error

	labelSelector := ""
	if len(req.Labels) > 0 {
		var labels []string
		for k, v := range req.Labels {
			labels = append(labels, fmt.Sprintf("%s=%s", k, v))
		}
		labelSelector = strings.Join(labels, ",")
	}

	if labelSelector != "" {
		list, err = s.configMapManager.ListConfigMapsBySelector(ctx, req.ClusterID, req.Namespace, labelSelector)
	} else {
		list, err = s.configMapManager.ListConfigMaps(ctx, req.ClusterID, req.Namespace)
	}
	if err != nil {
		s.logger.Error("获取ConfigMap列表失败", zap.Error(err),
			zap.Int("cluster_id", req.ClusterID))
		return model.ListResp[*model.K8sConfigMap]{}, fmt.Errorf("获取ConfigMap列表失败: %w", err)
	}

	entities := make([]*model.K8sConfigMap, 0, len(list.Items))
	for _, cm := range list.Items {
		entity := s.convertToK8sConfigMap(&cm, req.ClusterID)
		entities = append(entities, entity)
	}

	page := req.Page
	size := req.Size
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 10
	}
	start := (page - 1) * size
	end := start + size
	total := int64(len(entities))
	if start > len(entities) {
		start = len(entities)
	}
	if end > len(entities) {
		end = len(entities)
	}

	return model.ListResp[*model.K8sConfigMap]{Items: entities[start:end], Total: total}, nil
}

func (s *configMapService) GetConfigMap(ctx context.Context, req *model.GetConfigMapDetailsReq) (*model.K8sConfigMap, error) {
	if req == nil {
		return nil, fmt.Errorf("请求参数不能为空")
	}

	configMap, err := s.configMapManager.GetConfigMap(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, fmt.Errorf("ConfigMap不存在: %s/%s", req.Namespace, req.Name)
		}
		s.logger.Error("获取ConfigMap失败", zap.Error(err),
			zap.Int("cluster_id", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return nil, fmt.Errorf("获取ConfigMap失败: %w", err)
	}

	result := s.convertToK8sConfigMap(configMap, req.ClusterID)

	s.logger.Info("成功获取ConfigMap详情",
		zap.Int("cluster_id", req.ClusterID),
		zap.String("namespace", req.Namespace),
		zap.String("name", req.Name))

	return result, nil
}

func (s *configMapService) CreateConfigMap(ctx context.Context, req *model.CreateConfigMapReq) error {
	if req == nil {
		return fmt.Errorf("请求参数不能为空")
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
		BinaryData: map[string][]byte(req.BinaryData),
	}

	// 设置不可变标志
	if req.Immutable {
		configMap.Immutable = &req.Immutable
	}

	_, err := s.configMapManager.CreateConfigMap(ctx, req.ClusterID, configMap)
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

func (s *configMapService) UpdateConfigMap(ctx context.Context, req *model.UpdateConfigMapReq) error {
	if req == nil {
		return fmt.Errorf("请求参数不能为空")
	}

	// 获取现有的ConfigMap以获取ResourceVersion（用于乐观锁）
	existingConfigMap, err := s.configMapManager.GetConfigMap(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		if errors.IsNotFound(err) {
			return fmt.Errorf("ConfigMap不存在: %s/%s", req.Namespace, req.Name)
		}
		s.logger.Error("获取ConfigMap失败", zap.Error(err),
			zap.Int("cluster_id", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("获取ConfigMap失败: %w", err)
	}

	// 创建新的ConfigMap对象进行完全覆盖更新（参考Deployment模块）
	// 只保留必要的元数据字段
	updatedConfigMap := existingConfigMap.DeepCopy()

	// 完全覆盖数据字段
	updatedConfigMap.Data = req.Data
	updatedConfigMap.BinaryData = map[string][]byte(req.BinaryData)
	updatedConfigMap.Labels = req.Labels
	updatedConfigMap.Annotations = req.Annotations

	// Immutable字段在创建后通常不能修改，保持原值
	// 如果需要修改，K8s会返回错误

	_, err = s.configMapManager.UpdateConfigMap(ctx, req.ClusterID, updatedConfigMap)
	if err != nil {
		s.logger.Error("更新ConfigMap失败", zap.Error(err),
			zap.Int("cluster_id", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("更新ConfigMap失败: %w", err)
	}

	s.logger.Info("成功更新ConfigMap (完全覆盖)",
		zap.Int("cluster_id", req.ClusterID),
		zap.String("namespace", req.Namespace),
		zap.String("name", req.Name))

	return nil
}

func (s *configMapService) DeleteConfigMap(ctx context.Context, req *model.DeleteConfigMapReq) error {
	if req == nil {
		return fmt.Errorf("请求参数不能为空")
	}

	err := s.configMapManager.DeleteConfigMap(ctx, req.ClusterID, req.Namespace, req.Name, metav1.DeleteOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return fmt.Errorf("ConfigMap不存在: %s/%s", req.Namespace, req.Name)
		}
		s.logger.Error("删除ConfigMap失败", zap.Error(err),
			zap.Int("cluster_id", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("删除ConfigMap失败: %w", err)
	}

	s.logger.Info("成功删除ConfigMap",
		zap.Int("cluster_id", req.ClusterID),
		zap.String("namespace", req.Namespace),
		zap.String("name", req.Name))

	return nil
}

func (s *configMapService) GetConfigMapYAML(ctx context.Context, req *model.GetConfigMapYamlReq) (*model.K8sYaml, error) {
	if req == nil {
		return nil, fmt.Errorf("请求参数不能为空")
	}

	configMap, err := s.configMapManager.GetConfigMap(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, fmt.Errorf("ConfigMap不存在: %s/%s", req.Namespace, req.Name)
		}
		s.logger.Error("获取ConfigMap失败", zap.Error(err),
			zap.Int("cluster_id", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return nil, fmt.Errorf("获取ConfigMap失败: %w", err)
	}

	yamlStr, err := k8sutils.ConfigMapToYAML(configMap)
	if err != nil {
		s.logger.Error("转换ConfigMap为YAML失败", zap.Error(err))
		return nil, fmt.Errorf("转换ConfigMap为YAML失败: %w", err)
	}

	s.logger.Info("成功获取ConfigMap YAML",
		zap.Int("cluster_id", req.ClusterID),
		zap.String("namespace", req.Namespace),
		zap.String("name", req.Name))

	return &model.K8sYaml{
		YAML: yamlStr,
	}, nil
}

// convertToK8sConfigMap 将Kubernetes ConfigMap转换为模型对象
func (s *configMapService) convertToK8sConfigMap(configMap *corev1.ConfigMap, clusterID int) *model.K8sConfigMap {
	// 计算数据大小
	var totalSize int64
	for _, v := range configMap.Data {
		totalSize += int64(len(v))
	}
	for _, v := range configMap.BinaryData {
		totalSize += int64(len(v))
	}

	// 格式化大小
	size := k8sutils.FormatBytes(totalSize)

	// 计算数据条目数量
	dataCount := len(configMap.Data) + len(configMap.BinaryData)

	// 判断是否不可变
	immutable := false
	if configMap.Immutable != nil {
		immutable = *configMap.Immutable
	}

	return &model.K8sConfigMap{
		Name:         configMap.Name,
		Namespace:    configMap.Namespace,
		ClusterID:    clusterID,
		UID:          string(configMap.UID),
		Data:         configMap.Data,
		BinaryData:   model.BinaryDataMap(configMap.BinaryData),
		Labels:       configMap.Labels,
		Annotations:  configMap.Annotations,
		Immutable:    immutable,
		DataCount:    dataCount,
		Size:         size,
		CreatedAt:    configMap.CreationTimestamp.Time,
		Age:          time.Since(configMap.CreationTimestamp.Time).String(),
		RawConfigMap: configMap,
	}
}

func (s *configMapService) CreateConfigMapByYaml(ctx context.Context, req *model.CreateConfigMapByYamlReq) error {
	if req == nil {
		return fmt.Errorf("请求参数不能为空")
	}

	cm, err := k8sutils.YAMLToConfigMap(req.YAML)
	if err != nil {
		return fmt.Errorf("解析YAML失败: %w", err)
	}

	// 如果YAML中没有指定namespace，使用default命名空间
	if cm.Namespace == "" {
		cm.Namespace = "default"
		s.logger.Info("YAML中未指定namespace，使用default命名空间",
			zap.Int("cluster_id", req.ClusterID),
			zap.String("name", cm.Name))
	}

	_, err = s.configMapManager.CreateConfigMap(ctx, req.ClusterID, cm)
	if err != nil {
		if errors.IsAlreadyExists(err) {
			return fmt.Errorf("ConfigMap已存在: %s/%s", cm.Namespace, cm.Name)
		}
		s.logger.Error("通过YAML创建ConfigMap失败", zap.Error(err),
			zap.Int("cluster_id", req.ClusterID), zap.String("namespace", cm.Namespace), zap.String("name", cm.Name))
		return fmt.Errorf("创建ConfigMap %s/%s 失败: %w", cm.Namespace, cm.Name, err)
	}

	s.logger.Info("成功通过YAML创建ConfigMap",
		zap.Int("cluster_id", req.ClusterID),
		zap.String("namespace", cm.Namespace),
		zap.String("name", cm.Name))

	return nil
}

func (s *configMapService) UpdateConfigMapByYaml(ctx context.Context, req *model.UpdateConfigMapByYamlReq) error {
	if req == nil {
		return fmt.Errorf("请求参数不能为空")
	}

	s.logger.Info("开始通过YAML更新ConfigMap",
		zap.Int("cluster_id", req.ClusterID),
		zap.String("namespace", req.Namespace),
		zap.String("name", req.Name))

	// 解析YAML
	cm, err := k8sutils.YAMLToConfigMap(req.YAML)
	if err != nil {
		s.logger.Error("解析YAML失败", zap.Error(err))
		return fmt.Errorf("解析YAML失败: %w", err)
	}

	// 强制设置命名空间和名称（以URL参数为准）
	cm.Namespace = req.Namespace
	cm.Name = req.Name

	// 获取现有资源以获取ResourceVersion（用于乐观锁，避免并发冲突）
	existing, err := s.configMapManager.GetConfigMap(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		if errors.IsNotFound(err) {
			return fmt.Errorf("ConfigMap不存在: %s/%s", req.Namespace, req.Name)
		}
		s.logger.Error("获取现有ConfigMap失败", zap.Error(err))
		return fmt.Errorf("获取现有ConfigMap失败: %w", err)
	}

	// 保留ResourceVersion和UID等关键元数据
	cm.ResourceVersion = existing.ResourceVersion
	cm.UID = existing.UID

	// 执行完全覆盖式更新
	_, err = s.configMapManager.UpdateConfigMap(ctx, req.ClusterID, cm)
	if err != nil {
		s.logger.Error("通过YAML更新ConfigMap失败", zap.Error(err),
			zap.Int("cluster_id", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("通过YAML更新ConfigMap失败: %w", err)
	}

	s.logger.Info("成功通过YAML更新ConfigMap (完全覆盖)",
		zap.Int("cluster_id", req.ClusterID),
		zap.String("namespace", req.Namespace),
		zap.String("name", req.Name))

	return nil
}
