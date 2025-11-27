package utils

import "github.com/GoSimplicity/AI-CloudOps/internal/model"

// BuildUserForCreate 将注册请求转换为可持久化的用户实体
func BuildUserForCreate(req *model.UserSignUpReq, hashedPassword string) *model.User {
	return &model.User{
		Username:     req.Username,
		Password:     hashedPassword,
		RealName:     req.RealName,
		Desc:         req.Desc,
		Mobile:       req.Mobile,
		FeiShuUserId: req.FeiShuUserId,
		AccountType:  req.AccountType,
		HomePath:     req.HomePath,
		Enable:       req.Enable,
	}
}

// ApplyProfileUpdates 应用可编辑的用户资料字段
func ApplyProfileUpdates(user *model.User, req *model.UpdateProfileReq) {
	user.RealName = req.RealName
	user.Desc = req.Desc
	user.Mobile = req.Mobile
	user.FeiShuUserId = req.FeiShuUserId
	user.AccountType = req.AccountType
	user.HomePath = req.HomePath
	user.Enable = req.Enable
	user.Email = req.Email
	user.Avatar = req.Avatar
}
