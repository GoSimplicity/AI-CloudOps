package model

import "time"

const WorkorderNotificationResolveRoute = "/api/workorder/notification/events/:id/resolve"
const WorkorderNotificationCancelScheduledRoute = "/api/workorder/notification/events/:id/cancel"

// WorkorderNotificationClock records the explicit shared interpretation of DATETIME values.
type WorkorderNotificationClock struct {
	ID       int    `gorm:"primaryKey"`
	TimeZone string `gorm:"type:varchar(100);not null"`
}

func (WorkorderNotificationClock) TableName() string { return "cl_workorder_notification_clock" }

// WorkorderNotificationEvent 与工单变更同事务保存，供独立投递工作器恢复消费
type WorkorderNotificationEvent struct {
	ID             uint64     `gorm:"primaryKey"`
	ParentEventID  *uint64    `gorm:"uniqueIndex:idx_notification_occurrence;index"`
	NotificationID *int       `gorm:"uniqueIndex:idx_notification_occurrence"`
	Occurrence     *int       `gorm:"uniqueIndex:idx_notification_occurrence"`
	RepeatCount    int        `gorm:"not null;default:0"`
	RepeatMinutes  int        `gorm:"not null;default:0"`
	InstanceID     int        `gorm:"not null;index"`
	EventType      string     `gorm:"type:varchar(50);not null"`
	Plan           []byte     `gorm:"type:longblob"`
	Snapshot       []byte     `gorm:"type:longblob;not null"`
	Content        string     `gorm:"type:text"`
	SenderID       int        `gorm:"not null"`
	Status         string     `gorm:"type:varchar(20);not null;index"`
	Attempts       int        `gorm:"not null;default:0"`
	NextAttemptAt  time.Time  `gorm:"not null;index"`
	LeaseToken     string     `gorm:"type:varchar(64);not null;default:''"`
	LeaseUntil     *time.Time `gorm:"index"`
	LastError      string     `gorm:"type:varchar(500)"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (WorkorderNotificationEvent) TableName() string { return "cl_workorder_notification_event" }

// WorkorderNotificationDelivery 标识事件内已成功投递的具体接收人
type WorkorderNotificationDelivery struct {
	ID                   uint64     `gorm:"primaryKey"`
	EventID              uint64     `gorm:"not null;uniqueIndex:idx_event_delivery"`
	NotificationID       int        `gorm:"not null;uniqueIndex:idx_event_delivery"`
	Channel              string     `gorm:"type:varchar(20);not null;uniqueIndex:idx_event_delivery"`
	RecipientHash        string     `gorm:"type:char(64);not null;uniqueIndex:idx_event_delivery"`
	Status               string     `gorm:"type:varchar(20);not null"`
	Attempts             int        `gorm:"not null;default:0"`
	MaxRetries           int        `gorm:"not null;default:0"`
	RetryIntervalMinutes int        `gorm:"not null;default:0"`
	NextAttemptAt        *time.Time `gorm:"index"`
	UpdatedAt            time.Time
	CreatedAt            time.Time
}

func (WorkorderNotificationDelivery) TableName() string { return "cl_workorder_notification_delivery" }

type ListWorkorderNotificationEventsReq struct {
	Page   int    `form:"page"`
	Size   int    `form:"size"`
	Status string `form:"status" binding:"omitempty,oneof=pending processing success failed uncertain skipped planned cancelled"`
}
type WorkorderNotificationEventStatus struct {
	ID             uint64    `json:"id"`
	ParentEventID  *uint64   `json:"parent_event_id"`
	NotificationID *int      `json:"notification_id"`
	Occurrence     *int      `json:"occurrence"`
	RepeatCount    int       `json:"repeat_count"`
	RepeatMinutes  int       `json:"repeat_minutes"`
	InstanceID     int       `json:"instance_id"`
	EventType      string    `json:"event_type"`
	Status         string    `json:"status"`
	Attempts       int       `json:"attempts"`
	NextAttemptAt  time.Time `json:"next_attempt_at"`
	LastError      string    `json:"last_error"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
type WorkorderNotificationDeliveryStatus struct {
	ID                   uint64     `json:"id"`
	EventID              uint64     `json:"event_id"`
	NotificationID       int        `json:"notification_id"`
	Channel              string     `json:"channel"`
	RecipientHash        string     `json:"recipient_hash"`
	RecipientMasked      string     `json:"recipient_masked,omitempty"`
	Status               string     `json:"status"`
	Attempts             int        `json:"attempts"`
	MaxRetries           int        `json:"max_retries"`
	RetryIntervalMinutes int        `json:"retry_interval_minutes"`
	NextAttemptAt        *time.Time `json:"next_attempt_at"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

// WorkorderNotificationResolution 留存人工核对结论，不包含消息或凭据
type WorkorderNotificationResolution struct {
	ID         uint64    `gorm:"primaryKey" json:"id"`
	EventID    uint64    `gorm:"not null;index" json:"event_id"`
	DeliveryID uint64    `gorm:"not null;index" json:"delivery_id"`
	Action     string    `gorm:"type:varchar(20);not null" json:"action"`
	OperatorID int       `gorm:"not null" json:"operator_id"`
	Reason     string    `gorm:"type:varchar(500);not null" json:"reason"`
	CreatedAt  time.Time `json:"created_at"`
}

func (WorkorderNotificationResolution) TableName() string {
	return "cl_workorder_notification_resolution"
}

type ResolveWorkorderNotificationDeliveryReq struct {
	DeliveryID uint64 `json:"delivery_id" binding:"required"`
	Action     string `json:"action" binding:"required,oneof=delivered retry"`
	Reason     string `json:"reason" binding:"required,min=5,max=500"`
}

const WorkorderNotificationRetryEventRoute = "/api/workorder/notification/events/:id/retry"

type RetryWorkorderNotificationEventReq struct {
	Reason string `json:"reason" binding:"required,min=5,max=500"`
}
