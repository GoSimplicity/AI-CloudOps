package model

import (
	"encoding/json"
	"time"
)

// TerraformConfig 集中管理 Terraform 所需的所有配置
type TerraformConfig struct {
	ID          int             `json:"id"`          // 主键
	Region      string          `json:"region"`      // 阿里云的 Region
	Name        string          `json:"name"`        // 资源名称
	Instance    json.RawMessage `json:"instance"`    // ECS 实例配置，存储为 JSON
	VPC         json.RawMessage `json:"vpc"`         // VPC 配置，存储为 JSON
	Security    json.RawMessage `json:"security"`    // 安全组配置，存储为 JSON
	Env         string          `json:"env"`         // 环境标识，如 dev、stage、prod
	PayType     string          `json:"payType"`     // 付费类型，按量付费或包年包月
	Description string          `json:"description"` // 资源描述
	Tags        StringList      `json:"tags"`        // 资源标签，使用逗号分隔
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
	SecurityGroupName        string     `json:"security_group_name"`        // 安全组名称
	SecurityGroupDescription string     `json:"security_group_description"` // 安全组描述
	SecurityGroupVpcID       StringList `json:"security_group_vpc_id"`      // 关联的 VPC ID
}

type Task struct {
	TaskID       string          `json:"task_id"`
	Config       TerraformConfig `json:"config"`
	Status       string          `json:"status"`        // pending, processing, success, failed
	ErrorMessage string          `json:"error_message"` // 可选，用于记录错误信息
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
	Action       string          `json:"action"`      // create, update
	RetryCount   int             `json:"retry_count"` // 重试次数
}
