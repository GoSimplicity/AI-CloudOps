package model

import "time"

// WorkorderInboxMessage 同一事件、配置及收件人只产生一条消息；删除保留幂等墓碑。
type WorkorderInboxMessage struct {
	ID             uint64     `json:"id" gorm:"primaryKey"`
	EventID        uint64     `json:"event_id" gorm:"not null;uniqueIndex:idx_inbox_delivery;index"`
	NotificationID int        `json:"notification_id" gorm:"not null;uniqueIndex:idx_inbox_delivery"`
	UserID         int        `json:"-" gorm:"not null;uniqueIndex:idx_inbox_delivery;index:idx_inbox_user_deleted,priority:1"`
	InstanceID     int        `json:"instance_id" gorm:"not null"`
	EventType      string     `json:"event_type" gorm:"type:varchar(50);not null"`
	Subject        string     `json:"subject" gorm:"type:varchar(500);not null"`
	Content        string     `json:"content" gorm:"type:text;not null"`
	ReadAt         *time.Time `json:"read_at" gorm:"index"`
	DeletedAt      *time.Time `json:"-" gorm:"index:idx_inbox_user_deleted,priority:2"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

func (WorkorderInboxMessage) TableName() string { return "cl_workorder_inbox_message" }

type ListWorkorderInboxReq struct {
	Page   int  `form:"page"`
	Size   int  `form:"size"`
	Unread bool `form:"unread"`
}
