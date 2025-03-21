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

type RoleDAO interface {
	ListRoles(ctx context.Context, page, pageSize int) (*model.GenerateRoleResp, error)
	GetRolesByUserId(ctx context.Context, userId int, page, pageSize int) (*model.GenerateRoleResp, error)
	GetRolesByDomainId(ctx context.Context, domainId int, page, pageSize int) (*model.GenerateRoleResp, error)
}

type roleDAO struct {
	db *gorm.DB
	l  *zap.Logger
}

func NewRoleDAO(db *gorm.DB, l *zap.Logger) RoleDAO {
	return &roleDAO{
		db: db,
		l:  l,
	}
}

// GetRolesByDomainId implements RoleDAO.
func (r *roleDAO) GetRolesByDomainId(ctx context.Context, domainId int, page, pageSize int) (*model.GenerateRoleResp, error) {
	var roles []*model.CasbinRule
	var total int64

	query := r.db.WithContext(ctx).Where("v1 = ?", domainId)

	// 获取总数
	if err := query.Model(&model.CasbinRule{}).Count(&total).Error; err != nil {
		return nil, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Find(&roles).Error; err != nil {
		return nil, err
	}

	// 构建返回值
	items := make([]*model.Role, 0, len(roles))
	for _, role := range roles {
		items = append(items, &model.Role{
			Name:   role.V0,
			Domain: role.V1,
			Path:   role.V2,
			Method: role.V3,
		})
	}

	return &model.GenerateRoleResp{
		Total: int(total),
		Items: items,
	}, nil
}

// GetRolesByUserId implements RoleDAO.
func (r *roleDAO) GetRolesByUserId(ctx context.Context, userId int, page, pageSize int) (*model.GenerateRoleResp, error) {
	var roles []*model.CasbinRule
	var total int64

	query := r.db.WithContext(ctx).Where("v0 = ?", userId)

	// 获取总数
	if err := query.Model(&model.CasbinRule{}).Count(&total).Error; err != nil {
		return nil, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Find(&roles).Error; err != nil {
		return nil, err
	}

	// 构建返回值
	items := make([]*model.Role, 0, len(roles))
	for _, role := range roles {
		items = append(items, &model.Role{
			Name:   role.V0,
			Domain: role.V1,
			Path:   role.V2,
			Method: role.V3,
		})
	}

	return &model.GenerateRoleResp{
		Total: int(total),
		Items: items,
	}, nil
}

// ListRoles implements RoleDAO.
func (r *roleDAO) ListRoles(ctx context.Context, page int, pageSize int) (*model.GenerateRoleResp, error) {
	var roles []*model.CasbinRule
	var total int64

	// 获取总数
	if err := r.db.WithContext(ctx).Model(&model.CasbinRule{}).Count(&total).Error; err != nil {
		return nil, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := r.db.WithContext(ctx).Offset(offset).Limit(pageSize).Find(&roles).Error; err != nil {
		return nil, err
	}

	// 构建返回结果
	items := make([]*model.Role, 0, len(roles))
	for _, role := range roles {
		items = append(items, &model.Role{
			Name:   role.V0,
			Domain: role.V1,
			Path:   role.V2,
			Method: role.V3,
		})
	}

	return &model.GenerateRoleResp{
		Total: int(total),
		Items: items,
	}, nil
}
