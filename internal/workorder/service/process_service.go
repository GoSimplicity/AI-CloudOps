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
	userDao "github.com/GoSimplicity/AI-CloudOps/internal/user/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/dao"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"go.uber.org/zap"
)

type ProcessService interface {
	CreateProcess(ctx context.Context, req *model.CreateProcessReq) error
	UpdateProcess(ctx context.Context, req *model.UpdateProcessReq) error
	DeleteProcess(ctx context.Context, id int) error
	ListProcess(ctx context.Context, req model.ListProcessReq) ([]model.Process, error)
	DetailProcess(ctx context.Context, id int, userId int) (model.Process, error)
	PublishProcess(ctx context.Context, req model.PublishProcessReq) error
	CloneProcess(ctx context.Context, req model.CloneProcessReq) error
}

type processService struct {
	dao     dao.ProcessDAO
	userDao userDao.UserDAO
	l       *zap.Logger
}

func NewProcessService(dao dao.ProcessDAO, l *zap.Logger, userDao userDao.UserDAO) ProcessService {
	return &processService{
		dao:     dao,
		userDao: userDao,
		l:       l,
	}
}

// CreateProcess 创建流程
func (p *processService) CreateProcess(ctx context.Context, req *model.CreateProcessReq) error {
	process, err := utils.ConvertCreateProcessReq(req)
	if err != nil {
		return err
	}
	return p.dao.CreateProcess(ctx, process)
}

// DeleteProcess 删除流程
func (p *processService) DeleteProcess(ctx context.Context, id int) error {
	return p.dao.DeleteProcess(ctx, id)
}

// DetailProcess 流程详情
func (p *processService) DetailProcess(ctx context.Context, id int, userId int) (model.Process, error) {
	// 获取userid对应的中文名称
	user, err := p.userDao.GetUserByID(ctx, userId)
	if err != nil {
		return model.Process{}, err
	}

	process, err := p.dao.GetProcess(ctx, id)
	if err != nil {
		return model.Process{}, err
	}

	process.CreatorName = user.Username

	return process, nil
}

// ListProcess 流程列表
func (p *processService) ListProcess(ctx context.Context, req model.ListProcessReq) ([]model.Process, error) {
	return p.dao.ListProcess(ctx, req)
}

// UpdateProcess 更新流程
func (p *processService) UpdateProcess(ctx context.Context, req *model.UpdateProcessReq) error {
	process, err := utils.ConvertUpdateProcessReq(req)
	if err != nil {
		return err
	}
	return p.dao.UpdateProcess(ctx, process)
}

// PublishProcess 发布流程
func (p *processService) PublishProcess(ctx context.Context, req model.PublishProcessReq) error {
	return p.dao.PublishProcess(ctx, req.ID)
}

// CloneProcess 克隆流程
func (p *processService) CloneProcess(ctx context.Context, req model.CloneProcessReq) error {
	process, err := p.dao.GetProcess(ctx, req.ID)
	if err != nil {
		return err
	}
	process.ID = 0
	process.Name = req.Name
	return p.dao.CreateProcess(ctx, &process)
}
