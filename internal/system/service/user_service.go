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
	"strings"

	"github.com/GoSimplicity/AI-CloudOps/internal/constants"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/permission"
	"github.com/GoSimplicity/AI-CloudOps/internal/system/dao"
	userutils "github.com/GoSimplicity/AI-CloudOps/internal/system/utils"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type UserService interface {
	SignUp(ctx context.Context, user *model.UserSignUpReq) error
	Login(ctx context.Context, user *model.UserLoginReq, clientIP string) (*model.User, error)
	GetProfile(ctx context.Context, uid int) (*model.User, error)
	GetPermCode(ctx context.Context, uid int) ([]string, error)
	GetPermissions(ctx context.Context, uid int, catalog []permission.CatalogEntry) (*model.UserPermissionsResp, error)
	GetUserDetail(ctx context.Context, uid int) (*model.User, error)
	GetUserList(ctx context.Context, req *model.GetUserListReq) (model.ListResp[*model.User], error)
	ChangePassword(ctx context.Context, req *model.ChangePasswordReq) error
	WriteOff(ctx context.Context, uid int, password string) error
	UpdateProfile(ctx context.Context, req *model.UpdateProfileReq, allowAccountChanges bool) error
	DeleteUser(ctx context.Context, uid int) error
	GetUserStatistics(ctx context.Context) (*model.UserStatistics, error)
}

type userService struct {
	dao     dao.UserDAO
	logger  *zap.Logger
	roleDao dao.RoleDAO
	guard   LoginGuard
}

func NewUserService(dao dao.UserDAO, roleDao dao.RoleDAO, logger *zap.Logger, guard LoginGuard) UserService {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &userService{
		dao:     dao,
		roleDao: roleDao,
		logger:  logger,
		guard:   guard,
	}
}

// dummyPasswordHash 与真实口令使用相同的 bcrypt cost，账号不存在时也做一次比较。
func dummyPasswordHash() string {
	return userutils.LoginTimingHash()
}

var comparePassword = userutils.ComparePassword

// SignUp 用户注册
func (us *userService) SignUp(ctx context.Context, user *model.UserSignUpReq) error {
	user.Username = strings.TrimSpace(user.Username)
	if strings.EqualFold(user.Username, "admin") {
		return fmt.Errorf("保留用户名不可注册")
	}
	user.AccountType = 1
	user.Enable = 1
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

// Login 校验登录。用户不存在、密码错误、账号不可用对外都是同一凭证错误；
// 达到阈值时返回锁定错误。clientIP 只用于计数，密码不会写入日志。
func (us *userService) Login(ctx context.Context, req *model.UserLoginReq, clientIP string) (*model.User, error) {
	if req == nil {
		return nil, constants.ErrorInvalidCredentials
	}
	username := strings.TrimSpace(req.Username)
	if username == "" {
		_ = comparePassword(dummyPasswordHash(), req.Password)
		us.logger.Warn("登录失败", zap.String("reason", "empty_username"))
		return nil, constants.ErrorInvalidCredentials
	}
	if us.guard == nil {
		us.logger.Error("登录防护未配置")
		return nil, ErrLoginUnavailable
	}
	if err := us.guard.Check(ctx, username, clientIP); err != nil {
		return nil, err
	}

	user, err := us.dao.GetByUsername(ctx, username)
	hash := dummyPasswordHash()
	forced := false
	reason := ""
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		forced = true
		reason = "user_not_found"
		user = nil
	case err != nil:
		us.logger.Error("获取用户失败", zap.String("username", username), zap.Error(err))
		return nil, errors.New("登录失败")
	case user.Enable != 1:
		forced = true
		reason = "disabled"
		if user.Password != "" {
			hash = user.Password
		}
	default:
		if user.Password != "" {
			hash = user.Password
		}
	}

	if compareErr := comparePassword(hash, req.Password); compareErr != nil || forced {
		if reason == "" {
			reason = "password_mismatch"
		}
		us.logger.Warn("登录失败", zap.String("username", username), zap.String("reason", reason))
		if lockErr := us.guard.RecordFailure(ctx, username, clientIP); lockErr != nil {
			return nil, lockErr
		}
		return nil, constants.ErrorInvalidCredentials
	}

	if err = us.guard.Check(ctx, username, clientIP); err != nil {
		return nil, err
	}
	if err = us.guard.RecordSuccess(ctx, username); err != nil {
		us.logger.Error("清除登录失败计数失败", zap.String("username", username), zap.Error(err))
	}
	return user, nil
}

func (us *userService) GetProfile(ctx context.Context, uid int) (*model.User, error) {
	return us.dao.GetByID(ctx, uid)
}

func (us *userService) GetPermCode(ctx context.Context, uid int) ([]string, error) {
	codes, err := us.dao.GetPermCodes(ctx, uid)
	if err != nil {
		us.logger.Error("获取用户权限码失败", zap.Error(err))
		return nil, err
	}

	return codes, nil
}

// GetPermissions 把用户角色授予按与鉴权中间件相同的规则展开为 "METHOD /模板" 权限码。
// 超管只返回超管码 "*"。
func (us *userService) GetPermissions(ctx context.Context, uid int, catalog []permission.CatalogEntry) (*model.UserPermissionsResp, error) {
	roles, err := us.roleDao.GetRoles(ctx, uid)
	if err != nil {
		us.logger.Error("获取用户角色失败", zap.Int("uid", uid), zap.Error(err))
		return nil, err
	}
	if permission.HasSuperAdminRole(roles) {
		return &model.UserPermissionsResp{Version: model.UserPermissionsVersion, SuperAdmin: true, Permissions: []string{permission.SuperCode}}, nil
	}
	return &model.UserPermissionsResp{
		Version:     model.UserPermissionsVersion,
		Permissions: permission.Expand(permission.GrantsFromRoles(roles), catalog),
	}, nil
}

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
	if req == nil || req.UserID <= 0 {
		us.logger.Error("修改密码缺少有效的用户ID")
		return errors.New("无效的用户ID")
	}
	if req.Password == req.NewPassword {
		us.logger.Error("新密码不能与旧密码相同")
		return errors.New("新密码不能与旧密码相同")
	}

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

func (us *userService) UpdateProfile(ctx context.Context, req *model.UpdateProfileReq, allowAccountChanges bool) error {
	if req.DepartmentID != nil {
		return fmt.Errorf("部门归属只能通过管理员部门分配接口修改")
	}
	user, err := us.dao.GetByID(ctx, req.ID)
	if err != nil {
		us.logger.Error("获取用户失败", zap.Error(err))
		return err
	}

	userutils.ApplyProfileUpdates(user, req)
	if allowAccountChanges {
		if req.AccountType != 0 {
			user.AccountType = req.AccountType
		}
		if req.Enable != 0 {
			user.Enable = req.Enable
		}
	}
	return us.dao.Update(ctx, user)
}

// WriteOff 注销账号
func (us *userService) WriteOff(ctx context.Context, uid int, password string) error {
	user, err := us.dao.GetByID(ctx, uid)
	if err != nil {
		us.logger.Error("获取用户失败", zap.Error(err))
		return err
	}

	if err := userutils.ComparePassword(user.Password, password); err != nil {
		us.logger.Error("密码错误", zap.Error(err))
		return constants.ErrorPasswordIncorrect
	}

	return us.dao.WriteOff(ctx, uid)
}

func (us *userService) DeleteUser(ctx context.Context, uid int) error {
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
