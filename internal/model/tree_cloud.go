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

// CloudProvider 云厂商类型枚举
type CloudProvider string

const (
	CloudProviderAliyun CloudProvider = "aliyun" // 阿里云
	CloudProviderLocal  CloudProvider = "local"  // 本地环境
	// CloudProviderHuawei  CloudProvider = "huawei"  // 华为云
	// CloudProviderTencent CloudProvider = "tencent" // 腾讯云
	// CloudProviderAWS     CloudProvider = "aws"     // AWS
	// CloudProviderAzure   CloudProvider = "azure"   // Azure
	// CloudProviderGCP     CloudProvider = "gcp"     // Google Cloud
)

// CloudAccount 云账户信息
type CloudAccount struct {
	Model
	Name            string        `json:"name" gorm:"type:varchar(100);comment:账户名称"`
	Provider        CloudProvider `json:"provider" gorm:"type:varchar(50);comment:云厂商"`
	AccountId       string        `json:"accountId" gorm:"type:varchar(100);comment:账户ID"`
	AccessKey       string        `json:"accessKey" gorm:"type:varchar(100);comment:访问密钥ID"`
	EncryptedSecret string        `json:"encryptedSecret" gorm:"type:varchar(500);comment:加密的访问密钥"`
	Regions         StringList    `json:"regions" gorm:"type:varchar(500);comment:可用区域列表"`
	IsEnabled       bool          `json:"isEnabled" gorm:"comment:是否启用"`
	LastSyncTime    time.Time     `json:"lastSyncTime" gorm:"comment:最后同步时间"`
	Description     string        `json:"description" gorm:"type:text;comment:账户描述"`
}

// CreateCloudAccountReq 创建云账号请求
type CreateCloudAccountReq struct {
	Name        string        `json:"name" binding:"required" validate:"max=100"`
	Provider    CloudProvider `json:"provider" binding:"required"`
	AccountId   string        `json:"accountId" binding:"required" validate:"max=100"`
	AccessKey   string        `json:"accessKey" binding:"required" validate:"max=100"`
	SecretKey   string        `json:"secretKey" binding:"required"`
	Regions     []string      `json:"regions"`
	IsEnabled   bool          `json:"isEnabled"`
	Description string        `json:"description" validate:"max=500"`
}

// UpdateCloudAccountReq 更新云账号请求
type UpdateCloudAccountReq struct {
	ID          int           `json:"id"`
	Name        string        `json:"name" validate:"max=100"`
	Provider    CloudProvider `json:"provider"`
	AccountId   string        `json:"accountId" validate:"max=100"`
	AccessKey   string        `json:"accessKey" validate:"max=100"`
	SecretKey   string        `json:"secretKey"`
	Regions     []string      `json:"regions"`
	IsEnabled   bool          `json:"isEnabled"`
	Description string        `json:"description" validate:"max=500"`
}

// GetCloudAccountReq 获取云账号详情请求
type GetCloudAccountReq struct {
	ID int `json:"id" uri:"id" binding:"required"`
}

// ListCloudAccountsReq 获取云账号列表请求
type ListCloudAccountsReq struct {
	Page     int           `json:"page" form:"page"`
	PageSize int           `json:"pageSize" form:"pageSize"`
	Name     string        `json:"name" form:"name"`
	Provider CloudProvider `json:"provider" form:"provider"`
	Enabled  bool          `json:"enabled" form:"enabled"`
}

// TestCloudAccountReq 测试云账号连接请求
type TestCloudAccountReq struct {
	ID int `json:"id" uri:"id" binding:"required"`
}

// SyncCloudReq 同步云资源请求
type SyncCloudReq struct {
	AccountIds   []int    `json:"accountIds"`   // 要同步的账号ID列表，为空则同步所有启用的账号
	ResourceType string   `json:"resourceType"` // 资源类型：ecs,vpc,sg等，为空则同步所有类型
	Regions      []string `json:"regions"`      // 要同步的区域列表，为空则同步所有区域
	Force        bool     `json:"force"`        // 是否强制重新同步
}
