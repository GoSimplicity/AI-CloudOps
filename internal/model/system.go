package model

import (
	"fmt"
	"time"
)

// System 系统硬件信息
type System struct {
	Model
	Hostname       string  `json:"hostname" gorm:"type:varchar(255);comment:主机名"`         // 主机名
	OS             string  `json:"os" gorm:"type:varchar(100);comment:操作系统"`              // 操作系统
	OSVersion      string  `json:"os_version" gorm:"type:varchar(100);comment:操作系统版本"`    // 操作系统版本
	Arch           string  `json:"arch" gorm:"type:varchar(50);comment:系统架构"`             // 系统架构
	CPUModel       string  `json:"cpu_model" gorm:"type:varchar(255);comment:CPU型号"`      // CPU型号
	CPUCores       int     `json:"cpu_cores" gorm:"comment:CPU核心数"`                       // CPU核心数
	CPUUsage       float64 `json:"cpu_usage" gorm:"comment:CPU使用率"`                       // CPU使用率
	MemoryTotal    uint64  `json:"memory_total" gorm:"comment:总内存MB"`                     // 总内存（MB）
	MemoryUsed     uint64  `json:"memory_used" gorm:"comment:已用内存MB"`                     // 已用内存（MB）
	MemoryUsage    float64 `json:"memory_usage" gorm:"comment:内存使用率"`                     // 内存使用率
	DiskTotal      uint64  `json:"disk_total" gorm:"comment:总磁盘空间GB"`                     // 总磁盘空间（GB）
	DiskUsed       uint64  `json:"disk_used" gorm:"comment:已用磁盘空间GB"`                     // 已用磁盘空间（GB）
	DiskUsage      float64 `json:"disk_usage" gorm:"comment:磁盘使用率"`                       // 磁盘使用率
	NetworkIn      uint64  `json:"network_in" gorm:"comment:网络入流量MB"`                     // 网络入流量（MB）
	NetworkOut     uint64  `json:"network_out" gorm:"comment:网络出流量MB"`                    // 网络出流量（MB）
	Uptime         uint64  `json:"uptime" gorm:"comment:系统运行时长秒"`                         // 系统运行时长（秒）
	LoadAvg1       float64 `json:"load_avg_1" gorm:"comment:1分钟负载"`                       // 1分钟平均负载
	LoadAvg5       float64 `json:"load_avg_5" gorm:"comment:5分钟负载"`                       // 5分钟平均负载
	LoadAvg15      float64 `json:"load_avg_15" gorm:"comment:15分钟负载"`                     // 15分钟平均负载
	ProcessCount   int     `json:"process_count" gorm:"comment:进程数"`                      // 进程数
	LastUpdateTime int64   `json:"last_update_time" gorm:"comment:最后更新时间;autoUpdateTime"` // 最后更新时间
}

// GetMemoryUsagePercentage 获取内存使用率百分比
func (s *System) GetMemoryUsagePercentage() float64 {
	if s.MemoryTotal == 0 {
		return 0
	}
	return float64(s.MemoryUsed) / float64(s.MemoryTotal) * 100
}

// GetDiskUsagePercentage 获取磁盘使用率百分比
func (s *System) GetDiskUsagePercentage() float64 {
	if s.DiskTotal == 0 {
		return 0
	}
	return float64(s.DiskUsed) / float64(s.DiskTotal) * 100
}

// GetUptimeFormatted 获取格式化的运行时间
func (s *System) GetUptimeFormatted() string {
	duration := time.Duration(s.Uptime) * time.Second
	days := int(duration.Hours()) / 24
	hours := int(duration.Hours()) % 24
	minutes := int(duration.Minutes()) % 60

	if days > 0 {
		return fmt.Sprintf("%d天%d小时%d分钟", days, hours, minutes)
	} else if hours > 0 {
		return fmt.Sprintf("%d小时%d分钟", hours, minutes)
	} else {
		return fmt.Sprintf("%d分钟", minutes)
	}
}

// FormatBytes 格式化字节大小
func FormatBytes(bytes uint64, unit string) string {
	if unit == "GB" {
		if bytes < 1024 {
			return fmt.Sprintf("%.2f GB", float64(bytes))
		}
		tb := float64(bytes) / 1024
		return fmt.Sprintf("%.2f TB", tb)
	}

	if bytes < 1024 {
		return fmt.Sprintf("%.2f MB", float64(bytes))
	}
	gb := float64(bytes) / 1024
	return fmt.Sprintf("%.2f GB", gb)
}

// SystemInfoResponse 系统信息响应结构
type SystemInfoResponse struct {
	*System
	MemoryUsageFormatted string `json:"memory_usage_formatted"` // 格式化的内存使用情况
	DiskUsageFormatted   string `json:"disk_usage_formatted"`   // 格式化的磁盘使用情况
	UptimeFormatted      string `json:"uptime_formatted"`       // 格式化的运行时间
	SystemStatus         string `json:"system_status"`          // 系统状态
}

// ToResponse 转换为响应格式
func (s *System) ToResponse() *SystemInfoResponse {
	status := "健康"
	if s.CPUUsage > 80 || s.GetMemoryUsagePercentage() > 85 || s.GetDiskUsagePercentage() > 90 {
		status = "告警"
	} else if s.CPUUsage > 60 || s.GetMemoryUsagePercentage() > 70 || s.GetDiskUsagePercentage() > 80 {
		status = "注意"
	}

	return &SystemInfoResponse{
		System:               s,
		MemoryUsageFormatted: fmt.Sprintf("%s / %s", FormatBytes(s.MemoryUsed, "MB"), FormatBytes(s.MemoryTotal, "MB")),
		DiskUsageFormatted:   fmt.Sprintf("%s / %s", FormatBytes(s.DiskUsed, "GB"), FormatBytes(s.DiskTotal, "GB")),
		UptimeFormatted:      s.GetUptimeFormatted(),
		SystemStatus:         status,
	}
}
