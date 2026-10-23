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
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const (
	AdminUsername    = "admin"
	AdminAccountType = 2

	// MockAdminPasswordEnv 是 mock 管理员口令的环境变量名。未设置时不创建 mock 管理员。
	MockAdminPasswordEnv = "MOCK_ADMIN_PASSWORD"
	// MinMockAdminPasswordLen 与 BOOTSTRAP_ADMIN_PASSWORD 的最小长度保持一致。
	MinMockAdminPasswordLen = 12
)

// ErrMockAdminPasswordTooShort 表示 MOCK_ADMIN_PASSWORD 长度不足。
var ErrMockAdminPasswordTooShort = errors.New(MockAdminPasswordEnv + " 至少12字符")

type UserMock struct {
	db     *gorm.DB
	getenv func(string) string
}

func NewUserMock(db *gorm.DB) *UserMock {
	return &UserMock{
		db:     db,
		getenv: os.Getenv,
	}
}

// CreateUserAdmin 在空用户表上创建 mock 管理员。
// 口令只从环境变量 MOCK_ADMIN_PASSWORD 读取，代码中不再内置任何口令；
// 未设置时跳过创建（不生成、不落盘、不打印口令），设置但不足 12 字符时返回错误。
func (u *UserMock) CreateUserAdmin() error {
	// 检查是否已经初始化过用户
	var count int64
	if err := u.db.Model(&model.User{}).Count(&count).Error; err != nil {
		return fmt.Errorf("查询用户数量失败: %w", err)
	}
	if count > 0 {
		log.Println("[用户已经初始化过,跳过Mock]")
		return nil
	}

	password := u.getenv(MockAdminPasswordEnv)
	if password == "" {
		log.Printf("[用户模块Mock] 未设置 %s，跳过创建 mock 管理员；如需本地管理员，请设置该变量（至少%d字符）或使用 BOOTSTRAP_ADMIN_PASSWORD\n", MockAdminPasswordEnv, MinMockAdminPasswordLen)
		return nil
	}
	if len(password) < MinMockAdminPasswordLen {
		return ErrMockAdminPasswordTooShort
	}

	log.Println("[用户模块Mock开始]")

	// 生成加密密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
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
