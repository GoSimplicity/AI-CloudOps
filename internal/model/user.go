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
	ID           int    `json:"id" gorm:"primaryKey;autoIncrement;comment:主键ID"`                                       // 主键ID，自增
	CreatedAt    int64  `json:"created_at" gorm:"autoCreateTime;comment:创建时间"`                                         // 创建时间，自动记录
	UpdatedAt    int64  `json:"updated_at" gorm:"autoUpdateTime;comment:更新时间"`                                         // 更新时间，自动记录
	DeletedAt    int64  `json:"deleted_at" gorm:"index;default:0;comment:删除时间"`                                        // 软删除时间，使用普通索引
	Username     string `json:"username" gorm:"type:varchar(100);uniqueIndex:idx_username_del;not null;comment:用户登录名"` // 用户登录名，唯一且非空
	Password     string `json:"password" gorm:"type:varchar(255);not null;comment:用户登录密码"`                             // 用户登录密码，非空，JSON序列化时忽略
	RealName     string `json:"real_name" gorm:"type:varchar(100);comment:用户真实姓名"`                                     // 用户真实姓名
	Desc         string `json:"desc" gorm:"type:text;comment:用户描述"`                                                    // 用户描述，支持较长文本
	Mobile       string `json:"mobile" gorm:"type:varchar(20);uniqueIndex:idx_mobile_del;comment:手机号"`                 // 手机号，添加唯一索引
	FeiShuUserId string `json:"fei_shu_user_id" gorm:"type:varchar(50);uniqueIndex:idx_feishu_del;comment:飞书用户ID"`     // 飞书用户ID，添加唯一索引
	AccountType  int8   `json:"account_type" gorm:"default:1;comment:账号类型 1普通用户 2服务账号" binding:"omitempty,oneof=1 2"`  // 账号类型，使用int8节省空间
	HomePath     string `json:"home_path" gorm:"type:varchar(255);default:'/';comment:登录后的默认首页"`                       // 登录后的默认首页，添加默认值
	Enable       int8   `json:"enable" gorm:"default:1;comment:用户状态 1正常 2冻结" binding:"omitempty,oneof=1 2"`            // 用户状态，使用int8节省空间
	Apis         []*Api `json:"apis" gorm:"many2many:user_apis;comment:关联接口"`                                          // 多对多关联接口
}

// TokenRequest 刷新令牌请求
type TokenRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required"` // 刷新令牌
}

// ChangePasswordRequest 修改密码请求
type ChangePasswordRequest struct {
	Username        string `json:"username" binding:"required"`        // 用户名
	Password        string `json:"password" binding:"required"`        // 原密码
	NewPassword     string `json:"newPassword" binding:"required"`     // 新密码
	ConfirmPassword string `json:"confirmPassword" binding:"required"` // 确认密码
}

// WriteOffRequest 注销账号请求
type WriteOffRequest struct {
	Username string `json:"username" binding:"required"` // 用户名
	Password string `json:"password" binding:"required"` // 密码
}

// UpdateProfileRequest 更新用户信息请求
type UpdateProfileRequest struct {
	UserId       int    `json:"user_id" binding:"required"`                // 用户ID
	RealName     string `json:"real_name" binding:"required"`              // 真实姓名
	Desc         string `json:"desc"`                                      // 描述
	Mobile       string `json:"mobile" binding:"required,len=11"`          // 手机号
	FeiShuUserId string `json:"fei_shu_user_id"`                           // 飞书用户ID
	AccountType  int    `json:"account_type" binding:"required,oneof=1 2"` // 账号类型
	HomePath     string `json:"home_path" binding:"required"`              // 默认首页
	Enable       int    `json:"enable" binding:"required,oneof=1 2"`       // 用户状态
}

// DeleteUserRequest 删除用户请求
type DeleteUserRequest struct {
	UserId int `json:"user_id" binding:"required"` // 用户ID
}
