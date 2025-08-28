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
	corev1 "k8s.io/api/core/v1"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
)

// BuildServiceAccountResponse 构建ServiceAccount响应结构
func BuildServiceAccountResponse(sa *corev1.ServiceAccount, clusterID int) *model.K8sServiceAccountResponse {
	if sa == nil {
		return nil
	}

	response := &model.K8sServiceAccountResponse{
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
		AutomountServiceAccountToken: sa.AutomountServiceAccountToken,
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
