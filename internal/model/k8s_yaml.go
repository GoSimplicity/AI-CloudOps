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
	Name        string     `json:"name" gorm:"type:varchar(255);comment:YAML任务名称"`                // YAML任务名称
	UserID      int        `json:"user_id" gorm:"comment:创建者用户ID"`                                // 创建者用户ID
	TemplateID  int        `json:"template_id" gorm:"comment:关联的模板ID"`                            // 关联的模板ID
	ClusterID   int        `json:"cluster_id" gorm:"comment:集群ID"`                                // 集群ID
	Variables   StringList `json:"variables,omitempty" gorm:"type:text;comment:YAML变量，格式k=v,k=v"` // YAML变量
	Status      string     `json:"status,omitempty" gorm:"comment:当前状态"`                          // 当前状态
	ApplyResult string     `json:"apply_result,omitempty" gorm:"comment:apply后的返回数据"`             // apply结果
}

func (r *K8sYamlTask) TableName() string {
	return "cl_k8s_yaml_task"
}

// K8sYamlTemplate Kubernetes YAML 模板的配置
type K8sYamlTemplate struct {
	Model
	Name      string `json:"name" binding:"required,min=1,max=50" gorm:"size:100;comment:模板名称"` // 模板名称
	UserID    int    `json:"user_id" gorm:"comment:创建者用户ID"`                                    // 创建者用户ID
	Content   string `json:"content" binding:"required" gorm:"type:text;comment:YAML模板内容"`      // YAML模板内容
	ClusterID int    `json:"cluster_id" gorm:"comment:对应集群ID"`                                  // 对应集群ID
}

func (r *K8sYamlTemplate) TableName() string {
	return "cl_k8s_yaml_template"
}

// CreateResourceByYamlReq 通过YAML创建K8s资源的通用请求
type CreateResourceByYamlReq struct {
	ClusterID int    `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID
	YAML      string `json:"yaml" binding:"required" comment:"YAML内容"`                       // YAML内容
	Namespace string `json:"namespace" binding:"required" comment:"命名空间"`                    // 命名空间
}

// UpdateResourceByYamlReq 通过YAML更新K8s资源的通用请求
type UpdateResourceByYamlReq struct {
	ClusterID int    `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID
	Namespace string `json:"namespace" binding:"required" comment:"命名空间"`                    // 命名空间
	Name      string `json:"name" binding:"required" comment:"资源名称"`                         // 资源名称
	YAML      string `json:"yaml" binding:"required" comment:"YAML内容"`                       // YAML内容
}

// ApplyResourceByYamlReq 应用YAML到K8s集群的请求
type ApplyResourceByYamlReq struct {
	ClusterID int    `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID
	YAML      string `json:"yaml" binding:"required" comment:"YAML内容"`                       // YAML内容
	DryRun    bool   `json:"dry_run" comment:"是否为试运行"`                                       // 是否为试运行
}

// ValidateYamlReq 验证YAML格式的请求
type ValidateYamlReq struct {
	YAML string `json:"yaml" binding:"required" comment:"YAML内容"` // YAML内容
}

// ConvertToYamlReq 将资源配置转换为YAML的请求
type ConvertToYamlReq struct {
	ClusterID int         `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID
	Config    interface{} `json:"config" binding:"required" comment:"资源配置信息"`                     // 资源配置信息
}

// YamlTemplateCreateReq 创建YAML模板请求
type YamlTemplateCreateReq struct {
	Name      string `json:"name" binding:"required,min=1,max=50" comment:"模板名称"`            // 模板名称
	UserID    int    `json:"user_id" comment:"创建者用户ID"`                                      // 创建者用户ID（从token中获取）
	Content   string `json:"content" binding:"required" comment:"YAML模板内容"`                  // YAML模板内容
	ClusterID int    `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID
}

// YamlTemplateUpdateReq 更新YAML模板请求
type YamlTemplateUpdateReq struct {
	ID        int    `json:"id" binding:"required" comment:"模板ID"`                           // 模板ID
	Name      string `json:"name" binding:"required,min=1,max=50" comment:"模板名称"`            // 模板名称
	UserID    int    `json:"user_id" comment:"创建者用户ID"`                                      // 创建者用户ID（从token中获取）
	Content   string `json:"content" binding:"required" comment:"YAML模板内容"`                  // YAML模板内容
	ClusterID int    `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID
}

// YamlTemplateCheckReq 检查YAML模板请求
type YamlTemplateCheckReq struct {
	Name      string `json:"name" binding:"required,min=1,max=50" comment:"模板名称"`            // 模板名称
	Content   string `json:"content" binding:"required" comment:"YAML模板内容"`                  // YAML模板内容
	ClusterID int    `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID
}

// YamlTemplateListReq 获取YAML模板列表请求
type YamlTemplateListReq struct {
	ListReq
	ClusterID int `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID
}

// YamlTemplateDeleteReq 删除YAML模板请求
type YamlTemplateDeleteReq struct {
	ID        int `json:"id" binding:"required" comment:"模板ID"`                           // 模板ID
	ClusterID int `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID
	UserID    int `json:"user_id" comment:"用户ID"`                                         // 用户ID
}

// YamlTaskCreateReq 创建YAML任务请求
type YamlTaskCreateReq struct {
	Name       string     `json:"name" binding:"required,min=1,max=50" comment:"任务名称"`            // 任务名称
	UserID     int        `json:"user_id" comment:"创建者用户ID"`                                      // 创建者用户ID
	TemplateID int        `json:"template_id" binding:"required" comment:"模板ID"`                  // 模板ID
	ClusterID  int        `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID
	Variables  StringList `json:"variables" comment:"YAML变量"`                                     // YAML变量
}

// YamlTaskListReq 获取YAML任务列表请求
type YamlTaskListReq struct {
	ListReq
	ClusterID  int    `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID
	TemplateID int    `json:"template_id" form:"template_id" comment:"模板ID"`                  // 模板ID
	Status     string `json:"status" form:"status" comment:"任务状态"`                            // 任务状态
	Name       string `json:"name" form:"name" comment:"任务名称"`                                // 任务名称过滤
}

// YamlTaskExecuteReq 执行YAML任务请求
type YamlTaskExecuteReq struct {
	ID        int  `json:"id" binding:"required" comment:"任务ID"`                           // 任务ID
	DryRun    bool `json:"dry_run" comment:"是否为试运行"`                                       // 是否为试运行
	UserID    int  `json:"user_id" comment:"用户ID"`                                         // 用户ID
	ClusterID int  `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID
}

// YamlTaskUpdateReq 更新YAML任务请求
type YamlTaskUpdateReq struct {
	ID         int        `json:"id" binding:"required" comment:"任务ID"`
	Name       string     `json:"name" binding:"required,min=1,max=255" comment:"YAML任务名称"`
	UserID     int        `json:"user_id" comment:"创建者用户ID"`
	TemplateID int        `json:"template_id" comment:"关联的模板ID"`
	ClusterID  int        `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"`
	Variables  StringList `json:"variables" comment:"yaml变量，格式k=v,k=v"`
}

// YamlTaskDeleteReq 删除YAML任务请求
type YamlTaskDeleteReq struct {
	ID        int `json:"id" binding:"required" comment:"任务ID"`                           // 任务ID
	ClusterID int `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID
}

// YamlTemplateDetailReq 获取YAML模板详情请求
type YamlTemplateDetailReq struct {
	ID        int `json:"id" binding:"required" comment:"模板ID"`                           // 模板ID
	ClusterID int `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID
}
