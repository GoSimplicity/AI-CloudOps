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

type InstanceDAO interface {
	CreateInstanceOne(ctx context.Context, instance *model.K8sInstance) error
	GetInstanceAll(ctx context.Context) ([]model.K8sInstance, error)
	GetInstanceByApp(ctx context.Context, AppId int64) ([]model.K8sInstance, error)
	GetInstanceById(ctx context.Context, instanceId int64) (model.K8sInstance, error)
}
type instanceDAO struct {
	db *gorm.DB
	l  *zap.Logger
}

func NewInstanceDAO(db *gorm.DB, l *zap.Logger) InstanceDAO {
	return &instanceDAO{
		db: db,
		l:  l,
	}
}

func (i *instanceDAO) CreateInstanceOne(ctx context.Context, instance *model.K8sInstance) error {
	if err := i.db.WithContext(ctx).Create(instance).Error; err != nil {
		i.l.Error("CreateInstanceOne 创建Instance任务失败", zap.Error(err), zap.Any("instance", instance))
		return err
	}

	return nil
}
func (i *instanceDAO) GetInstanceAll(ctx context.Context) ([]model.K8sInstance, error) {
	var instances []model.K8sInstance
	if err := i.db.WithContext(ctx).Find(&instances).Error; err != nil {
		i.l.Error("GetInstanceAll 获取Instance任务失败", zap.Error(err))
		return nil, err
	}
	return instances, nil
}
func (i *instanceDAO) GetInstanceByApp(ctx context.Context, AppId int64) ([]model.K8sInstance, error) {
	var instances []model.K8sInstance
	if err := i.db.WithContext(ctx).Where("k8s_app_id = ?", AppId).Find(&instances).Error; err != nil {
		i.l.Error("GetInstanceByApp 获取Instance任务失败", zap.Error(err))
	}
	return instances, nil
}

func (i *instanceDAO) GetInstanceById(ctx context.Context, instanceId int64) (model.K8sInstance, error) {
	var instance model.K8sInstance
	if err := i.db.WithContext(ctx).Where("id =?", instanceId).Find(&instance).Error; err != nil {
		i.l.Error("GetInstanceById 获取Instance任务失败", zap.Error(err))
	}
	return instance, nil
}
