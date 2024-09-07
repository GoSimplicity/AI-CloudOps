package dao

import (
	"context"
	"errors"

	"github.com/GoSimplicity/CloudOps/internal/constants"
	"github.com/GoSimplicity/CloudOps/internal/model"
	"github.com/go-sql-driver/mysql"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type UserDAO interface {
	// CreateUser 新建用户
	CreateUser(ctx context.Context, user *model.User) error
	// GetUserByUsername 通过用户名获取用户
	GetUserByUsername(ctx context.Context, username string) (*model.User, error)
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
		if errors.As(err, &mysqlErr) && mysqlErr.Number == constants.ErrCodeDuplicateUserNameOrMobileNumber {
			return constants.ErrCodeDuplicateUserNameOrMobile
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

	if err := u.db.WithContext(ctx).Where("id = ?", id).First(&user).Error; err != nil {
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
