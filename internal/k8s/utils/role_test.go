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
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestConvertK8sRoleToRoleInfo(t *testing.T) {
	// 创建测试用的 K8s Role
	k8sRole := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-role",
			Namespace:         "default",
			UID:               "test-uid-123",
			CreationTimestamp: metav1.Time{Time: time.Now().Add(-time.Hour)},
			Labels: map[string]string{
				"test": "test",
			},
			Annotations: map[string]string{
				"est": "est",
			},
			ResourceVersion: "12345",
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"pods"},
				Verbs:     []string{"get"},
			},
		},
	}

	// 转换
	roleInfo := ConvertK8sRoleToRoleInfo(k8sRole, 1)

	// 验证基本字段
	if roleInfo.Name != "test-role" {
		t.Errorf("Expected name 'test-role', got '%s'", roleInfo.Name)
	}

	if roleInfo.Namespace != "default" {
		t.Errorf("Expected namespace 'default', got '%s'", roleInfo.Namespace)
	}

	if roleInfo.ClusterID != 1 {
		t.Errorf("Expected clusterID 1, got %d", roleInfo.ClusterID)
	}

	// 关键验证：Rules 不应该为空
	if len(roleInfo.Rules) == 0 {
		t.Fatal("Rules should not be empty!")
	}

	if len(roleInfo.Rules) != 1 {
		t.Fatalf("Expected 1 rule, got %d", len(roleInfo.Rules))
	}

	// 验证规则内容
	rule := roleInfo.Rules[0]
	if len(rule.APIGroups) != 1 || rule.APIGroups[0] != "" {
		t.Errorf("Expected APIGroups [''], got %v", rule.APIGroups)
	}

	if len(rule.Resources) != 1 || rule.Resources[0] != "pods" {
		t.Errorf("Expected Resources ['pods'], got %v", rule.Resources)
	}

	if len(rule.Verbs) != 1 || rule.Verbs[0] != "get" {
		t.Errorf("Expected Verbs ['get'], got %v", rule.Verbs)
	}

	t.Logf("✅ Test passed! Rules correctly converted: %+v", roleInfo.Rules)
}

func TestConvertPolicyRulesToK8s(t *testing.T) {
	// 测试带有空字符串的规则
	rules := []model.PolicyRule{
		{
			Verbs:     []string{"get"},
			APIGroups: []string{"", "test"}, // 包含空字符串
			Resources: []string{"", "pods"}, // 包含空字符串
		},
	}

	k8sRules := ConvertPolicyRulesToK8s(rules)

	if len(k8sRules) == 0 {
		t.Fatal("ConvertPolicyRulesToK8s should not return empty rules!")
	}

	if len(k8sRules) != 1 {
		t.Fatalf("Expected 1 rule, got %d", len(k8sRules))
	}

	rule := k8sRules[0]

	// 验证空字符串被过滤
	if len(rule.Resources) == 0 {
		t.Fatal("Resources should not be empty after filtering!")
	}

	// Resources 应该只包含 "pods"，空字符串应该被过滤
	foundPods := false
	for _, r := range rule.Resources {
		if r == "pods" {
			foundPods = true
		}
		if r == "" {
			t.Error("Empty string should be filtered out from resources")
		}
	}

	if !foundPods {
		t.Error("Resources should contain 'pods'")
	}

	t.Logf("✅ Test passed! Converted rules: %+v", k8sRules)
}

func TestConvertK8sPolicyRulesToModel(t *testing.T) {
	// 创建 K8s PolicyRule
	k8sRules := []rbacv1.PolicyRule{
		{
			APIGroups:       []string{""},
			Resources:       []string{"pods"},
			Verbs:           []string{"get", "list"},
			NonResourceURLs: []string{"/healthz"},
		},
	}

	// 转换为 Model
	modelRules := ConvertK8sPolicyRulesToModel(k8sRules)

	if len(modelRules) == 0 {
		t.Fatal("Converted model rules should not be empty!")
	}

	if len(modelRules) != 1 {
		t.Fatalf("Expected 1 rule, got %d", len(modelRules))
	}

	rule := modelRules[0]

	// 验证所有字段
	if len(rule.APIGroups) != 1 || rule.APIGroups[0] != "" {
		t.Errorf("APIGroups not correct: %v", rule.APIGroups)
	}

	if len(rule.Resources) != 1 || rule.Resources[0] != "pods" {
		t.Errorf("Resources not correct: %v", rule.Resources)
	}

	if len(rule.Verbs) != 2 {
		t.Errorf("Expected 2 verbs, got %d", len(rule.Verbs))
	}

	if len(rule.NonResourceURLs) != 1 || rule.NonResourceURLs[0] != "/healthz" {
		t.Errorf("NonResourceURLs not correct: %v", rule.NonResourceURLs)
	}

	t.Logf("✅ Test passed! Model rules: %+v", modelRules)
}
