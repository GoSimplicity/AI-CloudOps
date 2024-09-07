package service

import (
	"context"

	"golang.org/x/crypto/bcrypt"

	"github.com/GoSimplicity/CloudOps/internal/model"
	"github.com/GoSimplicity/CloudOps/internal/user/dao"
	"github.com/GoSimplicity/CloudOps/internal/user/dto"
)

type UserService interface {
	SignUp(ctx context.Context, user dto.UserDTO) error
	Login(ctx context.Context, user dto.UserDTO) error
	Profile(ctx context.Context, user dto.UserDTO) (dto.UserDTO, error)
}

type userService struct {
	dao dao.UserDAO
}

func NewUserService(dao dao.UserDAO) UserService {
	return &userService{
		dao: dao,
	}
}

func (u *userService) SignUp(ctx context.Context, user dto.UserDTO) error {
	// 验证用户名, 手机号唯一性
	// exist, err := u.dao.GetUserByUsername(ctx, user.UserName)
	// if err != gorm.ErrRecordNotFound && err != nil {
	// 	return Result{
	// 		Code: constants.UserSignUpFailedErrorCode,
	// 		Msg:  constants.ErrorUserSignUpFail.Error(),
	// 	}, err
	// }
	// if exist != nil {
	// 	return Result{
	// 		Code: constants.UserExistErrorCode,
	// 		Msg:  constants.ErrorUserExist.Error(),
	// 	}, err
	// }
	// exist, err = u.dao.GetUserByMobile(ctx, user.Mobile)
	// if err != gorm.ErrRecordNotFound && err != nil {
	// 	return Result{
	// 		Code: constants.UserSignUpFailedErrorCode,
	// 		Msg:  constants.ErrorUserSignUpFail.Error(),
	// 	}, err
	// }
	// if exist != nil {
	// 	return Result{
	// 		Code: constants.UserExistErrorCode,
	// 		Msg:  constants.ErrorUserExist.Error(),
	// 	}, err
	// }

	// 加密
	hash, err := bcrypt.GenerateFromPassword([]byte(user.PassWord), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user.PassWord = string(hash)

	return u.dao.CreateUser(ctx, u.toUserDAO(user))
}

func (u *userService) Login(ctx context.Context, user dto.UserDTO) error {
	// username, password
	return nil
}

func (u *userService) Profile(ctx context.Context, user dto.UserDTO) (dto.UserDTO, error) {

	return dto.UserDTO{}, nil
}

func (u *userService) toUserDAO(user dto.UserDTO) *model.User {
	return &model.User{
		// UserId:      user.UserID,
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
