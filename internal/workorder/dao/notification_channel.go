package dao

import (
	"fmt"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"gorm.io/gorm"
)

// notificationChannelFilter 在count与分页前精确匹配顶层JSON字符串元素
func notificationChannelFilter(db *gorm.DB, channel string) (*gorm.DB, error) {
	switch channel {
	case "":
		return db, nil
	case model.NotificationChannelEmail, model.NotificationChannelFeishu, model.NotificationChannelSMS, model.NotificationChannelWebhook:
	default:
		return nil, fmt.Errorf("不支持的通知渠道")
	}
	switch db.Dialector.Name() {
	case "mysql":
		// 发布数据库MySQL8.4支持JSON_OVERLAPS，避免嵌套数组或子串被当作顶层渠道
		safe := "CASE WHEN JSON_VALID(channels) THEN channels ELSE '[]' END"
		return db.Where("JSON_TYPE("+safe+") = 'ARRAY' AND JSON_OVERLAPS("+safe+", JSON_ARRAY(?))", channel), nil
	case "sqlite":
		safe := "CASE WHEN json_valid(channels) THEN channels ELSE '[]' END"
		return db.Where("json_type("+safe+") = 'array' AND EXISTS (SELECT 1 FROM json_each("+safe+") AS selected_channel WHERE selected_channel.type = 'text' AND selected_channel.value = ? COLLATE BINARY)", channel), nil
	default:
		return nil, fmt.Errorf("当前数据库不支持通知渠道筛选")
	}
}
