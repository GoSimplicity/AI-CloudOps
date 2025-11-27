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

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	userutils "github.com/GoSimplicity/AI-CloudOps/internal/system/utils"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type UserDAO interface {
	Create(ctx context.Context, user *model.User) error
	GetByUsername(ctx context.Context, username string) (*model.User, error)
	List(ctx context.Context, page, size int, search string, enable *int8, accountType *int8) ([]*model.User, int64, error)
	GetByID(ctx context.Context, id int) (*model.User, error)
	GetByIDs(ctx context.Context, ids []int) ([]*model.User, error)
	GetPermCodes(ctx context.Context, uid int) ([]string, error)
	ChangePassword(ctx context.Context, uid int, password string) error
	WriteOff(ctx context.Context, uid int) error
	Update(ctx context.Context, user *model.User) error
	Delete(ctx context.Context, uid int) error
	GetStatistics(ctx context.Context) (*model.UserStatistics, error)
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
func (d *userDAO) Create(ctx context.Context, user *model.User) error {
	if user == nil {
		return errors.New("用户信息不能为空")
	}
	if err := userutils.RequireNonEmpty(user.Username, "用户名"); err != nil {
		return err
	}

	// 使用事务和一次性查询检查唯一性约束
	err := d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var count int64
		query := tx.Model(&model.User{}).Where("deleted_at = ?", 0).
			Where("username = ? OR (mobile = ? AND mobile != '') OR (fei_shu_user_id = ? AND fei_shu_user_id != '')",
				user.Username, user.Mobile, user.FeiShuUserId)

		if err := query.Count(&count).Error; err != nil {
			d.l.Error("检查唯一性约束失败", zap.Error(err))
			return err
		}

		if count > 0 {
			// 进一步确定具体是哪个字段重复
			var existingUser model.User
			if err := tx.Where("username = ? AND deleted_at = ?", user.Username, 0).First(&existingUser).Error; err == nil {
				return errors.New("用户名已存在")
			}
			if user.Mobile != "" {
				if err := tx.Where("mobile = ? AND deleted_at = ?", user.Mobile, 0).First(&existingUser).Error; err == nil {
					return errors.New("手机号已存在")
				}
			}
			if user.FeiShuUserId != "" {
				if err := tx.Where("fei_shu_user_id = ? AND deleted_at = ?", user.FeiShuUserId, 0).First(&existingUser).Error; err == nil {
					return errors.New("飞书用户ID已存在")
				}
			}
		}

		// 创建用户
		if err := tx.Create(user).Error; err != nil {
			d.l.Error("创建用户失败", zap.Error(err), zap.String("username", user.Username))
			return err
		}

		return nil
	})

	return err
}

// GetUserByUsername 根据用户名获取用户信息
func (d *userDAO) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	if err := userutils.RequireNonEmpty(username, "用户名"); err != nil {
		return nil, err
	}

	var user model.User
	if err := d.db.WithContext(ctx).Where("username = ?", username).First(&user).Error; err != nil {
		d.l.Error("根据用户名获取用户失败", zap.String("username", username), zap.Error(err))
		return nil, err
	}

	return &user, nil
}

// GetAllUsers 获取所有用户
func (d *userDAO) List(ctx context.Context, page, size int, search string, enable *int8, accountType *int8) ([]*model.User, int64, error) {
	var users []*model.User
	var count int64

	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 10
	}

	query := d.db.WithContext(ctx).Model(&model.User{})
	if search != "" {
		query = query.Where("username LIKE ? OR real_name LIKE ?", "%"+search+"%", "%"+search+"%")
	}

	if enable != nil {
		query = query.Where("enable = ?", *enable)
	}

	if accountType != nil {
		query = query.Where("account_type = ?", *accountType)
	}

	if err := query.Count(&count).Error; err != nil {
		d.l.Error("获取用户总数失败", zap.Error(err))
		return nil, 0, err
	}

	if err := query.Order("id desc").Offset((page - 1) * size).Limit(size).Preload("Apis").Find(&users).Error; err != nil {
		d.l.Error("获取所有用户失败", zap.Error(err))
		return nil, 0, err
	}

	return users, count, nil
}

// GetUserByID 根据用户ID获取用户信息
func (d *userDAO) GetByID(ctx context.Context, id int) (*model.User, error) {
	if err := userutils.RequirePositiveID(id, "用户ID"); err != nil {
		return nil, err
	}

	var user model.User
	if err := d.db.WithContext(ctx).
		Where("id = ?", id).
		Preload("Apis").
		First(&user).Error; err != nil {
		d.l.Error("根据ID获取用户失败", zap.Int("id", id), zap.Error(err))
		return nil, err
	}

	return &user, nil
}

// GetPermCode 获取用户权限码
func (d *userDAO) GetPermCodes(ctx context.Context, uid int) ([]string, error) {
	if err := userutils.RequirePositiveID(uid, "用户ID"); err != nil {
		return nil, err
	}

	var codes []string
	if err := d.db.WithContext(ctx).
		Table("cl_system_apis AS a").
		Joins("JOIN cl_system_role_apis AS ra ON a.id = ra.api_id").
		Joins("JOIN cl_system_roles AS r ON ra.role_id = r.id").
		Joins("JOIN cl_system_user_roles AS ur ON ur.role_id = r.id").
		Where("ur.user_id = ? AND r.status = ?", uid, 1).
		Distinct().
		Pluck("a.path", &codes).Error; err != nil {
		d.l.Error("获取用户权限码失败", zap.Int("uid", uid), zap.Error(err))
		return nil, err
	}

	return codes, nil
}

// ChangePassword 修改密码
func (d *userDAO) ChangePassword(ctx context.Context, uid int, password string) error {
	if err := userutils.RequirePositiveID(uid, "用户ID"); err != nil {
		return err
	}
	if err := userutils.RequireNonEmpty(password, "密码"); err != nil {
		return err
	}

	result := d.db.WithContext(ctx).
		Model(&model.User{}).
		Where("id = ?", uid).
		Updates(map[string]interface{}{
			"password": password,
		})

	if result.Error != nil {
		d.l.Error("修改密码失败", zap.Int("uid", uid), zap.Error(result.Error))
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("用户不存在或已删除")
	}

	return nil
}

// UpdateProfile 更新用户信息
func (d *userDAO) Update(ctx context.Context, user *model.User) error {
	if user == nil || user.ID <= 0 {
		return errors.New("无效的用户信息")
	}
	if err := userutils.RequirePositiveID(user.ID, "用户ID"); err != nil {
		return err
	}

	// 使用事务和一次性查询检查唯一性约束
	err := d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var count int64
		query := tx.Model(&model.User{}).
			Where("id != ? AND ((mobile = ? AND mobile != '') OR (fei_shu_user_id = ? AND fei_shu_user_id != ''))",
				user.ID, user.Mobile, user.FeiShuUserId)

		if err := query.Count(&count).Error; err != nil {
			d.l.Error("检查唯一性约束失败", zap.Error(err))
			return err
		}

		if count > 0 {
			// 进一步确定具体是哪个字段重复
			var existingUser model.User
			if user.Mobile != "" {
				if err := tx.Where("id != ? AND mobile = ?", user.ID, user.Mobile).First(&existingUser).Error; err == nil {
					return errors.New("手机号已存在")
				}
			}
			if user.FeiShuUserId != "" {
				if err := tx.Where("id != ? AND fei_shu_user_id = ?", user.ID, user.FeiShuUserId).First(&existingUser).Error; err == nil {
					return errors.New("飞书用户ID已存在")
				}
			}
		}

		updates := map[string]interface{}{
			"real_name":       user.RealName,
			"desc":            user.Desc,
			"mobile":          user.Mobile,
			"fei_shu_user_id": user.FeiShuUserId,
			"account_type":    user.AccountType,
			"email":           user.Email,
			"avatar":          user.Avatar,
			"home_path":       user.HomePath,
			"enable":          user.Enable,
		}

		result := tx.Model(&model.User{}).
			Where("id = ?", user.ID).
			Updates(updates)

		if result.Error != nil {
			d.l.Error("更新用户信息失败", zap.Int("uid", user.ID), zap.Error(result.Error))
			return result.Error
		}

		if result.RowsAffected == 0 {
			return errors.New("用户不存在或已删除")
		}

		return nil
	})

	return err
}

// WriteOff 注销用户
func (d *userDAO) WriteOff(ctx context.Context, uid int) error {
	if err := userutils.RequirePositiveID(uid, "用户ID"); err != nil {
		return err
	}

	result := d.db.WithContext(ctx).
		Model(&model.User{}).
		Where("id = ?", uid).
		Delete(&model.User{})

	if result.Error != nil {
		d.l.Error("注销用户失败", zap.Int("uid", uid), zap.Error(result.Error))
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("用户不存在或已删除")
	}

	return nil
}

// DeleteUser 删除用户
func (d *userDAO) Delete(ctx context.Context, uid int) error {
	if err := userutils.RequirePositiveID(uid, "用户ID"); err != nil {
		return err
	}

	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 删除用户API关联
		if err := tx.Table("user_apis").Where("user_id = ?", uid).Delete(nil).Error; err != nil {
			d.l.Warn("删除用户API关联失败", zap.Int("uid", uid), zap.Error(err))
			// 继续执行，不中断事务
		}

		// 删除用户角色关联
		if err := tx.Table("cl_system_user_roles").Where("user_id = ?", uid).Delete(nil).Error; err != nil {
			d.l.Warn("删除用户角色关联失败", zap.Int("uid", uid), zap.Error(err))
		}

		// 删除用户
		result := tx.Where("id = ?", uid).Delete(&model.User{})
		if result.Error != nil {
			d.l.Error("删除用户失败", zap.Int("uid", uid), zap.Error(result.Error))
			return result.Error
		}

		if result.RowsAffected == 0 {
			return errors.New("用户不存在或已删除")
		}

		return nil
	})
}

// GetUserByIDs 批量获取用户信息
func (d *userDAO) GetByIDs(ctx context.Context, ids []int) ([]*model.User, error) {
	if len(ids) == 0 {
		return nil, errors.New("用户ID列表不能为空")
	}
	for _, id := range ids {
		if err := userutils.RequirePositiveID(id, "用户ID"); err != nil {
			return nil, err
		}
	}

	var users []*model.User
	if err := d.db.WithContext(ctx).
		Where("id in (?)", ids).
		Find(&users).Error; err != nil {
		d.l.Error("批量获取用户失败", zap.Ints("ids", ids), zap.Error(err))
		return nil, err
	}

	return users, nil
}

func (d *userDAO) GetStatistics(ctx context.Context) (*model.UserStatistics, error) {
	var statistics model.UserStatistics

	// 获取管理员总数
	if err := d.db.WithContext(ctx).Model(&model.User{}).Count(&statistics.AdminCount).Error; err != nil {
		d.l.Error("获取管理员总数失败", zap.Error(err))
		return nil, err
	}

	// 获取活跃用户数量
	if err := d.db.WithContext(ctx).Model(&model.User{}).Where("enable = ?", 1).Count(&statistics.ActiveUserCount).Error; err != nil {
		d.l.Error("获取活跃用户数量失败", zap.Error(err))
		return nil, err
	}

	return &statistics, nil
}
