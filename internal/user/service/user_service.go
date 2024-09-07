package service

import (
	"context"

	"golang.org/x/crypto/bcrypt"

	"github.com/GoSimplicity/CloudOps/internal/model"
	"github.com/GoSimplicity/CloudOps/internal/user/dao"
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
	un, err := u.dao.GetUserByUsername(ctx, user.Username)
	if err != nil && un != nil {
		return err
	}

	return u.dao.CreateUser(ctx, user)
}
