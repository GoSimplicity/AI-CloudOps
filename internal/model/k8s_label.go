package model

import "time"

// K8sLabelRequest 标签管理相关请求结构
type K8sLabelRequest struct {
	ClusterID     int               `json:"cluster_id" binding:"required"`    // 集群ID，必填
	Namespace     string            `json:"namespace"`                        // 命名空间，可选
	ResourceType  string            `json:"resource_type" binding:"required"` // 资源类型，必填
	ResourceName  string            `json:"resource_name"`                    // 资源名称，可选
	Labels        map[string]string `json:"labels"`                           // 标签键值对
	Annotations   map[string]string `json:"annotations"`                      // 注解键值对
	LabelSelector map[string]string `json:"label_selector"`                   // 标签选择器
	Operation     string            `json:"operation"`                        // 操作类型 (add, update, delete)
	ResourceNames []string          `json:"resource_names"`                   // 批量操作的资源名称列表
}

// K8sLabelResponse 标签管理响应结构
type K8sLabelResponse struct {
	ResourceType      string            `json:"resource_type"`      // 资源类型
	ResourceName      string            `json:"resource_name"`      // 资源名称
	Namespace         string            `json:"namespace"`          // 命名空间
	Labels            map[string]string `json:"labels"`             // 标签键值对
	Annotations       map[string]string `json:"annotations"`        // 注解键值对
	CreationTimestamp time.Time         `json:"creation_timestamp"` // 创建时间
}

// K8sLabelSelectorRequest 标签选择器查询请求
type K8sLabelSelectorRequest struct {
	ClusterID     int               `json:"cluster_id" binding:"required"`    // 集群ID，必填
	Namespace     string            `json:"namespace"`                        // 命名空间，可选
	ResourceType  string            `json:"resource_type" binding:"required"` // 资源类型，必填
	LabelSelector map[string]string `json:"label_selector"`                   // 标签选择器
	FieldSelector string            `json:"field_selector"`                   // 字段选择器
	Limit         int               `json:"limit"`                            // 限制数量
}

// K8sLabelPolicyRequest 标签策略请求
type K8sLabelPolicyRequest struct {
	ClusterID    int            `json:"cluster_id" binding:"required"`  // 集群ID，必填
	Namespace    string         `json:"namespace"`                      // 命名空间，可选
	PolicyName   string         `json:"policy_name" binding:"required"` // 策略名称，必填
	PolicyType   string         `json:"policy_type"`                    // 策略类型 (required, forbidden, preferred)
	ResourceType string         `json:"resource_type"`                  // 资源类型
	LabelRules   []K8sLabelRule `json:"label_rules"`                    // 标签规则
	Enabled      bool           `json:"enabled"`                        // 是否启用
	Description  string         `json:"description"`                    // 策略描述
}

// K8sLabelRule 标签规则
type K8sLabelRule struct {
	Key         string   `json:"key"`         // 标签键
	Values      []string `json:"values"`      // 标签值列表
	Operator    string   `json:"operator"`    // 操作符 (In, NotIn, Exists, DoesNotExist)
	Required    bool     `json:"required"`    // 是否必需
	Description string   `json:"description"` // 规则描述
}

// K8sLabelComplianceRequest 标签合规性检查请求
type K8sLabelComplianceRequest struct {
	ClusterID    int    `json:"cluster_id" binding:"required"` // 集群ID，必填
	Namespace    string `json:"namespace"`                     // 命名空间，可选
	ResourceType string `json:"resource_type"`                 // 资源类型，可选
	PolicyName   string `json:"policy_name"`                   // 策略名称，可选
	CheckAll     bool   `json:"check_all"`                     // 是否检查所有资源
}

// K8sLabelComplianceResponse 标签合规性检查响应
type K8sLabelComplianceResponse struct {
	ResourceType    string    `json:"resource_type"`    // 资源类型
	ResourceName    string    `json:"resource_name"`    // 资源名称
	Namespace       string    `json:"namespace"`        // 命名空间
	PolicyName      string    `json:"policy_name"`      // 策略名称
	Compliant       bool      `json:"compliant"`        // 是否合规
	ViolationReason string    `json:"violation_reason"` // 违规原因
	MissingLabels   []string  `json:"missing_labels"`   // 缺失的标签
	ExtraLabels     []string  `json:"extra_labels"`     // 多余的标签
	CheckTime       time.Time `json:"check_time"`       // 检查时间
}

// K8sLabelBatchRequest 批量标签操作请求
type K8sLabelBatchRequest struct {
	ClusterID     int               `json:"cluster_id" binding:"required"`    // 集群ID，必填
	Namespace     string            `json:"namespace"`                        // 命名空间，可选
	ResourceType  string            `json:"resource_type" binding:"required"` // 资源类型，必填
	ResourceNames []string          `json:"resource_names"`                   // 资源名称列表
	Operation     string            `json:"operation" binding:"required"`     // 操作类型 (add, update, delete)
	Labels        map[string]string `json:"labels"`                           // 标签键值对
	LabelSelector map[string]string `json:"label_selector"`                   // 标签选择器（用于批量选择）
}

// K8sLabelHistoryRequest 标签历史记录请求
type K8sLabelHistoryRequest struct {
	ClusterID    int        `json:"cluster_id" binding:"required"` // 集群ID，必填
	Namespace    string     `json:"namespace"`                     // 命名空间，可选
	ResourceType string     `json:"resource_type"`                 // 资源类型，可选
	ResourceName string     `json:"resource_name"`                 // 资源名称，可选
	StartTime    *time.Time `json:"start_time"`                    // 开始时间
	EndTime      *time.Time `json:"end_time"`                      // 结束时间
	Limit        int        `json:"limit"`                         // 限制数量
}

// K8sLabelHistoryResponse 标签历史记录响应
type K8sLabelHistoryResponse struct {
	ID           int               `json:"id"`            // 记录ID
	ClusterID    int               `json:"cluster_id"`    // 集群ID
	Namespace    string            `json:"namespace"`     // 命名空间
	ResourceType string            `json:"resource_type"` // 资源类型
	ResourceName string            `json:"resource_name"` // 资源名称
	Operation    string            `json:"operation"`     // 操作类型
	OldLabels    map[string]string `json:"old_labels"`    // 原标签
	NewLabels    map[string]string `json:"new_labels"`    // 新标签
	ChangedBy    string            `json:"changed_by"`    // 修改者
	ChangeTime   time.Time         `json:"change_time"`   // 修改时间
	ChangeReason string            `json:"change_reason"` // 修改原因
}
