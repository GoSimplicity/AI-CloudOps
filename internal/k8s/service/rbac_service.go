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

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/manager"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
)

type RBACService struct {
	manager manager.RBACManager
}

func NewRBACService(manager manager.RBACManager) *RBACService {
	return &RBACService{
		manager: manager,
	}
}

// GetRBACStatistics 获取RBAC统计信息
func (rs *RBACService) GetRBACStatistics(ctx context.Context, clusterID int) (*model.RBACStatistics, error) {
	return rs.manager.GetRBACStatistics(ctx, clusterID)
}

// CheckPermissions 检查权限
func (rs *RBACService) CheckPermissions(ctx context.Context, req *model.CheckPermissionsReq) ([]model.PermissionResult, error) {
	return rs.manager.CheckPermissions(ctx, req)
}

// GetSubjectPermissions 获取主体的有效权限列表
func (rs *RBACService) GetSubjectPermissions(ctx context.Context, req *model.SubjectPermissionsReq) (*model.SubjectPermissionsResponse, error) {
	return rs.manager.GetSubjectPermissions(ctx, req)
}

// GetResourceVerbs 获取预定义的资源和动作列表
func (rs *RBACService) GetResourceVerbs(ctx context.Context) (*model.ResourceVerbsResponse, error) {
	return rs.manager.GetResourceVerbs(ctx)
}
