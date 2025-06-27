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
	Model
	Name        string  `json:"name" gorm:"type:varchar(50);uniqueIndex:idx_name_del;not null;comment:API名称"`       // API名称，唯一且非空
	Path        string  `json:"path" gorm:"type:varchar(255);not null;comment:API路径"`                               // API路径，非空
	Method      int8    `json:"method" gorm:"type:tinyint(1);not null;comment:HTTP请求方法 1GET 2POST 3PUT 4DELETE"`    // 请求方法，使用int8节省空间
	Description string  `json:"description" gorm:"type:varchar(500);comment:API描述"`                                 // API描述
	Version     string  `json:"version" gorm:"type:varchar(20);default:v1;comment:API版本"`                           // API版本，默认v1
	Category    int8    `json:"category" gorm:"type:tinyint(1);not null;comment:API分类 1系统 2业务" binding:"oneof=1 2"` // API分类，使用int8节省空间
	IsPublic    int8    `json:"is_public" gorm:"type:tinyint(1);default:0;comment:是否公开 0否 1是" binding:"oneof=0 1"`  // 是否公开，使用int8节省空间
	Users       []*User `json:"users" gorm:"many2many:user_apis;comment:关联用户"`                                      // 多对多关联用户
}

type CreateApiRequest struct {
	Name        string `json:"name" binding:"required"`       // API名称
	Path        string `json:"path" binding:"required"`       // API路径
	Method      int    `json:"method" binding:"required"`     // 请求方法
	Description string `json:"description"`                   // API描述
	Version     string `json:"version"`                       // API版本
	Category    int    `json:"category"`                      // API分类
	IsPublic    int    `json:"is_public" binding:"oneof=1 2"` // 是否公开
}

type UpdateApiRequest struct {
	ID          int    `json:"id" binding:"required,gt=0"`    // API ID
	Name        string `json:"name" binding:"required"`       // API名称
	Path        string `json:"path" binding:"required"`       // API路径
	Method      int    `json:"method" binding:"required"`     // 请求方法
	Description string `json:"description"`                   // API描述
	Version     string `json:"version"`                       // API版本
	Category    int    `json:"category"`                      // API分类
	IsPublic    int    `json:"is_public" binding:"oneof=1 2"` // 是否公开
}

type DeleteApiRequest struct {
	ID int `json:"id" binding:"required,gt=0"` // API ID
}

type GetApiRequest struct {
	ID int `json:"id" binding:"required,gt=0"` // API ID
}

type ListApisRequest struct {
	ListReq
	IsPublic int `json:"is_public" form:"is_public"` // 是否公开
	Method   int `json:"method" form:"method"`       // 请求方法
}

type ApiStatistics struct {
	PublicCount  int64 `json:"public_count"`  // 公开API数量
	PrivateCount int64 `json:"private_count"` // 私有API数量
}
