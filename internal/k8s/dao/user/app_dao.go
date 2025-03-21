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

package user

import (
	"context"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type AppDAO interface {
	CreateAppOne(ctx context.Context, app *model.K8sApp) error
	GetAppById(ctx context.Context, id int64) (model.K8sApp, error)
	DeleteAppById(ctx context.Context, id int64) (model.K8sApp, error)
	UpdateAppById(ctx context.Context, id int64, app model.K8sApp) error
	GetAppsByProjectId(ctx context.Context, ids int64) ([]model.K8sApp, error)
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

func (a *appDAO) DeleteAppById(ctx context.Context, id int64) (model.K8sApp, error) {
	var app model.K8sApp
	// 先查询记录
	if err := a.db.WithContext(ctx).Where("id = ?", id).First(&app).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			a.l.Warn("DeleteAppById 应用不存在", zap.Int64("appId", id))
			return model.K8sApp{}, gorm.ErrRecordNotFound
		}
		a.l.Error("DeleteAppById 查询应用失败", zap.Int64("appId", id), zap.Error(err))
		return model.K8sApp{}, err
	}

	// 执行软删除（更新deleted_at）
	if err := a.db.WithContext(ctx).Model(&app).Update("deleted_at", 1).Error; err != nil {
		a.l.Error("DeleteAppById 更新删除状态失败", zap.Int64("appId", id), zap.Error(err))
		return model.K8sApp{}, err
	}

	return app, nil
}

//	func (a *appDAO) UpdateAppById(ctx context.Context, id int64, app model.K8sApp) error {
//		result := a.db.WithContext(ctx).
//			Model(&model.K8sApp{}).
//			Where("id = ?", id).
//			Updates(app)
//
//		if result.Error != nil {
//			a.l.Error("UpdateAppById 更新应用失败",
//				zap.Int64("appId", id),
//				zap.Error(result.Error))
//			return result.Error
//		}
//
//		if result.RowsAffected == 0 {
//			a.l.Warn("UpdateAppById 应用不存在", zap.Int64("appId", id))
//			return gorm.ErrRecordNotFound
//		}
//
//		return nil
//	}
func (a *appDAO) UpdateAppById(ctx context.Context, id int64, app model.K8sApp) error {
	return a.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. 先更新 K8sApp
		result := tx.Model(&model.K8sApp{}).Where("id = ?", id).Updates(app)
		if result.Error != nil {
			a.l.Error("UpdateAppById 更新应用失败", zap.Int64("appId", id), zap.Error(result.Error))
			return result.Error
		}
		if result.RowsAffected == 0 {
			a.l.Warn("UpdateAppById 应用不存在", zap.Int64("appId", id))
			return gorm.ErrRecordNotFound
		}

		// 2. 级联更新 K8sInstances
		for _, instance := range app.K8sInstances {
			instance.K8sAppID = int(id) // 确保实例正确关联到 K8sApp
			if err := tx.Model(&model.K8sInstance{}).
				Where("id = ? AND k8s_app_id = ?", instance.ID, id). // 确保只更新该实例
				Updates(instance).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// GetAppsByProjectId 根据多个 Project ID 查询关联的 K8sApp
func (a *appDAO) GetAppsByProjectId(ctx context.Context, ids int64) ([]model.K8sApp, error) {
	// 如果传入的 ID 列表为空，直接返回空结果

	var apps []model.K8sApp
	err := a.db.WithContext(ctx).Where("k8s_project_id = ?", ids).Find(&apps).Error
	if err != nil {
		return nil, err
	}
	return apps, nil
}
