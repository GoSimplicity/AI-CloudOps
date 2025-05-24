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
	GetProcess(ctx context.Context, processID int) (model.Process, error) // Renamed from GetWorkflow
	GetInstanceStatistics(ctx context.Context) (interface{}, error)
	GetInstanceTrend(ctx context.Context) ([]interface{}, error)

	// Attachment methods
	CreateInstanceAttachment(ctx context.Context, attachment *model.InstanceAttachment) (*model.InstanceAttachment, error)
	DeleteInstanceAttachment(ctx context.Context, instanceID int, attachmentID int) error
	GetInstanceAttachments(ctx context.Context, instanceID int) ([]model.InstanceAttachment, error)
	"go.uber.org/zap" // Added for logging
)

type instanceDAO struct {
	db     *gorm.DB
	logger *zap.Logger // Added logger
}

func NewInstanceDAO(db *gorm.DB, logger *zap.Logger) InstanceDAO { // Updated constructor
	return &instanceDAO{
		db:     db,
		logger: logger, // Set logger
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
	if req.Search != "" { // Changed from Keyword to Search (from embedded ListReq)
		db = db.Where("title LIKE ?", "%"+req.Search+"%")
	}
	if req.Status != nil { // Specific field from ListInstanceReq
		db = db.Where("status = ?", *req.Status)
	}
	// Date range filter from ListInstanceReq
	if req.StartDate != nil && req.EndDate != nil {
		db = db.Where("created_at BETWEEN ? AND ?", req.StartDate, req.EndDate)
	} else if req.StartDate != nil {
		db = db.Where("created_at >= ?", req.StartDate)
	} else if req.EndDate != nil {
		db = db.Where("created_at <= ?", req.EndDate)
	}

	if req.CreatorID != nil { // Specific field from ListInstanceReq
		db = db.Where("creator_id = ?", *req.CreatorID)
	}
	if req.AssigneeID != nil { // Specific field from ListInstanceReq
		db = db.Where("assignee_id = ?", *req.AssigneeID)
	}
	if req.ProcessID != nil { // Specific field from ListInstanceReq, was WorkflowID
		db = db.Where("process_id = ?", *req.ProcessID)
	}
	if req.CategoryID != nil { // Specific field from ListInstanceReq
		db = db.Where("category_id = ?", *req.CategoryID)
	}
	if req.Priority != nil { // Specific field from ListInstanceReq
		db = db.Where("priority = ?", *req.Priority)
	}
	if req.TemplateID != nil { // Specific field from ListInstanceReq
		db = db.Where("template_id = ?", *req.TemplateID)
	}
	// TODO: Handle req.Tags and req.Overdue if necessary.
	// Example for Tags (if stored as comma-separated string):
	// if len(req.Tags) > 0 {
	// 	for _, tag := range req.Tags {
	// 		db = db.Where("tags LIKE ?", "%"+tag+"%")
	// 	}
	// }
	// Example for Overdue:
	// if req.Overdue != nil && *req.Overdue {
	// 	db = db.Where("due_date < ? AND status NOT IN (?)", time.Now(), []int8{model.InstanceStatusCompleted, model.InstanceStatusCancelled, model.InstanceStatusRejected})
	// } else if req.Overdue != nil && !*req.Overdue {
	// 	db = db.Where("due_date >= ? OR status IN (?)", time.Now(), []int8{model.InstanceStatusCompleted, model.InstanceStatusCancelled, model.InstanceStatusRejected})
	// }
	}

	// 获取总数
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	if req.Page <= 0 { // Page from embedded ListReq
		req.Page = 1
	}
	if req.Size <= 0 { // Size from embedded ListReq (was PageSize)
		req.Size = 10
	}
	offset := (req.Page - 1) * req.Size // Use Size
	
	if err := db.Order("created_at DESC").Offset(offset).Limit(req.Size).Find(&instances).Error; err != nil { // Use Size
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

// GetProcess 获取流程定义
func (i *instanceDAO) GetProcess(ctx context.Context, processID int) (model.Process, error) { // Renamed from GetWorkflow
	var process model.Process
	if err := i.db.WithContext(ctx).Where("id = ?", processID).First(&process).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return model.Process{}, fmt.Errorf("流程定义不存在") // Changed error message
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

// CreateInstanceAttachment 创建工单附件记录
func (i *instanceDAO) CreateInstanceAttachment(ctx context.Context, attachment *model.InstanceAttachment) (*model.InstanceAttachment, error) {
	i.logger.Debug("开始创建工单附件 (DAO)", zap.Any("attachment", attachment))
	if err := i.db.WithContext(ctx).Create(attachment).Error; err != nil {
		i.logger.Error("创建工单附件失败 (DAO)", zap.Error(err), zap.Any("attachment", attachment))
		return nil, fmt.Errorf("创建工单附件失败: %w", err)
	}
	i.logger.Debug("工单附件创建成功 (DAO)", zap.Int("id", attachment.ID))
	return attachment, nil
}

// DeleteInstanceAttachment 删除工单附件记录
func (i *instanceDAO) DeleteInstanceAttachment(ctx context.Context, instanceID int, attachmentID int) error {
	i.logger.Debug("开始删除工单附件 (DAO)", zap.Int("instanceID", instanceID), zap.Int("attachmentID", attachmentID))
	result := i.db.WithContext(ctx).Where("id = ? AND instance_id = ?", attachmentID, instanceID).Delete(&model.InstanceAttachment{})
	if result.Error != nil {
		i.logger.Error("删除工单附件失败 (DAO)", zap.Error(result.Error), zap.Int("attachmentID", attachmentID))
		return fmt.Errorf("删除工单附件 (ID: %d) 失败: %w", attachmentID, result.Error)
	}
	if result.RowsAffected == 0 {
		i.logger.Warn("删除工单附件：未找到记录 (DAO)", zap.Int("attachmentID", attachmentID), zap.Int("instanceID", instanceID))
		return fmt.Errorf("未找到附件 (ID: %d) 或附件不属于工单 (ID: %d)", attachmentID, instanceID)
	}
	i.logger.Debug("工单附件删除成功 (DAO)", zap.Int("attachmentID", attachmentID))
	return nil
}

// GetInstanceAttachments 获取指定工单的所有附件记录
func (i *instanceDAO) GetInstanceAttachments(ctx context.Context, instanceID int) ([]model.InstanceAttachment, error) {
	i.logger.Debug("开始获取工单附件列表 (DAO)", zap.Int("instanceID", instanceID))
	var attachments []model.InstanceAttachment
	if err := i.db.WithContext(ctx).Where("instance_id = ?", instanceID).Order("created_at DESC").Find(&attachments).Error; err != nil {
		i.logger.Error("获取工单附件列表失败 (DAO)", zap.Error(err), zap.Int("instanceID", instanceID))
		return nil, fmt.Errorf("获取工单 (ID: %d) 的附件列表失败: %w", instanceID, err)
	}
	i.logger.Debug("工单附件列表获取成功 (DAO)", zap.Int("count", len(attachments)))
	return attachments, nil
}
