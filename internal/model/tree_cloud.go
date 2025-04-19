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
	CloudProviderLocal   CloudProvider = "local"   // 本地环境
	CloudProviderAliyun  CloudProvider = "aliyun"  // 阿里云
	CloudProviderHuawei  CloudProvider = "huawei"  // 华为云
	CloudProviderTencent CloudProvider = "tencent" // 腾讯云
	CloudProviderAWS     CloudProvider = "aws"     // AWS
	CloudProviderAzure   CloudProvider = "azure"   // Azure
	CloudProviderGCP     CloudProvider = "gcp"     // Google Cloud
)

// CloudAccount 云账户信息
type CloudAccount struct {
	Model
	Name            string        `json:"name" gorm:"type:varchar(100);comment:账户名称"`
	Provider        CloudProvider `json:"provider" gorm:"type:varchar(50);comment:云厂商"`
	AccountId       string        `json:"accountId" gorm:"type:varchar(100);comment:账户ID"`
	AccessKey       string        `json:"-" gorm:"type:varchar(100);comment:访问密钥ID"`
	EncryptedSecret string        `json:"-" gorm:"type:varchar(500);comment:加密的访问密钥"`
	Regions         StringList    `json:"regions" gorm:"type:varchar(500);comment:可用区域列表"`
	IsEnabled       bool          `json:"isEnabled" gorm:"comment:是否启用"`
	LastSyncTime    time.Time     `json:"lastSyncTime" gorm:"comment:最后同步时间"`
	Description     string        `json:"description" gorm:"type:text;comment:账户描述"`
}

// CloudProviderResp 云厂商响应
type CloudProviderResp struct {
	Provider  CloudProvider `json:"provider"`
	LocalName string        `json:"localName"`
}

type ListRegionsReq struct {
	Provider CloudProvider `json:"provider" binding:"required"`
}

type ListZonesReq struct {
}

type ListInstanceTypesReq struct {
	Provider CloudProvider `json:"provider" binding:"required"`
	Region   string        `json:"region" binding:"required"`
}

type ListImagesReq struct {
	Provider CloudProvider `json:"provider" binding:"required"`
	Region   string        `json:"region" binding:"required"`
}

type ListVpcsReq struct {
	Provider CloudProvider `json:"provider" binding:"required"`
	Region   string        `json:"region" binding:"required"`
}

type ListSecurityGroupsReq struct {
	Provider CloudProvider `json:"provider" binding:"required"`
	Region   string        `json:"region" binding:"required"`
}

// RegionResp 区域信息响应
type RegionResp struct {
	RegionId       string `json:"regionId"`       // 区域ID
	LocalName      string `json:"localName"`      // 区域名称
	RegionEndpoint string `json:"regionEndpoint"` // 区域终端节点
}

// ZoneResp 可用区信息响应
type ZoneResp struct {
	ZoneId    string `json:"zoneId"`
	LocalName string `json:"localName"`
}

// InstanceTypeResp 实例类型响应
type InstanceTypeResp struct {
	InstanceTypeId string `json:"instanceTypeId"`
	CpuCoreCount   int    `json:"cpuCoreCount"`
	MemorySize     int    `json:"memorySize"`
	Description    string `json:"description"`
}

// ImageResp 镜像响应
type ImageResp struct {
	ImageId     string `json:"imageId"`
	ImageName   string `json:"imageName"`
	OSType      string `json:"osType"`
	Description string `json:"description"`
}

// SecurityGroupResp 安全组响应
type SecurityGroupResp struct {
	SecurityGroupId   string `json:"securityGroupId"`
	SecurityGroupName string `json:"securityGroupName"`
	Description       string `json:"description"`
}
