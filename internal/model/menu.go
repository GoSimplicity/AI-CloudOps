package model

type Menu struct {
	Model
	Name       string    `json:"name" gorm:"type:varchar(100);uniqueIndex;not null;comment:菜单名称，必须唯一且非空"` // 菜单名称，唯一且非空
	Title      string    `json:"title" gorm:"type:varchar(100);comment:菜单的显示标题"`                          // 菜单标题，用于前端显示
	Pid        int       `json:"pid" gorm:"comment:父级菜单的ID"`                                              // 父级菜单ID，表示此菜单的上级菜单
	ParentMenu string    `json:"parentMenu" gorm:"type:varchar(50);comment:父级菜单标识符"`                      // 父级菜单标识，用于前端逻辑标识
	Icon       string    `json:"icon" gorm:"type:varchar(100);comment:菜单图标的路径或类名"`                        // 菜单图标，通常为图标的路径或类名
	Type       string    `json:"type" gorm:"type:varchar(10);comment:菜单类型，0=目录，1=子菜单"`                    // 菜单类型，0表示目录，1表示子菜单
	Show       bool      `json:"show" gorm:"type:bool;default:true;comment:显示状态，false=禁用，true=启用"`        // 菜单显示状态，true为启用，false为禁用
	OrderNo    int       `json:"orderNo" gorm:"comment:菜单排序号"`                                            // 排序号，决定菜单在前端展示时的顺序
	Component  string    `json:"component" gorm:"type:varchar(50);comment:前端组件名称，菜单对应LAYOUT"`             // 前端组件，菜单对应的前端组件名称
	Redirect   string    `json:"redirect" gorm:"type:varchar(100);comment:页面重定向路径"`                       // 重定向路径，当菜单被点击时跳转的默认页面
	Path       string    `json:"path" gorm:"type:varchar(100);comment:菜单的路由路径"`                           // 路由路径，用于匹配前端路由
	Remark     string    `json:"remark" gorm:"type:text;comment:菜单的备注信息"`                                 // 备注信息，提供关于此菜单的额外说明
	HomePath   string    `json:"homePath" gorm:"type:varchar(100);comment:登录后的默认首页路径"`                    // 用户登录后的默认首页路径
	Status     string    `json:"status" gorm:"type:varchar(10);default:1;comment:启用状态，0=禁用，1=启用"`         // 启用状态，0表示禁用，1表示启用
	Meta       *MenuMeta `json:"meta" gorm:"-"`                                                           // 元信息，存储菜单的额外属性，前端处理用，数据库不存储
	Children   []*Menu   `json:"children" gorm:"-"`                                                       // 子菜单列表，递归表示子级菜单，前端处理用，数据库不存储
	Roles      []*Role   `json:"roles" gorm:"many2many:role_menus;comment:多对多角色关联"`                       // 角色关联，表示菜单与角色的多对多关系
	Key        int       `json:"key" gorm:"-"`                                                            // 菜单的唯一标识符，前端使用
	Value      int       `json:"value" gorm:"-"`                                                          // 菜单的值，前端使用
}

type MenuMeta struct {
	Title           string `json:"title" gorm:"-"`           // 菜单标题，用于前端显示的标题
	Icon            string `json:"icon" gorm:"-"`            // 菜单图标，用于显示菜单的图标（类名或路径）
	ShowMenu        bool   `json:"showMenu" gorm:"-"`        // 是否显示菜单，true表示显示，false表示隐藏
	HideMenu        bool   `json:"hideMenu" gorm:"-"`        // 是否隐藏菜单，true表示隐藏，false表示不隐藏
	IgnoreKeepAlive bool   `json:"ignoreKeepAlive" gorm:"-"` // 是否禁用路由缓存，true表示禁用，false表示启用缓存
}
