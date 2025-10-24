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
	Name           string                `json:"name" gorm:"type:varchar(100);not null;comment:账户名称"`
	Provider       CloudProvider         `json:"provider" gorm:"type:tinyint(1);not null;comment:云厂商类型;default:1"`
	Region         string                `json:"region" gorm:"type:varchar(50);not null;comment:区域,如cn-hangzhou"`
	AccessKey      string                `json:"-" gorm:"type:varchar(500);not null;comment:访问密钥ID,加密存储"`
	SecretKey      string                `json:"-" gorm:"type:varchar(500);not null;comment:访问密钥Secret,加密存储"`
	AccountID      string                `json:"account_id" gorm:"type:varchar(100);comment:云账号ID"`
	AccountName    string                `json:"account_name" gorm:"type:varchar(100);comment:云账号名称"`
	AccountAlias   string                `json:"account_alias" gorm:"type:varchar(100);comment:账号别名"`
	Description    string                `json:"description" gorm:"type:text;comment:账户描述"`
	Status         CloudAccountStatus    `json:"status" gorm:"type:tinyint(1);not null;comment:账户状态,1:启用,2:禁用;default:1"`
	CreateUserID   int                   `json:"create_user_id" gorm:"comment:创建者ID;default:0"`
	CreateUserName string                `json:"create_user_name" gorm:"type:varchar(100);comment:创建者姓名"`
	CloudResources []*TreeCloudResource  `json:"cloud_resources,omitempty" gorm:"foreignKey:CloudAccountID"`
	Regions        []*CloudAccountRegion `json:"regions,omitempty" gorm:"foreignKey:CloudAccountID"`
}

func (c *CloudAccount) TableName() string {
	return "cl_cloud_account"
}

// GetCloudAccountListReq 获取云账户列表请求
type GetCloudAccountListReq struct {
	ListReq
	Provider CloudProvider      `json:"provider" form:"provider" binding:"omitempty,oneof=1 2 3 4 5 6"`
	Region   string             `json:"region" form:"region"`
	Status   CloudAccountStatus `json:"status" form:"status" binding:"omitempty,oneof=1 2"`
}

// GetCloudAccountDetailReq 获取云账户详情请求
type GetCloudAccountDetailReq struct {
	ID int `json:"id" form:"id" binding:"required,gt=0"`
}

// CreateCloudAccountReq 创建云账户请求
type CreateCloudAccountReq struct {
	Name           string                         `json:"name" binding:"required"`
	Provider       CloudProvider                  `json:"provider" binding:"required,oneof=1 2 3 4 5 6"`
	Region         string                         `json:"region" binding:"required"`
	AccessKey      string                         `json:"access_key" binding:"required"`
	SecretKey      string                         `json:"secret_key" binding:"required"`
	AccountID      string                         `json:"account_id"`
	AccountName    string                         `json:"account_name"`
	AccountAlias   string                         `json:"account_alias"`
	Description    string                         `json:"description"`
	CreateUserID   int                            `json:"create_user_id"`
	CreateUserName string                         `json:"create_user_name"`
	Regions        []CreateCloudAccountRegionItem `json:"regions,omitempty"`
}

// UpdateCloudAccountReq 更新云账户请求
type UpdateCloudAccountReq struct {
	ID           int    `json:"id" binding:"required,gt=0"`
	Name         string `json:"name"`
	AccessKey    string `json:"access_key"`
	SecretKey    string `json:"secret_key"`
	AccountID    string `json:"account_id"`
	AccountName  string `json:"account_name"`
	AccountAlias string `json:"account_alias"`
	Description  string `json:"description"`
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
	ID int `json:"id" binding:"required,gt=0"`
}
