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

package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gorm.io/plugin/soft_delete"
)

type Model struct {
	ID        int                   `json:"id" gorm:"primaryKey;autoIncrement;comment:主键ID"`
	CreatedAt time.Time             `json:"created_at" gorm:"autoCreateTime;comment:创建时间"`
	UpdatedAt time.Time             `json:"updated_at" gorm:"autoUpdateTime;comment:更新时间"`
	DeletedAt soft_delete.DeletedAt `json:"deleted_at" gorm:"index;comment:删除时间"`
}

// ListReq 列表请求
type ListReq struct {
	Page   int    `json:"page" form:"page" binding:"omitempty,min=1"`
	Size   int    `json:"size" form:"size" binding:"omitempty,min=10,max=100"`
	Search string `json:"search" form:"search" binding:"omitempty"`
}

// ListResp 列表响应
type ListResp[T any] struct {
	Items []T   `json:"items"` // 数据列表
	Total int64 `json:"total"` // 总数
}

type StringList []string

// Scan 从数据库转换
func (m *StringList) Scan(val interface{}) error {
	if val == nil {
		*m = StringList{}
		return nil
	}

	var str string
	switch v := val.(type) {
	case []byte:
		str = string(v)
	case string:
		str = v
	default:
		return fmt.Errorf("无法扫描 %T 到 StringList", val)
	}

	if str == "" || str == "[]" {
		*m = StringList{}
		return nil
	}

	// 尝试JSON解析
	var slice []string
	if err := json.Unmarshal([]byte(str), &slice); err == nil {
		*m = StringList(slice)
		return nil
	}

	// 如果JSON解析失败，尝试向后兼容的分割方式
	// 清理字符串，移除可能的引号和空格
	cleanStr := strings.TrimSpace(str)
	cleanStr = strings.Trim(cleanStr, `"'`)

	if cleanStr == "" {
		*m = StringList{}
		return nil
	}

	// 尝试按逗号分割
	if strings.Contains(cleanStr, ",") {
		parts := strings.Split(cleanStr, ",")
		for i, part := range parts {
			parts[i] = strings.TrimSpace(strings.Trim(part, `"'`))
		}
		*m = StringList(parts)
		return nil
	}

	// 尝试按竖线分割（向后兼容）
	if strings.Contains(cleanStr, "|") {
		*m = StringList(strings.Split(cleanStr, "|"))
		return nil
	}

	// 如果都失败了，将整个字符串作为一个元素
	*m = StringList{cleanStr}
	return nil
}

// Value 转换为数据库值
func (m StringList) Value() (driver.Value, error) {
	if len(m) == 0 {
		return "[]", nil
	}
	data, err := json.Marshal([]string(m))
	if err != nil {
		return nil, err
	}
	return string(data), nil
}

// MarshalJSON 序列化JSON
func (m StringList) MarshalJSON() ([]byte, error) {
	return json.Marshal([]string(m))
}

// UnmarshalJSON 反序列化JSON
func (m *StringList) UnmarshalJSON(data []byte) error {
	var ss []string
	if err := json.Unmarshal(data, &ss); err != nil {
		return err
	}
	*m = StringList(ss)
	return nil
}

// JSONMap JSON类型
type JSONMap map[string]interface{}

// Value 转为JSON存储
func (m JSONMap) Value() (driver.Value, error) {
	if m == nil {
		return nil, nil
	}
	return json.Marshal(m)
}

// Scan 从数据库转换JSON
func (m *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*m = nil
		return nil
	}

	var data []byte
	switch v := value.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return fmt.Errorf("无法扫描 %T 到 JSONMap", value)
	}

	return json.Unmarshal(data, m)
}
