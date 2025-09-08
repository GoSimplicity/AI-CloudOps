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
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	authv1 "k8s.io/api/authentication/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
)

// BuildServiceAccountResponse 构建ServiceAccount响应结构
func BuildServiceAccountResponse(sa *corev1.ServiceAccount, clusterID int) *model.K8sServiceAccount {
	if sa == nil {
		return nil
	}

	response := &model.K8sServiceAccount{
		Name:                         sa.Name,
		UID:                          string(sa.UID),
		Namespace:                    sa.Namespace,
		ClusterID:                    clusterID,
		Labels:                       sa.Labels,
		Annotations:                  sa.Annotations,
		CreationTimestamp:            sa.CreationTimestamp.Time,
		Age:                          utils.GetAge(sa.CreationTimestamp.Time),
		SecretsCount:                 len(sa.Secrets),
		ImagePullSecretsCount:        len(sa.ImagePullSecrets),
		AutomountServiceAccountToken: model.PtrBoolToPtrBoolValue(sa.AutomountServiceAccountToken),
	}

	// 构建Secrets列表
	if len(sa.Secrets) > 0 {
		response.Secrets = make([]model.ServiceAccountSecret, 0, len(sa.Secrets))
		for _, secret := range sa.Secrets {
			response.Secrets = append(response.Secrets, model.ServiceAccountSecret{
				Name:      secret.Name,
				Namespace: sa.Namespace,
				Type:      "kubernetes.io/service-account-token", // 默认类型
			})
		}
	}

	// 构建ImagePullSecrets列表
	if len(sa.ImagePullSecrets) > 0 {
		response.ImagePullSecrets = make([]model.ServiceAccountSecret, 0, len(sa.ImagePullSecrets))
		for _, secret := range sa.ImagePullSecrets {
			response.ImagePullSecrets = append(response.ImagePullSecrets, model.ServiceAccountSecret{
				Name:      secret.Name,
				Namespace: sa.Namespace,
				Type:      "kubernetes.io/dockercfg", // 默认类型
			})
		}
	}

	return response
}

// BuildServiceAccountListOptions 构建ServiceAccount列表选项
func BuildServiceAccountListOptions(req *model.GetServiceAccountListReq) metav1.ListOptions {
	options := metav1.ListOptions{}

	// 构建选项的逻辑可以在这里添加

	return options
}

// PaginateK8sServiceAccounts 对ServiceAccount列表进行分页
func PaginateK8sServiceAccounts(serviceAccounts []corev1.ServiceAccount, page, pageSize int) []corev1.ServiceAccount {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	start := (page - 1) * pageSize
	end := start + pageSize

	if start >= len(serviceAccounts) {
		return []corev1.ServiceAccount{}
	}

	if end > len(serviceAccounts) {
		end = len(serviceAccounts)
	}

	return serviceAccounts[start:end]
}

// ConvertToK8sServiceAccount 将内部模型转换为Kubernetes ServiceAccount对象
func ConvertToK8sServiceAccount(req *model.CreateServiceAccountReq) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:        req.Name,
			Namespace:   req.Namespace,
			Labels:      req.Labels,
			Annotations: req.Annotations,
		},
	}
}

// BuildK8sServiceAccount 构建K8s ServiceAccount对象
func BuildK8sServiceAccount(name, namespace string, labels, annotations model.KeyValueList) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Labels:      ConvertKeyValueListToLabels(labels),
			Annotations: ConvertKeyValueListToLabels(annotations),
		},
	}
}

// ServiceAccountToYAML 将ServiceAccount转换为YAML
func ServiceAccountToYAML(serviceAccount *corev1.ServiceAccount) (string, error) {
	if serviceAccount == nil {
		return "", fmt.Errorf("ServiceAccount不能为空")
	}

	data, err := yaml.Marshal(serviceAccount)
	if err != nil {
		return "", fmt.Errorf("转换为YAML失败: %w", err)
	}

	return string(data), nil
}

// YAMLToServiceAccount 将YAML转换为ServiceAccount
func YAMLToServiceAccount(yamlStr string) (*corev1.ServiceAccount, error) {
	if yamlStr == "" {
		return nil, fmt.Errorf("YAML字符串不能为空")
	}

	var serviceAccount corev1.ServiceAccount
	err := yaml.Unmarshal([]byte(yamlStr), &serviceAccount)
	if err != nil {
		return nil, fmt.Errorf("解析YAML失败: %w", err)
	}

	return &serviceAccount, nil
}

// GetServiceAccountToken 使用 TokenRequest API 获取 ServiceAccount 的短期令牌
func GetServiceAccountToken(ctx context.Context, kubeClient *kubernetes.Clientset, namespace, name string) (*model.K8sServiceAccountToken, error) {
	if kubeClient == nil {
		return nil, fmt.Errorf("kubeClient 不能为空")
	}

	tokenReq := &authv1.TokenRequest{ // 空的 Spec 使用集群默认过期时间
		Spec: authv1.TokenRequestSpec{},
	}

	tr, err := kubeClient.CoreV1().ServiceAccounts(namespace).CreateToken(ctx, name, tokenReq, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("创建 TokenRequest 失败: %w", err)
	}

	var expPtr *time.Time
	if !tr.Status.ExpirationTimestamp.IsZero() {
		t := tr.Status.ExpirationTimestamp.Time
		expPtr = &t
	}

	resp := &model.K8sServiceAccountToken{
		Token:               tr.Status.Token,
		ExpirationTimestamp: expPtr,
		Audience:            tr.Spec.Audiences,
		BoundObjectRef:      nil,
		CreatedAt:           time.Now(),
	}

	return resp, nil
}

// CreateServiceAccountToken 为 ServiceAccount 创建指定过期时间的令牌
func CreateServiceAccountToken(ctx context.Context, kubeClient *kubernetes.Clientset, namespace, name string, expiryTime *int64) (*model.K8sServiceAccountToken, error) {
	if kubeClient == nil {
		return nil, fmt.Errorf("kubeClient 不能为空")
	}

	tokenReq := &authv1.TokenRequest{Spec: authv1.TokenRequestSpec{}}
	if expiryTime != nil {
		tokenReq.Spec.ExpirationSeconds = expiryTime
	}

	tr, err := kubeClient.CoreV1().ServiceAccounts(namespace).CreateToken(ctx, name, tokenReq, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("创建 ServiceAccount Token 失败: %w", err)
	}

	var expPtr *time.Time
	if !tr.Status.ExpirationTimestamp.IsZero() {
		t := tr.Status.ExpirationTimestamp.Time
		expPtr = &t
	}

	resp := &model.K8sServiceAccountToken{
		Token:               tr.Status.Token,
		ExpirationTimestamp: expPtr,
		Audience:            tr.Spec.Audiences,
		BoundObjectRef:      nil,
		CreatedAt:           time.Now(),
	}

	return resp, nil
}
