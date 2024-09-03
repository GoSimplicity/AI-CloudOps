package dto

// UserDTO 用户的数据传输对象
type UserDTO struct {
	UserID      int    `json:"userId"`      // UserID 用户的唯一标识符
	UserName    string `json:"username"`    // UserName 用户的登录名
	PassWord    string `json:"password"`    // PassWord 用户的登录密码
	RealName    string `json:"realName"`    // RealName 用户的真实姓名
	Mobile      string `json:"mobile"`      // Mobile 用户的手机号
	LarkUserID  string `json:"larkUserID"`  // LarkUserID 飞书系统中的用户ID
	AccountType int    `json:"accountType"` // AccountType 表示账户类型（例如：1普通用户、2管理员）
	Enable      int    `json:"enable"`      // Enable 表示账户是否启用，1 为启用，0 为禁用
	HomePath    string `json:"homePath"`    // HomePath 用户进入前端的默认页面路径
	Avatar      string `json:"avatar"`      // Avatar 用户头像的 URL
	Desc        string `json:"desc"`        // Desc 用户的个人简介或描述
}
