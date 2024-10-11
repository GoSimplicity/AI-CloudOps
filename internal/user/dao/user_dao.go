package dao

import (
	"context"
	"errors"
	"github.com/GoSimplicity/AI-CloudOps/internal/constants"
	"strings"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/go-sql-driver/mysql"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type UserDAO interface {
	// CreateUser 新建用户
	CreateUser(ctx context.Context, user *model.User) error
	// GetUserByUsername 通过用户名获取用户
	GetUserByUsername(ctx context.Context, username string) (*model.User, error)
	// GetUserByUsernames 通过用户名批量获取用户
	GetUserByUsernames(ctx context.Context, usernames []string) ([]*model.User, error)
	// GetOrCreateUser 获取用户或创建新用户
	GetOrCreateUser(ctx context.Context, user *model.User) (*model.User, error)
	// UpdateUser 更新用户
	UpdateUser(ctx context.Context, user *model.User) error
	// GetAllUsers 获取所有用户
	GetAllUsers(ctx context.Context) ([]*model.User, error)
	// GetUserByID 通过ID获取用户
	GetUserByID(ctx context.Context, id int) (*model.User, error)
	// GetUserByRealName 通过名称获取用户
	GetUserByRealName(ctx context.Context, name string) (*model.User, error)
	// GetUserByMobile 通过手机号获取用户
	GetUserByMobile(ctx context.Context, mobile string) (*model.User, error)
	// GetPermCode 获取用户权限码
	GetPermCode(ctx context.Context, uid int) ([]string, error)
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

func (u *userDAO) GetUserByUsername(ctx context.Context, username string) (*model.User, error) {
	var user model.User

	if err := u.db.WithContext(ctx).Where("username = ?", username).First(&user).Error; err != nil {
		u.l.Error("get user by username failed", zap.String("username", username), zap.Error(err))
		return nil, err
	}

	return &user, nil
}

func (u *userDAO) GetOrCreateUser(ctx context.Context, user *model.User) (*model.User, error) {
	if err := u.db.WithContext(ctx).Where("username = ?", user.Username).FirstOrCreate(user).Error; err != nil {
		u.l.Error("get or create user failed", zap.String("username", user.Username), zap.Error(err))
		return nil, err
	}

	return user, nil
}

func (u *userDAO) UpdateUser(ctx context.Context, user *model.User) error {
	if err := u.db.WithContext(ctx).Model(user).Updates(user).Error; err != nil {
		u.l.Error("update user failed", zap.Error(err))
		return err
	}

	return nil
}

func (u *userDAO) GetAllUsers(ctx context.Context) ([]*model.User, error) {
	var users []*model.User

	if err := u.db.WithContext(ctx).Find(&users).Error; err != nil {
		u.l.Error("get all users failed", zap.Error(err))
		return nil, err
	}

	return users, nil
}

func (u *userDAO) GetUserByID(ctx context.Context, id int) (*model.User, error) {
	var user model.User

	if err := u.db.WithContext(ctx).Preload("Roles").Where("id = ?", id).First(&user).Error; err != nil {
		u.l.Error("get user by id failed", zap.Int("id", id), zap.Error(err))
		return nil, err
	}

	return &user, nil
}

func (u *userDAO) GetUserByRealName(ctx context.Context, name string) (*model.User, error) {
	var user model.User

	if err := u.db.WithContext(ctx).Where("real_name = ?", name).First(&user).Error; err != nil {
		u.l.Error("get user by real name failed", zap.String("real_name", name), zap.Error(err))
		return nil, err
	}

	return &user, nil
}

func (u *userDAO) GetUserByMobile(ctx context.Context, mobile string) (*model.User, error) {
	var user model.User

	if err := u.db.WithContext(ctx).Where("mobile = ?", mobile).First(&user).Error; err != nil {
		u.l.Error("get user by mobile failed", zap.String("mobile", mobile), zap.Error(err))
		return nil, err
	}

	return &user, nil
}

func (u *userDAO) GetPermCode(ctx context.Context, uid int) ([]string, error) {
	var user model.User

	// 根据 uid 查找用户，并预加载关联的 Roles
	if err := u.db.WithContext(ctx).Preload("Roles").Where("id = ?", uid).Find(&user).Error; err != nil {
		u.l.Error("get user by id failed", zap.Int("id", uid), zap.Error(err))
		return nil, err
	}

	// 用于存储所有的权限码
	var permCodes []string

	// 遍历用户的角色，提取每个角色的 Codes
	for _, role := range user.Roles {
		// Codes 字段存储为 "xxx,xxx,xxx" 格式的字符串，需要进行转换
		codes := strings.Split(role.Codes, ",")
		permCodes = append(permCodes, codes...)
	}

	return permCodes, nil
}

func (u *userDAO) GetUserByUsernames(ctx context.Context, usernames []string) ([]*model.User, error) {
	var users []*model.User

	if err := u.db.WithContext(ctx).Where("username in (?)", usernames).Find(&users).Error; err != nil {
		u.l.Error("get user by username failed", zap.Strings("usernames", usernames), zap.Error(err))
		return nil, err
	}

	return users, nil
}
