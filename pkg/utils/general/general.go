package general

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
	"time"
)

func ConvertToIntList(stringList []string) ([]int, error) {
	intList := make([]int, 0, len(stringList))
	for _, idStr := range stringList {
		id, err := strconv.Atoi(strings.TrimSpace(idStr)) // 去除空白并转换为整数
		if err != nil {
			return nil, fmt.Errorf("无法解析 leafNodeId: '%s' 为整数", idStr)
		}
		intList = append(intList, id)
	}

	return intList, nil
}

// IsType 判断两个值是否是相同类型
func IsType(value1, value2 interface{}) bool {
	return reflect.TypeOf(value1) == reflect.TypeOf(value2)
}

// GetDefaultValue 返回值的默认值（零值）
func GetDefaultValue(value interface{}) interface{} {
	v := reflect.ValueOf(value)
	if !v.IsValid() {
		return nil
	}

	// 创建零值的副本并返回
	return reflect.Zero(v.Type()).Interface()
}

// GetMax 返回两个数值中的最大值，支持 int, float64 等常见类型
func GetMax(value1, value2 interface{}) (interface{}, error) {
	switch v1 := value1.(type) {
	case int:
		v2, ok := value2.(int)
		if !ok {
			return nil, errors.New("类型不匹配")
		}
		if v1 > v2 {
			return v1, nil
		}
		return v2, nil
	case float64:
		v2, ok := value2.(float64)
		if !ok {
			return nil, errors.New("类型不匹配")
		}
		return math.Max(v1, v2), nil
	default:
		return nil, errors.New("不支持的类型")
	}
}

// GetMin 返回两个数值中的最小值，支持 int, float64 等常见类型
func GetMin(value1, value2 interface{}) (interface{}, error) {
	switch v1 := value1.(type) {
	case int:
		v2, ok := value2.(int)
		if !ok {
			return nil, errors.New("类型不匹配")
		}
		if v1 < v2 {
			return v1, nil
		}
		return v2, nil
	case float64:
		v2, ok := value2.(float64)
		if !ok {
			return nil, errors.New("类型不匹配")
		}
		return math.Min(v1, v2), nil
	default:
		return nil, errors.New("不支持的类型")
	}
}

// ToUpperCase 将字符串转换为大写
func ToUpperCase(str string) string {
	return strings.ToUpper(str)
}

// ToLowerCase 将字符串转换为小写
func ToLowerCase(str string) string {
	return strings.ToLower(str)
}

// TrimSpaces 去掉字符串的前后空格
func TrimSpaces(str string) string {
	return strings.TrimSpace(str)
}

// IsSameDay 判断两个日期是否为同一天
func IsSameDay(t1, t2 time.Time) bool {
	y1, m1, d1 := t1.Date()
	y2, m2, d2 := t2.Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}

// DaysBetween 计算两个日期之间的天数
func DaysBetween(t1, t2 time.Time) int {
	days := t2.Sub(t1).Hours() / 24
	return int(math.Abs(days))
}

// IsValidEmail 简单检查一个字符串是否是有效的电子邮件格式
func IsValidEmail(email string) bool {
	return strings.Contains(email, "@") && strings.Contains(email, ".")
}
