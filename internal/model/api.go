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

type Api struct {
	ID          int    `json:"id" gorm:"primaryKey;column:id;comment:主键ID"`
	Name        string `json:"name" gorm:"column:name;type:varchar(50);not null;comment:API名称"`
	Path        string `json:"path" gorm:"column:path;type:varchar(255);not null;comment:API路径"`
	Method      int    `json:"method" gorm:"column:method;type:tinyint(1);not null;comment:HTTP请求方法(1:GET,2:POST,3:PUT,4:DELETE)"`
	Description string `json:"description" gorm:"column:description;type:varchar(500);comment:API描述"`
	Version     string `json:"version" gorm:"column:version;type:varchar(20);default:v1;comment:API版本"`
	Category    int    `json:"category" gorm:"column:category;type:tinyint(1);not null;comment:API分类(1:系统,2:业务)"`
	IsPublic    int    `json:"is_public" gorm:"column:is_public;type:tinyint(1);default:0;comment:是否公开(0:否,1:是)"`
	CreateTime  int64  `json:"create_time" gorm:"column:create_time;autoCreateTime;comment:创建时间"`
	UpdateTime  int64  `json:"update_time" gorm:"column:update_time;autoUpdateTime;comment:更新时间"`
	IsDeleted   int    `json:"is_deleted" gorm:"column:is_deleted;type:tinyint(1);default:0;comment:是否删除(0:否,1:是)"`
}

type CreateApiRequest struct {
	Name        string `json:"name" binding:"required"`       // API名称
	Path        string `json:"path" binding:"required"`       // API路径
	Method      int    `json:"method" binding:"required"`     // 请求方法
	Description string `json:"description"`                   // API描述
	Version     string `json:"version"`                       // API版本
	Category    int    `json:"category"`                      // API分类
	IsPublic    int    `json:"is_public" binding:"oneof=0 1"` // 是否公开
}

type GetApiRequest struct {
	ID int `json:"id" binding:"required,gt=0"` // API ID
}

type UpdateApiRequest struct {
	ID          int    `json:"id" binding:"required,gt=0"`    // API ID
	Name        string `json:"name" binding:"required"`       // API名称
	Path        string `json:"path" binding:"required"`       // API路径
	Method      int    `json:"method" binding:"required"`     // 请求方法
	Description string `json:"description"`                   // API描述
	Version     string `json:"version"`                       // API版本
	Category    int    `json:"category"`                      // API分类
	IsPublic    int    `json:"is_public" binding:"oneof=0 1"` // 是否公开
}

type ListApisRequest struct {
	PageNumber int `json:"page_number" binding:"required,gt=0"` // 页码
	PageSize   int `json:"page_size" binding:"required,gt=0"`   // 每页数量
}
