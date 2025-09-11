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
	"errors"
	"fmt"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type ClusterDAO interface {
	GetClusterList(ctx context.Context, req *model.ListClustersReq) ([]*model.K8sCluster, int64, error)
	CreateCluster(ctx context.Context, cluster *model.K8sCluster) error
	UpdateCluster(ctx context.Context, cluster *model.K8sCluster) error
	UpdateClusterStatus(ctx context.Context, id int, status model.ClusterStatus) error
	DeleteCluster(ctx context.Context, id int) error
	GetClusterByID(ctx context.Context, id int) (*model.K8sCluster, error)
	GetClusterByName(ctx context.Context, name string) (*model.K8sCluster, error)
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

// GetClusterList 获取集群列表
func (c *clusterDAO) GetClusterList(ctx context.Context, req *model.ListClustersReq) ([]*model.K8sCluster, int64, error) {
	if req == nil {
		c.l.Error("GetClusterList: 请求参数不能为空")
		return nil, 0, errors.New("请求参数不能为空")
	}

	var clusters []*model.K8sCluster
	var total int64

	// 构建基础查询
	query := c.db.WithContext(ctx).Model(&model.K8sCluster{})

	// 应用过滤条件
	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	}

	if req.Env != "" {
		query = query.Where("env = ?", req.Env)
	}

	if req.Search != "" {
		query = query.Where("name LIKE ? OR api_server_addr LIKE ?",
			"%"+req.Search+"%", "%"+req.Search+"%")
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		c.l.Error("GetClusterList: 统计集群总数失败",
			zap.String("status", req.Status),
			zap.String("env", req.Env),
			zap.String("search", req.Search),
			zap.Error(err))
		return nil, 0, fmt.Errorf("统计集群总数失败: %w", err)
	}

	// 如果总数为0，直接返回空列表
	if total == 0 {
		return clusters, 0, nil
	}

	// 应用分页
	if req.Page > 0 && req.Size > 0 {
		offset := (req.Page - 1) * req.Size
		query = query.Offset(offset).Limit(req.Size)
	}

	// 执行查询并排序
	if err := query.Order("updated_at DESC, id DESC").Find(&clusters).Error; err != nil {
		c.l.Error("GetClusterList: 查询集群列表失败",
			zap.Int("page", req.Page),
			zap.Int("size", req.Size),
			zap.Error(err))
		return nil, 0, fmt.Errorf("查询集群列表失败: %w", err)
	}

	return clusters, total, nil
}

// CreateCluster 创建集群
func (c *clusterDAO) CreateCluster(ctx context.Context, cluster *model.K8sCluster) error {
	if cluster == nil {
		c.l.Error("CreateCluster: 集群信息不能为空")
		return errors.New("集群信息不能为空")
	}

	if cluster.Name == "" {
		c.l.Error("CreateCluster: 集群名称不能为空")
		return errors.New("集群名称不能为空")
	}

	if err := c.db.WithContext(ctx).Create(cluster).Error; err != nil {
		c.l.Error("CreateCluster: 创建集群失败",
			zap.String("name", cluster.Name),
			zap.String("api_server", cluster.ApiServerAddr),
			zap.Error(err))
		return fmt.Errorf("创建集群失败: %w", err)
	}

	return nil
}

// UpdateCluster 更新集群
func (c *clusterDAO) UpdateCluster(ctx context.Context, cluster *model.K8sCluster) error {
	if cluster == nil {
		c.l.Error("UpdateCluster: 集群信息不能为空")
		return errors.New("集群信息不能为空")
	}

	if cluster.ID <= 0 {
		c.l.Error("UpdateCluster: 集群ID不有效", zap.Int("id", cluster.ID))
		return errors.New("集群ID不有效")
	}

	if cluster.Name == "" {
		c.l.Error("UpdateCluster: 集群名称不能为空")
		return errors.New("集群名称不能为空")
	}

	// 使用普通更新而非事务，除非必要
	result := c.db.WithContext(ctx).Model(cluster).Where("id = ?", cluster.ID).Updates(cluster)
	if result.Error != nil {
		c.l.Error("UpdateCluster: 更新集群失败",
			zap.Int("id", cluster.ID),
			zap.String("name", cluster.Name),
			zap.Error(result.Error))
		return fmt.Errorf("更新集群失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		c.l.Warn("UpdateCluster: 未找到要更新的集群", zap.Int("id", cluster.ID))
		return fmt.Errorf("集群不存在，ID: %d", cluster.ID)
	}

	return nil
}

// UpdateClusterStatus 更新集群状态
func (c *clusterDAO) UpdateClusterStatus(ctx context.Context, id int, status model.ClusterStatus) error {
	if id <= 0 {
		c.l.Error("UpdateClusterStatus: 集群ID不有效", zap.Int("id", id))
		return errors.New("集群ID不有效")
	}

	if status <= 0 {
		c.l.Error("UpdateClusterStatus: 状态值无效", zap.Int8("status", int8(status)))
		return errors.New("状态值无效")
	}

	result := c.db.WithContext(ctx).Model(&model.K8sCluster{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status": int8(status),
	})

	if result.Error != nil {
		c.l.Error("UpdateClusterStatus: 更新集群状态失败",
			zap.Int("id", id),
			zap.Int8("status", int8(status)),
			zap.Error(result.Error))
		return fmt.Errorf("更新集群状态失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		c.l.Warn("UpdateClusterStatus: 未找到要更新的集群", zap.Int("id", id))
		return fmt.Errorf("集群不存在，ID: %d", id)
	}

	return nil
}

// DeleteCluster 删除集群
func (c *clusterDAO) DeleteCluster(ctx context.Context, id int) error {
	if id <= 0 {
		c.l.Error("DeleteCluster: 集群ID不有效", zap.Int("id", id))
		return errors.New("集群ID不有效")
	}

	result := c.db.WithContext(ctx).Where("id = ?", id).Delete(&model.K8sCluster{})
	if result.Error != nil {
		c.l.Error("DeleteCluster: 删除集群失败",
			zap.Int("id", id),
			zap.Error(result.Error))
		return fmt.Errorf("删除集群失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		c.l.Warn("DeleteCluster: 未找到要删除的集群", zap.Int("id", id))
		return fmt.Errorf("集群不存在，ID: %d", id)
	}

	return nil
}

// GetClusterByID 根据ID获取集群
func (c *clusterDAO) GetClusterByID(ctx context.Context, id int) (*model.K8sCluster, error) {
	if id <= 0 {
		c.l.Error("GetClusterByID: 集群ID不有效", zap.Int("id", id))
		return nil, errors.New("集群ID不有效")
	}

	var cluster model.K8sCluster
	if err := c.db.WithContext(ctx).Where("id = ?", id).First(&cluster).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.l.Debug("GetClusterByID: 集群不存在", zap.Int("id", id))
			return nil, err
		}
		c.l.Error("GetClusterByID: 查询集群失败",
			zap.Int("id", id),
			zap.Error(err))
		return nil, fmt.Errorf("查询集群失败: %w", err)
	}

	return &cluster, nil
}

// GetClusterByName 根据名称获取集群
func (c *clusterDAO) GetClusterByName(ctx context.Context, name string) (*model.K8sCluster, error) {
	if name == "" {
		c.l.Error("GetClusterByName: 集群名称不能为空")
		return nil, errors.New("集群名称不能为空")
	}

	var cluster model.K8sCluster
	if err := c.db.WithContext(ctx).Where("name = ?", name).First(&cluster).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.l.Debug("GetClusterByName: 集群不存在", zap.String("name", name))
			return nil, err
		}
		c.l.Error("GetClusterByName: 查询集群失败",
			zap.String("name", name),
			zap.Error(err))
		return nil, fmt.Errorf("查询集群失败: %w", err)
	}

	return &cluster, nil
}
