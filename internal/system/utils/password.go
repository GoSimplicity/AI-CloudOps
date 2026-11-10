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

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// loginTimingHash 只用于登录失败时做一次与真实口令相同 cost 的比较，不对应任何账号。
var loginTimingHash = newLoginTimingHash()

func newLoginTimingHash() string {
	hash, err := bcrypt.GenerateFromPassword([]byte{0xc1, 0x0d, 0x09, 0x5, 0x11}, bcrypt.DefaultCost)
	if err != nil {
		panic(err)
	}
	return string(hash)
}

// LoginTimingHash 返回登录耗时对齐用的 bcrypt 摘要。
func LoginTimingHash() string {
	return loginTimingHash
}

func HashPassword(password string) (string, error) {
	if err := RequireNonEmpty(password, "密码"); err != nil {
		return "", err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("生成密码失败: %w", err)
	}

	return string(hash), nil
}

// ComparePassword 对比明文密码与密文密码
func ComparePassword(hashedPassword, rawPassword string) error {
	if err := RequireNonEmpty(rawPassword, "密码"); err != nil {
		return err
	}

	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(rawPassword))
}
