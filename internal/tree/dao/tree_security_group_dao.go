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
	"gorm.io/gorm"
)

type TreeSecurityGroupDAO interface {
	// 安全组基础操作
	CreateSecurityGroup(ctx context.Context, securityGroup *model.ResourceSecurityGroup) error
	DeleteSecurityGroup(ctx context.Context, securityGroupID string) error
	GetSecurityGroupDetail(ctx context.Context, securityGroupID string) (*model.ResourceSecurityGroup, error)
	ListSecurityGroups(ctx context.Context, req *model.ListSecurityGroupsReq) ([]*model.ResourceSecurityGroup, error)
	UpdateSecurityGroup(ctx context.Context, securityGroup *model.ResourceSecurityGroup) error

	// 安全组规则操作
	AddSecurityGroupRule(ctx context.Context, rule *model.SecurityGroupRule) error
	RemoveSecurityGroupRule(ctx context.Context, securityGroupID string, ruleID string) error
	GetSecurityGroupRules(ctx context.Context, securityGroupID string) ([]*model.SecurityGroupRule, error)

	// 实例绑定操作
	BindInstanceToSecurityGroup(ctx context.Context, securityGroupID string, instanceID string) error
	UnbindInstanceFromSecurityGroup(ctx context.Context, securityGroupID string, instanceID string) error
	GetInstanceSecurityGroups(ctx context.Context, instanceID string) ([]*model.ResourceSecurityGroup, error)

	// 辅助查询方法
	SecurityGroupExists(ctx context.Context, securityGroupID string) (bool, error)
	GetSecurityGroupByName(ctx context.Context, name string) (*model.ResourceSecurityGroup, error)
	GetSecurityGroupsByRegion(ctx context.Context, region string) ([]*model.ResourceSecurityGroup, error)
}

type treeSecurityGroupDAO struct {
	db *gorm.DB
}

func NewTreeSecurityGroupDAO(db *gorm.DB) TreeSecurityGroupDAO {
	return &treeSecurityGroupDAO{
		db: db,
	}
}

// AddSecurityGroupRule implements TreeSecurityGroupDAO.
func (t *treeSecurityGroupDAO) AddSecurityGroupRule(ctx context.Context, rule *model.SecurityGroupRule) error {
	panic("unimplemented")
}

// BindInstanceToSecurityGroup implements TreeSecurityGroupDAO.
func (t *treeSecurityGroupDAO) BindInstanceToSecurityGroup(ctx context.Context, securityGroupID string, instanceID string) error {
	panic("unimplemented")
}

// CreateSecurityGroup implements TreeSecurityGroupDAO.
func (t *treeSecurityGroupDAO) CreateSecurityGroup(ctx context.Context, securityGroup *model.ResourceSecurityGroup) error {
	panic("unimplemented")
}

// DeleteSecurityGroup implements TreeSecurityGroupDAO.
func (t *treeSecurityGroupDAO) DeleteSecurityGroup(ctx context.Context, securityGroupID string) error {
	panic("unimplemented")
}

// GetInstanceSecurityGroups implements TreeSecurityGroupDAO.
func (t *treeSecurityGroupDAO) GetInstanceSecurityGroups(ctx context.Context, instanceID string) ([]*model.ResourceSecurityGroup, error) {
	panic("unimplemented")
}

// GetSecurityGroupByName implements TreeSecurityGroupDAO.
func (t *treeSecurityGroupDAO) GetSecurityGroupByName(ctx context.Context, name string) (*model.ResourceSecurityGroup, error) {
	panic("unimplemented")
}

// GetSecurityGroupDetail implements TreeSecurityGroupDAO.
func (t *treeSecurityGroupDAO) GetSecurityGroupDetail(ctx context.Context, securityGroupID string) (*model.ResourceSecurityGroup, error) {
	panic("unimplemented")
}

// GetSecurityGroupRules implements TreeSecurityGroupDAO.
func (t *treeSecurityGroupDAO) GetSecurityGroupRules(ctx context.Context, securityGroupID string) ([]*model.SecurityGroupRule, error) {
	panic("unimplemented")
}

// GetSecurityGroupsByRegion implements TreeSecurityGroupDAO.
func (t *treeSecurityGroupDAO) GetSecurityGroupsByRegion(ctx context.Context, region string) ([]*model.ResourceSecurityGroup, error) {
	panic("unimplemented")
}

// ListSecurityGroups implements TreeSecurityGroupDAO.
func (t *treeSecurityGroupDAO) ListSecurityGroups(ctx context.Context, req *model.ListSecurityGroupsReq) ([]*model.ResourceSecurityGroup, error) {
	panic("unimplemented")
}

// RemoveSecurityGroupRule implements TreeSecurityGroupDAO.
func (t *treeSecurityGroupDAO) RemoveSecurityGroupRule(ctx context.Context, securityGroupID string, ruleID string) error {
	panic("unimplemented")
}

// SecurityGroupExists implements TreeSecurityGroupDAO.
func (t *treeSecurityGroupDAO) SecurityGroupExists(ctx context.Context, securityGroupID string) (bool, error) {
	panic("unimplemented")
}

// UnbindInstanceFromSecurityGroup implements TreeSecurityGroupDAO.
func (t *treeSecurityGroupDAO) UnbindInstanceFromSecurityGroup(ctx context.Context, securityGroupID string, instanceID string) error {
	panic("unimplemented")
}

// UpdateSecurityGroup implements TreeSecurityGroupDAO.
func (t *treeSecurityGroupDAO) UpdateSecurityGroup(ctx context.Context, securityGroup *model.ResourceSecurityGroup) error {
	panic("unimplemented")
}
