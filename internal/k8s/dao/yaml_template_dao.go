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
	"fmt"

	"github.com/GoSimplicity/AI-CloudOps/internal/constants"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type YamlTemplateDAO interface {
	ListAllYamlTemplates(ctx context.Context, req *model.YamlTemplateListReq) ([]*model.K8sYamlTemplate, error)
	CreateYamlTemplate(ctx context.Context, yaml *model.K8sYamlTemplate) error
	UpdateYamlTemplate(ctx context.Context, yaml *model.K8sYamlTemplate, ownerID int) error
	DeleteYamlTemplate(ctx context.Context, id int, clusterId int, userID int) error
	GetYamlTemplateByID(ctx context.Context, id int, clusterId int) (*model.K8sYamlTemplate, error)
	GetOwnedYamlTemplate(ctx context.Context, id int, clusterID int, userID int) (*model.K8sYamlTemplate, error)
}

type yamlTemplateDAO struct {
	db *gorm.DB
	l  *zap.Logger
}

func NewYamlTemplateDAO(db *gorm.DB, l *zap.Logger) YamlTemplateDAO {
	return &yamlTemplateDAO{
		db: db,
		l:  l,
	}
}

// ListAllYamlTemplates 查询所有 YAML 模板
func (d *yamlTemplateDAO) ListAllYamlTemplates(ctx context.Context, req *model.YamlTemplateListReq) ([]*model.K8sYamlTemplate, error) {
	var yamls []*model.K8sYamlTemplate

	query := d.db.WithContext(ctx)

	// 根据集群ID过滤
	if req.ClusterID > 0 {
		query = query.Where("cluster_id = ?", req.ClusterID)
	}

	// 根据模板名称过滤（模糊匹配）
	if req.Search != "" {
		query = query.Where("name LIKE ?", "%"+req.Search+"%")
	}

	if req.Size > 0 {
		offset := (req.Page - 1) * req.Size
		query = query.Offset(offset).Limit(req.Size)
	}

	query = query.Order("created_at DESC")

	if err := query.Find(&yamls).Error; err != nil {
		d.l.Error("ListAllYamlTemplates 查询所有Yaml模板失败", zap.Error(err))
		return nil, err
	}

	return yamls, nil
}

func (d *yamlTemplateDAO) CreateYamlTemplate(ctx context.Context, yaml *model.K8sYamlTemplate) error {
	if err := d.db.WithContext(ctx).Create(&yaml).Error; err != nil {
		d.l.Error("CreateYamlTemplate 创建Yaml模板失败", zap.Error(err), zap.Any("yaml", yaml))
		return err
	}

	return nil
}

func (d *yamlTemplateDAO) UpdateYamlTemplate(ctx context.Context, yaml *model.K8sYamlTemplate, ownerID int) error {
	if yaml == nil || yaml.ID == 0 {
		d.l.Error("UpdateYamlTemplate ID 不能为空", zap.Any("yaml", yaml))
		return fmt.Errorf("invalid yaml ID")
	}
	if _, err := d.GetOwnedYamlTemplate(ctx, yaml.ID, yaml.ClusterID, ownerID); err != nil {
		return err
	}

	result := d.db.WithContext(ctx).Model(&model.K8sYamlTemplate{}).
		Where("id = ? AND cluster_id = ? AND user_id = ?", yaml.ID, yaml.ClusterID, ownerID).
		Omit("user_id").
		Updates(yaml)
	if result.Error != nil {
		d.l.Error("UpdateYamlTemplate 更新Yaml模板失败", zap.Int("yamlID", yaml.ID), zap.Error(result.Error))
		return result.Error
	}
	if result.RowsAffected == 0 {
		if _, err := d.GetOwnedYamlTemplate(ctx, yaml.ID, yaml.ClusterID, ownerID); err != nil {
			return err
		}
	}
	return nil
}

func (d *yamlTemplateDAO) DeleteYamlTemplate(ctx context.Context, id int, clusterId int, userID int) error {
	if id == 0 {
		d.l.Error("DeleteYamlTemplate ID 不能为空", zap.Int("id", id))
		return fmt.Errorf("invalid yaml template ID")
	}
	if userID <= 0 {
		return fmt.Errorf("%w: user id", constants.ErrInvalidParam)
	}

	if _, err := d.GetOwnedYamlTemplate(ctx, id, clusterId, userID); err != nil {
		return err
	}

	result := d.db.WithContext(ctx).Where("id = ? AND cluster_id = ? AND user_id = ?", id, clusterId, userID).Delete(&model.K8sYamlTemplate{})
	if result.Error != nil {
		d.l.Error("DeleteYamlTemplate 删除Yaml模板失败", zap.Int("yamlID", id), zap.Error(result.Error))
		return result.Error
	}
	if result.RowsAffected == 0 {
		return constants.ErrNotResourceOwner
	}
	return nil
}

func (d *yamlTemplateDAO) GetYamlTemplateByID(ctx context.Context, id int, clusterId int) (*model.K8sYamlTemplate, error) {
	if id == 0 {
		d.l.Error("GetYamlTemplateByID ID 不能为空", zap.Int("id", id))
		return nil, fmt.Errorf("invalid yaml template ID")
	}

	var yaml *model.K8sYamlTemplate

	if err := d.db.WithContext(ctx).Where("id = ? AND cluster_id = ?", id, clusterId).First(&yaml).Error; err != nil {
		d.l.Error("GetYamlTemplateByID 查询Yaml模板失败", zap.Int("yamlID", id), zap.Error(err))
		return nil, err
	}

	return yaml, nil
}

func (d *yamlTemplateDAO) GetOwnedYamlTemplate(ctx context.Context, id int, clusterID int, userID int) (*model.K8sYamlTemplate, error) {
	if userID <= 0 {
		return nil, fmt.Errorf("%w: user id", constants.ErrInvalidParam)
	}
	template, err := d.GetYamlTemplateByID(ctx, id, clusterID)
	if err != nil {
		return nil, err
	}
	if template.UserID != userID {
		d.l.Warn("YAML模板归属校验失败", zap.Int("yamlID", id), zap.Int("clusterID", clusterID), zap.Int("userID", userID))
		return nil, constants.ErrNotResourceOwner
	}
	return template, nil
}
