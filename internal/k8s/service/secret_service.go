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
	"sigs.k8s.io/yaml"
)

// SecretService Secret服务接口
type SecretService interface {
	GetSecretList(ctx context.Context, req *model.GetSecretListReq) (model.ListResp[*model.K8sSecretEntity], error)
	GetSecret(ctx context.Context, req *model.K8sSecretDeleteReq) (*model.K8sSecretEntity, error)
	CreateSecret(ctx context.Context, req *model.K8sSecretCreateReq) error
	UpdateSecret(ctx context.Context, req *model.K8sSecretUpdateReq) error
	// YAML相关方法
	CreateSecretByYaml(ctx context.Context, req *model.CreateResourceByYamlReq) error
	UpdateSecretByYaml(ctx context.Context, req *model.UpdateResourceByYamlReq) error
	DeleteSecret(ctx context.Context, req *model.K8sSecretDeleteReq) error

	GetSecretYAML(ctx context.Context, req *model.K8sSecretDeleteReq) (*model.K8sYaml, error)
}

// secretService Secret服务实现
type secretService struct {
	k8sClient     client.K8sClient      // 保持向后兼容
	secretManager manager.SecretManager // 新的依赖注入
	logger        *zap.Logger
}

// NewSecretService 创建新的Secret服务实例
func NewSecretService(k8sClient client.K8sClient, secretManager manager.SecretManager, logger *zap.Logger) SecretService {
	return &secretService{
		k8sClient:     k8sClient,
		secretManager: secretManager,
		logger:        logger,
	}
}

// GetSecretList 获取Secret列表
func (s *secretService) GetSecretList(ctx context.Context, req *model.GetSecretListReq) (model.ListResp[*model.K8sSecretEntity], error) {
	var list *corev1.SecretList
	var err error
	if req.LabelSelector != "" || req.FieldSelector != "" {
		list, err = s.secretManager.ListSecretsBySelectors(ctx, req.ClusterID, req.Namespace, req.LabelSelector, req.FieldSelector)
	} else {
		list, err = s.secretManager.ListSecrets(ctx, req.ClusterID, req.Namespace)
	}
	if err != nil {
		s.logger.Error("获取Secret列表失败", zap.Error(err),
			zap.Int("cluster_id", req.ClusterID), zap.String("namespace", req.Namespace))
		return model.ListResp[*model.K8sSecretEntity]{}, fmt.Errorf("获取Secret列表失败: %w", err)
	}

	entities := make([]*model.K8sSecretEntity, 0, len(list.Items))
	for _, item := range list.Items {
		entity := s.convertToK8sSecretEntity(&item)
		// 名称过滤
		if req.Name != "" && entity.Name != "" && !strings.Contains(entity.Name, req.Name) {
			continue
		}
		// 类型过滤
		if req.Type != "" && entity.Type != req.Type {
			continue
		}
		// 数据键过滤
		if req.DataKey != "" {
			matched := false
			for k := range entity.Data {
				if strings.Contains(k, req.DataKey) {
					matched = true
					break
				}
			}
			if !matched {
				for k := range entity.StringData {
					if strings.Contains(k, req.DataKey) {
						matched = true
						break
					}
				}
			}
			if !matched {
				continue
			}
		}
		entities = append(entities, entity)
	}

	page := req.Page
	size := req.PageSize
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

	return model.ListResp[*model.K8sSecretEntity]{Items: entities[start:end], Total: total}, nil
}

// GetSecret 获取单个Secret详情
func (s *secretService) GetSecret(ctx context.Context, req *model.K8sSecretDeleteReq) (*model.K8sSecretEntity, error) {
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

	result := s.convertToK8sSecretEntity(secret)

	s.logger.Info("成功获取Secret详情",
		zap.Int("cluster_id", req.ClusterID),
		zap.String("namespace", req.Namespace),
		zap.String("name", req.Name))

	return result, nil
}

// // CreateSecret 创建Secret
// func (s *secretService) CreateSecret(ctx context.Context, req *model.SecretCreateReq) error {
// 	// 构造Secret对象
// 	secret := &corev1.Secret{
// 		ObjectMeta: metav1.ObjectMeta{
// 			Name:        req.Name,
// 			Namespace:   req.Namespace,
// 			Labels:      req.Labels,
// 			Annotations: req.Annotations,
// 		},
// 		Type:       corev1.SecretType(req.Type),
// 		Data:       req.Data,
// 		StringData: req.StringData,
// 	}

// 	// 如果没有指定类型，默认为Opaque
// 	if secret.Type == "" {
// 		secret.Type = corev1.SecretTypeOpaque
// 	}

// 	// 使用 SecretManager 创建 Secret
// 	_, err := s.secretManager.CreateSecret(ctx, req.ClusterID, secret)
// 	if err != nil {
// 		if errors.IsAlreadyExists(err) {
// 			return fmt.Errorf("Secret已存在: %s/%s", req.Namespace, req.Name)
// 		}
// 		s.logger.Error("创建Secret失败", zap.Error(err),
// 			zap.Int("cluster_id", req.ClusterID),
// 			zap.String("namespace", req.Namespace),
// 			zap.String("name", req.Name))
// 		return fmt.Errorf("创建Secret失败: %w", err)
// 	}

// 	s.logger.Info("成功创建Secret",
// 		zap.Int("cluster_id", req.ClusterID),
// 		zap.String("namespace", req.Namespace),
// 		zap.String("name", req.Name))

// 	return nil
// }

// // UpdateSecret 更新Secret
// func (s *secretService) UpdateSecret(ctx context.Context, req *model.SecretUpdateReq) error {
// 	// 先获取现有的Secret
// 	existingSecret, err := s.secretManager.GetSecret(ctx, req.ClusterID, req.Namespace, req.ResourceName)
// 	if err != nil {
// 		if errors.IsNotFound(err) {
// 			return fmt.Errorf("Secret不存在: %s/%s", req.Namespace, req.ResourceName)
// 		}
// 		s.logger.Error("获取Secret失败", zap.Error(err),
// 			zap.Int("cluster_id", req.ClusterID),
// 			zap.String("namespace", req.Namespace),
// 			zap.String("name", req.ResourceName))
// 		return fmt.Errorf("获取Secret失败: %w", err)
// 	}

// 	// 更新Secret数据
// 	existingSecret.Data = req.Data
// 	existingSecret.StringData = req.StringData
// 	if req.Labels != nil {
// 		existingSecret.Labels = req.Labels
// 	}
// 	if req.Annotations != nil {
// 		existingSecret.Annotations = req.Annotations
// 	}

// 	// 使用 SecretManager 更新 Secret
// 	_, err = s.secretManager.UpdateSecret(ctx, req.ClusterID, existingSecret)
// 	if err != nil {
// 		s.logger.Error("更新Secret失败", zap.Error(err),
// 			zap.Int("cluster_id", req.ClusterID),
// 			zap.String("namespace", req.Namespace),
// 			zap.String("name", req.ResourceName))
// 		return fmt.Errorf("更新Secret失败: %w", err)
// 	}

// 	s.logger.Info("成功更新Secret",
// 		zap.Int("cluster_id", req.ClusterID),
// 		zap.String("namespace", req.Namespace),
// 		zap.String("name", req.ResourceName))

// 	return nil
// }

// // DeleteSecret 删除Secret
// func (s *secretService) DeleteSecret(ctx context.Context, req *model.K8sResourceIdentifierReq) error {
// 	// 使用 SecretManager 删除 Secret
// 	err := s.secretManager.DeleteSecret(ctx, req.ClusterID, req.Namespace, req.ResourceName, metav1.DeleteOptions{})
// 	if err != nil {
// 		if errors.IsNotFound(err) {
// 			return fmt.Errorf("Secret不存在: %s/%s", req.Namespace, req.ResourceName)
// 		}
// 		s.logger.Error("删除Secret失败", zap.Error(err),
// 			zap.Int("cluster_id", req.ClusterID),
// 			zap.String("namespace", req.Namespace),
// 			zap.String("name", req.ResourceName))
// 		return fmt.Errorf("删除Secret失败: %w", err)
// 	}

// 	s.logger.Info("成功删除Secret",
// 		zap.Int("cluster_id", req.ClusterID),
// 		zap.String("namespace", req.Namespace),
// 		zap.String("name", req.ResourceName))

// 	return nil
// }

// // GetSecretYAML 获取Secret的YAML配置
// func (s *secretService) GetSecretYAML(ctx context.Context, req *model.K8sResourceIdentifierReq) (string, error) {
// 	// 使用 SecretManager 获取 Secret
// 	secret, err := s.secretManager.GetSecret(ctx, req.ClusterID, req.Namespace, req.ResourceName)
// 	if err != nil {
// 		if errors.IsNotFound(err) {
// 			return "", fmt.Errorf("Secret不存在: %s/%s", req.Namespace, req.ResourceName)
// 		}
// 		s.logger.Error("获取Secret失败", zap.Error(err),
// 			zap.Int("cluster_id", req.ClusterID),
// 			zap.String("namespace", req.Namespace),
// 			zap.String("name", req.ResourceName))
// 		return "", fmt.Errorf("获取Secret失败: %w", err)
// 	}

// 	// 清除不需要的字段
// 	secret.ManagedFields = nil

// 	yamlData, err := yaml.Marshal(secret)
// 	if err != nil {
// 		s.logger.Error("转换Secret为YAML失败", zap.Error(err))
// 		return "", fmt.Errorf("转换Secret为YAML失败: %w", err)
// 	}

// 	s.logger.Info("成功获取Secret YAML",
// 		zap.Int("cluster_id", req.ClusterID),
// 		zap.String("namespace", req.Namespace),
// 		zap.String("name", req.ResourceName))

// 	return string(yamlData), nil
// }

// // CreateSecretByYaml 通过YAML创建Secret
// func (s *secretService) CreateSecretByYaml(ctx context.Context, req *model.CreateResourceByYamlReq) error {
// 	// TODO: 实现通过YAML创建Secret的逻辑
// 	return fmt.Errorf("CreateSecretByYaml方法暂未实现")
// }

// // UpdateSecretByYaml 通过YAML更新Secret
// func (s *secretService) UpdateSecretByYaml(ctx context.Context, req *model.UpdateResourceByYamlReq) error {
// 	// TODO: 实现通过YAML更新Secret的逻辑
// 	return fmt.Errorf("UpdateSecretByYaml方法暂未实现")
// }

// convertToK8sSecretEntity 将Kubernetes Secret转换为模型对象
func (s *secretService) convertToK8sSecretEntity(secret *corev1.Secret) *model.K8sSecretEntity {
	return &model.K8sSecretEntity{
		Name:              secret.Name,
		Namespace:         secret.Namespace,
		UID:               string(secret.UID),
		Type:              string(secret.Type),
		Data:              secret.Data,
		Labels:            secret.Labels,
		Annotations:       secret.Annotations,
		CreationTimestamp: secret.CreationTimestamp.Time,
		Age:               time.Since(secret.CreationTimestamp.Time).String(),
		DataCount:         len(secret.Data),
	}
}

// CreateSecret 创建Secret
func (s *secretService) CreateSecret(ctx context.Context, req *model.K8sSecretCreateReq) error {
	secret, err := k8sutils.BuildSecretFromRequest(req)
	if err != nil {
		s.logger.Error("构建Secret失败", zap.Error(err))
		return fmt.Errorf("构建Secret失败: %w", err)
	}
	if err := k8sutils.ValidateSecretData(secret.Type, secret.Data, secret.StringData); err != nil {
		return err
	}
	_, err = s.secretManager.CreateSecret(ctx, req.ClusterID, secret)
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
func (s *secretService) UpdateSecret(ctx context.Context, req *model.K8sSecretUpdateReq) error {
	// 获取现有 Secret
	existing, err := s.secretManager.GetSecret(ctx, req.ClusterID, req.Namespace, req.Name)
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

	updated, err := k8sutils.UpdateSecretFromRequest(existing, req)
	if err != nil {
		return err
	}
	if err := k8sutils.ValidateSecretData(updated.Type, updated.Data, updated.StringData); err != nil {
		return err
	}
	_, err = s.secretManager.UpdateSecret(ctx, req.ClusterID, updated)
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
func (s *secretService) DeleteSecret(ctx context.Context, req *model.K8sSecretDeleteReq) error {
	var options metav1.DeleteOptions
	if req.GracePeriodSeconds != nil {
		options.GracePeriodSeconds = req.GracePeriodSeconds
	} else if req.Force {
		var zero int64 = 0
		options.GracePeriodSeconds = &zero
	}
	if err := s.secretManager.DeleteSecret(ctx, req.ClusterID, req.Namespace, req.Name, options); err != nil {
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
func (s *secretService) GetSecretYAML(ctx context.Context, req *model.K8sSecretDeleteReq) (*model.K8sYaml, error) {
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
	cleaned := k8sutils.CleanSecretForYAML(secret)
	data, err := yaml.Marshal(cleaned)
	if err != nil {
		s.logger.Error("转换Secret为YAML失败", zap.Error(err))
		return nil, fmt.Errorf("转换Secret为YAML失败: %w", err)
	}
	return &model.K8sYaml{YAML: string(data)}, nil
}

// CreateSecretByYaml 通过YAML创建Secret
func (s *secretService) CreateSecretByYaml(ctx context.Context, req *model.CreateResourceByYamlReq) error {
	var sec corev1.Secret
	if err := yaml.Unmarshal([]byte(req.YAML), &sec); err != nil {
		return fmt.Errorf("解析YAML失败: %w", err)
	}
	if sec.Namespace == "" {
		sec.Namespace = req.Namespace
	}
	if _, err := s.secretManager.CreateSecret(ctx, req.ClusterID, &sec); err != nil {
		s.logger.Error("通过YAML创建Secret失败", zap.Error(err),
			zap.Int("cluster_id", req.ClusterID), zap.String("namespace", sec.Namespace), zap.String("name", sec.Name))
		return fmt.Errorf("通过YAML创建Secret失败: %w", err)
	}
	return nil
}

// UpdateSecretByYaml 通过YAML更新Secret
func (s *secretService) UpdateSecretByYaml(ctx context.Context, req *model.UpdateResourceByYamlReq) error {
	var sec corev1.Secret
	if err := yaml.Unmarshal([]byte(req.YAML), &sec); err != nil {
		return fmt.Errorf("解析YAML失败: %w", err)
	}
	sec.Namespace = req.Namespace
	sec.Name = req.Name
	if _, err := s.secretManager.UpdateSecret(ctx, req.ClusterID, &sec); err != nil {
		s.logger.Error("通过YAML更新Secret失败", zap.Error(err),
			zap.Int("cluster_id", req.ClusterID), zap.String("namespace", req.Namespace), zap.String("name", req.Name))
		return fmt.Errorf("通过YAML更新Secret失败: %w", err)
	}
	return nil
}
