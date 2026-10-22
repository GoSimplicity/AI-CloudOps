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
		Email:        req.Email,
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
	user.HomePath = req.HomePath
	user.Email = req.Email
	user.Avatar = req.Avatar
}
