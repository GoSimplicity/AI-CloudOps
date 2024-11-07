package model

import (
	"gorm.io/plugin/soft_delete"
	"time"
)

type Model struct {
	ID        int                   `json:"id" gorm:"primaryKey;autoIncrement"`     // 主键，自增
	CreatedAt time.Time             `json:"created_at" gorm:"autoCreateTime"`       // 自动记录创建时间
	UpdatedAt time.Time             `json:"updated_at" gorm:"autoUpdateTime"`       // 自动记录更新时间
	DeletedAt soft_delete.DeletedAt `json:"deleted_at" gorm:"uniqueIndex:udx_name"` // 软删除字段，自动管理
}

type NoUniqueIndexModel struct {
	ID        int                   `json:"id" gorm:"primaryKey;autoIncrement"` // 主键，自增
	CreatedAt time.Time             `json:"created_at" gorm:"autoCreateTime"`   // 自动记录创建时间
	UpdatedAt time.Time             `json:"updated_at" gorm:"autoUpdateTime"`   // 自动记录更新时间
	DeletedAt soft_delete.DeletedAt `json:"deleted_at" gorm:"index"`            // 软删除字段，自动管理
}
