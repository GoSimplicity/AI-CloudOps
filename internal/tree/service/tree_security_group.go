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

package service

import (
	"context"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/provider"
	"go.uber.org/zap"
)

type TreeSecurityGroupService interface {
	// 资源管理
	CreateSecurityGroup(ctx context.Context, req *model.CreateSecurityGroupReq) error
	DeleteSecurityGroup(ctx context.Context, req *model.DeleteSecurityGroupReq) error
	ListSecurityGroups(ctx context.Context, req *model.ListSecurityGroupsReq) (model.ListResp[*model.ResourceSecurityGroup], error)
	GetSecurityGroupDetail(ctx context.Context, req *model.GetSecurityGroupDetailReq) (*model.ResourceSecurityGroup, error)
	UpdateSecurityGroup(ctx context.Context, req *model.UpdateSecurityGroupReq) error
	AddSecurityGroupRule(ctx context.Context, req *model.AddSecurityGroupRuleReq) error
	RemoveSecurityGroupRule(ctx context.Context, req *model.RemoveSecurityGroupRuleReq) error
	BindInstanceToSecurityGroup(ctx context.Context, req *model.BindInstanceToSecurityGroupReq) error
	UnbindInstanceFromSecurityGroup(ctx context.Context, req *model.UnbindInstanceFromSecurityGroupReq) error
}

type treeSecurityGroupService struct {
	providerFactory *provider.ProviderFactory
	logger          *zap.Logger
}

func NewTreeSecurityGroupService(providerFactory *provider.ProviderFactory, logger *zap.Logger) TreeSecurityGroupService {
	return &treeSecurityGroupService{
		providerFactory: providerFactory,
		logger:          logger,
	}
}

// AddSecurityGroupRule implements TreeSecurityGroupService.
func (t *treeSecurityGroupService) AddSecurityGroupRule(ctx context.Context, req *model.AddSecurityGroupRuleReq) error {
	panic("unimplemented")
}

// BindInstanceToSecurityGroup implements TreeSecurityGroupService.
func (t *treeSecurityGroupService) BindInstanceToSecurityGroup(ctx context.Context, req *model.BindInstanceToSecurityGroupReq) error {
	panic("unimplemented")
}

// CreateSecurityGroup implements TreeSecurityGroupService.
func (t *treeSecurityGroupService) CreateSecurityGroup(ctx context.Context, req *model.CreateSecurityGroupReq) error {
	panic("unimplemented")
}

// DeleteSecurityGroup implements TreeSecurityGroupService.
func (t *treeSecurityGroupService) DeleteSecurityGroup(ctx context.Context, req *model.DeleteSecurityGroupReq) error {
	panic("unimplemented")
}

// GetSecurityGroupDetail implements TreeSecurityGroupService.
func (t *treeSecurityGroupService) GetSecurityGroupDetail(ctx context.Context, req *model.GetSecurityGroupDetailReq) (*model.ResourceSecurityGroup, error) {
	panic("unimplemented")
}

// ListSecurityGroups implements TreeSecurityGroupService.
func (t *treeSecurityGroupService) ListSecurityGroups(ctx context.Context, req *model.ListSecurityGroupsReq) (model.ListResp[*model.ResourceSecurityGroup], error) {
	panic("unimplemented")
}

// RemoveSecurityGroupRule implements TreeSecurityGroupService.
func (t *treeSecurityGroupService) RemoveSecurityGroupRule(ctx context.Context, req *model.RemoveSecurityGroupRuleReq) error {
	panic("unimplemented")
}

// UnbindInstanceFromSecurityGroup implements TreeSecurityGroupService.
func (t *treeSecurityGroupService) UnbindInstanceFromSecurityGroup(ctx context.Context, req *model.UnbindInstanceFromSecurityGroupReq) error {
	panic("unimplemented")
}

// UpdateSecurityGroup implements TreeSecurityGroupService.
func (t *treeSecurityGroupService) UpdateSecurityGroup(ctx context.Context, req *model.UpdateSecurityGroupReq) error {
	panic("unimplemented")
}
