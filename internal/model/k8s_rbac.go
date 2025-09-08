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

// ====================== Bool替代常量定义 ======================

// BoolValue Bool值的int8替代
type BoolValue int8

const (
	BoolTrue  BoolValue = 1 // true
	BoolFalse BoolValue = 2 // false
)

// BoolToBoolValue 将bool转换为BoolValue
func BoolToBoolValue(b bool) BoolValue {
	if b {
		return BoolTrue
	}
	return BoolFalse
}

// BoolValueToBool 将BoolValue转换为bool
func BoolValueToBool(bv BoolValue) bool {
	return bv == BoolTrue
}

// PtrBoolToPtrBoolValue 将*bool转换为*BoolValue
func PtrBoolToPtrBoolValue(b *bool) *BoolValue {
	if b == nil {
		return nil
	}
	bv := BoolToBoolValue(*b)
	return &bv
}

// PtrBoolValueToPtrBool 将*BoolValue转换为*bool
func PtrBoolValueToPtrBool(bv *BoolValue) *bool {
	if bv == nil {
		return nil
	}
	b := BoolValueToBool(*bv)
	return &b
}

// ====================== RBAC响应结构体 ======================

// RoleInfo Role信息
type RoleInfo struct {
	Name              string            `json:"name"`
	Namespace         string            `json:"namespace"`
	ClusterID         int               `json:"cluster_id"`
	UID               string            `json:"uid"`
	CreationTimestamp string            `json:"creation_timestamp"`
	Labels            map[string]string `json:"labels"`
	Annotations       map[string]string `json:"annotations"`
	Rules             []PolicyRule      `json:"rules"`
	ResourceVersion   string            `json:"resource_version"`
	Age               string            `json:"age"`
}

// ClusterRoleInfo ClusterRole信息
type ClusterRoleInfo struct {
	Name              string            `json:"name"`
	ClusterID         int               `json:"cluster_id"`
	UID               string            `json:"uid"`
	CreationTimestamp string            `json:"creation_timestamp"`
	Labels            map[string]string `json:"labels"`
	Annotations       map[string]string `json:"annotations"`
	Rules             []PolicyRule      `json:"rules"`
	ResourceVersion   string            `json:"resource_version"`
	Age               string            `json:"age"`
}

// PolicyRule 权限规则
type PolicyRule struct {
	APIGroups       []string `json:"api_groups"`
	Resources       []string `json:"resources"`
	Verbs           []string `json:"verbs"`
	ResourceNames   []string `json:"resource_names,omitempty"`
	NonResourceURLs []string `json:"non_resource_urls,omitempty"`
}

// RoleBindingInfo RoleBinding信息
type RoleBindingInfo struct {
	Name              string            `json:"name"`
	Namespace         string            `json:"namespace"`
	ClusterID         int               `json:"cluster_id"`
	UID               string            `json:"uid"`
	CreationTimestamp string            `json:"creation_timestamp"`
	Labels            map[string]string `json:"labels"`
	Annotations       map[string]string `json:"annotations"`
	RoleRef           RoleRef           `json:"role_ref"`
	Subjects          []Subject         `json:"subjects"`
	ResourceVersion   string            `json:"resource_version"`
	Age               string            `json:"age"`
}

// ClusterRoleBindingInfo ClusterRoleBinding信息
type ClusterRoleBindingInfo struct {
	Name              string            `json:"name"`
	ClusterID         int               `json:"cluster_id"`
	UID               string            `json:"uid"`
	CreationTimestamp string            `json:"creation_timestamp"`
	Labels            map[string]string `json:"labels"`
	Annotations       map[string]string `json:"annotations"`
	RoleRef           RoleRef           `json:"role_ref"`
	Subjects          []Subject         `json:"subjects"`
	ResourceVersion   string            `json:"resource_version"`
	Age               string            `json:"age"`
}

// RoleRef 角色引用
type RoleRef struct {
	APIGroup string `json:"api_group"`
	Kind     string `json:"kind"`
	Name     string `json:"name"`
}

// Subject 主体（用户、组或服务账户）
type Subject struct {
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Namespace string `json:"namespace,omitempty"`
	APIGroup  string `json:"api_group,omitempty"`
}

// RBACStatistics RBAC统计信息
type RBACStatistics struct {
	TotalRoles               int `json:"total_roles"`
	TotalClusterRoles        int `json:"total_cluster_roles"`
	TotalRoleBindings        int `json:"total_role_bindings"`
	TotalClusterRoleBindings int `json:"total_cluster_role_bindings"`
	ActiveSubjects           int `json:"active_subjects"`
	SystemRoles              int `json:"system_roles"`
	CustomRoles              int `json:"custom_roles"`
}

// ====================== Role请求结构体 ======================

// GetRoleListReq Role列表请求参数
type GetRoleListReq struct {
	ClusterID int    `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace string `json:"namespace" form:"namespace" comment:"命名空间"`
	Keyword   string `json:"keyword" form:"keyword" comment:"关键字搜索"`
	Page      int    `json:"page" form:"page" comment:"页码"`
	PageSize  int    `json:"page_size" form:"page_size" comment:"页面大小"`
}

// GetRoleDetailsReq 获取Role详情请求
type GetRoleDetailsReq struct {
	ClusterID int    `json:"cluster_id" comment:"集群ID"`
	Namespace string `json:"namespace" comment:"命名空间"`
	Name      string `json:"name" comment:"Role名称"`
}

// GetRoleYamlReq 获取Role YAML请求
type GetRoleYamlReq struct {
	ClusterID int    `json:"cluster_id" comment:"集群ID"`
	Namespace string `json:"namespace" comment:"命名空间"`
	Name      string `json:"name" comment:"Role名称"`
}

// UpdateRoleYamlReq 更新Role YAML请求
type UpdateRoleYamlReq struct {
	ClusterID   int    `json:"cluster_id" comment:"集群ID"`
	Namespace   string `json:"namespace" comment:"命名空间"`
	Name        string `json:"name" comment:"Role名称"`
	YamlContent string `json:"yaml_content" binding:"required" comment:"YAML内容"`
}

// GetRoleEventsReq 获取Role事件请求
type GetRoleEventsReq struct {
	ClusterID int    `json:"cluster_id" comment:"集群ID"`
	Namespace string `json:"namespace" comment:"命名空间"`
	Name      string `json:"name" comment:"Role名称"`
	Limit     int    `json:"limit" form:"limit" comment:"事件数量限制"`
}

// GetRoleUsageReq 获取Role使用情况请求
type GetRoleUsageReq struct {
	ClusterID int    `json:"cluster_id" comment:"集群ID"`
	Namespace string `json:"namespace" comment:"命名空间"`
	Name      string `json:"name" comment:"Role名称"`
}

// RoleListReq Role列表请求参数（兼容性）
type RoleListReq struct {
	ClusterID int    `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace string `json:"namespace" form:"namespace" comment:"命名空间"`
	Keyword   string `json:"keyword" form:"keyword" comment:"关键字搜索"`
	Page      int    `json:"page" form:"page" comment:"页码"`
	PageSize  int    `json:"page_size" form:"page_size" comment:"页面大小"`
}

// RoleGetReq 获取单个Role请求（兼容性）
type RoleGetReq struct {
	ClusterID int    `json:"cluster_id" uri:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace string `json:"namespace" uri:"namespace" binding:"required" comment:"命名空间"`
	Name      string `json:"name" uri:"name" binding:"required" comment:"Role名称"`
}

// CreateRoleReq 创建Role请求参数
type CreateRoleReq struct {
	ClusterID   int               `json:"cluster_id" binding:"required" comment:"集群ID"`
	Name        string            `json:"name" binding:"required" comment:"Role名称"`
	Namespace   string            `json:"namespace" binding:"required" comment:"命名空间"`
	Labels      map[string]string `json:"labels" comment:"标签"`
	Annotations map[string]string `json:"annotations" comment:"注解"`
	Rules       []PolicyRule      `json:"rules" binding:"required" comment:"权限规则"`
}

// UpdateRoleReq 更新Role请求参数
type UpdateRoleReq struct {
	ClusterID    int               `json:"cluster_id" binding:"required" comment:"集群ID"`
	Name         string            `json:"name" binding:"required" comment:"Role名称"`
	Namespace    string            `json:"namespace" binding:"required" comment:"命名空间"`
	OriginalName string            `json:"original_name" comment:"原始名称"`
	Labels       map[string]string `json:"labels" comment:"标签"`
	Annotations  map[string]string `json:"annotations" comment:"注解"`
	Rules        []PolicyRule      `json:"rules" binding:"required" comment:"权限规则"`
}

// DeleteRoleReq 删除Role请求
type DeleteRoleReq struct {
	ClusterID int    `json:"cluster_id" uri:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace string `json:"namespace" uri:"namespace" binding:"required" comment:"命名空间"`
	Name      string `json:"name" uri:"name" binding:"required" comment:"Role名称"`
}

// RoleYamlReq Role YAML请求
type RoleYamlReq struct {
	ClusterID   int    `json:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace   string `json:"namespace" binding:"required" comment:"命名空间"`
	Name        string `json:"name" binding:"required" comment:"Role名称"`
	YamlContent string `json:"yaml_content" binding:"required" comment:"YAML内容"`
}

// ====================== ClusterRole请求结构体 ======================

// GetClusterRoleListReq ClusterRole列表请求参数
type GetClusterRoleListReq struct {
	ClusterID int    `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"`
	Keyword   string `json:"keyword" form:"keyword" comment:"关键字搜索"`
	Page      int    `json:"page" form:"page" comment:"页码"`
	Size      int    `json:"size" form:"size" comment:"页面大小"`
	Status    string `json:"status" form:"status" comment:"状态过滤"`
}

// GetClusterRoleDetailsReq 获取ClusterRole详情请求
type GetClusterRoleDetailsReq struct {
	ClusterID int    `json:"cluster_id" comment:"集群ID"`
	Name      string `json:"name" comment:"ClusterRole名称"`
}

// GetClusterRoleYamlReq 获取ClusterRole YAML请求
type GetClusterRoleYamlReq struct {
	ClusterID int    `json:"cluster_id" comment:"集群ID"`
	Name      string `json:"name" comment:"ClusterRole名称"`
}

// UpdateClusterRoleYamlReq 更新ClusterRole YAML请求
type UpdateClusterRoleYamlReq struct {
	ClusterID   int    `json:"cluster_id" comment:"集群ID"`
	Name        string `json:"name" comment:"ClusterRole名称"`
	YamlContent string `json:"yaml_content" binding:"required" comment:"YAML内容"`
}

// GetClusterRoleEventsReq 获取ClusterRole事件请求
type GetClusterRoleEventsReq struct {
	ClusterID int    `json:"cluster_id" comment:"集群ID"`
	Name      string `json:"name" comment:"ClusterRole名称"`
	EventType string `json:"event_type" form:"event_type" comment:"事件类型"`
	Limit     int    `json:"limit" form:"limit" comment:"限制数量"`
}

// GetClusterRoleUsageReq 获取ClusterRole使用情况请求
type GetClusterRoleUsageReq struct {
	ClusterID int    `json:"cluster_id" comment:"集群ID"`
	Name      string `json:"name" comment:"ClusterRole名称"`
}

// ClusterRoleListReq ClusterRole列表请求参数（保持兼容性）
type ClusterRoleListReq struct {
	ClusterID int    `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"`
	Keyword   string `json:"keyword" form:"keyword" comment:"关键字搜索"`
	Page      int    `json:"page" form:"page" comment:"页码"`
	PageSize  int    `json:"page_size" form:"page_size" comment:"页面大小"`
}

// ClusterRoleGetReq 获取单个ClusterRole请求（保持兼容性）
type ClusterRoleGetReq struct {
	ClusterID int    `json:"cluster_id" uri:"cluster_id" binding:"required" comment:"集群ID"`
	Name      string `json:"name" uri:"name" binding:"required" comment:"ClusterRole名称"`
}

// CreateClusterRoleReq 创建ClusterRole请求参数
type CreateClusterRoleReq struct {
	ClusterID   int               `json:"cluster_id" binding:"required" comment:"集群ID"`
	Name        string            `json:"name" binding:"required" comment:"ClusterRole名称"`
	Labels      map[string]string `json:"labels" comment:"标签"`
	Annotations map[string]string `json:"annotations" comment:"注解"`
	Rules       []PolicyRule      `json:"rules" binding:"required" comment:"权限规则"`
}

// UpdateClusterRoleReq 更新ClusterRole请求参数
type UpdateClusterRoleReq struct {
	ClusterID    int               `json:"cluster_id" binding:"required" comment:"集群ID"`
	Name         string            `json:"name" binding:"required" comment:"ClusterRole名称"`
	OriginalName string            `json:"original_name" comment:"原始名称"`
	Labels       map[string]string `json:"labels" comment:"标签"`
	Annotations  map[string]string `json:"annotations" comment:"注解"`
	Rules        []PolicyRule      `json:"rules" binding:"required" comment:"权限规则"`
}

// DeleteClusterRoleReq 删除ClusterRole请求
type DeleteClusterRoleReq struct {
	ClusterID int    `json:"cluster_id" uri:"cluster_id" binding:"required" comment:"集群ID"`
	Name      string `json:"name" uri:"name" binding:"required" comment:"ClusterRole名称"`
}

// ClusterRoleYamlReq ClusterRole YAML请求
type ClusterRoleYamlReq struct {
	ClusterID   int    `json:"cluster_id" binding:"required" comment:"集群ID"`
	Name        string `json:"name" binding:"required" comment:"ClusterRole名称"`
	YamlContent string `json:"yaml_content" binding:"required" comment:"YAML内容"`
}

type CreateClusterRoleByYamlReq struct {
	ClusterID   int    `json:"cluster_id" binding:"required" comment:"集群ID"`
	YamlContent string `json:"yaml_content" binding:"required" comment:"YAML内容"`
}

// RoleBindingListReq RoleBinding列表请求参数
type RoleBindingListReq struct {
	ClusterID int    `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace string `json:"namespace" form:"namespace" comment:"命名空间"`
	Keyword   string `json:"keyword" form:"keyword" comment:"关键字搜索"`
	Page      int    `json:"page" form:"page" comment:"页码"`
	PageSize  int    `json:"page_size" form:"page_size" comment:"页面大小"`
}

// RoleBindingGetReq 获取单个RoleBinding请求
type RoleBindingGetReq struct {
	ClusterID int    `json:"cluster_id" uri:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace string `json:"namespace" uri:"namespace" binding:"required" comment:"命名空间"`
	Name      string `json:"name" uri:"name" binding:"required" comment:"RoleBinding名称"`
}

// CreateRoleBindingReq 创建RoleBinding请求参数
type CreateRoleBindingReq struct {
	ClusterID   int               `json:"cluster_id" binding:"required" comment:"集群ID"`
	Name        string            `json:"name" binding:"required" comment:"RoleBinding名称"`
	Namespace   string            `json:"namespace" binding:"required" comment:"命名空间"`
	Labels      map[string]string `json:"labels" comment:"标签"`
	Annotations map[string]string `json:"annotations" comment:"注解"`
	RoleRef     RoleRef           `json:"role_ref" binding:"required" comment:"角色引用"`
	Subjects    []Subject         `json:"subjects" binding:"required" comment:"主体列表"`
}

// UpdateRoleBindingReq 更新RoleBinding请求参数
type UpdateRoleBindingReq struct {
	ClusterID    int               `json:"cluster_id" binding:"required" comment:"集群ID"`
	Name         string            `json:"name" binding:"required" comment:"RoleBinding名称"`
	Namespace    string            `json:"namespace" binding:"required" comment:"命名空间"`
	OriginalName string            `json:"original_name" comment:"原始名称"`
	Labels       map[string]string `json:"labels" comment:"标签"`
	Annotations  map[string]string `json:"annotations" comment:"注解"`
	RoleRef      RoleRef           `json:"role_ref" binding:"required" comment:"角色引用"`
	Subjects     []Subject         `json:"subjects" binding:"required" comment:"主体列表"`
}

// DeleteRoleBindingReq 删除RoleBinding请求
type DeleteRoleBindingReq struct {
	ClusterID int    `json:"cluster_id" uri:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace string `json:"namespace" uri:"namespace" binding:"required" comment:"命名空间"`
	Name      string `json:"name" uri:"name" binding:"required" comment:"RoleBinding名称"`
}

// RoleBindingYamlReq RoleBinding YAML请求
type RoleBindingYamlReq struct {
	ClusterID   int    `json:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace   string `json:"namespace" binding:"required" comment:"命名空间"`
	Name        string `json:"name" binding:"required" comment:"RoleBinding名称"`
	YamlContent string `json:"yaml_content" binding:"required" comment:"YAML内容"`
}

// GetClusterRoleBindingListReq ClusterRoleBinding列表请求参数
type GetClusterRoleBindingListReq struct {
	ClusterID int    `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"`
	Keyword   string `json:"keyword" form:"keyword" comment:"关键字搜索"`
	Page      int    `json:"page" form:"page" comment:"页码"`
	PageSize  int    `json:"page_size" form:"page_size" comment:"页面大小"`
}

// GetClusterRoleBindingDetailsReq 获取ClusterRoleBinding详情请求
type GetClusterRoleBindingDetailsReq struct {
	ClusterID int    `json:"cluster_id" comment:"集群ID"`
	Name      string `json:"name" comment:"ClusterRoleBinding名称"`
}

// GetClusterRoleBindingYamlReq 获取ClusterRoleBinding YAML请求
type GetClusterRoleBindingYamlReq struct {
	ClusterID int    `json:"cluster_id" comment:"集群ID"`
	Name      string `json:"name" comment:"ClusterRoleBinding名称"`
}

// UpdateClusterRoleBindingYamlReq 更新ClusterRoleBinding YAML请求
type UpdateClusterRoleBindingYamlReq struct {
	ClusterID   int    `json:"cluster_id" comment:"集群ID"`
	Name        string `json:"name" comment:"ClusterRoleBinding名称"`
	YamlContent string `json:"yaml_content" binding:"required" comment:"YAML内容"`
}

// GetClusterRoleBindingEventsReq 获取ClusterRoleBinding事件请求
type GetClusterRoleBindingEventsReq struct {
	ClusterID int    `json:"cluster_id" comment:"集群ID"`
	Name      string `json:"name" comment:"ClusterRoleBinding名称"`
	Limit     int    `json:"limit" form:"limit" comment:"事件数量限制"`
}

// GetClusterRoleBindingUsageReq 获取ClusterRoleBinding使用情况请求
type GetClusterRoleBindingUsageReq struct {
	ClusterID int    `json:"cluster_id" comment:"集群ID"`
	Name      string `json:"name" comment:"ClusterRoleBinding名称"`
}

// ClusterRoleBindingListReq ClusterRoleBinding列表请求参数（兼容性）
type ClusterRoleBindingListReq struct {
	ClusterID int    `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"`
	Keyword   string `json:"keyword" form:"keyword" comment:"关键字搜索"`
	Page      int    `json:"page" form:"page" comment:"页码"`
	PageSize  int    `json:"page_size" form:"page_size" comment:"页面大小"`
}

// ClusterRoleBindingGetReq 获取单个ClusterRoleBinding请求（兼容性）
type ClusterRoleBindingGetReq struct {
	ClusterID int    `json:"cluster_id" uri:"cluster_id" binding:"required" comment:"集群ID"`
	Name      string `json:"name" uri:"name" binding:"required" comment:"ClusterRoleBinding名称"`
}

// CreateClusterRoleBindingReq 创建ClusterRoleBinding请求参数
type CreateClusterRoleBindingReq struct {
	ClusterID   int               `json:"cluster_id" binding:"required" comment:"集群ID"`
	Name        string            `json:"name" binding:"required" comment:"ClusterRoleBinding名称"`
	Labels      map[string]string `json:"labels" comment:"标签"`
	Annotations map[string]string `json:"annotations" comment:"注解"`
	RoleRef     RoleRef           `json:"role_ref" binding:"required" comment:"角色引用"`
	Subjects    []Subject         `json:"subjects" binding:"required" comment:"主体列表"`
}

// UpdateClusterRoleBindingReq 更新ClusterRoleBinding请求参数
type UpdateClusterRoleBindingReq struct {
	ClusterID    int               `json:"cluster_id" binding:"required" comment:"集群ID"`
	Name         string            `json:"name" binding:"required" comment:"ClusterRoleBinding名称"`
	OriginalName string            `json:"original_name" comment:"原始名称"`
	Labels       map[string]string `json:"labels" comment:"标签"`
	Annotations  map[string]string `json:"annotations" comment:"注解"`
	RoleRef      RoleRef           `json:"role_ref" binding:"required" comment:"角色引用"`
	Subjects     []Subject         `json:"subjects" binding:"required" comment:"主体列表"`
}

// DeleteClusterRoleBindingReq 删除ClusterRoleBinding请求
type DeleteClusterRoleBindingReq struct {
	ClusterID int    `json:"cluster_id" uri:"cluster_id" binding:"required" comment:"集群ID"`
	Name      string `json:"name" uri:"name" binding:"required" comment:"ClusterRoleBinding名称"`
}

// ClusterRoleBindingYamlReq ClusterRoleBinding YAML请求
type ClusterRoleBindingYamlReq struct {
	ClusterID   int    `json:"cluster_id" binding:"required" comment:"集群ID"`
	Name        string `json:"name" binding:"required" comment:"ClusterRoleBinding名称"`
	YamlContent string `json:"yaml_content" binding:"required" comment:"YAML内容"`
}

// ====================== 权限检查和其他请求结构体 ======================

// CheckPermissionsReq 权限检查请求
type CheckPermissionsReq struct {
	ClusterID int     `json:"cluster_id" binding:"required" comment:"集群ID"`
	Subject   Subject `json:"subject" binding:"required" comment:"主体"`
	Resources []struct {
		Namespace string `json:"namespace" comment:"命名空间"`
		Resource  string `json:"resource" binding:"required" comment:"资源"`
		Verb      string `json:"verb" binding:"required" comment:"动作"`
	} `json:"resources" binding:"required" comment:"资源列表"`
}

// SubjectPermissionsReq 获取主体权限请求
type SubjectPermissionsReq struct {
	ClusterID int     `json:"cluster_id" uri:"cluster_id" binding:"required" comment:"集群ID"`
	Subject   Subject `json:"subject" binding:"required" comment:"主体"`
}

// ResourceVerbsResponse 预定义资源和动作列表响应
type ResourceVerbsResponse struct {
	Resources []ResourceInfo `json:"resources"`
}

// ResourceInfo 资源信息
type ResourceInfo struct {
	APIGroup  string   `json:"api_group"`
	Resource  string   `json:"resource"`
	Verbs     []string `json:"verbs"`
	Shortname string   `json:"shortname,omitempty"`
}

// PermissionResult 权限检查结果
type PermissionResult struct {
	Namespace string    `json:"namespace"`
	Resource  string    `json:"resource"`
	Verb      string    `json:"verb"`
	Allowed   BoolValue `json:"allowed"`
	Reason    string    `json:"reason,omitempty"`
}

// SubjectPermissionsResponse 主体权限响应
type SubjectPermissionsResponse struct {
	Subject      Subject      `json:"subject"`
	Permissions  []PolicyRule `json:"permissions"`
	Roles        []string     `json:"roles"`
	ClusterRoles []string     `json:"cluster_roles"`
}

// ====================== 补充的响应结构体 ======================

// RoleListResponse Role列表响应
type RoleListResponse struct {
	Items      []RoleInfo `json:"items"`       // Role列表
	TotalCount int        `json:"total_count"` // 总数
}

// RoleDetailResponse Role详情响应
type RoleDetailResponse struct {
	Role     RoleInfo                `json:"role"`     // Role信息
	YAML     string                  `json:"yaml"`     // YAML内容
	Events   []RoleEventEntity       `json:"events"`   // 事件列表
	Bindings []RoleBindingSimpleInfo `json:"bindings"` // 关联的RoleBinding
	Usage    RoleUsageEntity         `json:"usage"`    // 使用情况
}

// RoleEventEntity Role事件实体
type RoleEventEntity struct {
	Type      string `json:"type"`       // 事件类型
	Reason    string `json:"reason"`     // 原因
	Message   string `json:"message"`    // 消息
	Source    string `json:"source"`     // 来源
	FirstTime string `json:"first_time"` // 首次时间
	LastTime  string `json:"last_time"`  // 最后时间
	Count     int32  `json:"count"`      // 次数
}

// RoleBindingSimpleInfo RoleBinding简要信息
type RoleBindingSimpleInfo struct {
	Name      string    `json:"name"`      // RoleBinding名称
	Namespace string    `json:"namespace"` // 命名空间
	Subjects  []Subject `json:"subjects"`  // 主体列表
}

// RoleUsageEntity Role使用情况实体
type RoleUsageEntity struct {
	TotalBindings int                     `json:"total_bindings"` // 绑定总数
	ActiveUsers   int                     `json:"active_users"`   // 活跃用户数
	Bindings      []RoleBindingSimpleInfo `json:"bindings"`       // 绑定列表
}

// ClusterRoleListResponse ClusterRole列表响应
type ClusterRoleListResponse struct {
	Items      []ClusterRoleInfo `json:"items"`       // ClusterRole列表
	TotalCount int               `json:"total_count"` // 总数
}

// ClusterRoleDetailResponse ClusterRole详情响应
type ClusterRoleDetailResponse struct {
	ClusterRole ClusterRoleInfo                `json:"cluster_role"` // ClusterRole信息
	YAML        string                         `json:"yaml"`         // YAML内容
	Events      []ClusterRoleEventEntity       `json:"events"`       // 事件列表
	Bindings    []ClusterRoleBindingSimpleInfo `json:"bindings"`     // 关联的ClusterRoleBinding
	Usage       ClusterRoleUsageEntity         `json:"usage"`        // 使用情况
}

// ClusterRoleEventEntity ClusterRole事件实体
type ClusterRoleEventEntity struct {
	Type      string `json:"type"`       // 事件类型
	Reason    string `json:"reason"`     // 原因
	Message   string `json:"message"`    // 消息
	Source    string `json:"source"`     // 来源
	FirstTime string `json:"first_time"` // 首次时间
	LastTime  string `json:"last_time"`  // 最后时间
	Count     int32  `json:"count"`      // 次数
}

// ClusterRoleBindingSimpleInfo ClusterRoleBinding简要信息
type ClusterRoleBindingSimpleInfo struct {
	Name     string    `json:"name"`     // ClusterRoleBinding名称
	Subjects []Subject `json:"subjects"` // 主体列表
}

// ClusterRoleUsageEntity ClusterRole使用情况实体
type ClusterRoleUsageEntity struct {
	TotalBindings int                            `json:"total_bindings"` // 绑定总数
	ActiveUsers   int                            `json:"active_users"`   // 活跃用户数
	Bindings      []ClusterRoleBindingSimpleInfo `json:"bindings"`       // 绑定列表
}

// RoleBindingListResponse RoleBinding列表响应
type RoleBindingListResponse struct {
	Items      []RoleBindingInfo `json:"items"`       // RoleBinding列表
	TotalCount int               `json:"total_count"` // 总数
}

// RoleBindingDetailResponse RoleBinding详情响应
type RoleBindingDetailResponse struct {
	RoleBinding RoleBindingInfo          `json:"role_binding"` // RoleBinding信息
	YAML        string                   `json:"yaml"`         // YAML内容
	Events      []RoleBindingEventEntity `json:"events"`       // 事件列表
	RoleDetail  RoleInfo                 `json:"role_detail"`  // 关联的Role详情
	Permissions []PolicyRule             `json:"permissions"`  // 有效权限
}

// RoleBindingEventEntity RoleBinding事件实体
type RoleBindingEventEntity struct {
	Type      string `json:"type"`       // 事件类型
	Reason    string `json:"reason"`     // 原因
	Message   string `json:"message"`    // 消息
	Source    string `json:"source"`     // 来源
	FirstTime string `json:"first_time"` // 首次时间
	LastTime  string `json:"last_time"`  // 最后时间
	Count     int32  `json:"count"`      // 次数
}

// ClusterRoleBindingListResponse ClusterRoleBinding列表响应
type ClusterRoleBindingListResponse struct {
	Items      []ClusterRoleBindingInfo `json:"items"`       // ClusterRoleBinding列表
	TotalCount int                      `json:"total_count"` // 总数
}

// ClusterRoleBindingDetailResponse ClusterRoleBinding详情响应
type ClusterRoleBindingDetailResponse struct {
	ClusterRoleBinding ClusterRoleBindingInfo          `json:"cluster_role_binding"` // ClusterRoleBinding信息
	YAML               string                          `json:"yaml"`                 // YAML内容
	Events             []ClusterRoleBindingEventEntity `json:"events"`               // 事件列表
	ClusterRoleDetail  ClusterRoleInfo                 `json:"cluster_role_detail"`  // 关联的ClusterRole详情
	Permissions        []PolicyRule                    `json:"permissions"`          // 有效权限
}

// ClusterRoleBindingEventEntity ClusterRoleBinding事件实体
type ClusterRoleBindingEventEntity struct {
	Type      string `json:"type"`       // 事件类型
	Reason    string `json:"reason"`     // 原因
	Message   string `json:"message"`    // 消息
	Source    string `json:"source"`     // 来源
	FirstTime string `json:"first_time"` // 首次时间
	LastTime  string `json:"last_time"`  // 最后时间
	Count     int32  `json:"count"`      // 次数
}

// CheckPermissionsResponse 权限检查响应
type CheckPermissionsResponse struct {
	Subject      Subject            `json:"subject"`       // 主体
	Results      []PermissionResult `json:"results"`       // 检查结果
	AllowedCount int                `json:"allowed_count"` // 允许的权限数
	DeniedCount  int                `json:"denied_count"`  // 拒绝的权限数
}

// RBACStatisticsResponse RBAC统计响应
type RBACStatisticsResponse struct {
	Statistics RBACStatistics             `json:"statistics"` // 统计信息
	Details    RBACStatisticsDetailEntity `json:"details"`    // 详细信息
}

// RBACStatisticsDetailEntity RBAC统计详细信息实体
type RBACStatisticsDetailEntity struct {
	SystemRolesList []string                  `json:"system_roles_list"` // 系统角色列表
	CustomRolesList []string                  `json:"custom_roles_list"` // 自定义角色列表
	NamespaceStats  []NamespaceRBACStatEntity `json:"namespace_stats"`   // 命名空间统计
	SubjectStats    []SubjectStatEntity       `json:"subject_stats"`     // 主体统计
	PermissionStats []PermissionStatEntity    `json:"permission_stats"`  // 权限统计
}

// NamespaceRBACStatEntity 命名空间RBAC统计实体
type NamespaceRBACStatEntity struct {
	Namespace        string `json:"namespace"`          // 命名空间
	RoleCount        int    `json:"role_count"`         // Role数量
	RoleBindingCount int    `json:"role_binding_count"` // RoleBinding数量
	SubjectCount     int    `json:"subject_count"`      // 主体数量
}

// SubjectStatEntity 主体统计实体
type SubjectStatEntity struct {
	Kind  string `json:"kind"`  // 主体类型
	Count int    `json:"count"` // 数量
}

// PermissionStatEntity 权限统计实体
type PermissionStatEntity struct {
	Resource string `json:"resource"` // 资源
	Verb     string `json:"verb"`     // 动作
	Count    int    `json:"count"`    // 使用次数
}

// RBACMatrixResponse RBAC权限矩阵响应
type RBACMatrixResponse struct {
	Subjects   []string                 `json:"subjects"`   // 主体列表
	Resources  []string                 `json:"resources"`  // 资源列表
	Verbs      []string                 `json:"verbs"`      // 动作列表
	Matrix     [][]RBACMatrixCellEntity `json:"matrix"`     // 权限矩阵
	Namespaces []string                 `json:"namespaces"` // 命名空间列表
}

// RBACMatrixCellEntity RBAC权限矩阵单元格实体
type RBACMatrixCellEntity struct {
	Allowed   BoolValue `json:"allowed"`   // 是否允许
	RoleName  string    `json:"role_name"` // 角色名称
	Namespace string    `json:"namespace"` // 命名空间
}

// RBACResourceVerbsResponse RBAC资源动作响应
type RBACResourceVerbsResponse struct {
	Resources []ResourceInfo `json:"resources"` // 资源信息列表
}

// RBACSecurityReportResponse RBAC安全报告响应
type RBACSecurityReportResponse struct {
	OverprivilegedRoles  []SecurityRoleEntity           `json:"overprivileged_roles"`   // 权限过大的角色
	UnusedRoles          []SecurityRoleEntity           `json:"unused_roles"`           // 未使用的角色
	DangerousPermissions []SecurityPermissionEntity     `json:"dangerous_permissions"`  // 危险权限
	OrphanedRoleBindings []SecurityBindingEntity        `json:"orphaned_role_bindings"` // 孤立的RoleBinding
	CrossNamespaceAccess []SecurityAccessEntity         `json:"cross_namespace_access"` // 跨命名空间访问
	SecurityScore        int                            `json:"security_score"`         // 安全评分
	Recommendations      []SecurityRecommendationEntity `json:"recommendations"`        // 安全建议
}

// SecurityRoleEntity 安全角色实体
type SecurityRoleEntity struct {
	Name        string   `json:"name"`        // 角色名称
	Namespace   string   `json:"namespace"`   // 命名空间
	Kind        string   `json:"kind"`        // 类型(Role/ClusterRole)
	Risk        string   `json:"risk"`        // 风险等级
	Permissions []string `json:"permissions"` // 权限列表
	Reason      string   `json:"reason"`      // 原因
}

// SecurityPermissionEntity 安全权限实体
type SecurityPermissionEntity struct {
	Resource string   `json:"resource"` // 资源
	Verb     string   `json:"verb"`     // 动作
	Risk     string   `json:"risk"`     // 风险等级
	Roles    []string `json:"roles"`    // 使用该权限的角色
	Reason   string   `json:"reason"`   // 风险原因
}

// SecurityBindingEntity 安全绑定实体
type SecurityBindingEntity struct {
	Name      string `json:"name"`      // 绑定名称
	Namespace string `json:"namespace"` // 命名空间
	Kind      string `json:"kind"`      // 类型(RoleBinding/ClusterRoleBinding)
	RoleName  string `json:"role_name"` // 角色名称
	Reason    string `json:"reason"`    // 孤立原因
}

// SecurityAccessEntity 安全访问实体
type SecurityAccessEntity struct {
	Subject    string `json:"subject"`    // 主体
	SourceNS   string `json:"source_ns"`  // 源命名空间
	TargetNS   string `json:"target_ns"`  // 目标命名空间
	Permission string `json:"permission"` // 权限
	Risk       string `json:"risk"`       // 风险等级
}

// SecurityRecommendationEntity 安全建议实体
type SecurityRecommendationEntity struct {
	Type        string `json:"type"`        // 建议类型
	Title       string `json:"title"`       // 标题
	Description string `json:"description"` // 描述
	Priority    string `json:"priority"`    // 优先级
	Action      string `json:"action"`      // 建议操作
}

// ====================== 补充缺失的RoleBinding请求结构体 ======================

// GetRoleBindingListReq 获取RoleBinding列表请求
type GetRoleBindingListReq struct {
	ClusterID int    `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace string `json:"namespace" form:"namespace" comment:"命名空间"`
	Name      string `json:"name" form:"name" comment:"RoleBinding名称过滤"`
	LabelKey  string `json:"label_key" form:"label_key" comment:"标签键过滤"`
	Keyword   string `json:"keyword" form:"keyword" comment:"关键字搜索"`
	Page      int    `json:"page" form:"page" comment:"页码"`
	PageSize  int    `json:"page_size" form:"page_size" comment:"页面大小"`
}

// GetRoleBindingDetailsReq 获取RoleBinding详情请求
type GetRoleBindingDetailsReq struct {
	ClusterID int    `json:"cluster_id" uri:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace string `json:"namespace" uri:"namespace" binding:"required" comment:"命名空间"`
	Name      string `json:"name" uri:"name" binding:"required" comment:"RoleBinding名称"`
}

// GetRoleBindingYamlReq 获取RoleBinding YAML请求
type GetRoleBindingYamlReq struct {
	ClusterID int    `json:"cluster_id" uri:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace string `json:"namespace" uri:"namespace" binding:"required" comment:"命名空间"`
	Name      string `json:"name" uri:"name" binding:"required" comment:"RoleBinding名称"`
}

// UpdateRoleBindingYamlReq 更新RoleBinding YAML请求
type UpdateRoleBindingYamlReq struct {
	ClusterID   int    `json:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace   string `json:"namespace" binding:"required" comment:"命名空间"`
	Name        string `json:"name" binding:"required" comment:"RoleBinding名称"`
	Yaml        string `json:"yaml" binding:"required" comment:"YAML内容"`
	YamlContent string `json:"yaml_content" binding:"required" comment:"YAML内容"`
}

// GetRoleBindingEventsReq 获取RoleBinding事件请求
type GetRoleBindingEventsReq struct {
	ClusterID int    `json:"cluster_id" uri:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace string `json:"namespace" uri:"namespace" binding:"required" comment:"命名空间"`
	Name      string `json:"name" uri:"name" binding:"required" comment:"RoleBinding名称"`
}

// GetRoleBindingUsageReq 获取RoleBinding使用情况请求
type GetRoleBindingUsageReq struct {
	ClusterID int    `json:"cluster_id" uri:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace string `json:"namespace" uri:"namespace" binding:"required" comment:"命名空间"`
	Name      string `json:"name" uri:"name" binding:"required" comment:"RoleBinding名称"`
}

// ====================== ServiceAccount请求结构体 ======================

// GetServiceAccountListReq 获取ServiceAccount列表请求
type GetServiceAccountListReq struct {
	ClusterID int    `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace string `json:"namespace" form:"namespace" comment:"命名空间"`
	Name      string `json:"name" form:"name" comment:"ServiceAccount名称过滤"`
	LabelKey  string `json:"label_key" form:"label_key" comment:"标签键过滤"`
	Keyword   string `json:"keyword" form:"keyword" comment:"关键字搜索"`
	Page      int    `json:"page" form:"page" comment:"页码"`
	PageSize  int    `json:"page_size" form:"page_size" comment:"页面大小"`
}

// GetServiceAccountDetailsReq 获取ServiceAccount详情请求
type GetServiceAccountDetailsReq struct {
	ClusterID int    `json:"cluster_id" uri:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace string `json:"namespace" uri:"namespace" binding:"required" comment:"命名空间"`
	Name      string `json:"name" uri:"name" binding:"required" comment:"ServiceAccount名称"`
}

// CreateServiceAccountReq 创建ServiceAccount请求
type CreateServiceAccountReq struct {
	ClusterID                    int               `json:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace                    string            `json:"namespace" binding:"required" comment:"命名空间"`
	Name                         string            `json:"name" binding:"required" comment:"ServiceAccount名称"`
	Labels                       map[string]string `json:"labels" comment:"标签"`
	Annotations                  map[string]string `json:"annotations" comment:"注解"`
	AutomountServiceAccountToken *BoolValue        `json:"automount_service_account_token" comment:"是否自动挂载服务账户令牌"`
}

// UpdateServiceAccountReq 更新ServiceAccount请求
type UpdateServiceAccountReq struct {
	ClusterID                    int               `json:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace                    string            `json:"namespace" binding:"required" comment:"命名空间"`
	Name                         string            `json:"name" binding:"required" comment:"ServiceAccount名称"`
	Labels                       map[string]string `json:"labels" comment:"标签"`
	Annotations                  map[string]string `json:"annotations" comment:"注解"`
	AutomountServiceAccountToken *BoolValue        `json:"automount_service_account_token" comment:"是否自动挂载服务账户令牌"`
}

// DeleteServiceAccountReq 删除ServiceAccount请求
type DeleteServiceAccountReq struct {
	ClusterID int    `json:"cluster_id" uri:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace string `json:"namespace" uri:"namespace" binding:"required" comment:"命名空间"`
	Name      string `json:"name" uri:"name" binding:"required" comment:"ServiceAccount名称"`
}

// GetServiceAccountYamlReq 获取ServiceAccount YAML请求
type GetServiceAccountYamlReq struct {
	ClusterID int    `json:"cluster_id" uri:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace string `json:"namespace" uri:"namespace" binding:"required" comment:"命名空间"`
	Name      string `json:"name" uri:"name" binding:"required" comment:"ServiceAccount名称"`
}

// UpdateServiceAccountYamlReq 更新ServiceAccount YAML请求
type UpdateServiceAccountYamlReq struct {
	ClusterID   int    `json:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace   string `json:"namespace" binding:"required" comment:"命名空间"`
	Name        string `json:"name" binding:"required" comment:"ServiceAccount名称"`
	Yaml        string `json:"yaml" binding:"required" comment:"YAML内容"`
	YamlContent string `json:"yaml_content" binding:"required" comment:"YAML内容"`
}

// GetServiceAccountEventsReq 获取ServiceAccount事件请求
type GetServiceAccountEventsReq struct {
	ClusterID int    `json:"cluster_id" uri:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace string `json:"namespace" uri:"namespace" binding:"required" comment:"命名空间"`
	Name      string `json:"name" uri:"name" binding:"required" comment:"ServiceAccount名称"`
}

// GetServiceAccountUsageReq 获取ServiceAccount使用情况请求
type GetServiceAccountUsageReq struct {
	ClusterID int    `json:"cluster_id" uri:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace string `json:"namespace" uri:"namespace" binding:"required" comment:"命名空间"`
	Name      string `json:"name" uri:"name" binding:"required" comment:"ServiceAccount名称"`
}

// GetServiceAccountTokenReq 获取ServiceAccount令牌请求
type GetServiceAccountTokenReq struct {
	ClusterID int    `json:"cluster_id" uri:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace string `json:"namespace" uri:"namespace" binding:"required" comment:"命名空间"`
	Name      string `json:"name" uri:"name" binding:"required" comment:"ServiceAccount名称"`
}

// CreateServiceAccountTokenReq 创建ServiceAccount令牌请求
type CreateServiceAccountTokenReq struct {
	ClusterID  int    `json:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace  string `json:"namespace" binding:"required" comment:"命名空间"`
	Name       string `json:"name" binding:"required" comment:"ServiceAccount名称"`
	ExpiryTime *int64 `json:"expiry_time" comment:"令牌过期时间（秒）"`
}
