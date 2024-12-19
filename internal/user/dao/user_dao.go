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

package dao

import (
	"context"
	"errors"

	"github.com/GoSimplicity/AI-CloudOps/internal/constants"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/go-sql-driver/mysql"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type UserDAO interface {
	CreateUser(ctx context.Context, user *model.User) error
	GetUserByUsername(ctx context.Context, username string) (*model.User, error)
	GetUserByUsernames(ctx context.Context, usernames []string) ([]*model.User, error)
	UpdateUser(ctx context.Context, user *model.User) error
	GetAllUsers(ctx context.Context) ([]*model.User, error)
	GetUserByID(ctx context.Context, id int) (*model.User, error)
	GetUserByIDs(ctx context.Context, ids []int) ([]*model.User, error)
	GetPermCode(ctx context.Context, uid int) ([]string, error)
	ChangePassword(ctx context.Context, uid int, password string) error
	WriteOff(ctx context.Context, username, password string) error
	UpdateProfile(ctx context.Context, user *model.User) error
	DeleteUser(ctx context.Context, uid int) error
}

type userDAO struct {
	db *gorm.DB
	l  *zap.Logger
}

func NewUserDAO(db *gorm.DB, l *zap.Logger) UserDAO {
	return &userDAO{
		db: db,
		l:  l,
	}
}

// CreateUser 创建用户
func (u *userDAO) CreateUser(ctx context.Context, user *model.User) error {
	if err := u.db.WithContext(ctx).Create(user).Error; err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == constants.ErrCodeDuplicate {
			return constants.ErrorUserExist
		}
		u.l.Error("create user failed", zap.Error(err))
		return err
	}

	return nil
}

// GetUserByUsername 根据用户名获取用户信息
func (u *userDAO) GetUserByUsername(ctx context.Context, username string) (*model.User, error) {
	var user model.User

	if err := u.db.WithContext(ctx).Where("username = ?", username).First(&user).Error; err != nil {
		u.l.Error("get user by username failed", zap.String("username", username), zap.Error(err))
		return nil, err
	}

	return &user, nil
}

// UpdateUser 更新用户信息
func (u *userDAO) UpdateUser(ctx context.Context, user *model.User) error {
	if err := u.db.WithContext(ctx).Model(user).Updates(user).Error; err != nil {
		u.l.Error("update user failed", zap.Error(err))
		return err
	}

	return nil
}

// GetAllUsers 获取所有用户
func (u *userDAO) GetAllUsers(ctx context.Context) ([]*model.User, error) {
	var users []*model.User

	if err := u.db.WithContext(ctx).Preload("Roles").Preload("Menus").Preload("Apis").Find(&users).Error; err != nil {
		u.l.Error("get all users failed", zap.Error(err))
		return nil, err
	}

	return users, nil
}

// GetUserByID 根据用户ID获取用户信息
func (u *userDAO) GetUserByID(ctx context.Context, id int) (*model.User, error) {
	var user model.User

	if err := u.db.WithContext(ctx).Preload("Roles").Preload("Apis").Where("id = ?", id).First(&user).Error; err != nil {
		u.l.Error("get user by id failed", zap.Int("id", id), zap.Error(err))
		return nil, err
	}

	// 获取用户的菜单并构建树状结构
	var menus []*model.Menu
	if err := u.db.WithContext(ctx).
		Table("menus").
		Joins("LEFT JOIN user_menus ON menus.id = user_menus.menu_id").
		Where("user_menus.user_id = ? AND menus.is_deleted = ?", id, 0).
		Order("menus.parent_id, menus.id").
		Find(&menus).Error; err != nil {
		u.l.Error("get user menus failed", zap.Int("id", id), zap.Error(err))
		return nil, err
	}

	// 构建菜单树
	menuMap := make(map[int]*model.Menu)
	var rootMenus []*model.Menu

	// 第一次遍历,建立id->menu的映射
	for _, menu := range menus {
		menuMap[menu.ID] = menu
	}

	// 第二次遍历,构建父子关系
	for _, menu := range menus {
		if menu.ParentID == 0 {
			rootMenus = append(rootMenus, menu)
		} else {
			if parent, ok := menuMap[menu.ParentID]; ok {
				parent.Children = append(parent.Children, menu)
			}
		}
	}

	user.Menus = rootMenus

	return &user, nil
}

// GetPermCode 根据用户ID获取用户的所有权限码
func (u *userDAO) GetPermCode(ctx context.Context, uid int) ([]string, error) {
	// var user model.User

	// // 根据 uid 查找用户，并预加载关联的 Roles
	// if err := u.db.WithContext(ctx).Preload("Roles").Where("id = ?", uid).Find(&user).Error; err != nil {
	// 	u.l.Error("get user by id failed", zap.Int("id", uid), zap.Error(err))
	// 	return nil, err
	// }

	// // 用于存储所有的权限码
	// var permCodes []string

	// // 遍历用户的角色，提取每个角色的 Codes
	// for _, role := range user.Roles {
	// 	// Codes 字段存储为 "xxx,xxx,xxx" 格式的字符串，需要进行转换
	// 	codes := strings.Split(role.Codes, ",")
	// 	permCodes = append(permCodes, codes...)
	// }

	// return permCodes, nil

	return nil, nil
}

// GetUserByUsernames 根据用户名批量获取用户信息
func (u *userDAO) GetUserByUsernames(ctx context.Context, usernames []string) ([]*model.User, error) {
	var users []*model.User

	if err := u.db.WithContext(ctx).Where("username in (?)", usernames).Find(&users).Error; err != nil {
		u.l.Error("get user by username failed", zap.Strings("usernames", usernames), zap.Error(err))
		return nil, err
	}

	return users, nil
}

// ChangePassword 修改密码
func (u *userDAO) ChangePassword(ctx context.Context, uid int, password string) error {
	if err := u.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", uid).Update("password", password).Error; err != nil {
		u.l.Error("update password failed", zap.Int("uid", uid), zap.Error(err))
		return err
	}

	return nil
}

// UpdateProfile 更新用户信息
func (u *userDAO) UpdateProfile(ctx context.Context, user *model.User) error {
	if err := u.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", user.ID).Updates(map[string]interface{}{
		"real_name":       user.RealName,
		"desc":            user.Desc,
		"mobile":          user.Mobile,
		"fei_shu_user_id": user.FeiShuUserId,
		"account_type":    user.AccountType,
		"home_path":       user.HomePath,
		"enable":          user.Enable,
	}).Error; err != nil {
		u.l.Error("update user profile failed", zap.Int("uid", user.ID), zap.Error(err))
		return err
	}

	return nil
}

// WriteOff 注销用户
func (u *userDAO) WriteOff(ctx context.Context, username string, password string) error {
	if err := u.db.WithContext(ctx).Where("username = ?", username).Delete(&model.User{}).Error; err != nil {
		u.l.Error("write off user failed", zap.String("username", username), zap.Error(err))
		return err
	}

	return nil
}

// DeleteUser 删除用户
func (u *userDAO) DeleteUser(ctx context.Context, uid int) error {
	if err := u.db.WithContext(ctx).Where("id = ?", uid).Delete(&model.User{}).Error; err != nil {
		u.l.Error("delete user failed", zap.Int("uid", uid), zap.Error(err))
		return err
	}

	return nil
}

// GetUserByIDs 根据用户ID批量获取用户信息
func (u *userDAO) GetUserByIDs(ctx context.Context, ids []int) ([]*model.User, error) {
	var users []*model.User

	if err := u.db.WithContext(ctx).Where("id in (?)", ids).Find(&users).Error; err != nil {
		u.l.Error("get user by ids failed", zap.Ints("ids", ids), zap.Error(err))
		return nil, err
	}

	return users, nil
}
