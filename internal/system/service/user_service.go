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
	"fmt"

	"github.com/GoSimplicity/AI-CloudOps/internal/constants"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/system/dao"
	userutils "github.com/GoSimplicity/AI-CloudOps/internal/system/utils"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type UserService interface {
	SignUp(ctx context.Context, user *model.UserSignUpReq) error
	Login(ctx context.Context, user *model.UserLoginReq) (*model.User, error)
	GetProfile(ctx context.Context, uid int) (*model.User, error)
	GetPermCode(ctx context.Context, uid int) ([]string, error)
	GetUserDetail(ctx context.Context, uid int) (*model.User, error)
	GetUserList(ctx context.Context, req *model.GetUserListReq) (model.ListResp[*model.User], error)
	ChangePassword(ctx context.Context, req *model.ChangePasswordReq) error
	WriteOff(ctx context.Context, uid int, password string) error
	UpdateProfile(ctx context.Context, req *model.UpdateProfileReq) error
	DeleteUser(ctx context.Context, uid int) error
	GetUserStatistics(ctx context.Context) (*model.UserStatistics, error)
}

type userService struct {
	dao     dao.UserDAO
	logger  *zap.Logger
	roleDao dao.RoleDAO
}

func NewUserService(dao dao.UserDAO, roleDao dao.RoleDAO, logger *zap.Logger) UserService {
	return &userService{
		dao:     dao,
		roleDao: roleDao,
		logger:  logger,
	}
}

// SignUp 用户注册
func (us *userService) SignUp(ctx context.Context, user *model.UserSignUpReq) error {
	hash, err := userutils.HashPassword(user.Password)
	if err != nil {
		us.logger.Error("生成密码失败", zap.Error(err))
		return err
	}

	if err := us.dao.Create(ctx, userutils.BuildUserForCreate(user, hash)); err != nil {
		us.logger.Error("创建用户失败", zap.Error(err))
		return err
	}

	return nil
}

// Login 用户登录
func (us *userService) Login(ctx context.Context, user *model.UserLoginReq) (*model.User, error) {
	u, err := us.dao.GetByUsername(ctx, user.Username)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &model.User{}, constants.ErrorUserNotExist
	} else if err != nil {
		us.logger.Error("获取用户失败", zap.Error(err))
		return &model.User{}, err
	}

	if err = userutils.ComparePassword(u.Password, user.Password); err != nil {
		us.logger.Error("密码错误", zap.Error(err))
		return &model.User{}, constants.ErrorPasswordIncorrect
	}

	return u, nil
}

// GetProfile 获取用户信息
func (us *userService) GetProfile(ctx context.Context, uid int) (*model.User, error) {
	return us.dao.GetByID(ctx, uid)
}

// GetPermCode 获取用户权限
func (us *userService) GetPermCode(ctx context.Context, uid int) ([]string, error) {
	codes, err := us.dao.GetPermCodes(ctx, uid)
	if err != nil {
		us.logger.Error("获取用户权限码失败", zap.Error(err))
		return nil, err
	}

	return codes, nil
}

// GetUserList 获取用户列表
func (us *userService) GetUserList(ctx context.Context, req *model.GetUserListReq) (model.ListResp[*model.User], error) {
	page := req.Page
	if page <= 0 {
		page = 1
	}
	size := req.Size
	if size <= 0 {
		size = 10
	}

	if req.Enable != nil {
		if *req.Enable == 0 {
			req.Enable = nil
		} else if *req.Enable != 1 && *req.Enable != 2 {
			return model.ListResp[*model.User]{}, fmt.Errorf("用户状态只支持 1(正常)/2(冻结)")
		}
	}

	if req.AccountType != nil {
		if *req.AccountType == 0 {
			req.AccountType = nil
		} else if *req.AccountType != 1 && *req.AccountType != 2 {
			return model.ListResp[*model.User]{}, fmt.Errorf("账号类型只支持 1(普通)/2(服务)")
		}
	}

	users, count, err := us.dao.List(ctx, page, size, req.Search, req.Enable, req.AccountType)
	if err != nil {
		us.logger.Error("获取用户列表失败", zap.Error(err))
		return model.ListResp[*model.User]{}, err
	}

	return model.ListResp[*model.User]{
		Items: users,
		Total: count,
	}, nil
}

// ChangePassword 修改密码
func (us *userService) ChangePassword(ctx context.Context, req *model.ChangePasswordReq) error {
	if req.Password == req.NewPassword {
		us.logger.Error("新密码不能与旧密码相同")
		return errors.New("新密码不能与旧密码相同")
	}

	// 验证旧密码是否正确
	user, err := us.dao.GetByID(ctx, req.UserID)
	if err != nil {
		us.logger.Error("获取用户失败", zap.Error(err))
		return err
	}

	if err := userutils.ComparePassword(user.Password, req.Password); err != nil {
		us.logger.Error("密码错误", zap.Error(err))
		return constants.ErrorPasswordIncorrect
	}

	hash, err := userutils.HashPassword(req.NewPassword)
	if err != nil {
		us.logger.Error("生成密码失败", zap.Error(err))
		return err
	}

	// 修改密码
	return us.dao.ChangePassword(ctx, req.UserID, hash)
}

// UpdateProfile 修改用户信息
func (us *userService) UpdateProfile(ctx context.Context, req *model.UpdateProfileReq) error {
	// 验证用户是否存在
	user, err := us.dao.GetByID(ctx, req.ID)
	if err != nil {
		us.logger.Error("获取用户失败", zap.Error(err))
		return err
	}

	// 更新用户信息
	userutils.ApplyProfileUpdates(user, req)
	return us.dao.Update(ctx, user)
}

// WriteOff 注销账号
func (us *userService) WriteOff(ctx context.Context, uid int, password string) error {
	// 验证用户是否存在
	user, err := us.dao.GetByID(ctx, uid)
	if err != nil {
		us.logger.Error("获取用户失败", zap.Error(err))
		return err
	}

	// 验证密码是否正确
	if err := userutils.ComparePassword(user.Password, password); err != nil {
		us.logger.Error("密码错误", zap.Error(err))
		return constants.ErrorPasswordIncorrect
	}

	return us.dao.WriteOff(ctx, uid)
}

func (us *userService) DeleteUser(ctx context.Context, uid int) error {
	// 删除用户角色关联
	if err := us.roleDao.RevokeRolesFromUser(ctx, uid, nil); err != nil {
		us.logger.Error("删除用户角色关联失败", zap.Int("uid", uid), zap.Error(err))
		return err
	}

	return us.dao.Delete(ctx, uid)
}

func (us *userService) GetUserDetail(ctx context.Context, uid int) (*model.User, error) {
	user, err := us.dao.GetByID(ctx, uid)
	if err != nil {
		us.logger.Error("获取用户详情失败", zap.Int("uid", uid), zap.Error(err))
		return nil, err
	}

	return user, nil
}

func (us *userService) GetUserStatistics(ctx context.Context) (*model.UserStatistics, error) {
	statistics, err := us.dao.GetStatistics(ctx)
	if err != nil {
		us.logger.Error("获取用户统计失败", zap.Error(err))
		return nil, err
	}

	return statistics, nil
}
