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
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

func BuildRoleBindingListOptions(req *model.GetRoleBindingListReq) metav1.ListOptions {
	options := metav1.ListOptions{}

	return options
}

func ConvertToK8sRoleBinding(req *model.CreateRoleBindingReq) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:        req.Name,
			Namespace:   req.Namespace,
			Labels:      req.Labels,
			Annotations: req.Annotations,
		},
		RoleRef:  ConvertRoleRefToK8s(req.RoleRef),
		Subjects: ConvertSubjectsToK8s(req.Subjects),
	}
}

// PaginateK8sRoleBindings 对RoleBinding列表进行分页（基于内部模型）
func PaginateK8sRoleBindings(roleBindings []*model.K8sRoleBinding, page, pageSize int) (model.ListResp[*model.K8sRoleBinding], error) {
	resp := model.ListResp[*model.K8sRoleBinding]{
		Items: []*model.K8sRoleBinding{},
		Total: int64(len(roleBindings)),
	}
	if len(roleBindings) == 0 {
		return resp, nil
	}

	if page <= 0 || pageSize <= 0 {
		resp.Items = roleBindings
		return resp, nil
	}

	start := (page - 1) * pageSize
	end := start + pageSize
	if start >= len(roleBindings) {
		return resp, nil
	}
	if end > len(roleBindings) {
		end = len(roleBindings)
	}
	resp.Items = roleBindings[start:end]
	return resp, nil
}

func BuildK8sRoleBinding(name, namespace string, labels, annotations model.KeyValueList, roleRef model.RoleRef, subjects []model.Subject) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Labels:      ConvertKeyValueListToLabels(labels),
			Annotations: ConvertKeyValueListToLabels(annotations),
		},
		RoleRef:  ConvertRoleRefToK8s(roleRef),
		Subjects: ConvertSubjectsToK8s(subjects),
	}
}

func ConvertRoleRefToK8s(roleRef model.RoleRef) rbacv1.RoleRef {
	return rbacv1.RoleRef{
		APIGroup: roleRef.APIGroup,
		Kind:     roleRef.Kind,
		Name:     roleRef.Name,
	}
}

func ConvertSubjectsToK8s(subjects []model.Subject) []rbacv1.Subject {
	if len(subjects) == 0 {
		return nil
	}

	k8sSubjects := make([]rbacv1.Subject, 0, len(subjects))
	for _, subject := range subjects {
		k8sSubjects = append(k8sSubjects, rbacv1.Subject{
			Kind:      subject.Kind,
			APIGroup:  subject.APIGroup,
			Name:      subject.Name,
			Namespace: subject.Namespace,
		})
	}

	return k8sSubjects
}

// RoleBindingToYAML 将RoleBinding转换为YAML
func RoleBindingToYAML(roleBinding *rbacv1.RoleBinding) (string, error) {
	if roleBinding == nil {
		return "", fmt.Errorf("RoleBinding不能为空")
	}

	data, err := yaml.Marshal(roleBinding)
	if err != nil {
		return "", fmt.Errorf("转换为YAML失败: %w", err)
	}

	return string(data), nil
}

// YAMLToRoleBinding 将YAML转换为RoleBinding
func YAMLToRoleBinding(yamlStr string) (*rbacv1.RoleBinding, error) {
	if yamlStr == "" {
		return nil, fmt.Errorf("YAML字符串不能为空")
	}

	var roleBinding rbacv1.RoleBinding
	err := yaml.Unmarshal([]byte(yamlStr), &roleBinding)
	if err != nil {
		return nil, fmt.Errorf("解析YAML失败: %w", err)
	}

	return &roleBinding, nil
}

func ConvertK8sRoleBindingToRoleBindingInfo(roleBinding *rbacv1.RoleBinding, clusterID int) *model.K8sRoleBinding {
	if roleBinding == nil {
		return nil
	}

	return &model.K8sRoleBinding{
		Name:            roleBinding.Name,
		Namespace:       roleBinding.Namespace,
		ClusterID:       clusterID,
		UID:             string(roleBinding.UID),
		CreatedAt:       roleBinding.CreationTimestamp.Time.Format(time.RFC3339),
		Labels:          roleBinding.Labels,
		Annotations:     roleBinding.Annotations,
		RoleRef:         ConvertK8sRoleRefToModel(roleBinding.RoleRef),
		Subjects:        ConvertK8sSubjectsToModel(roleBinding.Subjects),
		ResourceVersion: roleBinding.ResourceVersion,
		Age:             CalculateAge(roleBinding.CreationTimestamp.Time),
		RawRoleBinding:  roleBinding,
	}
}
