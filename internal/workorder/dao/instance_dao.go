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
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"gorm.io/gorm"
)

type InstanceDAO interface {
	CreateInstance(ctx context.Context, instance model.Instance) error
	UpdateInstance(ctx context.Context, instance *model.Instance) error
	DeleteInstance(ctx context.Context, id int) error
	ListInstance(ctx context.Context, req model.ListInstanceReq) ([]model.Instance, int64, error)
	GetInstance(ctx context.Context, id int) (model.Instance, error)
	CreateInstanceFlow(ctx context.Context, flow model.InstanceFlow) error
	CreateInstanceComment(ctx context.Context, comment model.InstanceComment) error
	GetInstanceFlows(ctx context.Context, instanceID int) ([]model.InstanceFlow, error)
	GetInstanceComments(ctx context.Context, instanceID int) ([]model.InstanceComment, error)
	GetWorkflow(ctx context.Context, workflowID int) (model.Process, error)
	GetInstanceStatistics(ctx context.Context) (interface{}, error)
	GetInstanceTrend(ctx context.Context) ([]interface{}, error)
}

type instanceDAO struct {
	db *gorm.DB
}

func NewInstanceDAO(db *gorm.DB) InstanceDAO {
	return &instanceDAO{
		db: db,
	}
}

// CreateInstance 创建工单实例
func (i *instanceDAO) CreateInstance(ctx context.Context, instance model.Instance) error {
	// 处理零值时间
	if instance.CompletedAt != nil && instance.CompletedAt.IsZero() {
		instance.CompletedAt = nil
	}
	if instance.DueDate != nil && instance.DueDate.IsZero() {
		instance.DueDate = nil
	}

	if err := i.db.WithContext(ctx).Create(&instance).Error; err != nil {
		if err == gorm.ErrDuplicatedKey {
			return fmt.Errorf("工单实例已存在")
		}
		return err
	}
	return nil
}

// DeleteInstance 删除工单实例
func (i *instanceDAO) DeleteInstance(ctx context.Context, id int) error {
	if err := i.db.WithContext(ctx).Delete(&model.Instance{}, id).Error; err != nil {
		return err
	}
	return nil
}

// GetInstance 获取工单实例详情
func (i *instanceDAO) GetInstance(ctx context.Context, id int) (model.Instance, error) {
	var instance model.Instance
	if err := i.db.WithContext(ctx).Where("id = ?", id).First(&instance).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return model.Instance{}, fmt.Errorf("工单实例不存在")
		}
		return model.Instance{}, err
	}
	return instance, nil
}

// ListInstance 获取工单实例列表
func (i *instanceDAO) ListInstance(ctx context.Context, req model.ListInstanceReq) ([]model.Instance, int64, error) {
	var instances []model.Instance
	var total int64
	db := i.db.WithContext(ctx).Model(&model.Instance{})

	// 构建查询条件
	if req.Keyword != "" {
		db = db.Where("title LIKE ?", "%"+req.Keyword+"%")
	}
	if req.Status != 0 {
		db = db.Where("status = ?", req.Status)
	}
	if len(req.DateRange) == 2 {
		db = db.Where("created_at BETWEEN ? AND ?", req.DateRange[0], req.DateRange[1])
	}
	if req.CreatorID != 0 {
		db = db.Where("creator_id = ?", req.CreatorID)
	}
	if req.AssigneeID != 0 {
		db = db.Where("assignee_id = ?", req.AssigneeID)
	}
	if req.WorkflowID != 0 {
		db = db.Where("workflow_id = ?", req.WorkflowID)
	}

	// 获取总数
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 10
	}
	offset := (req.Page - 1) * req.PageSize
	
	if err := db.Order("created_at DESC").Offset(offset).Limit(req.PageSize).Find(&instances).Error; err != nil {
		return nil, 0, err
	}

	return instances, total, nil
}

// UpdateInstance 更新工单实例
func (i *instanceDAO) UpdateInstance(ctx context.Context, instance *model.Instance) error {
	if instance == nil {
		return fmt.Errorf("实例对象为空")
	}
	if instance.ID == 0 {
		return fmt.Errorf("实例ID无效")
	}

	// 处理零值时间
	if instance.CompletedAt != nil && instance.CompletedAt.IsZero() {
		var nilTime *time.Time = nil
		instance.CompletedAt = nilTime
	}
	if instance.DueDate != nil && instance.DueDate.IsZero() {
		var nilTime *time.Time = nil
		instance.DueDate = nilTime
	}

	result := i.db.WithContext(ctx).Model(&model.Instance{}).Where("id = ?", instance.ID).Updates(instance)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("未找到ID为%d的工单实例", instance.ID)
	}

	return nil
}

// CreateInstanceFlow 创建工单流程记录
func (i *instanceDAO) CreateInstanceFlow(ctx context.Context, flow model.InstanceFlow) error {
	if err := i.db.WithContext(ctx).Create(&flow).Error; err != nil {
		return err
	}
	return nil
}

// CreateInstanceComment 创建工单评论
func (i *instanceDAO) CreateInstanceComment(ctx context.Context, comment model.InstanceComment) error {
	if err := i.db.WithContext(ctx).Create(&comment).Error; err != nil {
		return err
	}
	return nil
}

// GetInstanceFlows 获取工单流程记录
func (i *instanceDAO) GetInstanceFlows(ctx context.Context, instanceID int) ([]model.InstanceFlow, error) {
	var flows []model.InstanceFlow
	if err := i.db.WithContext(ctx).Where("instance_id = ?", instanceID).Order("created_at ASC").Find(&flows).Error; err != nil {
		return nil, err
	}
	return flows, nil
}

// GetInstanceComments 获取工单评论
func (i *instanceDAO) GetInstanceComments(ctx context.Context, instanceID int) ([]model.InstanceComment, error) {
	var comments []model.InstanceComment
	if err := i.db.WithContext(ctx).Where("instance_id = ?", instanceID).Order("created_at ASC").Find(&comments).Error; err != nil {
		return nil, err
	}
	return comments, nil
}

// GetWorkflow 获取工作流定义
func (i *instanceDAO) GetWorkflow(ctx context.Context, workflowID int) (model.Process, error) {
	var process model.Process
	if err := i.db.WithContext(ctx).Where("id = ?", workflowID).First(&process).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return model.Process{}, fmt.Errorf("工作流定义不存在")
		}
		return model.Process{}, err
	}
	return process, nil
}

// GetInstanceStatistics 获取工单统计信息
func (i *instanceDAO) GetInstanceStatistics(ctx context.Context) (interface{}, error) {
	var result []struct {
		Status int   `json:"status"`
		Count  int64 `json:"count"`
	}
	
	if err := i.db.WithContext(ctx).Model(&model.Instance{}).
		Select("status, count(*) as count").
		Group("status").
		Find(&result).Error; err != nil {
		return nil, err
	}
	
	return result, nil
}

// GetInstanceTrend 获取工单趋势
func (i *instanceDAO) GetInstanceTrend(ctx context.Context) ([]interface{}, error) {
	// 获取最近30天的工单创建趋势
	var result []struct {
		Date  string `json:"date"`
		Count int64  `json:"count"`
	}
	
	if err := i.db.WithContext(ctx).Model(&model.Instance{}).
		Select("DATE(created_at) as date, count(*) as count").
		Where("created_at >= ?", time.Now().AddDate(0, 0, -30)).
		Group("DATE(created_at)").
		Order("date ASC").
		Find(&result).Error; err != nil {
		return nil, err
	}
	
	// 转换为interface{}切片
	trend := make([]interface{}, len(result))
	for i, v := range result {
		trend[i] = v
	}
	
	return trend, nil
}
