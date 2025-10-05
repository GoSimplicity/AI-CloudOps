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
	"fmt"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

func BuildClusterRoleBindingListOptions(req *model.GetClusterRoleBindingListReq) metav1.ListOptions {
	options := metav1.ListOptions{}

	return options
}

// ConvertToK8sClusterRoleBinding 将 K8s 对象构建为模型对象
func ConvertToK8sClusterRoleBinding(crb *rbacv1.ClusterRoleBinding) *model.K8sClusterRoleBinding {
	if crb == nil {
		return nil
	}
	return &model.K8sClusterRoleBinding{
		Name:                  crb.Name,
		UID:                   string(crb.UID),
		CreationTimestamp:     crb.CreationTimestamp.Time.Format("2006-01-02T15:04:05Z07:00"),
		Labels:                crb.Labels,
		Annotations:           crb.Annotations,
		RoleRef:               ConvertK8sRoleRefToModel(crb.RoleRef),
		Subjects:              ConvertK8sSubjectsToModel(crb.Subjects),
		ResourceVersion:       crb.ResourceVersion,
		Age:                   crb.CreationTimestamp.Time.Format("2006-01-02T15:04:05Z07:00"),
		RawClusterRoleBinding: crb,
	}
}

// PaginateK8sClusterRoleBindings 对ClusterRoleBinding列表进行分页
func PaginateK8sClusterRoleBindings(clusterRoleBindings []*model.K8sClusterRoleBinding, page, pageSize int) ([]*model.K8sClusterRoleBinding, int64) {
	total := int64(len(clusterRoleBindings))
	if total == 0 {
		return []*model.K8sClusterRoleBinding{}, 0
	}
	if page <= 0 || pageSize <= 0 {
		return clusterRoleBindings, total
	}
	start := int64((page - 1) * pageSize)
	end := start + int64(pageSize)
	if start >= total {
		return []*model.K8sClusterRoleBinding{}, total
	}
	if end > total {
		end = total
	}
	return clusterRoleBindings[start:end], total
}

func ConvertK8sClusterRoleBindingToClusterRoleBindingInfo(clusterRoleBinding *rbacv1.ClusterRoleBinding, clusterID int) model.K8sClusterRoleBinding {
	if clusterRoleBinding == nil {
		return model.K8sClusterRoleBinding{}
	}

	age := clusterRoleBinding.CreationTimestamp.Time.Format("2006-01-02T15:04:05Z07:00")

	return model.K8sClusterRoleBinding{
		Name:              clusterRoleBinding.Name,
		ClusterID:         clusterID,
		UID:               string(clusterRoleBinding.UID),
		CreationTimestamp: clusterRoleBinding.CreationTimestamp.Time.Format("2006-01-02T15:04:05Z07:00"),
		Labels:            clusterRoleBinding.Labels,
		Annotations:       clusterRoleBinding.Annotations,
		RoleRef:           ConvertK8sRoleRefToModel(clusterRoleBinding.RoleRef),
		Subjects:          ConvertK8sSubjectsToModel(clusterRoleBinding.Subjects),
		ResourceVersion:   clusterRoleBinding.ResourceVersion,
		Age:               age,
	}
}

func BuildK8sClusterRoleBinding(name string, labels, annotations model.KeyValueList, roleRef model.RoleRef, subjects []model.Subject) *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Labels:      ConvertKeyValueListToLabels(labels),
			Annotations: ConvertKeyValueListToLabels(annotations),
		},
		RoleRef:  ConvertRoleRefToK8s(roleRef),
		Subjects: ConvertSubjectsToK8s(subjects),
	}
}

func ConvertK8sRoleRefToModel(roleRef rbacv1.RoleRef) model.RoleRef {
	return model.RoleRef{
		APIGroup: roleRef.APIGroup,
		Kind:     roleRef.Kind,
		Name:     roleRef.Name,
	}
}

func ConvertK8sSubjectsToModel(subjects []rbacv1.Subject) []model.Subject {
	if len(subjects) == 0 {
		return nil
	}

	modelSubjects := make([]model.Subject, 0, len(subjects))
	for _, subject := range subjects {
		modelSubjects = append(modelSubjects, model.Subject{
			Kind:      subject.Kind,
			APIGroup:  subject.APIGroup,
			Name:      subject.Name,
			Namespace: subject.Namespace,
		})
	}

	return modelSubjects
}

// ClusterRoleBindingToYAML 将ClusterRoleBinding转换为YAML
func ClusterRoleBindingToYAML(clusterRoleBinding *rbacv1.ClusterRoleBinding) (string, error) {
	if clusterRoleBinding == nil {
		return "", fmt.Errorf("ClusterRoleBinding不能为空")
	}

	data, err := yaml.Marshal(clusterRoleBinding)
	if err != nil {
		return "", fmt.Errorf("转换为YAML失败: %w", err)
	}

	return string(data), nil
}

// YAMLToClusterRoleBinding 将YAML转换为ClusterRoleBinding
func YAMLToClusterRoleBinding(yamlStr string) (*rbacv1.ClusterRoleBinding, error) {
	if yamlStr == "" {
		return nil, fmt.Errorf("YAML字符串不能为空")
	}

	var clusterRoleBinding rbacv1.ClusterRoleBinding
	err := yaml.Unmarshal([]byte(yamlStr), &clusterRoleBinding)
	if err != nil {
		return nil, fmt.Errorf("解析YAML失败: %w", err)
	}

	return &clusterRoleBinding, nil
}
