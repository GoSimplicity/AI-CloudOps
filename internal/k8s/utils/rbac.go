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
	"time"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/util/duration"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
)

// ConvertK8sRoleToRoleInfo 将K8s Role转换为RoleInfo
func ConvertK8sRoleToRoleInfo(role *rbacv1.Role, clusterID int) model.RoleInfo {
	rules := make([]model.PolicyRule, 0, len(role.Rules))
	for _, rule := range role.Rules {
		rules = append(rules, model.PolicyRule{
			APIGroups:       rule.APIGroups,
			Resources:       rule.Resources,
			Verbs:           rule.Verbs,
			ResourceNames:   rule.ResourceNames,
			NonResourceURLs: rule.NonResourceURLs,
		})
	}

	return model.RoleInfo{
		Name:              role.Name,
		Namespace:         role.Namespace,
		ClusterID:         clusterID,
		UID:               string(role.UID),
		CreationTimestamp: role.CreationTimestamp.Format(time.RFC3339),
		Labels:            role.Labels,
		Annotations:       role.Annotations,
		Rules:             rules,
		ResourceVersion:   role.ResourceVersion,
		Age:               duration.HumanDuration(time.Since(role.CreationTimestamp.Time)),
	}
}

// ConvertK8sClusterRoleToClusterRoleInfo 将K8s ClusterRole转换为ClusterRoleInfo
func ConvertK8sClusterRoleToClusterRoleInfo(clusterRole *rbacv1.ClusterRole, clusterID int) model.ClusterRoleInfo {
	rules := make([]model.PolicyRule, 0, len(clusterRole.Rules))
	for _, rule := range clusterRole.Rules {
		rules = append(rules, model.PolicyRule{
			APIGroups:       rule.APIGroups,
			Resources:       rule.Resources,
			Verbs:           rule.Verbs,
			ResourceNames:   rule.ResourceNames,
			NonResourceURLs: rule.NonResourceURLs,
		})
	}

	return model.ClusterRoleInfo{
		Name:              clusterRole.Name,
		ClusterID:         clusterID,
		UID:               string(clusterRole.UID),
		CreationTimestamp: clusterRole.CreationTimestamp.Format(time.RFC3339),
		Labels:            clusterRole.Labels,
		Annotations:       clusterRole.Annotations,
		Rules:             rules,
		ResourceVersion:   clusterRole.ResourceVersion,
		Age:               duration.HumanDuration(time.Since(clusterRole.CreationTimestamp.Time)),
	}
}

// ConvertK8sRoleBindingToRoleBindingInfo 将K8s RoleBinding转换为RoleBindingInfo
func ConvertK8sRoleBindingToRoleBindingInfo(roleBinding *rbacv1.RoleBinding, clusterID int) model.RoleBindingInfo {
	subjects := make([]model.Subject, 0, len(roleBinding.Subjects))
	for _, subject := range roleBinding.Subjects {
		subjects = append(subjects, model.Subject{
			Kind:      subject.Kind,
			Name:      subject.Name,
			Namespace: subject.Namespace,
			APIGroup:  subject.APIGroup,
		})
	}

	return model.RoleBindingInfo{
		Name:              roleBinding.Name,
		Namespace:         roleBinding.Namespace,
		ClusterID:         clusterID,
		UID:               string(roleBinding.UID),
		CreationTimestamp: roleBinding.CreationTimestamp.Format(time.RFC3339),
		Labels:            roleBinding.Labels,
		Annotations:       roleBinding.Annotations,
		RoleRef: model.RoleRef{
			APIGroup: roleBinding.RoleRef.APIGroup,
			Kind:     roleBinding.RoleRef.Kind,
			Name:     roleBinding.RoleRef.Name,
		},
		Subjects:        subjects,
		ResourceVersion: roleBinding.ResourceVersion,
		Age:             duration.HumanDuration(time.Since(roleBinding.CreationTimestamp.Time)),
	}
}

// ConvertK8sClusterRoleBindingToClusterRoleBindingInfo 将K8s ClusterRoleBinding转换为ClusterRoleBindingInfo
func ConvertK8sClusterRoleBindingToClusterRoleBindingInfo(clusterRoleBinding *rbacv1.ClusterRoleBinding, clusterID int) model.ClusterRoleBindingInfo {
	subjects := make([]model.Subject, 0, len(clusterRoleBinding.Subjects))
	for _, subject := range clusterRoleBinding.Subjects {
		subjects = append(subjects, model.Subject{
			Kind:      subject.Kind,
			Name:      subject.Name,
			Namespace: subject.Namespace,
			APIGroup:  subject.APIGroup,
		})
	}

	return model.ClusterRoleBindingInfo{
		Name:              clusterRoleBinding.Name,
		ClusterID:         clusterID,
		UID:               string(clusterRoleBinding.UID),
		CreationTimestamp: clusterRoleBinding.CreationTimestamp.Format(time.RFC3339),
		Labels:            clusterRoleBinding.Labels,
		Annotations:       clusterRoleBinding.Annotations,
		RoleRef: model.RoleRef{
			APIGroup: clusterRoleBinding.RoleRef.APIGroup,
			Kind:     clusterRoleBinding.RoleRef.Kind,
			Name:     clusterRoleBinding.RoleRef.Name,
		},
		Subjects:        subjects,
		ResourceVersion: clusterRoleBinding.ResourceVersion,
		Age:             duration.HumanDuration(time.Since(clusterRoleBinding.CreationTimestamp.Time)),
	}
}

// ConvertPolicyRulesToK8s 将模型PolicyRule转换为K8s PolicyRule
func ConvertPolicyRulesToK8s(rules []model.PolicyRule) []rbacv1.PolicyRule {
	k8sRules := make([]rbacv1.PolicyRule, 0, len(rules))
	for _, rule := range rules {
		k8sRules = append(k8sRules, rbacv1.PolicyRule{
			APIGroups:       rule.APIGroups,
			Resources:       rule.Resources,
			Verbs:           rule.Verbs,
			ResourceNames:   rule.ResourceNames,
			NonResourceURLs: rule.NonResourceURLs,
		})
	}
	return k8sRules
}

// ConvertRoleRefToK8s 将模型RoleRef转换为K8s RoleRef
func ConvertRoleRefToK8s(roleRef model.RoleRef) rbacv1.RoleRef {
	return rbacv1.RoleRef{
		APIGroup: roleRef.APIGroup,
		Kind:     roleRef.Kind,
		Name:     roleRef.Name,
	}
}

// ConvertSubjectsToK8s 将模型Subjects转换为K8s Subjects
func ConvertSubjectsToK8s(subjects []model.Subject) []rbacv1.Subject {
	k8sSubjects := make([]rbacv1.Subject, 0, len(subjects))
	for _, subject := range subjects {
		k8sSubjects = append(k8sSubjects, rbacv1.Subject{
			Kind:      subject.Kind,
			Name:      subject.Name,
			Namespace: subject.Namespace,
			APIGroup:  subject.APIGroup,
		})
	}
	return k8sSubjects
}
