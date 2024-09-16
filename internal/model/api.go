package model

type Api struct {
	Model
	Path     string  `gorm:"type:varchar(100);not null;comment:路由路径"`            // 路由路径，非空，表示API的具体访问路径
	Method   string  `gorm:"type:varchar(20);not null;comment:HTTP请求方法"`         // HTTP请求方法，非空，如 GET、POST、PUT 等
	Pid      int     `gorm:"comment:父级API的ID"`                                    // 父级API的ID，用于构建API的树状结构
	Title    string  `gorm:"type:varchar(100);uniqueIndex;not null;comment:API名称"` // API名称，唯一且非空，用于描述此API的功能
	Roles    []*Role `gorm:"many2many:role_apis;comment:关联的角色"`                 // 关联的角色，多对多关系，表示哪些角色可以访问该API
	Type     string  `gorm:"type:varchar(100);default:1;comment:类型 0=父级 1=子级"` // API类型，0表示父级API，1表示子级API，默认值为1（子级）
	Key      uint    `json:"key" gorm:"-"`                                           // 用于前端显示的唯一键，不存储在数据库
	Value    uint    `json:"value" gorm:"-"`                                         // 用于前端显示的值，不存储在数据库
	Children []*Api  `json:"children" gorm:"-"`                                      // 子API列表，递归定义，用于前端构建API树结构，不存储在数据库
}
