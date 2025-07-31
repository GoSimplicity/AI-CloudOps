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

import (
	"gorm.io/datatypes"
	"time"
)

// 模板状态常量
const (
	TemplateStatusDisabled int8 = 1 // 禁用
	TemplateStatusEnabled  int8 = 2 // 启用
)

// 模板可见性常量
const (
	TemplateVisibilityPrivate = "private" // 私有
	TemplateVisibilityPublic  = "public"  // 公开
	TemplateVisibilityShared  = "shared"  // 共享
)

// WorkorderTemplateDefaultValues 工单模板默认值结构
type WorkorderTemplateDefaultValues struct {
	FormFields    map[string]interface{} `json:"form_fields"`    // 表单字段默认值
	AssigneeIDs   []int                  `json:"assignee_ids"`   // 默认处理人ID列表
	Priority      int8                   `json:"priority"`       // 默认优先级
	DueHours      *int                   `json:"due_hours"`      // 默认处理时限(小时)
	CategoryID    *int                   `json:"category_id"`    // 默认分类ID
	Tags          []string               `json:"tags"`           // 默认标签
	NotifyUsers   []int                  `json:"notify_users"`   // 默认通知用户
	AutoApprove   bool                   `json:"auto_approve"`   // 是否自动审批
	RequireFiles  bool                   `json:"require_files"`  // 是否需要附件
	AllowReopen   bool                   `json:"allow_reopen"`   // 是否允许重新打开
	Props         map[string]interface{} `json:"props"`          // 其他属性
}

// WorkorderTemplate 工单模板实体
type WorkorderTemplate struct {
	Model
	Name          string                          `json:"name" gorm:"column:name;type:varchar(200);not null;index;comment:模板名称"`
	Description   string                          `json:"description" gorm:"column:description;type:varchar(1000);comment:模板描述"`
	ProcessID     int                             `json:"process_id" gorm:"column:process_id;not null;index;comment:关联的流程ID"`
	FormDesignID  int                             `json:"form_design_id" gorm:"column:form_design_id;not null;index;comment:关联的表单设计ID"`
	DefaultValues datatypes.JSON                 `json:"default_values" gorm:"column:default_values;type:json;comment:默认值JSON"`
	Icon          string                          `json:"icon" gorm:"column:icon;type:varchar(500);comment:模板图标URL"`
	Color         string                          `json:"color" gorm:"column:color;type:varchar(20);default:'#1890ff';comment:模板颜色"`
	Status        int8                            `json:"status" gorm:"column:status;not null;default:1;index;comment:状态：0-禁用，1-启用"`
	SortOrder     int                             `json:"sort_order" gorm:"column:sort_order;not null;default:0;index;comment:排序顺序"`
	CategoryID    *int                            `json:"category_id" gorm:"column:category_id;index;comment:分类ID"`
	CreatorID     int                             `json:"creator_id" gorm:"column:creator_id;not null;index;comment:创建人ID"`
	CreatorName   string                          `json:"creator_name" gorm:"-"`
	Tags          StringList                      `json:"tags" gorm:"column:tags;comment:标签"`
	UseCount      int                             `json:"use_count" gorm:"column:use_count;not null;default:0;comment:使用次数"`
	Visibility    string                          `json:"visibility" gorm:"column:visibility;type:varchar(20);not null;default:'public';comment:可见性"`
	IsRecommended bool                            `json:"is_recommended" gorm:"column:is_recommended;not null;default:false;comment:是否推荐"`
	ValidFrom     *time.Time                      `json:"valid_from" gorm:"column:valid_from;comment:有效期开始时间"`
	ValidTo       *time.Time                      `json:"valid_to" gorm:"column:valid_to;comment:有效期结束时间"`

	// 关联信息（不存储到数据库）
	ProcessName    string `json:"process_name,omitempty" gorm:"-"`
	FormDesignName string `json:"form_design_name,omitempty" gorm:"-"`
	CategoryName   string `json:"category_name,omitempty" gorm:"-"`
}

// TableName 指定工单模板表名
func (WorkorderTemplate) TableName() string {
	return "cl_workorder_template"
}

// CreateWorkorderTemplateReq 创建工单模板请求
type CreateWorkorderTemplateReq struct {
	Name          string                          `json:"name" binding:"required,min=1,max=200"`
	Description   string                          `json:"description" binding:"omitempty,max=1000"`
	ProcessID     int                             `json:"process_id" binding:"required,min=1"`
	FormDesignID  int                             `json:"form_design_id" binding:"required,min=1"`
	DefaultValues WorkorderTemplateDefaultValues `json:"default_values"`
	Icon          string                          `json:"icon" binding:"omitempty,max=500"`
	Color         string                          `json:"color" binding:"omitempty,max=20"`
	CategoryID    *int                            `json:"category_id" binding:"omitempty,min=1"`
	SortOrder     int                             `json:"sort_order" binding:"omitempty,min=0"`
	Tags          []string                        `json:"tags" binding:"omitempty"`
	Visibility    string                          `json:"visibility" binding:"omitempty,oneof=private public shared"`
	IsRecommended bool                            `json:"is_recommended"`
	ValidFrom     *time.Time                      `json:"valid_from"`
	ValidTo       *time.Time                      `json:"valid_to"`
}

// UpdateWorkorderTemplateReq 更新工单模板请求
type UpdateWorkorderTemplateReq struct {
	ID            int                             `json:"id" binding:"required,min=1"`
	Name          string                          `json:"name" binding:"required,min=1,max=200"`
	Description   string                          `json:"description" binding:"omitempty,max=1000"`
	ProcessID     int                             `json:"process_id" binding:"required,min=1"`
	FormDesignID  int                             `json:"form_design_id" binding:"required,min=1"`
	DefaultValues WorkorderTemplateDefaultValues `json:"default_values"`
	Icon          string                          `json:"icon" binding:"omitempty,max=500"`
	Color         string                          `json:"color" binding:"omitempty,max=20"`
	CategoryID    *int                            `json:"category_id" binding:"omitempty,min=1"`
	SortOrder     int                             `json:"sort_order" binding:"omitempty,min=0"`
	Status        int8                            `json:"status" binding:"required,oneof=0 1"`
	Tags          []string                        `json:"tags" binding:"omitempty"`
	Visibility    string                          `json:"visibility" binding:"omitempty,oneof=private public shared"`
	IsRecommended bool                            `json:"is_recommended"`
	ValidFrom     *time.Time                      `json:"valid_from"`
	ValidTo       *time.Time                      `json:"valid_to"`
}

// DeleteWorkorderTemplateReq 删除工单模板请求
type DeleteWorkorderTemplateReq struct {
	ID int `json:"id" form:"id" binding:"required,min=1"`
}

// DetailWorkorderTemplateReq 获取工单模板详情请求
type DetailWorkorderTemplateReq struct {
	ID int `json:"id" form:"id" binding:"required,min=1"`
}

// ListWorkorderTemplateReq 工单模板列表请求
type ListWorkorderTemplateReq struct {
	ListReq
	CategoryID    *int    `json:"category_id" form:"category_id" binding:"omitempty,min=1"`
	ProcessID     *int    `json:"process_id" form:"process_id" binding:"omitempty,min=1"`
	FormDesignID  *int    `json:"form_design_id" form:"form_design_id" binding:"omitempty,min=1"`
	Status        *int8   `json:"status" form:"status" binding:"omitempty,oneof=0 1"`
	Visibility    *string `json:"visibility" form:"visibility" binding:"omitempty,oneof=private public shared"`
	IsRecommended *bool   `json:"is_recommended" form:"is_recommended"`
	Tags          []string `json:"tags" form:"tags"`
}

// CloneWorkorderTemplateReq 克隆工单模板请求
type CloneWorkorderTemplateReq struct {
	ID   int    `json:"id" binding:"required,min=1"`
	Name string `json:"name" binding:"required,min=1,max=200"`
}

// SortWorkorderTemplateReq 排序工单模板请求
type SortWorkorderTemplateReq struct {
	Items []TemplateSortItem `json:"items" binding:"required,min=1"`
}

// TemplateSortItem 模板排序项
type TemplateSortItem struct {
	ID        int `json:"id" binding:"required,min=1"`
	SortOrder int `json:"sort_order" binding:"required,min=0"`
}

// BatchUpdateTemplateStatusReq 批量更新模板状态请求
type BatchUpdateTemplateStatusReq struct {
	IDs    []int `json:"ids" binding:"required,min=1,dive,min=1"`
	Status int8  `json:"status" binding:"required,oneof=0 1"`
}

// BatchUpdateTemplateVisibilityReq 批量更新模板可见性请求
type BatchUpdateTemplateVisibilityReq struct {
	IDs        []int  `json:"ids" binding:"required,min=1,dive,min=1"`
	Visibility string `json:"visibility" binding:"required,oneof=private public shared"`
}

// SetRecommendedTemplateReq 设置推荐模板请求
type SetRecommendedTemplateReq struct {
	ID            int  `json:"id" binding:"required,min=1"`
	IsRecommended bool `json:"is_recommended"`
}

// PreviewWorkorderTemplateReq 预览工单模板请求
type PreviewWorkorderTemplateReq struct {
	ID int `json:"id" form:"id" binding:"required,min=1"`
}

// GetTemplatesByProcessReq 根据流程获取模板请求
type GetTemplatesByProcessReq struct {
	ProcessID int `json:"process_id" form:"process_id" binding:"required,min=1"`
}

// GetTemplatesByCategoryReq 根据分类获取模板请求
type GetTemplatesByCategoryReq struct {
	CategoryID int `json:"category_id" form:"category_id" binding:"required,min=1"`
}

// GetRecommendedTemplatesReq 获取推荐模板请求
type GetRecommendedTemplatesReq struct {
	ListReq
	CategoryID *int `json:"category_id" form:"category_id" binding:"omitempty,min=1"`
}

// GetPopularTemplatesReq 获取热门模板请求
type GetPopularTemplatesReq struct {
	ListReq
	CategoryID *int `json:"category_id" form:"category_id" binding:"omitempty,min=1"`
	Days       int  `json:"days" form:"days" binding:"omitempty,min=1,max=365"`
}

// WorkorderTemplateStatistics 工单模板统计
type WorkorderTemplateStatistics struct {
	EnabledCount      int64 `json:"enabled_count"`      // 启用数量
	DisabledCount     int64 `json:"disabled_count"`     // 禁用数量
	RecommendedCount  int64 `json:"recommended_count"`  // 推荐数量
	PublicCount       int64 `json:"public_count"`       // 公开数量
	PrivateCount      int64 `json:"private_count"`      // 私有数量
	SharedCount       int64 `json:"shared_count"`       // 共享数量
	TotalUseCount     int64 `json:"total_use_count"`    // 总使用次数
	AvgUsePerTemplate int64 `json:"avg_use_per_template"` // 平均每模板使用次数
}

// WorkorderTemplateUsage 工单模板使用情况
type WorkorderTemplateUsage struct {
	TemplateID   int    `json:"template_id"`
	TemplateName string `json:"template_name"`
	UseCount     int    `json:"use_count"`
	LastUsed     int64  `json:"last_used"`
}

// WorkorderTemplateTree 工单模板树结构
type WorkorderTemplateTree struct {
	ID          int                     `json:"id"`
	Name        string                  `json:"name"`
	Description string                  `json:"description"`
	Icon        string                  `json:"icon"`
	Color       string                  `json:"color"`
	Status      int8                    `json:"status"`
	UseCount    int                     `json:"use_count"`
	CategoryID  *int                    `json:"category_id"`
	Children    []WorkorderTemplateTree `json:"children,omitempty"`
}
