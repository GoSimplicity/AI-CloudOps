package service

import (
	"context"
	"errors"
	"github.com/GoSimplicity/CloudOps/internal/model"
	"github.com/GoSimplicity/CloudOps/internal/user/dao"
	"gorm.io/gorm"
)

type UserService interface {
	Create(ctx context.Context, user *model.User) error
}

type userService struct {
	dao dao.UserDAO
}

func NewUserService(dao dao.UserDAO) UserService {
	return &userService{
		dao: dao,
	}
}

func (u *userService) Create(ctx context.Context, user *model.User) error {
	_, err := u.dao.GetUserByUsername(ctx, user.Username)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	return u.dao.CreateUser(ctx, user)
}
