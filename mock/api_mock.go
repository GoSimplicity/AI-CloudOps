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

package mock

import (
	"log"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"gorm.io/gorm"
)

type ApiMock struct {
	db *gorm.DB
}

func NewApiMock(db *gorm.DB) *ApiMock {
	return &ApiMock{
		db: db,
	}
}

func (m *ApiMock) InitApi() error {
	// 检查是否已经初始化过API
	var count int64
	m.db.Model(&model.Api{}).Count(&count)
	if count > 0 {
		log.Println("[API已经初始化过,跳过Mock]")
		return nil
	}

	log.Println("[API Mock开始]")

	apis := []model.Api{
		{Model: model.Model{ID: 1}, Path: "/*", Method: 1, Name: "所有接口GET权限", Description: "所有接口GET权限", Version: "v1", Category: 1, IsPublic: 1},
		{Model: model.Model{ID: 2}, Path: "/*", Method: 2, Name: "所有接口POST权限", Description: "所有接口POST权限", Version: "v1", Category: 1, IsPublic: 1},
		{Model: model.Model{ID: 3}, Path: "/*", Method: 3, Name: "所有接口PUT权限", Description: "所有接口PUT权限", Version: "v1", Category: 1, IsPublic: 1},
		{Model: model.Model{ID: 4}, Path: "/*", Method: 4, Name: "所有接口DELETE权限", Description: "所有接口DELETE权限", Version: "v1", Category: 1, IsPublic: 1}}

	for _, api := range apis {
		if err := m.db.Create(&api).Error; err != nil {
			// 使用FirstOrCreate方法,如果记录存在则跳过,不存在则创建
			result := m.db.Where("id = ?", api.ID).FirstOrCreate(&api)
			if result.Error != nil {
				return result.Error
			}

			if result.RowsAffected == 1 {
				log.Printf("创建API [%s] 成功", api.Name)
			} else {
				log.Printf("API [%s] 已存在,跳过创建", api.Name)
			}
		}
	}

	log.Println("[API Mock结束]")

	return nil
}
