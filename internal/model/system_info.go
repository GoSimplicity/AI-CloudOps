package model

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

// SystemInfoResponse 系统信息响应结构
type SystemInfoResponse struct {
	*System
	MemoryUsageFormatted string `json:"memory_usage_formatted"` // 格式化的内存使用情况
	DiskUsageFormatted   string `json:"disk_usage_formatted"`   // 格式化的磁盘使用情况
	UptimeFormatted      string `json:"uptime_formatted"`       // 格式化的运行时间
	SystemStatus         string `json:"system_status"`          // 系统状态
}
