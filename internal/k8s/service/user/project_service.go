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

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao/admin"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao/user"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
)

type ProjectService interface {
	CreateProject(ctx context.Context, req *model.CreateK8sProjectRequest) error
	GetProjectList(ctx context.Context, req *model.GetK8sProjectListRequest) ([]model.K8sProject, error)
	GetprojectListByIds(ctx context.Context, ids []int64) ([]model.K8sProject, error)
	DeleteProject(ctx context.Context, id int64) error
	UpdateProject(ctx context.Context, req *model.UpdateK8sProjectRequest) error
}
type projectService struct {
	client     client.K8sClient
	l          *zap.Logger
	dao        admin.ClusterDAO
	projectdao user.ProjectDAO
}

func NewProjectService(dao admin.ClusterDAO, projectdao user.ProjectDAO, client client.K8sClient, l *zap.Logger) ProjectService {
	return &projectService{
		dao:        dao,
		projectdao: projectdao,
		client:     client,
		l:          l,
	}
}

// CreateProject implements ProjectService.
func (p *projectService) CreateProject(ctx context.Context, req *model.CreateK8sProjectRequest) error {
	panic("unimplemented")
}

// DeleteProject implements ProjectService.
func (p *projectService) DeleteProject(ctx context.Context, id int64) error {
	panic("unimplemented")
}

// GetProjectList implements ProjectService.
func (p *projectService) GetProjectList(ctx context.Context, req *model.GetK8sProjectListRequest) ([]model.K8sProject, error) {
	panic("unimplemented")
}

// GetprojectListByIds implements ProjectService.
func (p *projectService) GetprojectListByIds(ctx context.Context, ids []int64) ([]model.K8sProject, error) {
	panic("unimplemented")
}

// UpdateProject implements ProjectService.
func (p *projectService) UpdateProject(ctx context.Context, req *model.UpdateK8sProjectRequest) error {
	panic("unimplemented")
}
