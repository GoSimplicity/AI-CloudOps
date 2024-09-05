package dto

import userdto "github.com/GoSimplicity/CloudOps/internal/user/dto"

// MenuDTO 菜单数据传输对象
type MenuDTO struct {
	ID         uint      `json:"id"`          // 菜单ID
	Name       string    `json:"name"`        // 菜单名称
	Title      string    `json:"title"`       // 菜单标题
	Pid        int       `json:"pid"`         // 父级菜单ID
	ParentMenu string    `json:"parent_menu"` // 父级菜单标识
	Icon       string    `json:"icon"`        // 菜单图标
	Type       string    `json:"type"`        // 菜单类型，0=目录 1=子菜单
	Show       string    `json:"show"`        // 显示状态，0=禁用 1=启用
	OrderNo    int       `json:"order_no"`    // 排序号
	Component  string    `json:"component"`   // 前端组件
	Redirect   string    `json:"redirect"`    // 重定向路径
	Meta       MenuMeta  `json:"meta"`
	Path       string    `json:"path"`      // 路由路径
	Remark     string    `json:"remark"`    // 备注
	HomePath   string    `json:"home_path"` // 登录后的默认首页
	Status     string    `json:"status"`    // 启用状态，0=禁用 1=启用
	Roles      []RoleDTO `json:"roles"`     // 关联的角色
}

// RoleDTO 角色数据传输对象
type RoleDTO struct {
	ID        uint              `json:"id"`         // 角色ID
	OrderNo   int               `json:"order_no"`   // 排序号
	RoleName  string            `json:"role_name"`  // 角色名称
	RoleValue string            `json:"role_value"` // 角色值
	Remark    string            `json:"remark"`     // 备注
	HomePath  string            `json:"home_path"`  // 默认首页
	Status    string            `json:"status"`     // 角色状态，1=正常 2=冻结
	Users     []userdto.UserDTO `json:"users"`      // 关联的用户
	Menus     []MenuDTO         `json:"menus"`      // 关联的菜单
	Apis      []ApiDTO          `json:"apis"`       // 关联的API
}

// ApiDTO API数据传输对象
type ApiDTO struct {
	ID     uint      `json:"id"`     // API ID
	Path   string    `json:"path"`   // 路由路径
	Method string    `json:"method"` // HTTP 请求方法
	Pid    int       `json:"pid"`    // 父级API的ID
	Title  string    `json:"title"`  // API名称
	Roles  []RoleDTO `json:"roles"`  // 关联的角色
	Type   string    `json:"type"`   // 类型，0=父级 1=子级
}

type MenuMeta struct {
	Title           string `json:"title"`             // 菜单标题
	Icon            string `json:"icon"`              // 菜单图标
	ShowMenu        bool   `json:"show_menu"`         // 是否显示菜单
	HideMenu        bool   `json:"hide_menu"`         // 是否隐藏菜单
	IgnoreKeepAlive bool   `json:"ignore_keep_alive"` // 是否忽略页面缓存
}
