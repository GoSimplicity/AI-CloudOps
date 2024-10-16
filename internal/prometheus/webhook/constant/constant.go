package constant

// AlertSeverity 表示告警的严重性等级
type AlertSeverity string

// AlertStatus 表示告警的状态
type AlertStatus string

const (
	// 定义告警严重性等级常量
	AlertSeverityCritical AlertSeverity = "critical" // 严重
	AlertSeverityWarning  AlertSeverity = "warning"  // 警告
	AlertSeverityInfo     AlertSeverity = "info"     // 信息

	// 定义告警状态常量
	AlertStatusFiring   AlertStatus = "firing"   // 触发中
	AlertStatusResolved AlertStatus = "resolved" // 已恢复
)

// SeverityTitleColorMap 将告警严重性映射到标题颜色
var SeverityTitleColorMap = map[AlertSeverity]string{
	AlertSeverityCritical: "red",    // 严重 - 红色
	AlertSeverityWarning:  "yellow", // 警告 - 黄色
	AlertSeverityInfo:     "blue",   // 信息 - 蓝色
}

// StatusColorMap 将告警状态映射到颜色
var StatusColorMap = map[AlertStatus]string{
	AlertStatusFiring:   "red",   // 触发中 - 红色
	AlertStatusResolved: "green", // 已恢复 - 绿色
}

// StatusChineseMap 将告警状态映射到中文描述
var StatusChineseMap = map[AlertStatus]string{
	AlertStatusFiring:   "触发中", // 触发中
	AlertStatusResolved: "已恢复", // 已恢复
}

// URL 模板常量
const (
	SendGroupURLTemplate     = "%s/%s?id=%v"                            // 发送组 URL 模板
	RenderingURLTemplate     = "%s/%s?fingerprint=%v"                   // 渲染 URL 模板
	SilenceURLTemplate       = "%s/%s?fingerprint=%v&hour=%v"           // 静音 URL 模板
	SilenceByNameURLTemplate = "%s/%s?fingerprint=%v&hour=%v&by_name=1" // 按名称静音 URL 模板
	UnsilenceURLTemplate     = "%s/%s?fingerprint=%v"                   // 取消静音 URL 模板

	// DefaultUpgradeMinutes 默认告警升级时间（分钟）
	DefaultUpgradeMinutes = 30 // 默认告警升级时间为30分钟
)
