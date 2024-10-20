package model

import (
	"encoding/json"
	"gorm.io/gorm"
	"time"
)

// TerraformConfig 集中管理 Terraform 所需的所有配置
type TerraformConfig struct {
	ID        int             `gorm:"primaryKey" json:"id"` // 主键
	Region    string          `json:"region"`               // 阿里云的 Region
	Name      string          `json:"name"`                 // 资源名称
	Instance  json.RawMessage `json:"instance"`             // ECS 实例配置，存储为 JSON
	VPC       json.RawMessage `json:"vpc"`                  // VPC 配置，存储为 JSON
	Security  json.RawMessage `json:"security"`             // 安全组配置，存储为 JSON
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
	DeletedAt gorm.DeletedAt  `gorm:"index" json:"-"` // 软删除
}

// InstanceConfig 表示 ECS 实例的配置
type InstanceConfig struct {
	AvailabilityZone        string `json:"instance_availability_zone"` // 可用区 ID
	InstanceType            string `json:"instance_type"`              // ECS 实例类型
	SystemDiskCategory      string `json:"system_disk_category"`       // 系统盘类型
	SystemDiskName          string `json:"system_disk_name"`           // 系统盘名称
	SystemDiskDescription   string `json:"system_disk_description"`    // 系统盘描述
	ImageID                 string `json:"image_id"`                   // 镜像 ID
	InstanceName            string `json:"instance_name"`              // 实例名称
	VSwitchID               string `json:"instance_vswitch_id"`        // 关联的 VSwitch ID
	InternetMaxBandwidthOut int    `json:"internet_max_bandwidth_out"` // 最大公网带宽
}

// VPCConfig 表示 VPC 和相关资源的配置
type VPCConfig struct {
	VpcName     string `json:"vpc_name"`     // VPC 名称
	CidrBlock   string `json:"cidr_block"`   // VPC 的网段
	VSwitchCidr string `json:"vswitch_cidr"` // VSwitch 的网段
	ZoneID      string `json:"zone_id"`      // 可用区 ID
}

// SecurityConfig 表示安全组的配置
type SecurityConfig struct {
	SecurityGroupName        string `json:"security_group_name"`        // 安全组名称
	SecurityGroupDescription string `json:"security_group_description"` // 安全组描述
	SecurityGroupVpcID       string `json:"security_group_vpc_id"`      // 关联的 VPC ID
}
