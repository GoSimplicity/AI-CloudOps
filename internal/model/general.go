package model

import (
	"gorm.io/gorm"
	"time"
)

type Model struct {
	ID        int            `gorm:"primaryKey;autoIncrement"` // 主键，自增
	CreatedAt time.Time      `gorm:"autoCreateTime"`           // 自动记录创建时间
	UpdatedAt time.Time      `gorm:"autoUpdateTime"`           // 自动记录更新时间
	DeletedAt gorm.DeletedAt `gorm:"index"`                    // 软删除字段，自动管理
}
