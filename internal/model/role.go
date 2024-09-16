package model

type Role struct {
	Model
	OrderNo   int     `json:"orderNo" gorm:"comment:排序"`                                              // 排序号，决定角色在显示或列表中的顺序
	RoleName  string  `json:"roleName" gorm:"type:varchar(100);uniqueIndex;comment:角色中文名称"`       // 角色的中文名称，必须唯一，便于识别和显示
	RoleValue string  `json:"roleValue" gorm:"type:varchar(100);uniqueIndex;comment:角色值"`            // 角色的标识符，用于在权限控制等场景中标识角色，必须唯一
	Remark    string  `json:"remark" gorm:"comment:角色描述"`                                           // 对角色的简要描述，通常用于说明角色的功能或用途
	HomePath  string  `json:"homePath" gorm:"comment:登录后的默认首页"`                                 // 用户登录后默认的首页路径，根据角色定义不同的首页
	Codes     string  `json:"codes" gorm:"type:varchar(100);comment:权限码"`                            // 前端校验权限码
	Status    string  `json:"status" gorm:"type:varchar(100);default:1;comment:角色状态 1=正常 2=冻结"` // 角色状态，1 表示正常，2 表示被冻结
	Users     []*User `json:"users" gorm:"many2many:user_roles;comment:关联的用户"`                     // 多对多关联用户，表示哪些用户属于该角色
	Menus     []*Menu `json:"menus" gorm:"many2many:role_menus;comment:关联的菜单"`                     // 多对多关联菜单，表示该角色可以访问的菜单
	Apis      []*Api  `json:"apis" gorm:"many2many:role_apis;comment:关联的API"`                        // 多对多关联API，表示该角色可以调用的API接口
	MenuIds   []int   `json:"menuIds" gorm:"-"`                                                         // 前端使用的菜单ID，用于构建角色与菜单的关系，不存储在数据库中
	ApiIds    []int   `json:"apiIds" gorm:"-"`                                                          // 前端使用的API ID，用于构建角色与API的关系，不存储在数据库中
}
