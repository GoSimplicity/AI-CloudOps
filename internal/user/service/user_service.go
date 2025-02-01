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

package service

import (
	"context"
	"errors"

	"github.com/GoSimplicity/AI-CloudOps/internal/constants"
	"github.com/GoSimplicity/AI-CloudOps/internal/system/service"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/user/dao"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserService interface {
	SignUp(ctx context.Context, user *model.User) error
	Login(ctx context.Context, user *model.User) (*model.User, error)
	GetProfile(ctx context.Context, uid int) (*model.User, error)
	GetPermCode(ctx context.Context, uid int) ([]string, error)
	GetUserList(ctx context.Context) ([]*model.User, error)
	ChangePassword(ctx context.Context, uid int, oldPassword, newPassword string) error
	WriteOff(ctx context.Context, username, password string) error
	UpdateProfile(ctx context.Context, uid int, user *model.User) error
	DeleteUser(ctx context.Context, uid int) error
}

type userService struct {
	dao           dao.UserDAO
	roleSvc       service.RoleService
	permissionSvc service.PermissionService
}

func NewUserService(dao dao.UserDAO, roleSvc service.RoleService, permissionSvc service.PermissionService) UserService {
	return &userService{
		dao:           dao,
		roleSvc:       roleSvc,
		permissionSvc: permissionSvc,
	}
}

// SignUp 用户注册
func (us *userService) SignUp(ctx context.Context, user *model.User) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user.Password = string(hash)

	if err := us.dao.CreateUser(ctx, user); err != nil {
		return err
	}

	// 为新用户分配默认角色
	role, err := us.roleSvc.GetRoleByName(ctx, "user")
	if err != nil {
		return err
	}

	if err := us.permissionSvc.AssignRoleToUser(ctx, user.ID, []int{role.ID}, nil); err != nil {
		return err
	}

	return nil
}

// Login 用户登录
func (us *userService) Login(ctx context.Context, user *model.User) (*model.User, error) {
	u, err := us.dao.GetUserByUsername(ctx, user.Username)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &model.User{}, constants.ErrorUserNotExist
	} else if err != nil {
		return &model.User{}, err
	}

	err = bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(user.Password))
	if err != nil {
		return &model.User{}, constants.ErrorPasswordIncorrect
	}

	return u, nil
}

// GetProfile 获取用户信息
func (us *userService) GetProfile(ctx context.Context, uid int) (*model.User, error) {
	return us.dao.GetUserByID(ctx, uid)
}

// GetPermCode 获取用户权限
func (us *userService) GetPermCode(ctx context.Context, uid int) ([]string, error) {
	codes, err := us.dao.GetPermCode(ctx, uid)
	if err != nil {
		return nil, err
	}

	return codes, nil
}

// GetUserList 获取用户列表
func (us *userService) GetUserList(ctx context.Context) ([]*model.User, error) {
	return us.dao.GetAllUsers(ctx)
}

// ChangePassword 修改密码
func (us *userService) ChangePassword(ctx context.Context, uid int, oldPassword string, newPassword string) error {
	// 验证旧密码是否正确
	user, err := us.dao.GetUserByID(ctx, uid)
	if err != nil {
		return err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(oldPassword)); err != nil {
		return constants.ErrorPasswordIncorrect
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	// 修改密码
	return us.dao.ChangePassword(ctx, uid, string(hash))
}

// UpdateProfile 修改用户信息
func (us *userService) UpdateProfile(ctx context.Context, uid int, req *model.User) error {
	// 验证用户是否存在
	user, err := us.dao.GetUserByID(ctx, uid)
	if err != nil {
		return err
	}

	// 更新用户信息
	user.RealName = req.RealName
	user.Desc = req.Desc
	user.Mobile = req.Mobile
	user.FeiShuUserId = req.FeiShuUserId
	user.AccountType = req.AccountType
	user.HomePath = req.HomePath
	user.Enable = req.Enable

	return us.dao.UpdateProfile(ctx, user)
}

// WriteOff 注销账号
func (us *userService) WriteOff(ctx context.Context, username string, password string) error {
	// 验证用户是否存在
	user, err := us.dao.GetUserByUsername(ctx, username)
	if err != nil {
		return err
	}

	// 验证密码是否正确
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return constants.ErrorPasswordIncorrect
	}

	// 注销账号
	return us.dao.WriteOff(ctx, username, password)
}

func (us *userService) DeleteUser(ctx context.Context, uid int) error {
	return us.dao.DeleteUser(ctx, uid)
}
