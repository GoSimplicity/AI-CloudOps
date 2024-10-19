package model

type User struct {
	Model
	Username     string  `json:"username" gorm:"type:varchar(100);uniqueIndex;not null;comment:用户登录名"`                // 用户登录名，唯一且非空
	Password     string  `json:"password" gorm:"type:varchar(255);not null;comment:用户登录密码"`                           // 用户登录密码，非空
	RealName     string  `json:"realName" gorm:"type:varchar(100);comment:用户真实姓名"`                                    // 用户真实姓名
	Desc         string  `json:"desc" gorm:"type:text;comment:用户描述"`                                                  // 用户描述，支持较长文本
	Mobile       string  `json:"mobile" gorm:"type:varchar(20);uniqueIndex;comment:手机号"`                              // 手机号，唯一索引
	FeiShuUserId string  `json:"feiShuUserId" gorm:"type:varchar(50);comment:飞书用户ID"`                                 // 飞书用户ID
	AccountType  int     `json:"accountType" gorm:"default:1;comment:账号类型 1普通用户 2服务账号" binding:"omitempty,oneof=1 2"` // 账号类型，默认为普通用户
	HomePath     string  `json:"homePath" gorm:"type:varchar(255);comment:登录后的默认首页"`                                  // 登录后的默认首页
	Enable       int     `json:"enable" gorm:"default:1;comment:用户状态 1正常 2冻结" binding:"omitempty,oneof=1 2"`          // 用户状态，默认为正常
	Roles        []*Role `json:"roles" gorm:"many2many:user_roles;comment:关联角色"`                                      // 多对多关联角色
}
