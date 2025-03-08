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

package uesr

import (
	"context"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type AppDAO interface {
	CreateAppOne(ctx context.Context, app *model.K8sApp) error
	GetAppById(ctx context.Context, id int64) (model.K8sApp, error)
}
type appDAO struct {
	db *gorm.DB
	l  *zap.Logger
}

func NewAppDAO(db *gorm.DB, l *zap.Logger) AppDAO {
	return &appDAO{
		db: db,
		l:  l,
	}
}
func (a *appDAO) CreateAppOne(ctx context.Context, app *model.K8sApp) error {
	if err := a.db.WithContext(ctx).Create(app).Error; err != nil {
		a.l.Error("CreateAppOne 创建k8sApp失败", zap.Error(err))
		return err
	}
	return nil
}
func (a *appDAO) GetAppById(ctx context.Context, id int64) (model.K8sApp, error) {
	var app model.K8sApp
	err := a.db.WithContext(ctx).
		Where("id = ?", id).
		First(&app).
		Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			a.l.Warn("GetAppById 应用不存在", zap.Int64("appId", id))
			return model.K8sApp{}, gorm.ErrRecordNotFound
		}
		a.l.Error("GetAppById 获取应用失败", zap.Int64("appId", id), zap.Error(err))
		return model.K8sApp{}, err
	}
	return app, nil
}
