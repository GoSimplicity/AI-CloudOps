package model

import "gorm.io/gorm"

type Menu struct {
	gorm.Model
	Name       string  `gorm:"type:varchar(100);uniqueIndex;not null;comment:菜单名称"` // 菜单名称，唯一且非空
	Title      string  `gorm:"type:varchar(100);comment:菜单标题"`                      // 菜单标题
	Pid        int     `gorm:"comment:父级菜单ID"`                                      // 父级菜单ID
	ParentMenu string  `gorm:"type:varchar(50);comment:父级菜单标识"`                     // 父级菜单标识
	Icon       string  `gorm:"type:varchar(100);comment:菜单图标"`                      // 菜单图标
	Type       string  `gorm:"type:varchar(10);comment:菜单类型 0=目录 1=子菜单"`            // 菜单类型，0=目录 1=子菜单
	Show       string  `gorm:"type:varchar(10);default:1;comment:显示状态 0=禁用 1=启用"`   // 显示状态，0=禁用 1=启用
	OrderNo    int     `gorm:"comment:排序号"`                                         // 排序号
	Component  string  `gorm:"type:varchar(50);comment:前端组件 菜单就是LAYOUT"`            // 前端组件，菜单对应的前端组件
	Redirect   string  `gorm:"type:varchar(100);comment:重定向路径"`                     // 重定向路径
	Path       string  `gorm:"type:varchar(100);comment:路由路径"`                      // 路由路径
	Remark     string  `gorm:"type:text;comment:备注"`                                // 备注，支持较长文本
	HomePath   string  `gorm:"type:varchar(100);comment:登录后的默认首页"`                  // 登录后的默认首页
	Status     string  `gorm:"type:varchar(10);default:1;comment:启用状态 0=禁用 1=启用"`   // 启用状态，0=禁用 1=启用
	Roles      []*Role `gorm:"many2many:role_menus;comment:多对多角色关联"`                // 多对多角色关联
}
