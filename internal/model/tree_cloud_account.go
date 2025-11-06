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

// CloudAccountStatus 云账户状态
type CloudAccountStatus int8

const (
	CloudAccountEnabled  CloudAccountStatus = iota + 1 // 启用
	CloudAccountDisabled                               // 禁用
)

// CloudAccount 云账户管理
type CloudAccount struct {
	Model
	Name           string             `json:"name" gorm:"type:varchar(100);not null;uniqueIndex:idx_name_provider;comment:账户名称"`
	Provider       CloudProvider      `json:"provider" gorm:"type:tinyint(1);not null;uniqueIndex:idx_name_provider;comment:云厂商类型;default:1"`
	AccessKey      string             `json:"-" gorm:"type:varchar(500);not null;comment:访问密钥ID,加密存储"`
	SecretKey      string             `json:"-" gorm:"type:varchar(500);not null;comment:访问密钥Secret,加密存储"`
	AccountID      string             `json:"account_id" gorm:"type:varchar(100);index;comment:云账号ID"`
	AccountName    string             `json:"account_name" gorm:"type:varchar(100);comment:云账号名称"`
	AccountAlias   string             `json:"account_alias" gorm:"type:varchar(100);comment:账号别名"`
	Description    string             `json:"description" gorm:"type:varchar(500);comment:账户描述"`
	Status         CloudAccountStatus `json:"status" gorm:"type:tinyint(1);not null;index;comment:账户状态,1:启用,2:禁用;default:1"`
	CreateUserID   int                `json:"create_user_id" gorm:"not null;comment:创建者ID;default:0"`
	CreateUserName string             `json:"create_user_name" gorm:"type:varchar(100);not null;comment:创建者姓名"`
	// 关联关系
	CloudResources []*TreeCloudResource  `json:"cloud_resources,omitempty" gorm:"foreignKey:CloudAccountID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;comment:云账户资源"`
	Regions        []*CloudAccountRegion `json:"regions,omitempty" gorm:"foreignKey:CloudAccountID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;comment:云账户区域"`
}

func (c *CloudAccount) TableName() string {
	return "cl_tree_cloud_account"
}

// GetCloudAccountListReq 获取云账户列表请求
type GetCloudAccountListReq struct {
	ListReq
	Provider CloudProvider      `json:"provider" form:"provider" binding:"omitempty,oneof=1 2 3 4 5 6"`                // 云厂商筛选
	Status   CloudAccountStatus `json:"status" form:"status" binding:"omitempty,oneof=1 2"`                            // 状态筛选
	OrderBy  string             `json:"order_by" form:"order_by" binding:"omitempty,oneof=created_at updated_at name"` // 排序字段
	Order    string             `json:"order" form:"order" binding:"omitempty,oneof=asc desc"`                         // 排序方向
}

// GetCloudAccountDetailReq 获取云账户详情请求
type GetCloudAccountDetailReq struct {
	ID int `json:"id" form:"id" binding:"required,gt=0"`
}

// CreateCloudAccountReq 创建云账户请求
type CreateCloudAccountReq struct {
	Name         string                         `json:"name" binding:"required,min=2,max=100"`         // 账户名称
	Provider     CloudProvider                  `json:"provider" binding:"required,oneof=1 2 3 4 5 6"` // 云厂商类型
	AccessKey    string                         `json:"access_key" binding:"required,min=10,max=500"`  // 访问密钥ID
	SecretKey    string                         `json:"secret_key" binding:"required,min=10,max=500"`  // 访问密钥Secret
	AccountID    string                         `json:"account_id" binding:"omitempty,max=100"`        // 云账号ID
	AccountName  string                         `json:"account_name" binding:"omitempty,max=100"`      // 云账号名称
	AccountAlias string                         `json:"account_alias" binding:"omitempty,max=100"`     // 账号别名
	Description  string                         `json:"description" binding:"omitempty,max=500"`       // 账户描述
	Regions      []CreateCloudAccountRegionItem `json:"regions" binding:"required,min=1,dive"`         // 区域配置（至少一个）
}

// UpdateCloudAccountReq 更新云账户请求
type UpdateCloudAccountReq struct {
	ID           int                            `json:"id" binding:"required,gt=0"`                    // 账户ID
	Name         string                         `json:"name" binding:"omitempty,min=2,max=100"`        // 账户名称
	AccessKey    string                         `json:"access_key" binding:"omitempty,min=10,max=500"` // 访问密钥ID
	SecretKey    string                         `json:"secret_key" binding:"omitempty,min=10,max=500"` // 访问密钥Secret
	AccountID    string                         `json:"account_id" binding:"omitempty,max=100"`        // 云账号ID
	AccountName  string                         `json:"account_name" binding:"omitempty,max=100"`      // 云账号名称
	AccountAlias string                         `json:"account_alias" binding:"omitempty,max=100"`     // 账号别名
	Description  string                         `json:"description" binding:"omitempty,max=500"`       // 账户描述
	Regions      []CreateCloudAccountRegionItem `json:"regions" binding:"omitempty,min=1,dive"`        // 区域配置（可选，如果提供则至少一个）
}

// DeleteCloudAccountReq 删除云账户请求
type DeleteCloudAccountReq struct {
	ID int `json:"id" binding:"required,gt=0"`
}

// UpdateCloudAccountStatusReq 更新云账户状态请求
type UpdateCloudAccountStatusReq struct {
	ID     int                `json:"id" binding:"required,gt=0"`
	Status CloudAccountStatus `json:"status" binding:"required,oneof=1 2"`
}

// VerifyCloudAccountReq 验证云账户凭证请求
type VerifyCloudAccountReq struct {
	ID int `json:"id" binding:"required,gt=0"` // 账户ID
}

// BatchDeleteCloudAccountReq 批量删除云账户请求
type BatchDeleteCloudAccountReq struct {
	IDs []int `json:"ids" binding:"required,min=1,max=100,dive,gt=0"` // 账户ID列表
}

// BatchUpdateCloudAccountStatusReq 批量更新云账户状态请求
type BatchUpdateCloudAccountStatusReq struct {
	IDs    []int              `json:"ids" binding:"required,min=1,max=100,dive,gt=0"` // 账户ID列表
	Status CloudAccountStatus `json:"status" binding:"required,oneof=1 2"`            // 目标状态
}

// ImportCloudAccountReq 导入云账户请求
type ImportCloudAccountReq struct {
	Accounts []CreateCloudAccountReq `json:"accounts" binding:"required,min=1,max=50,dive"` // 账户列表
}

// ExportCloudAccountReq 导出云账户请求
type ExportCloudAccountReq struct {
	IDs      []int         `json:"ids" binding:"omitempty,max=100,dive,gt=0"`      // 指定账户ID，为空则导出全部
	Provider CloudProvider `json:"provider" binding:"omitempty,oneof=1 2 3 4 5 6"` // 按云厂商过滤
	Format   string        `json:"format" binding:"omitempty,oneof=json csv"`      // 导出格式：json或csv
}

// ImportCloudAccountResp 导入云账户响应
type ImportCloudAccountResp struct {
	SuccessCount int      `json:"success_count"` // 成功数量
	FailedCount  int      `json:"failed_count"`  // 失败数量
	FailedItems  []string `json:"failed_items"`  // 失败的账户名称列表
	Message      string   `json:"message"`       // 提示信息
}

// ExportRegion 导出区域信息
type ExportRegion struct {
	Region      string `json:"region"`      // 区域代码
	RegionName  string `json:"region_name"` // 区域名称
	IsDefault   bool   `json:"is_default"`  // 是否为默认区域
	Description string `json:"description"` // 区域描述
}

// ExportAccount 导出账户信息（不包含敏感信息）
type ExportAccount struct {
	ID           int            `json:"id"`            // 账户ID
	Name         string         `json:"name"`          // 账户名称
	Provider     CloudProvider  `json:"provider"`      // 云厂商类型
	ProviderName string         `json:"provider_name"` // 云厂商名称
	AccountID    string         `json:"account_id"`    // 云账号ID
	AccountName  string         `json:"account_name"`  // 云账号名称
	AccountAlias string         `json:"account_alias"` // 账号别名
	Description  string         `json:"description"`   // 账户描述
	Status       int8           `json:"status"`        // 账户状态
	Regions      []ExportRegion `json:"regions"`       // 区域列表
	CreatedAt    string         `json:"created_at"`    // 创建时间
}
