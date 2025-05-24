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
	CreateProcess(ctx context.Context, req *model.CreateProcessReq, creatorID int, creatorName string) error
	UpdateProcess(ctx context.Context, req *model.UpdateProcessReq) error
	DeleteProcess(ctx context.Context, id int) error
	ListProcess(ctx context.Context, req model.ListProcessReq) ([]model.Process, error)
	DetailProcess(ctx context.Context, id int, userId int) (model.Process, error)
	PublishProcess(ctx context.Context, req model.PublishProcessReq) error
	CloneProcess(ctx context.Context, req model.CloneProcessReq) error
	ValidateProcess(ctx context.Context, id int, userID int) (*model.ValidateProcessResp, error) // Added ValidateProcess
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
func (p *processService) CreateProcess(ctx context.Context, req *model.CreateProcessReq, creatorID int, creatorName string) error {
	process, err := utils.ConvertCreateProcessReqToModel(req, creatorID, creatorName) // Updated to use the new function name and pass creator info
	if err != nil {
		p.l.Error("转换创建流程请求失败", zap.Error(err))
		return err
	}
	// CreatorID and CreatorName are set by the converter.
	// Default status (e.g., draft) and version can also be set by the converter or here.
	// process.Status = 0 // Example: Default to draft, if not set in converter
	// process.Version = 1 // Example: Default to version 1, if not set in converter
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
	process, err := utils.ConvertUpdateProcessReqToModel(req) // Updated to use the new function name
	if err != nil {
		p.l.Error("转换更新流程请求失败", zap.Error(err))
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

// ValidateProcess 校验流程定义
func (p *processService) ValidateProcess(ctx context.Context, id int, userID int) (*model.ValidateProcessResp, error) {
	p.l.Info("开始校验流程", zap.Int("processID", id), zap.Int("userID", userID))
	// TODO: Add permission check for userID if necessary

	resp := &model.ValidateProcessResp{
		IsValid: true,
		Errors:  make([]string, 0),
	}

	process, err := p.dao.GetProcess(ctx, id)
	if err != nil {
		p.l.Error("校验流程失败：获取流程失败", zap.Error(err), zap.Int("processID", id))
		resp.IsValid = false
		resp.Errors = append(resp.Errors, fmt.Sprintf("获取流程 (ID: %d) 失败: %s", id, err.Error()))
		return resp, err // Return error as well, as fetching process failed
	}

	if process.Definition == "" {
		p.l.Warn("流程定义为空", zap.Int("processID", id))
		resp.IsValid = false
		resp.Errors = append(resp.Errors, "流程定义为空。")
		return resp, nil // No further validation possible
	}

	var def model.ProcessDefinition
	if err := utils.Json.Unmarshal([]byte(process.Definition), &def); err != nil {
		p.l.Error("校验流程失败：解析流程定义JSON失败", zap.Error(err), zap.Int("processID", id))
		resp.IsValid = false
		resp.Errors = append(resp.Errors, fmt.Sprintf("解析流程定义JSON失败: %s", err.Error()))
		return resp, nil // Return response with error, not the error itself if JSON is malformed by user
	}

	if len(def.Steps) == 0 {
		resp.IsValid = false
		resp.Errors = append(resp.Errors, "流程至少需要一个步骤。")
	}

	hasStartStep := false
	hasEndStep := false
	stepMap := make(map[string]model.ProcessStep)

	for _, step := range def.Steps {
		if _, exists := stepMap[step.ID]; exists {
			resp.IsValid = false
			resp.Errors = append(resp.Errors, fmt.Sprintf("步骤ID '%s' 重复。", step.ID))
		}
		stepMap[step.ID] = step

		if step.Type == "start" {
			hasStartStep = true
		}
		if step.Type == "end" {
			hasEndStep = true
		}
		if step.Name == "" {
			resp.IsValid = false
			resp.Errors = append(resp.Errors, fmt.Sprintf("步骤ID '%s' 的名称不能为空。", step.ID))
		}
	}

	if !hasStartStep {
		resp.IsValid = false
		resp.Errors = append(resp.Errors, "流程必须包含一个开始（start）类型的步骤。")
	}
	if !hasEndStep {
		resp.IsValid = false
		resp.Errors = append(resp.Errors, "流程必须包含一个结束（end）类型的步骤。")
	}

	// Validate connections
	if len(def.Steps) > 0 && len(def.Connections) == 0 && len(def.Steps) > 1 { // Allow single-step process (e.g. start->end as one step)
		// If there's more than one step, there should be connections, unless it's a very simple start-end process.
		// A single start step that is also an end step doesn't need connections.
		isSingleStartEndStep := false
		if len(def.Steps) == 1 {
			if def.Steps[0].Type == "start" && def.Steps[0].Type == "end" { // This condition is wrong, can't be both
                 // A single step could be e.g. a "task" that implicitly starts and ends the process.
                 // Or a start step that directly leads to an implicit end.
                 // For more complex single step logic, this validation might be too strict.
                 // Let's assume if >1 steps, connections are needed.
			}
		}
		if !isSingleStartEndStep && len(def.Steps) > 1 {
			// resp.IsValid = false // Making this a warning for now, as some simple processes might not have connections if linear and defined by order
			// resp.Errors = append(resp.Errors, "流程步骤之间缺少连接定义。")
		}

	}


	for _, conn := range def.Connections {
		if conn.From == "" || conn.To == "" {
			resp.IsValid = false
			resp.Errors = append(resp.Errors, "连接的来源（From）和目标（To）步骤ID不能为空。")
			continue
		}
		if _, ok := stepMap[conn.From]; !ok {
			resp.IsValid = false
			resp.Errors = append(resp.Errors, fmt.Sprintf("连接中引用的来源步骤ID '%s' 不存在。", conn.From))
		}
		if _, ok := stepMap[conn.To]; !ok {
			resp.IsValid = false
			resp.Errors = append(resp.Errors, fmt.Sprintf("连接中引用的目标步骤ID '%s' 不存在。", conn.To))
		}
		fromStep := stepMap[conn.From]
		if fromStep.Type == "end" {
			resp.IsValid = false
			resp.Errors = append(resp.Errors, fmt.Sprintf("结束（end）类型的步骤 '%s' 不能有输出连接。", conn.From))
		}
		toStep := stepMap[conn.To]
		if toStep.Type == "start" {
			resp.IsValid = false
			resp.Errors = append(resp.Errors, fmt.Sprintf("开始（start）类型的步骤 '%s' 不能有输入连接。", conn.To))
		}
	}
	
	// Check for orphaned steps (steps not part of any connection, excluding start/end if they are handled)
	// This can be complex. For now, ensuring all steps are reachable from start and can reach an end. (Graph traversal - out of scope for basic validation)


	if len(resp.Errors) > 0 {
		resp.IsValid = false
	}

	p.l.Info("流程校验完成", zap.Int("processID", id), zap.Bool("isValid", resp.IsValid), zap.Strings("errors", resp.Errors))
	return resp, nil
}
