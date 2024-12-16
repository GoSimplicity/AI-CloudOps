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

type User struct {
	Model                // 软删除字段，自动管理
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
	Menus        []*Menu `json:"menus" gorm:"many2many:user_menus;comment:关联菜单"`                                      // 多对多关联菜单
	Apis         []*Api  `json:"apis" gorm:"many2many:user_apis;comment:关联接口"`                                        // 多对多关联接口
}
