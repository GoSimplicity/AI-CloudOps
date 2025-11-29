package utils

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// HashPassword 对密码进行统一校验与加密
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
