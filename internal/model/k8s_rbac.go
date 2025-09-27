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

package model

// AnalyzeRBACPermissionsReq 分析RBAC权限请求
type AnalyzeRBACPermissionsReq struct {
	ClusterID int     `json:"cluster_id" binding:"required" comment:"集群ID"`
	Subject   Subject `json:"subject" binding:"required" comment:"主体信息"`
}

// CheckRBACPermissionReq 检查RBAC权限请求
type CheckRBACPermissionReq struct {
	ClusterID int     `json:"cluster_id" binding:"required" comment:"集群ID"`
	Subject   Subject `json:"subject" binding:"required" comment:"主体信息"`
	Resource  string  `json:"resource" binding:"required" comment:"资源类型"`
	Verb      string  `json:"verb" binding:"required" comment:"动作"`
	Namespace string  `json:"namespace" comment:"命名空间"`
}

// EffectivePermissions 有效权限
type EffectivePermissions struct {
	Subject     Subject             `json:"subject"`
	ClusterID   int                 `json:"cluster_id"`
	Permissions map[string][]string `json:"permissions"` // resource -> verbs
	Sources     []PermissionSource  `json:"sources"`     // 权限来源
}

// PermissionSource 权限来源
type PermissionSource struct {
	Type        string `json:"type"` // Role, ClusterRole
	Name        string `json:"name"`
	Namespace   string `json:"namespace,omitempty"`
	BindingName string `json:"binding_name"`
}

// PermissionCheckResult 权限检查结果
type PermissionCheckResult struct {
	Allowed bool   `json:"allowed"`
	Source  string `json:"source,omitempty"`
	Reason  string `json:"reason,omitempty"`
}
