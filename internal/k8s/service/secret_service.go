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

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/manager"
	k8sutils "github.com/GoSimplicity/AI-CloudOps/internal/k8s/utils"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SecretService Secret服务接口
type SecretService interface {
	GetSecretList(ctx context.Context, req *model.GetSecretListReq) (model.ListResp[*model.K8sSecret], error)
	GetSecret(ctx context.Context, req *model.GetSecretDetailsReq) (*model.K8sSecret, error)
	CreateSecret(ctx context.Context, req *model.CreateSecretReq) error
	UpdateSecret(ctx context.Context, req *model.UpdateSecretReq) error
	// YAML相关方法
	CreateSecretByYaml(ctx context.Context, req *model.CreateSecretByYamlReq) error
	UpdateSecretByYaml(ctx context.Context, req *model.UpdateSecretByYamlReq) error
	DeleteSecret(ctx context.Context, req *model.DeleteSecretReq) error
	GetSecretYAML(ctx context.Context, req *model.GetSecretYamlReq) (string, error)
}

// secretService Secret服务实现
type secretService struct {
	secretManager manager.SecretManager
	logger        *zap.Logger
}

// NewSecretService 创建新的Secret服务实例
func NewSecretService(secretManager manager.SecretManager, logger *zap.Logger) SecretService {
	return &secretService{
		secretManager: secretManager,
		logger:        logger,
	}
}

// GetSecretList 获取Secret列表
func (s *secretService) GetSecretList(ctx context.Context, req *model.GetSecretListReq) (model.ListResp[*model.K8sSecret], error) {
	var list *corev1.SecretList
	var err error

	// 构建标签选择器
	labelSelector := ""
	if len(req.Labels) > 0 {
		var labels []string
		for k, v := range req.Labels {
			labels = append(labels, fmt.Sprintf("%s=%s", k, v))
		}
		labelSelector = strings.Join(labels, ",")
	}

	if labelSelector != "" {
		list, err = s.secretManager.ListSecretsBySelectors(ctx, req.ClusterID, req.Namespace, labelSelector, "")
	} else {
		list, err = s.secretManager.ListSecrets(ctx, req.ClusterID, req.Namespace)
	}
	if err != nil {
		s.logger.Error("获取Secret列表失败", zap.Error(err),
			zap.Int("cluster_id", req.ClusterID), zap.String("namespace", req.Namespace))
		return model.ListResp[*model.K8sSecret]{}, fmt.Errorf("获取Secret列表失败: %w", err)
	}

	entities := make([]*model.K8sSecret, 0, len(list.Items))
	for _, item := range list.Items {
		entity := s.convertToK8sSecret(&item, req.ClusterID)
		// 类型过滤
		if req.Type != "" && string(entity.Type) != string(req.Type) {
			continue
		}
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

	s.logger.Info("成功获取Secret列表",
		zap.Int("cluster_id", req.ClusterID),
		zap.String("namespace", req.Namespace),
		zap.Int("count", len(entities)))

	return model.ListResp[*model.K8sSecret]{Items: entities[start:end], Total: total}, nil
}

// GetSecret 获取单个Secret详情
func (s *secretService) GetSecret(ctx context.Context, req *model.GetSecretDetailsReq) (*model.K8sSecret, error) {
	// 使用 SecretManager 获取 Secret
	secret, err := s.secretManager.GetSecret(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, fmt.Errorf("Secret不存在: %s/%s", req.Namespace, req.Name)
		}
		s.logger.Error("获取Secret失败", zap.Error(err),
			zap.Int("cluster_id", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return nil, fmt.Errorf("获取Secret失败: %w", err)
	}

	result := s.convertToK8sSecret(secret, req.ClusterID)

	s.logger.Info("成功获取Secret详情",
		zap.Int("cluster_id", req.ClusterID),
		zap.String("namespace", req.Namespace),
		zap.String("name", req.Name))

	return result, nil
}

// convertToK8sSecret 将Kubernetes Secret转换为模型对象
func (s *secretService) convertToK8sSecret(secret *corev1.Secret, clusterID int) *model.K8sSecret {
	// 计算数据大小
	var totalSize int64
	for _, v := range secret.Data {
		totalSize += int64(len(v))
	}
	for _, v := range secret.StringData {
		totalSize += int64(len(v))
	}

	// 格式化大小
	size := formatBytes(totalSize)

	// 计算数据条目数量
	dataCount := len(secret.Data) + len(secret.StringData)

	// 判断是否不可变
	immutable := false
	if secret.Immutable != nil {
		immutable = *secret.Immutable
	}

	// 构建StringData，如果存在的话
	stringData := make(map[string]string)
	for k, v := range secret.StringData {
		stringData[k] = v
	}

	return &model.K8sSecret{
		Name:        secret.Name,
		Namespace:   secret.Namespace,
		ClusterID:   clusterID,
		UID:         string(secret.UID),
		Type:        model.K8sSecretType(secret.Type),
		Data:        secret.Data,
		StringData:  stringData,
		Labels:      secret.Labels,
		Annotations: secret.Annotations,
		Immutable:   immutable,
		DataCount:   dataCount,
		Size:        size,
		Age:         time.Since(secret.CreationTimestamp.Time).String(),
		CreatedAt:   secret.CreationTimestamp.Time,
		UpdatedAt:   secret.CreationTimestamp.Time, // K8s doesn't track update time separately
		RawSecret:   secret,
	}
}

// CreateSecret 创建Secret
func (s *secretService) CreateSecret(ctx context.Context, req *model.CreateSecretReq) error {
	// 构造Secret对象
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        req.Name,
			Namespace:   req.Namespace,
			Labels:      req.Labels,
			Annotations: req.Annotations,
		},
		Type:       corev1.SecretType(req.Type),
		Data:       req.Data,
		StringData: req.StringData,
	}

	// 如果没有指定类型，默认为 Opaque
	if secret.Type == "" {
		secret.Type = corev1.SecretTypeOpaque
	}

	// 设置不可变标志
	if req.Immutable {
		secret.Immutable = &req.Immutable
	}

	if err := k8sutils.ValidateSecretData(secret.Type, secret.Data, secret.StringData); err != nil {
		return err
	}
	_, err := s.secretManager.CreateSecret(ctx, req.ClusterID, secret)
	if err != nil {
		if errors.IsAlreadyExists(err) {
			return fmt.Errorf("Secret已存在: %s/%s", req.Namespace, req.Name)
		}
		s.logger.Error("创建Secret失败", zap.Error(err),
			zap.Int("cluster_id", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("创建Secret失败: %w", err)
	}
	s.logger.Info("成功创建Secret",
		zap.Int("cluster_id", req.ClusterID),
		zap.String("namespace", req.Namespace),
		zap.String("name", req.Name))
	return nil
}

// UpdateSecret 更新Secret
func (s *secretService) UpdateSecret(ctx context.Context, req *model.UpdateSecretReq) error {
	// 先获取现有的Secret
	existingSecret, err := s.secretManager.GetSecret(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		if errors.IsNotFound(err) {
			return fmt.Errorf("Secret不存在: %s/%s", req.Namespace, req.Name)
		}
		s.logger.Error("获取Secret失败", zap.Error(err),
			zap.Int("cluster_id", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("获取Secret失败: %w", err)
	}

	// 更新Secret数据
	if req.Data != nil {
		existingSecret.Data = req.Data
	}
	if req.StringData != nil {
		existingSecret.StringData = req.StringData
	}
	if req.Labels != nil {
		existingSecret.Labels = req.Labels
	}
	if req.Annotations != nil {
		existingSecret.Annotations = req.Annotations
	}

	// 使用 SecretManager 更新 Secret
	_, err = s.secretManager.UpdateSecret(ctx, req.ClusterID, existingSecret)
	if err != nil {
		s.logger.Error("更新Secret失败", zap.Error(err),
			zap.Int("cluster_id", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("更新Secret失败: %w", err)
	}

	s.logger.Info("成功更新Secret",
		zap.Int("cluster_id", req.ClusterID),
		zap.String("namespace", req.Namespace),
		zap.String("name", req.Name))

	return nil
}

// DeleteSecret 删除Secret
func (s *secretService) DeleteSecret(ctx context.Context, req *model.DeleteSecretReq) error {
	// 使用 SecretManager 删除 Secret
	err := s.secretManager.DeleteSecret(ctx, req.ClusterID, req.Namespace, req.Name, metav1.DeleteOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return fmt.Errorf("Secret不存在: %s/%s", req.Namespace, req.Name)
		}
		s.logger.Error("删除Secret失败", zap.Error(err),
			zap.Int("cluster_id", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("删除Secret失败: %w", err)
	}

	s.logger.Info("成功删除Secret",
		zap.Int("cluster_id", req.ClusterID),
		zap.String("namespace", req.Namespace),
		zap.String("name", req.Name))

	return nil
}

// GetSecretYAML 获取Secret的YAML配置
func (s *secretService) GetSecretYAML(ctx context.Context, req *model.GetSecretYamlReq) (string, error) {
	// 使用 SecretManager 获取 Secret
	secret, err := s.secretManager.GetSecret(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		if errors.IsNotFound(err) {
			return "", fmt.Errorf("Secret不存在: %s/%s", req.Namespace, req.Name)
		}
		s.logger.Error("获取Secret失败", zap.Error(err),
			zap.Int("cluster_id", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return "", fmt.Errorf("获取Secret失败: %w", err)
	}

	yamlStr, err := k8sutils.SecretToYAML(secret)
	if err != nil {
		s.logger.Error("转换Secret为YAML失败", zap.Error(err))
		return "", fmt.Errorf("转换Secret为YAML失败: %w", err)
	}

	s.logger.Info("成功获取Secret YAML",
		zap.Int("cluster_id", req.ClusterID),
		zap.String("namespace", req.Namespace),
		zap.String("name", req.Name))

	return yamlStr, nil
}

// CreateSecretByYaml 通过YAML创建Secret
func (s *secretService) CreateSecretByYaml(ctx context.Context, req *model.CreateSecretByYamlReq) error {
	sec, err := k8sutils.YAMLToSecret(req.YAML)
	if err != nil {
		return err
	}

	_, err = s.secretManager.CreateSecret(ctx, req.ClusterID, sec)
	if err != nil {
		s.logger.Error("通过YAML创建Secret失败", zap.Error(err),
			zap.Int("cluster_id", req.ClusterID), zap.String("namespace", sec.Namespace), zap.String("name", sec.Name))
		return fmt.Errorf("通过YAML创建Secret失败: %w", err)
	}

	return nil
}

// UpdateSecretByYaml 通过YAML更新Secret
func (s *secretService) UpdateSecretByYaml(ctx context.Context, req *model.UpdateSecretByYamlReq) error {
	sec, err := k8sutils.YAMLToSecret(req.YAML)
	if err != nil {
		return err
	}
	sec.Namespace = req.Namespace
	sec.Name = req.Name
	_, err = s.secretManager.UpdateSecret(ctx, req.ClusterID, sec)
	if err != nil {
		s.logger.Error("通过YAML更新Secret失败", zap.Error(err),
			zap.Int("cluster_id", req.ClusterID), zap.String("namespace", req.Namespace), zap.String("name", req.Name))
		return fmt.Errorf("通过YAML更新Secret失败: %w", err)
	}
	return nil
}
