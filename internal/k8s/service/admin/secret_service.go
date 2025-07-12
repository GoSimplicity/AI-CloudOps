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

package admin

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"sync"
	
	pkg "github.com/GoSimplicity/AI-CloudOps/pkg/utils"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao/admin"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type SecretService interface {
	GetSecretsByNamespace(ctx context.Context, id int, namespace string) ([]*corev1.Secret, error)
	CreateSecret(ctx context.Context, req *model.K8sSecretRequest) error
	CreateEncryptedSecret(ctx context.Context, req *model.K8sSecretEncryptionRequest) error
	UpdateSecret(ctx context.Context, req *model.K8sSecretRequest) error
	DeleteSecret(ctx context.Context, id int, namespace, secretName string) error
	BatchDeleteSecret(ctx context.Context, id int, namespace string, secretNames []string) error
	GetSecretYaml(ctx context.Context, id int, namespace, secretName string) (string, error)
	GetSecretStatus(ctx context.Context, id int, namespace, secretName string) (*model.K8sSecretStatus, error)
	GetSupportedSecretTypes(ctx context.Context, id int) (map[string]interface{}, error)
	DecryptSecret(ctx context.Context, id int, namespace, secretName string) (map[string]interface{}, error)
}

type secretService struct {
	dao    admin.ClusterDAO
	client client.K8sClient
	logger *zap.Logger
	encryptionKey []byte
}

// NewSecretService 创建新的 SecretService 实例
func NewSecretService(dao admin.ClusterDAO, client client.K8sClient, logger *zap.Logger) SecretService {
	// 生成默认加密密钥（在生产环境中应从配置或密钥管理系统获取）
	key := make([]byte, 32) // AES-256
	if _, err := rand.Read(key); err != nil {
		logger.Error("生成加密密钥失败", zap.Error(err))
	}
	
	return &secretService{
		dao:           dao,
		client:        client,
		logger:        logger,
		encryptionKey: key,
	}
}

// GetSecretsByNamespace 获取指定命名空间下的所有 Secret
func (s *secretService) GetSecretsByNamespace(ctx context.Context, id int, namespace string) ([]*corev1.Secret, error) {
	kubeClient, err := pkg.GetKubeClient(id, s.client, s.logger)
	if err != nil {
		s.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", id))
		return nil, fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	secrets, err := kubeClient.CoreV1().Secrets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		s.logger.Error("获取 Secret 列表失败", zap.Error(err), zap.Int("cluster_id", id), zap.String("namespace", namespace))
		return nil, fmt.Errorf("failed to get Secret list: %w", err)
	}

	result := make([]*corev1.Secret, len(secrets.Items))
	for i := range secrets.Items {
		// 为了安全，移除敏感数据
		secret := secrets.Items[i].DeepCopy()
		secret.Data = nil
		secret.StringData = nil
		result[i] = secret
	}

	s.logger.Info("成功获取 Secret 列表", zap.Int("cluster_id", id), zap.String("namespace", namespace), zap.Int("count", len(result)))
	return result, nil
}

// CreateSecret 创建 Secret
func (s *secretService) CreateSecret(ctx context.Context, req *model.K8sSecretRequest) error {
	kubeClient, err := pkg.GetKubeClient(req.ClusterID, s.client, s.logger)
	if err != nil {
		s.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", req.ClusterID))
		return fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	_, err = kubeClient.CoreV1().Secrets(req.Namespace).Create(ctx, req.SecretYaml, metav1.CreateOptions{})
	if err != nil {
		s.logger.Error("创建 Secret 失败", zap.Error(err), zap.String("secret_name", req.SecretYaml.Name), zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID))
		return fmt.Errorf("failed to create Secret: %w", err)
	}

	s.logger.Info("成功创建 Secret", zap.String("secret_name", req.SecretYaml.Name), zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID))
	return nil
}

// CreateEncryptedSecret 创建加密的 Secret
func (s *secretService) CreateEncryptedSecret(ctx context.Context, req *model.K8sSecretEncryptionRequest) error {
	kubeClient, err := pkg.GetKubeClient(req.ClusterID, s.client, s.logger)
	if err != nil {
		s.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", req.ClusterID))
		return fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	// 创建 Secret 对象
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.Name,
			Namespace: req.Namespace,
		},
		Type: req.Type,
		Data: make(map[string][]byte),
	}

	// 加密数据
	for key, value := range req.Data {
		encryptedValue, err := s.encryptData(value)
		if err != nil {
			s.logger.Error("加密数据失败", zap.Error(err), zap.String("key", key))
			return fmt.Errorf("failed to encrypt data for key %s: %w", key, err)
		}
		secret.Data[key] = encryptedValue
	}

	// 处理字符串数据
	for key, value := range req.StringData {
		encryptedValue, err := s.encryptData(value)
		if err != nil {
			s.logger.Error("加密字符串数据失败", zap.Error(err), zap.String("key", key))
			return fmt.Errorf("failed to encrypt string data for key %s: %w", key, err)
		}
		secret.Data[key] = encryptedValue
	}

	if req.Immutable != nil {
		secret.Immutable = req.Immutable
	}

	_, err = kubeClient.CoreV1().Secrets(req.Namespace).Create(ctx, secret, metav1.CreateOptions{})
	if err != nil {
		s.logger.Error("创建加密 Secret 失败", zap.Error(err), zap.String("secret_name", req.Name), zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID))
		return fmt.Errorf("failed to create encrypted Secret: %w", err)
	}

	s.logger.Info("成功创建加密 Secret", zap.String("secret_name", req.Name), zap.String("namespace", req.Namespace), zap.String("type", string(req.Type)), zap.Int("cluster_id", req.ClusterID))
	return nil
}

// UpdateSecret 更新 Secret
func (s *secretService) UpdateSecret(ctx context.Context, req *model.K8sSecretRequest) error {
	kubeClient, err := pkg.GetKubeClient(req.ClusterID, s.client, s.logger)
	if err != nil {
		s.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", req.ClusterID))
		return fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	existingSecret, err := kubeClient.CoreV1().Secrets(req.Namespace).Get(ctx, req.SecretYaml.Name, metav1.GetOptions{})
	if err != nil {
		s.logger.Error("获取现有 Secret 失败", zap.Error(err), zap.String("secret_name", req.SecretYaml.Name), zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID))
		return fmt.Errorf("failed to get existing Secret: %w", err)
	}

	// 更新 Secret 数据
	existingSecret.Data = req.SecretYaml.Data
	existingSecret.StringData = req.SecretYaml.StringData
	existingSecret.Type = req.SecretYaml.Type
	if req.SecretYaml.Immutable != nil {
		existingSecret.Immutable = req.SecretYaml.Immutable
	}

	_, err = kubeClient.CoreV1().Secrets(req.Namespace).Update(ctx, existingSecret, metav1.UpdateOptions{})
	if err != nil {
		s.logger.Error("更新 Secret 失败", zap.Error(err), zap.String("secret_name", req.SecretYaml.Name), zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID))
		return fmt.Errorf("failed to update Secret: %w", err)
	}

	s.logger.Info("成功更新 Secret", zap.String("secret_name", req.SecretYaml.Name), zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID))
	return nil
}

// GetSecretYaml 获取指定 Secret 的 YAML 定义
func (s *secretService) GetSecretYaml(ctx context.Context, id int, namespace, secretName string) (string, error) {
	kubeClient, err := pkg.GetKubeClient(id, s.client, s.logger)
	if err != nil {
		s.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", id))
		return "", fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	secret, err := kubeClient.CoreV1().Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		s.logger.Error("获取 Secret 失败", zap.Error(err), zap.String("secret_name", secretName), zap.String("namespace", namespace), zap.Int("cluster_id", id))
		return "", fmt.Errorf("failed to get Secret: %w", err)
	}

	// 为了安全，移除敏感数据
	secretCopy := secret.DeepCopy()
	secretCopy.Data = nil
	secretCopy.StringData = nil

	yamlData, err := yaml.Marshal(secretCopy)
	if err != nil {
		s.logger.Error("序列化 Secret YAML 失败", zap.Error(err), zap.String("secret_name", secretName))
		return "", fmt.Errorf("failed to serialize Secret YAML: %w", err)
	}

	s.logger.Info("成功获取 Secret YAML", zap.String("secret_name", secretName), zap.String("namespace", namespace), zap.Int("cluster_id", id))
	return string(yamlData), nil
}

// BatchDeleteSecret 批量删除 Secret
func (s *secretService) BatchDeleteSecret(ctx context.Context, id int, namespace string, secretNames []string) error {
	kubeClient, err := pkg.GetKubeClient(id, s.client, s.logger)
	if err != nil {
		s.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", id))
		return fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	var wg sync.WaitGroup
	errChan := make(chan error, len(secretNames))

	for _, name := range secretNames {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			if err := kubeClient.CoreV1().Secrets(namespace).Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
				s.logger.Error("删除 Secret 失败", zap.Error(err), zap.String("secret_name", name), zap.String("namespace", namespace), zap.Int("cluster_id", id))
				errChan <- fmt.Errorf("failed to delete Secret '%s': %w", name, err)
			}
		}(name)
	}

	wg.Wait()
	close(errChan)

	var errs []error
	for err := range errChan {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		s.logger.Error("批量删除 Secret 部分失败", zap.Int("failed_count", len(errs)), zap.Int("total_count", len(secretNames)), zap.String("namespace", namespace), zap.Int("cluster_id", id))
		return fmt.Errorf("errors occurred while deleting Secrets: %v", errs)
	}

	s.logger.Info("成功批量删除 Secret", zap.Int("count", len(secretNames)), zap.String("namespace", namespace), zap.Int("cluster_id", id))
	return nil
}

// DeleteSecret 删除指定的 Secret
func (s *secretService) DeleteSecret(ctx context.Context, id int, namespace, secretName string) error {
	kubeClient, err := pkg.GetKubeClient(id, s.client, s.logger)
	if err != nil {
		s.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", id))
		return fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	if err := kubeClient.CoreV1().Secrets(namespace).Delete(ctx, secretName, metav1.DeleteOptions{}); err != nil {
		s.logger.Error("删除 Secret 失败", zap.Error(err), zap.String("secret_name", secretName), zap.String("namespace", namespace), zap.Int("cluster_id", id))
		return fmt.Errorf("failed to delete Secret '%s': %w", secretName, err)
	}

	s.logger.Info("成功删除 Secret", zap.String("secret_name", secretName), zap.String("namespace", namespace), zap.Int("cluster_id", id))
	return nil
}

// GetSecretStatus 获取 Secret 状态
func (s *secretService) GetSecretStatus(ctx context.Context, id int, namespace, secretName string) (*model.K8sSecretStatus, error) {
	kubeClient, err := pkg.GetKubeClient(id, s.client, s.logger)
	if err != nil {
		s.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", id))
		return nil, fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	secret, err := kubeClient.CoreV1().Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		s.logger.Error("获取 Secret 失败", zap.Error(err), zap.String("secret_name", secretName), zap.String("namespace", namespace), zap.Int("cluster_id", id))
		return nil, fmt.Errorf("failed to get Secret: %w", err)
	}

	// 提取数据键（不包含敏感值）
	var dataKeys []string
	totalSize := 0
	for key, value := range secret.Data {
		dataKeys = append(dataKeys, key)
		totalSize += len(value)
	}

	status := &model.K8sSecretStatus{
		Name:              secret.Name,
		Namespace:         secret.Namespace,
		Type:              secret.Type,
		DataKeys:          dataKeys,
		DataSize:          totalSize,
		Immutable:         secret.Immutable,
		CreationTimestamp: secret.CreationTimestamp.Time,
	}

	s.logger.Info("成功获取 Secret 状态", zap.String("secret_name", secretName), zap.String("namespace", namespace), zap.String("type", string(secret.Type)), zap.Int("data_keys_count", len(dataKeys)), zap.Int("cluster_id", id))
	return status, nil
}

// GetSupportedSecretTypes 获取支持的 Secret 类型
func (s *secretService) GetSupportedSecretTypes(ctx context.Context, id int) (map[string]interface{}, error) {
	types := map[string]interface{}{
		"supported_types": []map[string]interface{}{
			{
				"type":        "Opaque",
				"description": "不透明数据，可以包含任意用户定义的数据",
				"examples":    []string{"api-key", "password", "config-file"},
			},
			{
				"type":        "kubernetes.io/service-account-token",
				"description": "服务账户令牌",
				"examples":    []string{"token", "ca.crt", "namespace"},
			},
			{
				"type":        "kubernetes.io/dockercfg",
				"description": "~/.dockercfg 文件的序列化形式",
				"examples":    []string{".dockercfg"},
			},
			{
				"type":        "kubernetes.io/dockerconfigjson",
				"description": "~/.docker/config.json 文件的序列化形式",
				"examples":    []string{".dockerconfigjson"},
			},
			{
				"type":        "kubernetes.io/basic-auth",
				"description": "基本身份验证的凭据",
				"examples":    []string{"username", "password"},
			},
			{
				"type":        "kubernetes.io/ssh-auth",
				"description": "SSH 身份验证的凭据",
				"examples":    []string{"ssh-privatekey"},
			},
			{
				"type":        "kubernetes.io/tls",
				"description": "TLS 客户端或服务器的数据",
				"examples":    []string{"tls.crt", "tls.key"},
			},
		},
		"default_type": "Opaque",
		"notes": []string{
			"Opaque 类型是默认的 Secret 类型，可以存储任意用户定义的数据",
			"其他类型的 Secret 由 Kubernetes 强制要求特定的键名",
			"选择正确的类型有助于 Kubernetes 验证数据格式",
		},
	}

	s.logger.Info("成功获取支持的 Secret 类型", zap.Int("cluster_id", id))
	return types, nil
}

// DecryptSecret 解密 Secret 数据 (仅用于演示，实际生产中需要权限控制)
func (s *secretService) DecryptSecret(ctx context.Context, id int, namespace, secretName string) (map[string]interface{}, error) {
	kubeClient, err := pkg.GetKubeClient(id, s.client, s.logger)
	if err != nil {
		s.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", id))
		return nil, fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	secret, err := kubeClient.CoreV1().Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		s.logger.Error("获取 Secret 失败", zap.Error(err), zap.String("secret_name", secretName), zap.String("namespace", namespace), zap.Int("cluster_id", id))
		return nil, fmt.Errorf("failed to get Secret: %w", err)
	}

	result := map[string]interface{}{
		"name":      secret.Name,
		"namespace": secret.Namespace,
		"type":      secret.Type,
		"data":      make(map[string]string),
		"warning":   "此操作仅用于调试，生产环境中应谨慎使用",
	}

	// 解码 base64 数据
	data := make(map[string]string)
	for key, value := range secret.Data {
		data[key] = string(value)
	}

	result["data"] = data

	s.logger.Info("成功解密 Secret 数据", zap.String("secret_name", secretName), zap.String("namespace", namespace), zap.Int("keys_count", len(secret.Data)), zap.Int("cluster_id", id))
	return result, nil
}

// encryptData 加密数据
func (s *secretService) encryptData(plaintext string) ([]byte, error) {
	block, err := aes.NewCipher(s.encryptionKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return []byte(base64.StdEncoding.EncodeToString(ciphertext)), nil
}

// decryptData 解密数据
func (s *secretService) decryptData(ciphertext []byte) (string, error) {
	data, err := base64.StdEncoding.DecodeString(string(ciphertext))
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(s.encryptionKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, cipherData := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, cipherData, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}