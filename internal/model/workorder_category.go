package model

import "time"

// Category 分类实体（DAO层）
type Category struct {
	ID          int        `json:"id" gorm:"primaryKey;column:id;comment:主键ID"`
	Name        string     `json:"name" gorm:"column:name;not null;comment:分类名称"`
	ParentID    *int       `json:"parent_id" gorm:"column:parent_id;comment:父分类ID"`
	Icon        string     `json:"icon" gorm:"column:icon;comment:图标"`
	SortOrder   int        `json:"sort_order" gorm:"column:sort_order;default:0;comment:排序顺序"`
	Status      int8       `json:"status" gorm:"column:status;not null;default:1;comment:状态：0-禁用，1-启用"`
	Description string     `json:"description" gorm:"column:description;comment:分类描述"`
	CreatedAt   time.Time  `json:"created_at" gorm:"column:created_at;not null;comment:创建时间"`
	UpdatedAt   time.Time  `json:"updated_at" gorm:"column:updated_at;not null;comment:更新时间"`
	DeletedAt   *time.Time `json:"deleted_at" gorm:"column:deleted_at;index;comment:删除时间"`
	CreatorID   int        `json:"creator_id" gorm:"column:creator_id;not null;comment:创建人ID"`
	CreatorName string     `json:"creator_name" gorm:"-"`

	Children []Category `json:"children" gorm:"-"`
	Parent   *Category  `json:"parent,omitempty" gorm:"foreignKey:ParentID"`
}

func (Category) TableName() string {
	return "workorder_category"
}

// 分类请求结构
type CreateCategoryReq struct {
	Name        string `json:"name" binding:"required,min=1,max=100"`
	ParentID    *int   `json:"parent_id"`
	Icon        string `json:"icon" binding:"omitempty,url"`
	SortOrder   int    `json:"sort_order"`
	Description string `json:"description" binding:"omitempty,max=500"`
}

type UpdateCategoryReq struct {
	ID          int    `json:"id" binding:"required"`
	Name        string `json:"name" binding:"required,min=1,max=100"`
	ParentID    *int   `json:"parent_id"`
	Icon        string `json:"icon" binding:"omitempty,url"`
	SortOrder   int    `json:"sort_order"`
	Description string `json:"description" binding:"omitempty,max=500"`
	Status      int8   `json:"status" binding:"required,oneof=0 1"`
}

type DeleteCategoryReq struct {
	ID int `json:"id" form:"id" binding:"required"`
}

type ListCategoryReq struct {
	Name     string `json:"name" form:"name"`
	Status   *int8  `json:"status" form:"status"`
	Page     int    `json:"page" form:"page" binding:"required,min=1"`
	PageSize int    `json:"page_size" form:"page_size" binding:"required,min=1,max=100"`
}

type DetailCategoryReq struct {
	ID int `json:"id" uri:"id" binding:"required"`
}

// TreeCategoryReq 获取分类树请求
type TreeCategoryReq struct {
	Status *int8 `json:"status" form:"status"`
}

// 分类响应结构
type CategoryResp struct {
	ID          int            `json:"id"`
	Name        string         `json:"name"`
	ParentID    *int           `json:"parent_id"`
	Icon        string         `json:"icon"`
	SortOrder   int            `json:"sort_order"`
	Status      int8           `json:"status"`
	Description string         `json:"description"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	CreatorName string         `json:"creator_name"`
	Children    []CategoryResp `json:"children,omitempty"`
}
