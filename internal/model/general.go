package model

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

import (
	"gorm.io/plugin/soft_delete"
	"time"
)

type Model struct {
	ID        int                   `json:"id" gorm:"primaryKey;autoIncrement"`     // 主键，自增
	CreatedAt time.Time             `json:"created_at" gorm:"autoCreateTime"`       // 自动记录创建时间
	UpdatedAt time.Time             `json:"updated_at" gorm:"autoUpdateTime"`       // 自动记录更新时间
	DeletedAt soft_delete.DeletedAt `json:"deleted_at" gorm:"uniqueIndex:udx_name"` // 软删除字段，自动管理
}

type NoUniqueIndexModel struct {
	ID        int                   `json:"id" gorm:"primaryKey;autoIncrement"` // 主键，自增
	CreatedAt time.Time             `json:"created_at" gorm:"autoCreateTime"`   // 自动记录创建时间
	UpdatedAt time.Time             `json:"updated_at" gorm:"autoUpdateTime"`   // 自动记录更新时间
	DeletedAt soft_delete.DeletedAt `json:"deleted_at" gorm:"index"`            // 软删除字段，自动管理
}
