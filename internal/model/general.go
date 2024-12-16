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
	"time"

	"gorm.io/plugin/soft_delete"
)

type Model struct {
	ID        int                   `json:"id" gorm:"primaryKey;autoIncrement;comment:主键ID"` // 主键ID，自增
	CreatedAt time.Time             `json:"created_at" gorm:"autoCreateTime;comment:创建时间"`   // 创建时间，自动记录
	UpdatedAt time.Time             `json:"updated_at" gorm:"autoUpdateTime;comment:更新时间"`   // 更新时间，自动记录
	DeletedAt soft_delete.DeletedAt `json:"deleted_at" gorm:"index;comment:删除时间"`            // 软删除时间，使用普通索引
}
