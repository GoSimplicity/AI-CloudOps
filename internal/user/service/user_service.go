package service

import (
	"context"
	"github.com/GoSimplicity/CloudOps/internal/model"
	"github.com/GoSimplicity/CloudOps/internal/user/dao"
	"github.com/GoSimplicity/CloudOps/internal/user/dto"
)

type UserService interface {
	Create(ctx context.Context, user dto.UserDTO) error
}

type userService struct {
	dao dao.UserDAO
}

func NewUserService(dao dao.UserDAO) UserService {
	return &userService{
		dao: dao,
	}
}

func (u *userService) Create(ctx context.Context, user dto.UserDTO) error {
	return u.dao.Create(ctx, u.toUserDAO(user))
}

func (u *userService) toUserDAO(user dto.UserDTO) model.User {
	return model.User{
		UserId:      user.UserID,
		Username:    user.UserName,
		Password:    user.PassWord,
		RealName:    user.RealName,
		Desc:        user.Desc,
		Mobile:      user.Mobile,
		LarkUserId:  user.LarkUserID,
		AccountType: user.AccountType,
		HomePath:    user.HomePath,
		Enable:      user.Enable,
	}
}
