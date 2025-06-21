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

package mock

import (
	"log"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/casbin/casbin/v2"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const (
	AdminUsername    = "admin"
	AdminPassword    = "admin"
	AdminAccountType = 2
)

type UserMock struct {
	db *gorm.DB
	ce *casbin.Enforcer
}

func NewUserMock(db *gorm.DB, ce *casbin.Enforcer) *UserMock {
	return &UserMock{
		db: db,
		ce: ce,
	}
}

func (u *UserMock) CreateUserAdmin() error {
	// 检查是否已经初始化过用户
	var count int64
	u.db.Model(&model.User{}).Count(&count)
	if count > 0 {
		log.Println("[用户已经初始化过,跳过Mock]")
		return nil
	}

	log.Println("[用户模块Mock开始]")

	// 生成加密密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(AdminPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("生成密码失败: %v\n", err)
		log.Println("[用户模块Mock结束]")
		return err
	}

	// 创建管理员用户实例
	adminUser := model.User{
		Username:     AdminUsername,
		Password:     string(hashedPassword),
		RealName:     "管理员账号",
		AccountType:  AdminAccountType,
		Enable:       1,
		HomePath:     "/",
		Mobile:       "123123123",
		FeiShuUserId: "123123123",
	}

	// 使用 FirstOrCreate 方法查找或创建管理员用户
	result := u.db.Where("username = ?", adminUser.Username).FirstOrCreate(&adminUser)

	// 检查操作是否成功
	if result.Error != nil {
		log.Printf("创建或获取管理员用户失败: %v\n", result.Error)
		log.Println("[用户模块Mock结束]")
		return result.Error
	}

	// 根据 RowsAffected 判断用户是否已存在或新创建
	if result.RowsAffected == 1 {
		log.Println("管理员用户创建成功")
	} else {
		log.Println("管理员用户已存在，跳过创建")
	}

	log.Println("[用户模块Mock结束]")
	return nil
}
