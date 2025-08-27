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

// User 用户模型
type User struct {
	Model
	Username     string `json:"username" gorm:"type:varchar(100);not null;comment:用户登录名"`                             // 用户登录名，唯一且非空
	Password     string `json:"-" gorm:"type:varchar(255);not null;comment:用户登录密码"`                                   // 用户登录密码，非空，JSON序列化时忽略
	RealName     string `json:"real_name" gorm:"type:varchar(100);comment:用户真实姓名"`                                    // 用户真实姓名
	Domain       string `json:"domain" gorm:"type:varchar(100);default:'default';comment:用户域"`                        // 用户域，默认default
	Desc         string `json:"desc" gorm:"type:text;comment:用户描述"`                                                   // 用户描述，支持较长文本
	Avatar       string `json:"avatar" gorm:"type:longblob;comment:用户头像"`                                             // 用户头像
	Mobile       string `json:"mobile" gorm:"type:varchar(20);comment:手机号"`                                           // 手机号，添加唯一索引
	Email        string `json:"email" gorm:"type:varchar(100);comment:邮箱"`                                            // 邮箱，添加唯一索引
	FeiShuUserId string `json:"fei_shu_user_id" gorm:"type:varchar(50);comment:飞书用户ID"`                               // 飞书用户ID，添加唯一索引
	AccountType  int8   `json:"account_type" gorm:"default:1;comment:账号类型 1普通用户 2服务账号" binding:"omitempty,oneof=1 2"` // 账号类型，使用int8节省空间
	HomePath     string `json:"home_path" gorm:"type:varchar(255);default:'/';comment:登录后的默认首页"`                      // 登录后的默认首页，添加默认值
	Enable       int8   `json:"enable" gorm:"default:1;comment:用户状态 1正常 2冻结" binding:"omitempty,oneof=1 2"`           // 用户状态，使用int8节省空间
	Apis         []*Api `json:"apis" gorm:"many2many:cl_user_apis;comment:关联接口"`                                      // 多对多关联接口
}

func (u *User) TableName() string {
	return "cl_system_users"
}

// UserStatistics 用户统计
type UserStatistics struct {
	AdminCount      int64 `json:"admin_count"`       // 管理员数量
	ActiveUserCount int64 `json:"active_user_count"` // 活跃用户数量
}

// UserLoginReq 用户登录请求
type UserLoginReq struct {
	Username string `json:"username" binding:"required"` // 用户名
	Password string `json:"password" binding:"required"` // 密码
}

// UserSignUpReq 用户注册请求
type UserSignUpReq struct {
	Username     string `json:"username" binding:"required"`                      // 用户名
	Password     string `json:"password" binding:"required,min=6"`                // 密码，至少6位
	Mobile       string `json:"mobile" binding:"required"`                        // 手机号
	Email        string `json:"email"`                                            // 邮箱
	RealName     string `json:"real_name" binding:"required"`                     // 真实姓名
	Avatar       string `json:"avatar"`                                           // 用户头像
	Desc         string `json:"desc"`                                             // 用户描述
	FeiShuUserId string `json:"fei_shu_user_id"`                                  // 飞书用户ID
	AccountType  int8   `json:"account_type" binding:"required,oneof=1 2"`        // 账号类型 1普通用户 2服务账号
	HomePath     string `json:"home_path" binding:"omitempty" default:"/"`        // 默认首页
	Enable       int8   `json:"enable" binding:"omitempty,oneof=1 2" default:"1"` // 用户状态 1正常 2冻结
}

// TokenRequest 刷新令牌请求
type TokenRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required"`           // 刷新令牌
	UserID       int    `json:"user_id" binding:"required"`                // 用户ID
	Username     string `json:"username" binding:"required"`               // 用户名
	Ssid         string `json:"ssid" binding:"required"`                   // 会话ID
	AccountType  int8   `json:"account_type" binding:"required,oneof=1 2"` // 账号类型 1普通用户 2服务账号
}

// ProfileReq 获取用户信息请求
type ProfileReq struct {
	ID int `json:"id" binding:"required"` // 用户ID
}

// GetPermCodeReq 获取权限码请求
type GetPermCodeReq struct {
	ID int `json:"id" binding:"required"` // 用户ID
}

// ChangePasswordReq 修改密码请求
type ChangePasswordReq struct {
	UserID          int    `json:"user_id" form:"user_id" binding:"required"`                                       // 用户ID
	Username        string `json:"username" form:"username" binding:"required"`                                     // 用户名
	Password        string `json:"password" form:"password" binding:"required"`                                     // 原密码
	NewPassword     string `json:"new_password" form:"new_password" binding:"required,min=6"`                       // 新密码，至少6位
	ConfirmPassword string `json:"confirm_password" form:"confirm_password" binding:"required,eqfield=NewPassword"` // 确认密码，必须与新密码相同
}

// GetUserListReq 获取用户列表请求
type GetUserListReq struct {
	ListReq
	Enable      int8 `json:"enable" form:"enable" default:"1"`             // 用户状态 1正常 2冻结
	AccountType int8 `json:"account_type" form:"account_type" default:"1"` // 账号类型 1普通用户 2服务账号
}

// WriteOffReq 注销账号请求
type WriteOffReq struct {
	Username string `json:"username" binding:"required"` // 用户名
	Password string `json:"password" binding:"required"` // 密码
}

// UpdateProfileReq 更新用户信息请求
type UpdateProfileReq struct {
	ID           int    `json:"id" form:"id" binding:"required"`                  // 用户ID
	RealName     string `json:"real_name" binding:"required"`                     // 真实姓名
	Desc         string `json:"desc"`                                             // 描述
	Avatar       string `json:"avatar"`                                           // 用户头像
	Mobile       string `json:"mobile" binding:"required"`                        // 手机号
	Email        string `json:"email"`                                            // 邮箱
	FeiShuUserId string `json:"fei_shu_user_id"`                                  // 飞书用户ID
	AccountType  int8   `json:"account_type" binding:"required,oneof=1 2"`        // 账号类型
	HomePath     string `json:"home_path" binding:"required"`                     // 默认首页
	Enable       int8   `json:"enable" binding:"omitempty,oneof=1 2" default:"1"` // 用户状态
}

// DeleteUserReq 删除用户请求
type DeleteUserReq struct {
	ID int `json:"id" form:"id" binding:"required"` // 用户ID
}

// GetUserDetailReq 获取用户详情请求
type GetUserDetailReq struct {
	ID int `json:"id" form:"id" binding:"required"`
}
