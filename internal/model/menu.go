/*
 * MIT License
 *
 * Copyright (c) 2024 Bamboo
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 * THE SOFTWARE.
 *
 */

package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// Meta 菜单元数据
type Meta struct {
	Order      int    `json:"order,omitempty"`      // 排序
	Title      string `json:"title"`                // 标题
	AffixTab   bool   `json:"affixTab,omitempty"`   // 是否固定标签页
	HideInMenu bool   `json:"hideInMenu,omitempty"` // 是否在菜单中隐藏
	Icon       string `json:"icon"`                 // 图标
}

type Menu struct {
	ID           int       `json:"id" gorm:"primaryKey;autoIncrement;comment:主键ID"`                                     // 主键ID，自增
	CreatedAt    int64     `json:"created_at" gorm:"autoCreateTime;comment:创建时间"`                                       // 创建时间，自动记录
	UpdatedAt    int64     `json:"updated_at" gorm:"autoUpdateTime;comment:更新时间"`                                       // 更新时间，自动记录
	DeletedAt    int64     `json:"deleted_at" gorm:"index;default:0;comment:删除时间"`                                      // 软删除时间，使用普通索引
	Name         string    `json:"name" gorm:"type:varchar(50);not null;comment:菜单显示名称"`                                // 菜单显示名称，非空
	ParentID     int       `json:"parent_id" gorm:"default:0;comment:上级菜单ID,0表示顶级菜单"`                                 // 上级菜单ID,0表示顶级菜单
	Path         string    `json:"path" gorm:"type:varchar(255);not null;comment:前端路由访问路径"`                            // 前端路由访问路径，非空
	Component    string    `json:"component" gorm:"type:varchar(255);not null;comment:前端组件文件路径"`                       // 前端组件文件路径，非空
	RouteName    string    `json:"route_name" gorm:"type:varchar(50);uniqueIndex:idx_route_del;not null;comment:前端路由名称"` // 前端路由名称，唯一且非空
	Hidden       int8      `json:"hidden" gorm:"type:tinyint(1);default:0;comment:菜单是否隐藏 0显示 1隐藏"`                    // 菜单是否隐藏，使用int8节省空间
	Redirect     string    `json:"redirect" gorm:"type:varchar(255);default:'';comment:重定向路径"`                         // 重定向路径
	Meta         MetaField `json:"meta" gorm:"type:json;serializer:json;comment:菜单元数据"`                                // 菜单元数据，使用JSON存储
	Children     []*Menu   `json:"children" gorm:"-"`                                                                  // 子菜单列表,不映射到数据库
	Users        []*User   `json:"users" gorm:"many2many:user_menus;comment:关联用户"`                                    // 多对多关联用户
	Roles        []*Role   `json:"roles" gorm:"many2many:role_menus;comment:关联角色"`                                    // 多对多关联角色
}

type CreateMenuRequest struct {
	Name      string    `json:"name" binding:"required"`    // 菜单名称
	Path      string    `json:"path" binding:"required"`    // 菜单路径
	ParentId  int       `json:"parent_id" binding:"gte=0"`  // 父菜单ID
	Component string    `json:"component"`                  // 组件
	RouteName string    `json:"route_name"`                 // 路由名称
	Hidden    int       `json:"hidden" binding:"oneof=0 1"` // 是否隐藏
	Redirect  string    `json:"redirect"`                   // 重定向路径
	Meta      MetaField `json:"meta"`                       // 元数据
	Children  []*Menu   `json:"children" gorm:"-"`
}

type GetMenuRequest struct {
	Id int `json:"id" binding:"required,gt=0"` // 菜单ID
}

type UpdateMenuRequest struct {
	Id        int       `json:"id" binding:"required,gt=0"` // 菜单ID
	Name      string    `json:"name" binding:"required"`    // 菜单名称
	Path      string    `json:"path" binding:"required"`    // 菜单路径
	ParentId  int       `json:"parent_id" binding:"gte=0"`  // 父菜单ID
	Component string    `json:"component"`                  // 组件
	Icon      string    `json:"icon"`                       // 图标
	SortOrder int       `json:"sort_order" binding:"gte=0"` // 排序
	RouteName string    `json:"route_name"`                 // 路由名称
	Hidden    int       `json:"hidden" binding:"oneof=0 1"` // 是否隐藏
	Redirect  string    `json:"redirect"`                   // 重定向路径
	Meta      MetaField `json:"meta"`                       // 元数据
}

type DeleteMenuRequest struct {
	Id int `json:"id" binding:"required,gt=0"` // 菜单ID
}

type ListMenusRequest struct {
	PageNumber int `json:"page_number" binding:"required,gt=0"` // 页码
	PageSize   int `json:"page_size" binding:"required,gt=0"`   // 每页数量
}

type UpdateUserMenuRequest struct {
	UserId  int   `json:"user_id" binding:"required,gt=0"`  // 用户ID
	MenuIds []int `json:"menu_ids" binding:"required,gt=0"` // 菜单ID
}

type MetaField Meta

func (m *MetaField) Scan(value interface{}) error {
	if value == nil {
		*m = MetaField{}
		return nil
	}

	byteValue, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("invalid type for MetaField: %T", value)
	}

	// 将 JSON 字符串解析为 Meta 结构体
	if err := json.Unmarshal(byteValue, m); err != nil {
		return fmt.Errorf("error unmarshaling MetaField: %v", err)
	}

	return nil
}

func (m *MetaField) Value() (driver.Value, error) {
	if m == nil {
		return nil, nil
	}
	return json.Marshal(m)
}
