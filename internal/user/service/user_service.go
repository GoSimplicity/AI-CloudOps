package service

import (
	"context"
	"errors"
	"github.com/GoSimplicity/CloudOps/internal/constants"

	"github.com/GoSimplicity/CloudOps/internal/model"
	"github.com/GoSimplicity/CloudOps/internal/user/dao"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserService interface {
	SignUp(ctx context.Context, user *model.User) error
	Login(ctx context.Context, user *model.User) (*model.User, error)
	GetProfile(ctx context.Context, uid int) (*model.User, error)
	GetPermCode(ctx context.Context, uid int) ([]string, error)
}

type userService struct {
	dao dao.UserDAO
}

func NewUserService(dao dao.UserDAO) UserService {
	return &userService{
		dao: dao,
	}
}

func (us *userService) SignUp(ctx context.Context, user *model.User) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user.Password = string(hash)

	return us.dao.CreateUser(ctx, user)
}

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

func (us *userService) GetProfile(ctx context.Context, uid int) (*model.User, error) {
	return us.dao.GetUserByID(ctx, uid)
}

func (us *userService) GetPermCode(ctx context.Context, uid int) ([]string, error) {
	codes, err := us.dao.GetPermCode(ctx, uid)
	if err != nil {
		return nil, err
	}

	return codes, nil
}
