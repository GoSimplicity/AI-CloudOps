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

// import (
// 	"context"
// 	"fmt"
// 	"time"

// 	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
// 	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/manager"
// 	"github.com/GoSimplicity/AI-CloudOps/internal/model"
// 	"go.uber.org/zap"
// 	corev1 "k8s.io/api/core/v1"
// 	"k8s.io/apimachinery/pkg/api/errors"
// 	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
// 	"sigs.k8s.io/yaml"
// )

// type SecretService interface {
// 	GetSecretList(ctx context.Context, req *model.K8sListReq) ([]*model.K8sSecret, error)
// 	GetSecret(ctx context.Context, req *model.K8sResourceIdentifierReq) (*model.K8sSecret, error)
// 	CreateSecret(ctx context.Context, req *model.SecretCreateReq) error
// 	UpdateSecret(ctx context.Context, req *model.SecretUpdateReq) error
// 	// YAML相关方法
// 	CreateSecretByYaml(ctx context.Context, req *model.CreateResourceByYamlReq) error
// 	UpdateSecretByYaml(ctx context.Context, req *model.UpdateResourceByYamlReq) error
// 	DeleteSecret(ctx context.Context, req *model.K8sResourceIdentifierReq) error

// 	GetSecretYAML(ctx context.Context, req *model.K8sResourceIdentifierReq) (string, error)
// }

// type secretService struct {
// 	k8sClient     client.K8sClient      // 保持向后兼容
// 	secretManager manager.SecretManager // 新的依赖注入
// 	logger        *zap.Logger
// }

// func NewSecretService(k8sClient client.K8sClient, secretManager manager.SecretManager, logger *zap.Logger) SecretService {
// 	return &secretService{
// 		k8sClient:     k8sClient,
// 		secretManager: secretManager,
// 		logger:        logger,
// 	}
// }

// // GetSecretList 获取Secret列表
// func (s *secretService) GetSecretList(ctx context.Context, req *model.K8sListReq) ([]*model.K8sSecret, error) {
// 	// 使用 SecretManager 获取 Secret 列表
// 	secretList, err := s.secretManager.ListSecrets(ctx, req.ClusterID, req.Namespace)
// 	if err != nil {
// 		s.logger.Error("获取Secret列表失败", zap.Error(err),
// 			zap.Int("cluster_id", req.ClusterID), zap.String("namespace", req.Namespace))
// 		return nil, fmt.Errorf("获取Secret列表失败: %w", err)
// 	}

// 	result := make([]*model.K8sSecret, 0, len(secretList.Items))
// 	for _, secret := range secretList.Items {
// 		k8sSecret := s.convertToK8sSecret(&secret)
// 		result = append(result, k8sSecret)
// 	}

// 	s.logger.Info("成功获取Secret列表",
// 		zap.Int("cluster_id", req.ClusterID),
// 		zap.String("namespace", req.Namespace),
// 		zap.Int("count", len(result)))

// 	return result, nil
// }

// // GetSecret 获取单个Secret详情
// func (s *secretService) GetSecret(ctx context.Context, req *model.K8sResourceIdentifierReq) (*model.K8sSecret, error) {
// 	// 使用 SecretManager 获取 Secret
// 	secret, err := s.secretManager.GetSecret(ctx, req.ClusterID, req.Namespace, req.ResourceName)
// 	if err != nil {
// 		if errors.IsNotFound(err) {
// 			return nil, fmt.Errorf("Secret不存在: %s/%s", req.Namespace, req.ResourceName)
// 		}
// 		s.logger.Error("获取Secret失败", zap.Error(err),
// 			zap.Int("cluster_id", req.ClusterID),
// 			zap.String("namespace", req.Namespace),
// 			zap.String("name", req.ResourceName))
// 		return nil, fmt.Errorf("获取Secret失败: %w", err)
// 	}

// 	result := s.convertToK8sSecret(secret)

// 	s.logger.Info("成功获取Secret详情",
// 		zap.Int("cluster_id", req.ClusterID),
// 		zap.String("namespace", req.Namespace),
// 		zap.String("name", req.ResourceName))

// 	return result, nil
// }

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

// // convertToK8sSecret 将Kubernetes Secret转换为模型对象
// func (s *secretService) convertToK8sSecret(secret *corev1.Secret) *model.K8sSecret {
// 	// 对于安全考虑，不在响应中返回实际的数据内容，只返回键的信息
// 	stringData := make(map[string]string)
// 	for key := range secret.Data {
// 		stringData[key] = "*** HIDDEN ***"
// 	}

// 	return &model.K8sSecret{
// 		Name:              secret.Name,
// 		UID:               string(secret.UID),
// 		Namespace:         secret.Namespace,
// 		Type:              string(secret.Type),
// 		Data:              secret.Data, // 在实际应用中可能需要隐藏敏感数据
// 		StringData:        stringData,  // 显示键但隐藏值
// 		Labels:            secret.Labels,
// 		Annotations:       secret.Annotations,
// 		CreationTimestamp: secret.CreationTimestamp.Time,
// 		Age:               time.Since(secret.CreationTimestamp.Time).String(),
// 	}
// }
