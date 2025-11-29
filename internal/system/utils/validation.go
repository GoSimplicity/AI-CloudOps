package utils

import (
	"fmt"
	"strings"
)

// RequireNonEmpty 校验字符串是否为空
func RequireNonEmpty(value, field string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s不能为空", field)
	}
	return nil
}

// RequirePositiveID 校验 ID 是否大于零
func RequirePositiveID(id int, field string) error {
	if id <= 0 {
		return fmt.Errorf("%s必须为正整数", field)
	}
	return nil
}
