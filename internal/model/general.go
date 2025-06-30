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
	Page   int    `json:"page" form:"page" binding:"required,min=1"`
	Size   int    `json:"size" form:"size" binding:"required,min=10,max=100"`
	Search string `json:"search" form:"search" binding:"omitempty"`
}

// ListResp 通用列表响应
type ListResp[T any] struct {
	Items []T   `json:"items"` // 数据列表
	Total int64 `json:"total"` // 总数
}

type StringList []string

// Scan 从数据库值转换为 StringList
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

	if str == "" {
		*m = StringList{}
		return nil
	}

	*m = strings.Split(str, "|")
	return nil
}

// Value 将 StringList 转换为数据库值
func (m StringList) Value() (driver.Value, error) {
	return strings.Join(m, "|"), nil
}

// MarshalJSON 将 StringList 序列化为 JSON
func (m StringList) MarshalJSON() ([]byte, error) {
	return json.Marshal([]string(m))
}

// UnmarshalJSON 将 JSON 反序列化为 StringList
func (m *StringList) UnmarshalJSON(data []byte) error {
	var ss []string
	if err := json.Unmarshal(data, &ss); err != nil {
		return err
	}
	*m = StringList(ss)
	return nil
}
