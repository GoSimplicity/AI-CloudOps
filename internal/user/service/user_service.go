package service

import (
	"context"
	"errors"
	"github.com/GoSimplicity/AI-CloudOps/internal/constants"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/user/dao"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

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

type UserService interface {
	SignUp(ctx context.Context, user *model.User) error
	Login(ctx context.Context, user *model.User) (*model.User, error)
	GetProfile(ctx context.Context, uid int) (*model.User, error)
	GetPermCode(ctx context.Context, uid int) ([]string, error)
	GetUserList(ctx context.Context) ([]*model.User, error)
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

func (us *userService) GetUserList(ctx context.Context) ([]*model.User, error) {
	return us.dao.GetAllUsers(ctx)
}
