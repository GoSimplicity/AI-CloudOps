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

	"github.com/GoSimplicity/AI-CloudOps/pkg/base"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	authv1 "k8s.io/api/authentication/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
)

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
		CreatedAt:                    sa.CreationTimestamp.Time.Format(time.RFC3339),
		Age:                          base.GetAge(sa.CreationTimestamp.Time),
		AutomountServiceAccountToken: sa.AutomountServiceAccountToken,
		ResourceVersion:              sa.ResourceVersion,
		RawServiceAccount:            sa,
	}

	if len(sa.Secrets) > 0 {
		response.Secrets = make([]string, 0, len(sa.Secrets))
		for _, secret := range sa.Secrets {
			response.Secrets = append(response.Secrets, secret.Name)
		}
	}

	if len(sa.ImagePullSecrets) > 0 {
		response.ImagePullSecrets = make([]string, 0, len(sa.ImagePullSecrets))
		for _, secret := range sa.ImagePullSecrets {
			response.ImagePullSecrets = append(response.ImagePullSecrets, secret.Name)
		}
	}

	return response
}

func BuildServiceAccountListOptions(req *model.GetServiceAccountListReq) metav1.ListOptions {
	options := metav1.ListOptions{}

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

func ConvertToK8sServiceAccount(req *model.CreateServiceAccountReq) *corev1.ServiceAccount {
	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:        req.Name,
			Namespace:   req.Namespace,
			Labels:      req.Labels,
			Annotations: req.Annotations,
		},
	}

	if req.AutomountServiceAccountToken != nil {
		sa.AutomountServiceAccountToken = req.AutomountServiceAccountToken
	}

	if len(req.ImagePullSecrets) > 0 {
		sa.ImagePullSecrets = make([]corev1.LocalObjectReference, 0, len(req.ImagePullSecrets))
		for _, n := range req.ImagePullSecrets {
			sa.ImagePullSecrets = append(sa.ImagePullSecrets, corev1.LocalObjectReference{Name: n})
		}
	}

	if len(req.Secrets) > 0 {
		sa.Secrets = make([]corev1.ObjectReference, 0, len(req.Secrets))
		for _, n := range req.Secrets {
			sa.Secrets = append(sa.Secrets, corev1.ObjectReference{Name: n, Namespace: req.Namespace})
		}
	}

	return sa
}

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

func GetServiceAccountToken(ctx context.Context, kubeClient *kubernetes.Clientset, namespace, name string) (*model.ServiceAccountTokenInfo, error) {
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

	resp := &model.ServiceAccountTokenInfo{
		Token:             tr.Status.Token,
		ExpirationSeconds: tr.Spec.ExpirationSeconds,
		CreatedAt:         time.Now().Format(time.RFC3339),
		ExpirationTime:    "",
	}

	if expPtr != nil {
		resp.ExpirationTime = expPtr.Format(time.RFC3339)
	}

	return resp, nil
}

func CreateServiceAccountToken(ctx context.Context, kubeClient *kubernetes.Clientset, namespace, name string, expiryTime *int64) (*model.ServiceAccountTokenInfo, error) {
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

	resp := &model.ServiceAccountTokenInfo{
		Token:             tr.Status.Token,
		ExpirationSeconds: tr.Spec.ExpirationSeconds,
		CreatedAt:         time.Now().Format(time.RFC3339),
		ExpirationTime:    "",
	}

	if expPtr != nil {
		resp.ExpirationTime = expPtr.Format(time.RFC3339)
	}

	return resp, nil
}

// IsServiceAccountTokenSecretFor 判断 secret 是否是 sa 自己的 token Secret（SEC-29）：
// 类型必须是 kubernetes.io/service-account-token，注解 kubernetes.io/service-account.name 必须等于 SA 名称，
// 若存在注解 kubernetes.io/service-account.uid 则必须等于 SA 的 UID，且命名空间一致、token 非空。
// sa.secrets 只是引用列表，任何能更新 SA 的人都能把别的 SA 的 token Secret 挂进来，因此不能仅凭引用信任。
func IsServiceAccountTokenSecretFor(secret *corev1.Secret, sa *corev1.ServiceAccount) bool {
	if secret == nil || sa == nil || sa.Name == "" {
		return false
	}
	if secret.Type != corev1.SecretTypeServiceAccountToken {
		return false
	}
	if secret.Namespace != "" && sa.Namespace != "" && secret.Namespace != sa.Namespace {
		return false
	}
	if secret.Annotations[corev1.ServiceAccountNameKey] != sa.Name {
		return false
	}
	if uid, ok := secret.Annotations[corev1.ServiceAccountUIDKey]; ok && uid != string(sa.UID) {
		return false
	}
	return len(secret.Data[corev1.ServiceAccountTokenKey]) > 0
}

// SelectServiceAccountTokenSecret 从 SA 引用的 Secrets 中挑出第一个属于该 SA 的 token Secret；
// rejected 返回被拒绝的 service-account-token 类型 Secret 名称（只含名称，便于记录日志）。
func SelectServiceAccountTokenSecret(sa *corev1.ServiceAccount, secrets []corev1.Secret) (selected *corev1.Secret, rejected []string) {
	for i := range secrets {
		secret := &secrets[i]
		if IsServiceAccountTokenSecretFor(secret, sa) {
			if selected == nil {
				selected = secret
			}
			continue
		}
		if secret.Type == corev1.SecretTypeServiceAccountToken {
			rejected = append(rejected, secret.Name)
		}
	}
	return selected, rejected
}

// ServiceAccountSecretRefNames 返回 SA secrets 引用中的名称列表
func ServiceAccountSecretRefNames(refs []corev1.ObjectReference) []string {
	names := make([]string, 0, len(refs))
	for _, ref := range refs {
		names = append(names, ref.Name)
	}
	return names
}
