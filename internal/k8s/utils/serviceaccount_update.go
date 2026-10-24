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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/util/retry"
)

// ServiceAccountMutateFunc 把请求中允许修改的字段应用到从集群读取的最新 SA 上。
// 每次冲突重试都会以新读取的对象重新调用，实现必须是幂等的。
type ServiceAccountMutateFunc func(latest *corev1.ServiceAccount) error

// UpdateServiceAccountWithRetry 以 retry.RetryOnConflict(retry.DefaultRetry) 更新 SA：
// 每次尝试先 Get 最新版本（携带最新 resourceVersion），调用 mutate 应用允许修改的字段，再 Update。
// 无论 mutate 做了什么，name 与 secrets 引用都以集群中最新版本为准（SEC-29：secrets 由集群维护）。
// Get 失败（如 NotFound）直接返回且不调用 Update；连续冲突超过重试次数时返回最后一次的 Conflict 错误。
func UpdateServiceAccountWithRetry(ctx context.Context, saClient corev1client.ServiceAccountInterface, name string, mutate ServiceAccountMutateFunc) (*corev1.ServiceAccount, error) {
	var updated *corev1.ServiceAccount
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		latest, err := saClient.Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		keptName, keptNamespace := latest.Name, latest.Namespace
		keptSecrets := append([]corev1.ObjectReference(nil), latest.Secrets...)
		if mutate != nil {
			if err := mutate(latest); err != nil {
				return err
			}
		}
		latest.Name, latest.Namespace = keptName, keptNamespace
		latest.Secrets = keptSecrets
		updated, err = saClient.Update(ctx, latest, metav1.UpdateOptions{})
		return err
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

// UpdateServiceAccountWithResourceVersion 严格乐观锁更新：调用方提交了 resourceVersion（如 YAML 中的 metadata.resourceVersion）时使用。
// 先 Get 当前对象作为基础（保留不可编辑字段与集群维护的 secrets 引用），调用 mutate 应用允许修改的字段，
// 再以调用方提交的 resourceVersion 只做一次 Update：版本过期时由 API Server 返回 409 Conflict，原样返回，不重试。
// Get 失败（如 NotFound）直接返回且不调用 Update。
func UpdateServiceAccountWithResourceVersion(ctx context.Context, saClient corev1client.ServiceAccountInterface, name, resourceVersion string, mutate ServiceAccountMutateFunc) (*corev1.ServiceAccount, error) {
	latest, err := saClient.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	keptName, keptNamespace := latest.Name, latest.Namespace
	keptSecrets := append([]corev1.ObjectReference(nil), latest.Secrets...)
	if mutate != nil {
		if err := mutate(latest); err != nil {
			return nil, err
		}
	}
	latest.Name, latest.Namespace = keptName, keptNamespace
	latest.Secrets = keptSecrets
	latest.ResourceVersion = resourceVersion
	return saClient.Update(ctx, latest, metav1.UpdateOptions{})
}

// ApplyServiceAccountEditableFields 把 desired 中允许通过更新接口修改的字段整体覆盖到 latest：
// labels、annotations、automountServiceAccountToken、imagePullSecrets。
// 其余字段（name/namespace、secrets、resourceVersion、uid、ownerReferences、finalizers 等）保持 latest 的值。
func ApplyServiceAccountEditableFields(latest, desired *corev1.ServiceAccount) {
	if latest == nil || desired == nil {
		return
	}
	latest.Labels = desired.Labels
	latest.Annotations = desired.Annotations
	latest.AutomountServiceAccountToken = desired.AutomountServiceAccountToken
	latest.ImagePullSecrets = desired.ImagePullSecrets
}
