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

package dao

import (
	"context"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type ClusterDAO interface {
	ListAllClusters(ctx context.Context) ([]*model.K8sCluster, error)
	CreateCluster(ctx context.Context, cluster *model.K8sCluster) error
	UpdateCluster(ctx context.Context, cluster *model.K8sCluster) error
	UpdateClusterStatus(ctx context.Context, id int, status string) error
	DeleteCluster(ctx context.Context, id int) error
	GetClusterByID(ctx context.Context, id int) (*model.K8sCluster, error)
	GetClusterByName(ctx context.Context, name string) (*model.K8sCluster, error)
	BatchDeleteClusters(ctx context.Context, ids []int) error
}

type clusterDAO struct {
	db *gorm.DB
	l  *zap.Logger
}

func NewClusterDAO(db *gorm.DB, l *zap.Logger) ClusterDAO {
	return &clusterDAO{
		db: db,
		l:  l,
	}
}

// ListAllClusters 获取所有集群
func (c *clusterDAO) ListAllClusters(ctx context.Context) ([]*model.K8sCluster, error) {
	var clusters []*model.K8sCluster

	if err := c.db.WithContext(ctx).Find(&clusters).Error; err != nil {
		c.l.Error("ListAllClusters 查询所有集群失败", zap.Error(err))
		return nil, err
	}

	return clusters, nil
}

// CreateCluster 创建集群
func (c *clusterDAO) CreateCluster(ctx context.Context, cluster *model.K8sCluster) error {
	if err := c.db.WithContext(ctx).Create(cluster).Error; err != nil {
		c.l.Error("CreateCluster 创建集群失败", zap.Error(err))
		return err
	}

	return nil
}

// UpdateCluster 更新集群
func (c *clusterDAO) UpdateCluster(ctx context.Context, cluster *model.K8sCluster) error {
	tx := c.db.WithContext(ctx).Begin()

	if err := tx.Model(cluster).Where("id = ?", cluster.ID).Updates(cluster).Error; err != nil {
		tx.Rollback()
		c.l.Error("UpdateCluster 更新集群失败", zap.Int("id", cluster.ID), zap.Error(err))
		return err
	}

	tx.Commit()
	return nil
}

// UpdateClusterStatus 更新集群状态
func (c *clusterDAO) UpdateClusterStatus(ctx context.Context, id int, status string) error {
	if err := c.db.WithContext(ctx).Model(&model.K8sCluster{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status": status,
	}).Error; err != nil {
		c.l.Error("UpdateClusterStatus 更新集群状态失败", zap.Int("id", id), zap.String("status", status), zap.Error(err))
		return err
	}

	return nil
}

// DeleteCluster 删除集群
func (c *clusterDAO) DeleteCluster(ctx context.Context, id int) error {
	if err := c.db.WithContext(ctx).Where("id = ?", id).Delete(&model.K8sCluster{}).Error; err != nil {
		c.l.Error("DeleteCluster 删除集群失败", zap.Int("id", id), zap.Error(err))
		return err
	}

	return nil
}

// GetClusterByID 根据ID获取集群
func (c *clusterDAO) GetClusterByID(ctx context.Context, id int) (*model.K8sCluster, error) {
	var cluster model.K8sCluster

	if err := c.db.WithContext(ctx).Where("id = ?", id).First(&cluster).Error; err != nil {
		c.l.Error("GetClusterByID 查询集群失败", zap.Int("id", id), zap.Error(err))
		return nil, err
	}

	return &cluster, nil
}

// GetClusterByName 根据名称获取集群
func (c *clusterDAO) GetClusterByName(ctx context.Context, name string) (*model.K8sCluster, error) {
	var cluster model.K8sCluster

	if err := c.db.WithContext(ctx).Where("name = ?", name).First(&cluster).Error; err != nil {
		c.l.Error("GetClusterByName 查询集群失败", zap.String("name", name), zap.Error(err))
		return nil, err
	}

	return &cluster, nil
}

// BatchDeleteClusters 批量删除集群
func (c *clusterDAO) BatchDeleteClusters(ctx context.Context, ids []int) error {
	if err := c.db.WithContext(ctx).Where("id IN ?", ids).Delete(&model.K8sCluster{}).Error; err != nil {
		c.l.Error("BatchDeleteClusters 批处理删除集群失败", zap.Error(err))
		return err
	}

	return nil
}
