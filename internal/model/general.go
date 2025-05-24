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
	Status *int8  `json:"status" form:"status" binding:"omitempty"`
}

// 通用分页请求
type PageRequest struct {
}
type ListResponse struct {
	Total int         `json:"total"`
	Items interface{} `json:"items"`
}

type StringList []string

func (m *StringList) Scan(val interface{}) error {
	s := val.([]uint8)
	ss := strings.Split(string(s), "|")
	*m = ss
	return nil
}

func (m StringList) Value() (driver.Value, error) {
	str := strings.Join(m, "|")
	return str, nil
}
