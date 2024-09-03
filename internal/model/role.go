package model

import "gorm.io/gorm"

type Role struct {
	gorm.Model
	OrderNo   int     ` gorm:"comment:排序"`                                   // 排序
	RoleName  string  ` gorm:"type:varchar(100);uniqueIndex;comment:角色中文名称"` // 角色名称
	RoleValue string  ` gorm:"type:varchar(100);uniqueIndex;comment:角色值"`    // 角色值
	Remark    string  ` gorm:"comment:用户描述"`                                 // 备注
	HomePath  string  ` gorm:"comment:登陆后的默认首页"`                             // 默认首页
	Status    string  ` gorm:"default:1;comment:角色是否被冻结 1正常 2冻结"`            // 角色状态
	Users     []*User ` gorm:"many2many:user_roles;"`                        // 多对多用户关联
	Menus     []*Menu ` gorm:"many2many:role_menus;"`                        // 多对多菜单关联
	Apis      []*Api  ` gorm:"many2many:role_apis;"`                         // 多对多API关联
}
