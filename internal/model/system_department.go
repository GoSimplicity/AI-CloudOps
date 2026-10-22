package model

import (
	"gorm.io/plugin/soft_delete"
	"time"
)

// Department is a single-parent organization unit. Zero ParentID denotes a root.
type Department struct {
	ID          int                   `json:"id" gorm:"primaryKey;autoIncrement"`
	CreatedAt   time.Time             `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time             `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt   soft_delete.DeletedAt `json:"deleted_at" gorm:"softDelete:nano;uniqueIndex:idx_department_sibling,priority:3"`
	Name        string                `json:"name" gorm:"size:100;not null;uniqueIndex:idx_department_sibling,priority:2"`
	ParentID    int                   `json:"parent_id" gorm:"not null;default:0;index;uniqueIndex:idx_department_sibling,priority:1"`
	Description string                `json:"description" gorm:"size:500"`
}

func (Department) TableName() string { return "cl_system_departments" }

type DepartmentWriteReq struct {
	Name        string `json:"name" binding:"required,max=100"`
	ParentID    int    `json:"parent_id" binding:"gte=0"`
	Description string `json:"description" binding:"max=500"`
}

type AssignUserDepartmentReq struct {
	DepartmentID *int `json:"department_id" binding:"required,gte=0"`
}
