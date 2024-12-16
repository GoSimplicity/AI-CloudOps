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

package api

type TokenRequest struct {
	RefreshToken string `json:"refreshToken"`
}

type ChangePasswordRequest struct {
	Username        string `json:"username"`
	Password        string `json:"password"`
	NewPassword     string `json:"newPassword"`
	ConfirmPassword string `json:"confirmPassword"`
}

type WriteOffRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type UpdateProfileRequest struct {
	UserId       int    `json:"user_id"`
	RealName     string `json:"real_name"`
	Desc         string `json:"desc"`
	Mobile       string `json:"mobile"`
	FeiShuUserId string `json:"fei_shu_user_id"`
	AccountType  int    `json:"account_type"`
	HomePath     string `json:"home_path"`
	Enable       int    `json:"enable"`
}

type DeleteUserRequest struct {
	UserId int `json:"user_id"`
}
