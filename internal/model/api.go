package model

import "gorm.io/gorm"

type Api struct {
	gorm.Model
	Path   string  `gorm:"type:varchar(100);not null;comment:路由路径"`              // 路由路径，非空
	Method string  `gorm:"type:varchar(20);not null;comment:HTTP请求方法"`           // HTTP请求方法，非空
	Pid    int     `gorm:"comment:父级API的ID"`                                     // 父级API的ID，用于构建API树结构
	Title  string  `gorm:"type:varchar(100);uniqueIndex;not null;comment:API名称"` // API名称，唯一且非空
	Roles  []*Role `gorm:"many2many:role_apis;comment:关联的角色"`                    // 关联的角色，多对多关系
	Type   string  `gorm:"type:varchar(10);default:1;comment:类型 0=父级 1=子级"`      // API类型，0=父级 1=子级，默认子级
}
