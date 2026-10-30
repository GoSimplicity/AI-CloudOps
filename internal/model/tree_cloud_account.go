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

import "time"

type CloudAccountStatus int8

const (
	CloudAccountEnabled  CloudAccountStatus = iota + 1 // 启用
	CloudAccountDisabled                               // 禁用
)

// CloudAccount 云账户管理
type CloudAccount struct {
	Model
	Name                      string             `json:"name" gorm:"type:varchar(100);not null;uniqueIndex:idx_name_provider;comment:账户名称"`
	Provider                  CloudProvider      `json:"provider" gorm:"type:tinyint(1);not null;uniqueIndex:idx_name_provider;comment:云厂商类型;default:1"`
	AccessKey                 string             `json:"-" gorm:"type:varchar(500);not null;comment:访问密钥ID,加密存储"`
	SecretKey                 string             `json:"-" gorm:"type:varchar(500);not null;comment:访问密钥Secret,加密存储"`
	CredentialType            string             `json:"credential_type" gorm:"type:varchar(20);not null;default:long_term"`
	SessionToken              string             `json:"-" gorm:"type:text"`
	ProviderCredentials       string             `json:"-" gorm:"type:text"`
	CredentialExpiresAt       *time.Time         `json:"credential_expires_at,omitempty" gorm:"column:credential_expires_at"`
	CredentialVersion         uint64             `json:"-" gorm:"not null;default:0"`
	ExpectedCredentialVersion *uint64            `json:"-" gorm:"-"`
	AccountID                 string             `json:"account_id" gorm:"type:varchar(100);index;comment:云账号ID"`
	AccountName               string             `json:"account_name" gorm:"type:varchar(100);comment:云账号名称"`
	AccountAlias              string             `json:"account_alias" gorm:"type:varchar(100);comment:账号别名"`
	Description               string             `json:"description" gorm:"type:varchar(500);comment:账户描述"`
	Status                    CloudAccountStatus `json:"status" gorm:"type:tinyint(1);not null;index;comment:账户状态,1:启用,2:禁用;default:1"`
	CreateUserID              int                `json:"create_user_id" gorm:"not null;comment:创建者ID;default:0"`
	CreateUserName            string             `json:"create_user_name" gorm:"type:varchar(100);not null;comment:创建者姓名"`
	// 关联关系
	CloudResources []*TreeCloudResource  `json:"cloud_resources,omitempty" gorm:"foreignKey:CloudAccountID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;comment:云账户资源"`
	Regions        []*CloudAccountRegion `json:"regions,omitempty" gorm:"foreignKey:CloudAccountID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;comment:云账户区域"`
}

func (c *CloudAccount) TableName() string {
	return "cl_tree_cloud_account"
}

type CloudAccountRegionStatus int8

const (
	CloudAccountRegionEnabled  CloudAccountRegionStatus = iota + 1 // 启用
	CloudAccountRegionDisabled                                     // 禁用
)

// CloudAccountRegion 云账号区域关联表
type CloudAccountRegion struct {
	Model
	CloudAccountID int                      `json:"cloud_account_id" gorm:"not null;comment:云账户ID;index:idx_account_region,unique"`
	CloudAccount   *CloudAccount            `json:"cloud_account,omitempty" gorm:"foreignKey:CloudAccountID"`
	Region         string                   `json:"region" gorm:"type:varchar(50);not null;comment:区域,如cn-hangzhou;index:idx_account_region,unique"`
	RegionName     string                   `json:"region_name" gorm:"type:varchar(100);comment:区域名称,如华东1(杭州)"`
	ProjectID      *string                  `json:"project_id,omitempty" gorm:"type:varchar(64);comment:该地域对应的项目ID"`
	Status         CloudAccountRegionStatus `json:"status" gorm:"type:tinyint(1);not null;comment:区域状态,1:启用,2:禁用;default:1"`
	IsDefault      bool                     `json:"is_default" gorm:"comment:是否为默认区域;default:false"`
	Description    string                   `json:"description" gorm:"type:text;comment:区域描述"`
	LastSyncTime   *time.Time               `json:"last_sync_time" gorm:"type:datetime;comment:最后同步时间"`
	CreateUserID   int                      `json:"create_user_id" gorm:"comment:创建者ID;default:0"`
	CreateUserName string                   `json:"create_user_name" gorm:"type:varchar(100);comment:创建者姓名"`
}

func (c *CloudAccountRegion) TableName() string {
	return "cl_tree_cloud_account_region"
}

type GetCloudAccountListReq struct {
	ListReq
	Provider CloudProvider      `json:"provider" form:"provider" binding:"omitempty,oneof=1 2 3 4 5 6"`                // 云厂商筛选
	Status   CloudAccountStatus `json:"status" form:"status" binding:"omitempty,oneof=1 2"`                            // 状态筛选
	OrderBy  string             `json:"order_by" form:"order_by" binding:"omitempty,oneof=created_at updated_at name"` // 排序字段
	Order    string             `json:"order" form:"order" binding:"omitempty,oneof=asc desc"`                         // 排序方向
}

type GetCloudAccountDetailReq struct {
	ID int `json:"id" form:"id" binding:"required,gt=0"`
}

// ProviderCredentialInput is a write-only, complete provider credential tuple.
type ProviderCredentialInput struct {
	TenantID           string `json:"tenant_id,omitempty"`
	ClientID           string `json:"client_id,omitempty"`
	ClientSecret       string `json:"client_secret,omitempty"`
	SubscriptionID     string `json:"subscription_id,omitempty"`
	ProjectID          string `json:"project_id,omitempty"`
	ServiceAccountJSON string `json:"service_account_json,omitempty"`
}

func (ProviderCredentialInput) String() string   { return "[REDACTED]" }
func (ProviderCredentialInput) GoString() string { return "[REDACTED]" }

type CreateCloudAccountReq struct {
	Status              *int                           `json:"status" binding:"omitempty,oneof=1 2"` // 缺省启用；支持先保存停用账户
	ProviderCredentials *ProviderCredentialInput       `json:"provider_credentials,omitempty"`
	Name                string                         `json:"name" binding:"required,min=2,max=100"`         // 账户名称
	Provider            CloudProvider                  `json:"provider" binding:"required,oneof=1 2 3 4 5 6"` // 云厂商类型
	AccessKey           string                         `json:"access_key" binding:"omitempty,min=10,max=500"` // 访问密钥ID
	SecretKey           string                         `json:"secret_key" binding:"omitempty,min=10,max=500"` // 访问密钥Secret
	CredentialType      string                         `json:"credential_type" binding:"omitempty,oneof=long_term temporary"`
	SessionToken        string                         `json:"session_token" binding:"omitempty,max=8192"`
	CredentialExpiresAt *time.Time                     `json:"credential_expires_at"`
	AccountID           string                         `json:"account_id" binding:"omitempty,max=100"`    // 云账号ID
	AccountName         string                         `json:"account_name" binding:"omitempty,max=100"`  // 云账号名称
	AccountAlias        string                         `json:"account_alias" binding:"omitempty,max=100"` // 账号别名
	Description         string                         `json:"description" binding:"omitempty,max=500"`   // 账户描述
	Regions             []CreateCloudAccountRegionItem `json:"regions" binding:"required,min=1,dive"`     // 区域配置（至少一个）
}

type UpdateCloudAccountReq struct {
	ProviderCredentials *ProviderCredentialInput       `json:"provider_credentials,omitempty"`
	ID                  int                            `json:"id" binding:"required,gt=0"`                    // 账户ID
	Name                string                         `json:"name" binding:"omitempty,min=2,max=100"`        // 账户名称
	AccessKey           string                         `json:"access_key" binding:"omitempty,min=10,max=500"` // 访问密钥ID
	SecretKey           string                         `json:"secret_key" binding:"omitempty,min=10,max=500"` // 访问密钥Secret
	CredentialType      string                         `json:"credential_type" binding:"omitempty,oneof=long_term temporary"`
	SessionToken        string                         `json:"session_token" binding:"omitempty,max=8192"`
	CredentialExpiresAt *time.Time                     `json:"credential_expires_at"`
	AccountID           string                         `json:"account_id" binding:"omitempty,max=100"`    // 云账号ID
	AccountName         string                         `json:"account_name" binding:"omitempty,max=100"`  // 云账号名称
	AccountAlias        string                         `json:"account_alias" binding:"omitempty,max=100"` // 账号别名
	Description         string                         `json:"description" binding:"omitempty,max=500"`   // 账户描述
	Regions             []CreateCloudAccountRegionItem `json:"regions" binding:"omitempty,min=1,dive"`    // 区域配置（可选，如果提供则至少一个）
}

type DeleteCloudAccountReq struct {
	ID int `json:"id" binding:"required,gt=0"`
}

type UpdateCloudAccountStatusReq struct {
	ID     int                `json:"id" binding:"required,gt=0"`
	Status CloudAccountStatus `json:"status" binding:"required,oneof=1 2"`
}

type VerifyCloudAccountReq struct {
	ID int `json:"id" binding:"required,gt=0"` // 账户ID
}

type BatchDeleteCloudAccountReq struct {
	IDs []int `json:"ids" binding:"required,min=1,max=100,dive,gt=0"` // 账户ID列表
}

type BatchUpdateCloudAccountStatusReq struct {
	IDs    []int              `json:"ids" binding:"required,min=1,max=100,dive,gt=0"` // 账户ID列表
	Status CloudAccountStatus `json:"status" binding:"required,oneof=1 2"`            // 目标状态
}

type ImportCloudAccountReq struct {
	Accounts []CreateCloudAccountReq `json:"accounts" binding:"required,min=1,max=50,dive"` // 账户列表
}

type ExportCloudAccountReq struct {
	IDs      []int         `json:"ids" binding:"omitempty,max=100,dive,gt=0"`      // 指定账户ID，为空则导出全部
	Provider CloudProvider `json:"provider" binding:"omitempty,oneof=1 2 3 4 5 6"` // 按云厂商过滤
	Format   string        `json:"format" binding:"omitempty,oneof=json csv"`      // 导出格式：json或csv
}

type ImportCloudAccountResp struct {
	SuccessCount int      `json:"success_count"` // 成功数量
	FailedCount  int      `json:"failed_count"`  // 失败数量
	FailedItems  []string `json:"failed_items"`  // 失败的账户名称列表
	Message      string   `json:"message"`       // 提示信息
}

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

type GetCloudAccountRegionListReq struct {
	ListReq
	CloudAccountID int                      `json:"cloud_account_id" form:"cloud_account_id" binding:"omitempty,gt=0"`
	Region         string                   `json:"region" form:"region"`
	Status         CloudAccountRegionStatus `json:"status" form:"status" binding:"omitempty,oneof=1 2"`
}

type CreateCloudAccountRegionReq struct {
	ProjectID      *string `json:"project_id,omitempty" binding:"omitempty,max=64"`
	CloudAccountID int     `json:"cloud_account_id" binding:"required,gt=0"`
	Region         string  `json:"region" binding:"required"`
	RegionName     string  `json:"region_name"`
	IsDefault      bool    `json:"is_default"`
	Description    string  `json:"description"`
	CreateUserID   int     `json:"create_user_id"`
	CreateUserName string  `json:"create_user_name"`
}

type UpdateCloudAccountRegionReq struct {
	ID          int    `json:"id" binding:"required,gt=0"`
	RegionName  string `json:"region_name"`
	IsDefault   bool   `json:"is_default"`
	Description string `json:"description"`
}

type DeleteCloudAccountRegionReq struct {
	ID int `json:"id" binding:"required,gt=0"`
}

type UpdateCloudAccountRegionStatusReq struct {
	ID     int                      `json:"id" binding:"required,gt=0"`
	Status CloudAccountRegionStatus `json:"status" binding:"required,oneof=1 2"`
}

// BatchCreateCloudAccountRegionReq 批量创建云账号区域关联请求
type BatchCreateCloudAccountRegionReq struct {
	CloudAccountID int                            `json:"cloud_account_id" binding:"required,gt=0"`
	Regions        []CreateCloudAccountRegionItem `json:"regions" binding:"required,min=1"`
	CreateUserID   int                            `json:"create_user_id"`
	CreateUserName string                         `json:"create_user_name"`
}

type CreateCloudAccountRegionItem struct {
	Region      string  `json:"region" binding:"required"` // 区域,如cn-hangzhou
	RegionName  string  `json:"region_name"`               // 区域名称,如华东1(杭州)
	IsDefault   bool    `json:"is_default"`                // 是否为默认区域
	Description string  `json:"description"`               // 区域描述
	ProjectID   *string `json:"project_id,omitempty" binding:"omitempty,max=64"`
}

type GetAvailableRegionsReq struct {
	CloudAccountID int           `json:"cloud_account_id" form:"cloud_account_id" binding:"omitempty,gt=0"`
	Provider       CloudProvider `json:"provider" form:"provider" binding:"omitempty,oneof=1 2 3 4 5 6"`
	AccessKey      string        `json:"access_key" form:"access_key"`
	SecretKey      string        `json:"secret_key" form:"secret_key"`
	SessionToken   string        `json:"-" form:"session_token"`
}

type AvailableRegion struct {
	Region     string `json:"region"`      // 区域代码，如cn-hangzhou
	RegionName string `json:"region_name"` // 区域名称，如华东1(杭州)
	Available  bool   `json:"available"`   // 是否可用
}

type GetAvailableRegionsResp struct {
	Regions []AvailableRegion `json:"regions"`
}
