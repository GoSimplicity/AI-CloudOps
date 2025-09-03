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

// K8sYamlTask Kubernetes YAML 任务的配置
type K8sYamlTask struct {
	Model
	Name        string     `json:"name" gorm:"type:varchar(255);comment:YAML 任务名称"`                 // YAML 任务名称
	UserID      int        `json:"user_id" gorm:"comment:创建者用户ID"`                                  // 创建者用户ID
	TemplateID  int        `json:"template_id" gorm:"comment:关联的模板ID"`                              // 关联的模板ID
	ClusterId   int        `json:"cluster_id,omitempty" gorm:"comment:集群名称"`                        // 集群名称
	Variables   StringList `json:"variables,omitempty" gorm:"type:text;comment:yaml 变量，格式 k=v,k=v"` // YAML 变量
	Status      string     `json:"status,omitempty" gorm:"comment:当前状态"`                            // 当前状态
	ApplyResult string     `json:"apply_result,omitempty" gorm:"comment:apply 后的返回数据"`              // apply 结果
}

func (r *K8sYamlTask) TableName() string {
	return "cl_k8s_yaml_task"
}

// K8sYamlTemplate Kubernetes YAML 模板的配置
type K8sYamlTemplate struct {
	Model
	Name      string `json:"name" binding:"required,min=1,max=50" gorm:"size:100;comment:模板名称"` // 模板名称
	UserID    int    `json:"user_id" gorm:"comment:创建者用户ID"`                                    // 创建者用户ID
	Content   string `json:"content,omitempty" gorm:"type:text;comment:yaml 模板内容"`              // YAML 模板内容
	ClusterId int    `json:"cluster_id,omitempty" gorm:"comment:对应集群id"`
}

func (r *K8sYamlTemplate) TableName() string {
	return "cl_k8s_yaml_template"
}

// K8sResourceType K8s资源类型枚举
type K8sResourceType string

const (
	ResourceTypeDeployment  K8sResourceType = "deployment"
	ResourceTypeService     K8sResourceType = "service"
	ResourceTypePod         K8sResourceType = "pod"
	ResourceTypeConfigMap   K8sResourceType = "configmap"
	ResourceTypeSecret      K8sResourceType = "secret"
	ResourceTypeIngress     K8sResourceType = "ingress"
	ResourceTypePV          K8sResourceType = "persistentvolume"
	ResourceTypePVC         K8sResourceType = "persistentvolumeclaim"
	ResourceTypeDaemonSet   K8sResourceType = "daemonset"
	ResourceTypeStatefulSet K8sResourceType = "statefulset"
	ResourceTypeJob         K8sResourceType = "job"
	ResourceTypeCronJob     K8sResourceType = "cronjob"
)

// CreateResourceByYamlReq 通过YAML创建K8s资源的通用请求
type CreateResourceByYamlReq struct {
	ClusterID    int             `json:"cluster_id" binding:"required"`    // 集群ID
	ResourceType K8sResourceType `json:"resource_type" binding:"required"` // 资源类型
	YAML         string          `json:"yaml" binding:"required"`          // YAML内容
}

// UpdateResourceByYamlReq 通过YAML更新K8s资源的通用请求
type UpdateResourceByYamlReq struct {
	ClusterID    int             `json:"cluster_id" binding:"required"`    // 集群ID
	ResourceType K8sResourceType `json:"resource_type" binding:"required"` // 资源类型
	Namespace    string          `json:"namespace" binding:"required"`     // 命名空间
	Name         string          `json:"name" binding:"required"`          // 资源名称
	YAML         string          `json:"yaml" binding:"required"`          // YAML内容
}

// ApplyResourceByYamlReq 应用YAML到K8s集群的请求
type ApplyResourceByYamlReq struct {
	ClusterID int    `json:"cluster_id" binding:"required"` // 集群ID
	YAML      string `json:"yaml" binding:"required"`       // YAML内容
	DryRun    bool   `json:"dry_run"`                       // 是否为试运行
}

// ValidateYamlReq 验证YAML格式的请求
type ValidateYamlReq struct {
	YAML string `json:"yaml" binding:"required"` // YAML内容
}

// ConvertToYamlReq 将资源配置转换为YAML的请求
type ConvertToYamlReq struct {
	ClusterID    int             `json:"cluster_id" binding:"required"`    // 集群ID
	ResourceType K8sResourceType `json:"resource_type" binding:"required"` // 资源类型
	Config       interface{}     `json:"config" binding:"required"`        // 资源配置信息
}

// YamlTemplateUpdateReq 更新 YAML 模板请求
type YamlTemplateUpdateReq struct {
	ID        int    `json:"id" binding:"required"`                // 模板ID
	Name      string `json:"name" binding:"required,min=1,max=50"` // 模板名称
	UserID    int    `json:"user_id"`                              // 创建者用户ID（从token中获取）
	Content   string `json:"content" binding:"required"`           // YAML 模板内容
	ClusterId int    `json:"cluster_id" binding:"required"`        // 集群ID
}

// YamlTemplateCheckReq 检查 YAML 模板请求
type YamlTemplateCheckReq struct {
	Name      string `json:"name" binding:"required,min=1,max=50"` // 模板名称
	Content   string `json:"content" binding:"required"`           // YAML 模板内容
	ClusterId int    `json:"cluster_id" binding:"required"`        // 集群ID
}

// YamlTemplateListResponse YAML模板列表响应
type YamlTemplateListResponse struct {
	Items      []K8sYamlTemplate `json:"items"`       // 模板列表
	TotalCount int               `json:"total_count"` // 总数
}

// YamlTaskListResponse YAML任务列表响应
type YamlTaskListResponse struct {
	Items      []K8sYamlTask `json:"items"`       // 任务列表
	TotalCount int           `json:"total_count"` // 总数
}

// YamlTaskExecuteResponse YAML任务执行响应
type YamlTaskExecuteResponse struct {
	TaskID      int    `json:"task_id"`      // 任务ID
	Status      string `json:"status"`       // 执行状态
	Message     string `json:"message"`      // 执行消息
	ApplyResult string `json:"apply_result"` // apply结果
	DryRun      bool   `json:"dry_run"`      // 是否为试运行
}

// YamlTaskLogsResponse YAML任务日志响应
type YamlTaskLogsResponse struct {
	TaskID   int                 `json:"task_id"`   // 任务ID
	TaskName string              `json:"task_name"` // 任务名称
	Logs     []YamlTaskLogEntity `json:"logs"`      // 日志列表
}

// YamlTaskLogEntity YAML任务日志实体
type YamlTaskLogEntity struct {
	Timestamp string `json:"timestamp"` // 时间戳
	Level     string `json:"level"`     // 日志级别
	Message   string `json:"message"`   // 日志消息
	Source    string `json:"source"`    // 日志来源
}

// YamlTaskStatusResponse YAML任务状态响应
type YamlTaskStatusResponse struct {
	TaskID      int    `json:"task_id"`      // 任务ID
	TaskName    string `json:"task_name"`    // 任务名称
	Status      string `json:"status"`       // 当前状态
	Progress    int    `json:"progress"`     // 进度百分比
	StartTime   string `json:"start_time"`   // 开始时间
	EndTime     string `json:"end_time"`     // 结束时间
	Duration    string `json:"duration"`     // 执行时长
	ApplyResult string `json:"apply_result"` // apply结果
	ErrorMsg    string `json:"error_msg"`    // 错误消息
}

// RenderYamlResponse 渲染YAML响应
type RenderYamlResponse struct {
	TemplateID     int               `json:"template_id"`     // 模板ID
	TemplateName   string            `json:"template_name"`   // 模板名称
	RenderedYaml   string            `json:"rendered_yaml"`   // 渲染后的YAML
	Variables      map[string]string `json:"variables"`       // 变量列表
	ValidateResult bool              `json:"validate_result"` // 验证结果
	ValidateMsg    string            `json:"validate_msg"`    // 验证消息
}
