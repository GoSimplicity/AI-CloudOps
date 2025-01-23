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
	"strconv"
	"time"

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
		CreatedAt:    time.Now().Unix(),
		UpdatedAt:    time.Now().Unix(),
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

	// 为管理员用户添加所有权限
	userIDStr := strconv.FormatInt(int64(adminUser.ID), 10)
	paths := []string{"/*"}
	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}

	for _, path := range paths {
		for _, method := range methods {
			if ok, err := u.ce.AddPolicy(userIDStr, path, method); err == nil && ok {
				log.Printf("成功添加权限策略: 用户=%s, 路径=%s, 方法=%s", userIDStr, path, method)
			} else if err != nil {
				log.Printf("添加权限策略失败: %v", err)
			} else {
				log.Printf("权限策略已存在: 用户=%s, 路径=%s, 方法=%s", userIDStr, path, method)
			}
		}
	}

	err = u.ce.SavePolicy()
	if err != nil {
		log.Printf("保存策略失败: %v\n", err)
		return err
	}

	// 先检查菜单是否存在
	menuIds := []int{
		1, 2, 3, 4, 5, 6, 7, 8, 9, 10,
		11, 12, 13, 14, 15, 16, 17, 18,
		19, 20, 21, 22, 23, 24, 25, 26,
		27, 28, 29, 30,
	}

	var existingMenus []int
	if err := u.db.Model(&model.Menu{}).Where("id IN ?", menuIds).Pluck("id", &existingMenus).Error; err != nil {
		log.Printf("查询菜单失败: %v", err)
		return err
	}

	// 构建批量插入的数据,只包含存在的菜单ID
	userMenus := make([]map[string]interface{}, 0, len(existingMenus))
	for _, menuId := range existingMenus {
		userMenus = append(userMenus, map[string]interface{}{
			"user_id": adminUser.ID,
			"menu_id": menuId,
		})
	}

	// 先删除已有的关联
	if err := u.db.Table("user_menus").Where("user_id = ?", adminUser.ID).Delete(nil).Error; err != nil {
		log.Printf("删除已有用户菜单关联失败: %v", err)
		return err
	}

	// 批量创建新的关联
	if len(userMenus) > 0 {
		if err := u.db.Table("user_menus").Create(userMenus).Error; err != nil {
			log.Printf("添加用户菜单关联失败: %v", err)
			return err
		}
	}

	log.Println("[用户模块Mock结束]")
	return nil
}
