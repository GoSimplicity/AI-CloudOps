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

// CloudAccountRegionStatus 云账号区域状态
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
	Status         CloudAccountRegionStatus `json:"status" gorm:"type:tinyint(1);not null;comment:区域状态,1:启用,2:禁用;default:1"`
	IsDefault      bool                     `json:"is_default" gorm:"comment:是否为默认区域;default:false"`
	Description    string                   `json:"description" gorm:"type:text;comment:区域描述"`
	LastSyncTime   *time.Time               `json:"last_sync_time" gorm:"type:datetime;comment:最后同步时间"`
	CreateUserID   int                      `json:"create_user_id" gorm:"comment:创建者ID;default:0"`
	CreateUserName string                   `json:"create_user_name" gorm:"type:varchar(100);comment:创建者姓名"`
}

func (c *CloudAccountRegion) TableName() string {
	return "cl_cloud_account_region"
}

// GetCloudAccountRegionListReq 获取云账号区域列表请求
type GetCloudAccountRegionListReq struct {
	ListReq
	CloudAccountID int                      `json:"cloud_account_id" form:"cloud_account_id" binding:"omitempty,gt=0"`
	Region         string                   `json:"region" form:"region"`
	Status         CloudAccountRegionStatus `json:"status" form:"status" binding:"omitempty,oneof=1 2"`
}

// CreateCloudAccountRegionReq 创建云账号区域关联请求
type CreateCloudAccountRegionReq struct {
	CloudAccountID int    `json:"cloud_account_id" binding:"required,gt=0"`
	Region         string `json:"region" binding:"required"`
	RegionName     string `json:"region_name"`
	IsDefault      bool   `json:"is_default"`
	Description    string `json:"description"`
	CreateUserID   int    `json:"create_user_id"`
	CreateUserName string `json:"create_user_name"`
}

// UpdateCloudAccountRegionReq 更新云账号区域关联请求
type UpdateCloudAccountRegionReq struct {
	ID          int    `json:"id" binding:"required,gt=0"`
	RegionName  string `json:"region_name"`
	IsDefault   bool   `json:"is_default"`
	Description string `json:"description"`
}

// DeleteCloudAccountRegionReq 删除云账号区域关联请求
type DeleteCloudAccountRegionReq struct {
	ID int `json:"id" binding:"required,gt=0"`
}

// UpdateCloudAccountRegionStatusReq 更新云账号区域状态请求
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

// CreateCloudAccountRegionItem 创建云账号区域项
type CreateCloudAccountRegionItem struct {
	Region      string `json:"region" binding:"required"` // 区域,如cn-hangzhou
	RegionName  string `json:"region_name"`               // 区域名称,如华东1(杭州)
	IsDefault   bool   `json:"is_default"`                // 是否为默认区域
	Description string `json:"description"`               // 区域描述
}

// GetAvailableRegionsReq 获取可用区域列表请求
type GetAvailableRegionsReq struct {
	Provider  CloudProvider `json:"provider" form:"provider" binding:"required,oneof=1 2 3 4 5 6"`
	AccessKey string        `json:"access_key" form:"access_key"` // 可选，提供时会通过API动态获取
	SecretKey string        `json:"secret_key" form:"secret_key"` // 可选，提供时会通过API动态获取
}

// AvailableRegion 可用区域信息
type AvailableRegion struct {
	Region     string `json:"region"`      // 区域代码，如cn-hangzhou
	RegionName string `json:"region_name"` // 区域名称，如华东1(杭州)
	Available  bool   `json:"available"`   // 是否可用
}

// GetAvailableRegionsResp 获取可用区域列表响应
type GetAvailableRegionsResp struct {
	Regions []AvailableRegion `json:"regions"`
}
