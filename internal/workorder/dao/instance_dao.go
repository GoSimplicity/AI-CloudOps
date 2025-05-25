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
	"strings"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	ErrInstanceNotFound    = fmt.Errorf("工单实例不存在")
	ErrInstanceExists      = fmt.Errorf("工单实例已存在")
	ErrInstanceInvalidID   = fmt.Errorf("工单实例ID无效")
	ErrInstanceNilPointer  = fmt.Errorf("工单实例对象为空")
	ErrAttachmentNotFound  = fmt.Errorf("附件不存在")
	ErrAttachmentNotBelong = fmt.Errorf("附件不属于指定工单")
)

type InstanceDAO interface {
	CreateInstance(ctx context.Context, instance *model.Instance) error
	UpdateInstance(ctx context.Context, instance *model.Instance) error
	DeleteInstance(ctx context.Context, id int) error
	ListInstance(ctx context.Context, req *model.ListInstanceReq) (*model.ListResp[model.Instance], error)
	GetInstance(ctx context.Context, id int) (*model.Instance, error)
	GetInstanceWithRelations(ctx context.Context, id int) (*model.Instance, error)
	BatchUpdateInstanceStatus(ctx context.Context, ids []int, status int8) error

	// Flow methods
	CreateInstanceFlow(ctx context.Context, flow *model.InstanceFlow) error
	GetInstanceFlows(ctx context.Context, instanceID int) ([]model.InstanceFlow, error)
	BatchCreateInstanceFlows(ctx context.Context, flows []model.InstanceFlow) error

	// Comment methods
	CreateInstanceComment(ctx context.Context, comment *model.InstanceComment) error
	GetInstanceComments(ctx context.Context, instanceID int) ([]model.InstanceComment, error)
	GetInstanceCommentsTree(ctx context.Context, instanceID int) ([]model.InstanceComment, error)

	// Attachment methods
	CreateInstanceAttachment(ctx context.Context, attachment *model.InstanceAttachment) (*model.InstanceAttachment, error)
	DeleteInstanceAttachment(ctx context.Context, instanceID int, attachmentID int) error
	GetInstanceAttachments(ctx context.Context, instanceID int) ([]model.InstanceAttachment, error)
	BatchDeleteInstanceAttachments(ctx context.Context, instanceID int, attachmentIDs []int) error

	// Process methods
	GetProcess(ctx context.Context, processID int) (*model.Process, error)

	// Statistics methods
	GetInstanceStatistics(ctx context.Context, req *model.OverviewStatsReq) (*model.OverviewStatsResp, error)
	GetInstanceTrend(ctx context.Context, req *model.TrendStatsReq) (*model.TrendStatsResp, error)
	GetCategoryStatistics(ctx context.Context, req *model.CategoryStatsReq) (*model.CategoryStatsResp, error)
	GetUserPerformanceStatistics(ctx context.Context, req *model.PerformanceStatsReq) (*model.PerformanceStatsResp, error)

	// Business methods
	GetMyInstances(ctx context.Context, userID int, req *model.MyInstanceReq) (*model.ListResp[model.Instance], error)
	GetOverdueInstances(ctx context.Context) ([]model.Instance, error)
	TransferInstance(ctx context.Context, instanceID int, fromUserID int, toUserID int, comment string) error
}

type instanceDAO struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewInstanceDAO(db *gorm.DB, logger *zap.Logger) InstanceDAO {
	return &instanceDAO{
		db:     db,
		logger: logger,
	}
}

// CreateInstance 创建工单实例
func (d *instanceDAO) CreateInstance(ctx context.Context, instance *model.Instance) error {
	if instance == nil {
		return ErrInstanceNilPointer
	}

	// 处理零值时间
	d.normalizeTimeFields(instance)

	if err := d.db.WithContext(ctx).Create(instance).Error; err != nil {
		if d.isDuplicateKeyError(err) {
			d.logger.Warn("工单实例已存在", zap.String("title", instance.Title))
			return ErrInstanceExists
		}
		d.logger.Error("创建工单实例失败", zap.Error(err), zap.String("title", instance.Title))
		return fmt.Errorf("创建工单实例失败: %w", err)
	}

	d.logger.Info("创建工单实例成功", zap.Int("id", instance.ID), zap.String("title", instance.Title))
	return nil
}

// UpdateInstance 更新工单实例
func (d *instanceDAO) UpdateInstance(ctx context.Context, instance *model.Instance) error {
	if instance == nil {
		return ErrInstanceNilPointer
	}
	if instance.ID == 0 {
		return ErrInstanceInvalidID
	}

	// 处理零值时间
	d.normalizeTimeFields(instance)

	updateData := map[string]interface{}{
		"title":        instance.Title,
		"form_data":    instance.FormData,
		"current_step": instance.CurrentStep,
		"status":       instance.Status,
		"priority":     instance.Priority,
		"description":  instance.Description,
		"assignee_id":  instance.AssigneeID,
		"completed_at": instance.CompletedAt,
		"due_date":     instance.DueDate,
		"tags":         instance.Tags,
		"process_data": instance.ProcessData,
		"updated_at":   time.Now(),
	}

	result := d.db.WithContext(ctx).
		Model(&model.Instance{}).
		Where("id = ?", instance.ID).
		Updates(updateData)

	if result.Error != nil {
		d.logger.Error("更新工单实例失败", zap.Error(result.Error), zap.Int("id", instance.ID))
		return fmt.Errorf("更新工单实例失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		d.logger.Warn("工单实例不存在", zap.Int("id", instance.ID))
		return ErrInstanceNotFound
	}

	d.logger.Info("更新工单实例成功", zap.Int("id", instance.ID), zap.String("title", instance.Title))
	return nil
}

// DeleteInstance 删除工单实例
func (d *instanceDAO) DeleteInstance(ctx context.Context, id int) error {
	if id <= 0 {
		return ErrInstanceInvalidID
	}

	// 使用事务删除相关数据
	err := d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 删除相关的流转记录
		if err := tx.Where("instance_id = ?", id).Delete(&model.InstanceFlow{}).Error; err != nil {
			return fmt.Errorf("删除工单流转记录失败: %w", err)
		}

		// 删除相关的评论
		if err := tx.Where("instance_id = ?", id).Delete(&model.InstanceComment{}).Error; err != nil {
			return fmt.Errorf("删除工单评论失败: %w", err)
		}

		// 删除相关的附件记录（注意：这里只删除数据库记录，不删除实际文件）
		if err := tx.Where("instance_id = ?", id).Delete(&model.InstanceAttachment{}).Error; err != nil {
			return fmt.Errorf("删除工单附件记录失败: %w", err)
		}

		// 删除工单实例
		result := tx.Delete(&model.Instance{}, id)
		if result.Error != nil {
			return fmt.Errorf("删除工单实例失败: %w", result.Error)
		}

		if result.RowsAffected == 0 {
			return ErrInstanceNotFound
		}

		return nil
	})

	if err != nil {
		d.logger.Error("删除工单实例失败", zap.Error(err), zap.Int("id", id))
		return err
	}

	d.logger.Info("删除工单实例成功", zap.Int("id", id))
	return nil
}

// GetInstance 获取工单实例详情
func (d *instanceDAO) GetInstance(ctx context.Context, id int) (*model.Instance, error) {
	if id <= 0 {
		return nil, ErrInstanceInvalidID
	}

	var instance model.Instance
	err := d.db.WithContext(ctx).First(&instance, id).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			d.logger.Warn("工单实例不存在", zap.Int("id", id))
			return nil, ErrInstanceNotFound
		}
		d.logger.Error("获取工单实例失败", zap.Error(err), zap.Int("id", id))
		return nil, fmt.Errorf("获取工单实例失败: %w", err)
	}

	return &instance, nil
}

// GetInstanceWithRelations 获取工单实例及其关联数据
func (d *instanceDAO) GetInstanceWithRelations(ctx context.Context, id int) (*model.Instance, error) {
	if id <= 0 {
		return nil, ErrInstanceInvalidID
	}

	var instance model.Instance
	err := d.db.WithContext(ctx).
		Preload("Template").
		Preload("Process").
		Preload("Category").
		First(&instance, id).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			d.logger.Warn("工单实例不存在", zap.Int("id", id))
			return nil, ErrInstanceNotFound
		}
		d.logger.Error("获取工单实例及关联数据失败", zap.Error(err), zap.Int("id", id))
		return nil, fmt.Errorf("获取工单实例及关联数据失败: %w", err)
	}

	return &instance, nil
}

// ListInstance 获取工单实例列表
func (d *instanceDAO) ListInstance(ctx context.Context, req *model.ListInstanceReq) (*model.ListResp[model.Instance], error) {
	var instances []model.Instance
	var total int64

	db := d.db.WithContext(ctx).Model(&model.Instance{})

	// 构建查询条件
	db = d.buildInstanceListQuery(db, req)

	// 获取总数
	if err := db.Count(&total).Error; err != nil {
		d.logger.Error("获取工单实例总数失败", zap.Error(err))
		return nil, fmt.Errorf("获取工单实例总数失败: %w", err)
	}

	// 分页查询
	offset := (req.Page - 1) * req.Size
	err := db.Preload("Template").
		Preload("Process").
		Preload("Category").
		Order("created_at DESC").
		Offset(offset).
		Limit(req.Size).
		Find(&instances).Error

	if err != nil {
		d.logger.Error("获取工单实例列表失败", zap.Error(err))
		return nil, fmt.Errorf("获取工单实例列表失败: %w", err)
	}

	result := &model.ListResp[model.Instance]{
		Items: instances,
		Total: total,
	}

	d.logger.Info("获取工单实例列表成功",
		zap.Int("count", len(instances)),
		zap.Int64("total", total),
		zap.Int("page", req.Page),
		zap.Int("size", req.Size))

	return result, nil
}

// BatchUpdateInstanceStatus 批量更新工单状态
func (d *instanceDAO) BatchUpdateInstanceStatus(ctx context.Context, ids []int, status int8) error {
	if len(ids) == 0 {
		return fmt.Errorf("ID列表不能为空")
	}

	result := d.db.WithContext(ctx).
		Model(&model.Instance{}).
		Where("id IN ?", ids).
		Updates(map[string]interface{}{
			"status":     status,
			"updated_at": time.Now(),
		})

	if result.Error != nil {
		d.logger.Error("批量更新工单状态失败", zap.Error(result.Error), zap.Ints("ids", ids), zap.Int8("status", status))
		return fmt.Errorf("批量更新工单状态失败: %w", result.Error)
	}

	d.logger.Info("批量更新工单状态成功", zap.Ints("ids", ids), zap.Int8("status", status), zap.Int64("affected", result.RowsAffected))
	return nil
}

// CreateInstanceFlow 创建工单流程记录
func (d *instanceDAO) CreateInstanceFlow(ctx context.Context, flow *model.InstanceFlow) error {
	if flow == nil {
		return fmt.Errorf("流程记录对象为空")
	}

	if err := d.db.WithContext(ctx).Create(flow).Error; err != nil {
		d.logger.Error("创建工单流程记录失败", zap.Error(err), zap.Int("instanceID", flow.InstanceID))
		return fmt.Errorf("创建工单流程记录失败: %w", err)
	}

	d.logger.Info("创建工单流程记录成功", zap.Int("id", flow.ID), zap.Int("instanceID", flow.InstanceID))
	return nil
}

// GetInstanceFlows 获取工单流程记录
func (d *instanceDAO) GetInstanceFlows(ctx context.Context, instanceID int) ([]model.InstanceFlow, error) {
	if instanceID <= 0 {
		return nil, fmt.Errorf("工单ID无效")
	}

	var flows []model.InstanceFlow
	err := d.db.WithContext(ctx).
		Where("instance_id = ?", instanceID).
		Order("created_at ASC").
		Find(&flows).Error

	if err != nil {
		d.logger.Error("获取工单流程记录失败", zap.Error(err), zap.Int("instanceID", instanceID))
		return nil, fmt.Errorf("获取工单流程记录失败: %w", err)
	}

	return flows, nil
}

// BatchCreateInstanceFlows 批量创建工单流程记录
func (d *instanceDAO) BatchCreateInstanceFlows(ctx context.Context, flows []model.InstanceFlow) error {
	if len(flows) == 0 {
		return nil
	}

	if err := d.db.WithContext(ctx).CreateInBatches(flows, 100).Error; err != nil {
		d.logger.Error("批量创建工单流程记录失败", zap.Error(err), zap.Int("count", len(flows)))
		return fmt.Errorf("批量创建工单流程记录失败: %w", err)
	}

	d.logger.Info("批量创建工单流程记录成功", zap.Int("count", len(flows)))
	return nil
}

// CreateInstanceComment 创建工单评论
func (d *instanceDAO) CreateInstanceComment(ctx context.Context, comment *model.InstanceComment) error {
	if comment == nil {
		return fmt.Errorf("评论对象为空")
	}

	if err := d.db.WithContext(ctx).Create(comment).Error; err != nil {
		d.logger.Error("创建工单评论失败", zap.Error(err), zap.Int("instanceID", comment.InstanceID))
		return fmt.Errorf("创建工单评论失败: %w", err)
	}

	d.logger.Info("创建工单评论成功", zap.Int("id", comment.ID), zap.Int("instanceID", comment.InstanceID))
	return nil
}

// GetInstanceComments 获取工单评论
func (d *instanceDAO) GetInstanceComments(ctx context.Context, instanceID int) ([]model.InstanceComment, error) {
	if instanceID <= 0 {
		return nil, fmt.Errorf("工单ID无效")
	}

	var comments []model.InstanceComment
	err := d.db.WithContext(ctx).
		Where("instance_id = ?", instanceID).
		Order("created_at ASC").
		Find(&comments).Error

	if err != nil {
		d.logger.Error("获取工单评论失败", zap.Error(err), zap.Int("instanceID", instanceID))
		return nil, fmt.Errorf("获取工单评论失败: %w", err)
	}

	return comments, nil
}

// GetInstanceCommentsTree 获取工单评论树结构
func (d *instanceDAO) GetInstanceCommentsTree(ctx context.Context, instanceID int) ([]model.InstanceComment, error) {
	comments, err := d.GetInstanceComments(ctx, instanceID)
	if err != nil {
		return nil, err
	}

	// 构建评论树结构（这里需要根据实际的Comment结构来实现）
	// 简化实现，返回按层级排序的评论
	return comments, nil
}

// CreateInstanceAttachment 创建工单附件记录
func (d *instanceDAO) CreateInstanceAttachment(ctx context.Context, attachment *model.InstanceAttachment) (*model.InstanceAttachment, error) {
	if attachment == nil {
		return nil, fmt.Errorf("附件对象为空")
	}

	d.logger.Debug("开始创建工单附件", zap.Int("instanceID", attachment.InstanceID), zap.String("fileName", attachment.FileName))

	if err := d.db.WithContext(ctx).Create(attachment).Error; err != nil {
		d.logger.Error("创建工单附件失败", zap.Error(err), zap.Int("instanceID", attachment.InstanceID))
		return nil, fmt.Errorf("创建工单附件失败: %w", err)
	}

	d.logger.Info("创建工单附件成功", zap.Int("id", attachment.ID), zap.String("fileName", attachment.FileName))
	return attachment, nil
}

// DeleteInstanceAttachment 删除工单附件记录
func (d *instanceDAO) DeleteInstanceAttachment(ctx context.Context, instanceID int, attachmentID int) error {
	if instanceID <= 0 || attachmentID <= 0 {
		return fmt.Errorf("工单ID或附件ID无效")
	}

	d.logger.Debug("开始删除工单附件", zap.Int("instanceID", instanceID), zap.Int("attachmentID", attachmentID))

	result := d.db.WithContext(ctx).
		Where("id = ? AND instance_id = ?", attachmentID, instanceID).
		Delete(&model.InstanceAttachment{})

	if result.Error != nil {
		d.logger.Error("删除工单附件失败", zap.Error(result.Error), zap.Int("attachmentID", attachmentID))
		return fmt.Errorf("删除工单附件失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		d.logger.Warn("附件不存在或不属于指定工单", zap.Int("attachmentID", attachmentID), zap.Int("instanceID", instanceID))
		return ErrAttachmentNotBelong
	}

	d.logger.Info("删除工单附件成功", zap.Int("attachmentID", attachmentID))
	return nil
}

// GetInstanceAttachments 获取指定工单的所有附件记录
func (d *instanceDAO) GetInstanceAttachments(ctx context.Context, instanceID int) ([]model.InstanceAttachment, error) {
	if instanceID <= 0 {
		return nil, fmt.Errorf("工单ID无效")
	}

	d.logger.Debug("开始获取工单附件列表", zap.Int("instanceID", instanceID))

	var attachments []model.InstanceAttachment
	err := d.db.WithContext(ctx).
		Where("instance_id = ?", instanceID).
		Order("created_at DESC").
		Find(&attachments).Error

	if err != nil {
		d.logger.Error("获取工单附件列表失败", zap.Error(err), zap.Int("instanceID", instanceID))
		return nil, fmt.Errorf("获取工单附件列表失败: %w", err)
	}

	d.logger.Debug("获取工单附件列表成功", zap.Int("count", len(attachments)))
	return attachments, nil
}

// BatchDeleteInstanceAttachments 批量删除工单附件
func (d *instanceDAO) BatchDeleteInstanceAttachments(ctx context.Context, instanceID int, attachmentIDs []int) error {
	if instanceID <= 0 || len(attachmentIDs) == 0 {
		return fmt.Errorf("工单ID无效或附件ID列表为空")
	}

	result := d.db.WithContext(ctx).
		Where("instance_id = ? AND id IN ?", instanceID, attachmentIDs).
		Delete(&model.InstanceAttachment{})

	if result.Error != nil {
		d.logger.Error("批量删除工单附件失败", zap.Error(result.Error), zap.Ints("attachmentIDs", attachmentIDs))
		return fmt.Errorf("批量删除工单附件失败: %w", result.Error)
	}

	d.logger.Info("批量删除工单附件成功", zap.Ints("attachmentIDs", attachmentIDs), zap.Int64("affected", result.RowsAffected))
	return nil
}

// GetProcess 获取流程定义
func (d *instanceDAO) GetProcess(ctx context.Context, processID int) (*model.Process, error) {
	if processID <= 0 {
		return nil, fmt.Errorf("流程ID无效")
	}

	var process model.Process
	err := d.db.WithContext(ctx).
		Preload("FormDesign").
		Preload("Category").
		First(&process, processID).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			d.logger.Warn("流程定义不存在", zap.Int("processID", processID))
			return nil, ErrProcessNotFound
		}
		d.logger.Error("获取流程定义失败", zap.Error(err), zap.Int("processID", processID))
		return nil, fmt.Errorf("获取流程定义失败: %w", err)
	}

	return &process, nil
}

// GetInstanceStatistics 获取工单统计信息
func (d *instanceDAO) GetInstanceStatistics(ctx context.Context, req *model.OverviewStatsReq) (*model.OverviewStatsResp, error) {
	var stats model.OverviewStatsResp

	db := d.db.WithContext(ctx).Model(&model.Instance{})

	// 应用时间范围过滤
	if req.StartDate != nil && req.EndDate != nil {
		db = db.Where("created_at BETWEEN ? AND ?", req.StartDate, req.EndDate)
	}

	// 获取各状态统计
	var statusStats []struct {
		Status int8  `json:"status"`
		Count  int64 `json:"count"`
	}

	if err := db.Select("status, count(*) as count").Group("status").Find(&statusStats).Error; err != nil {
		d.logger.Error("获取状态统计失败", zap.Error(err))
		return nil, fmt.Errorf("获取状态统计失败: %w", err)
	}

	// 处理统计结果
	for _, stat := range statusStats {
		switch stat.Status {
		case model.InstanceStatusCompleted:
			stats.CompletedCount = stat.Count
		case model.InstanceStatusProcessing:
			stats.ProcessingCount = stat.Count
		case model.InstanceStatusPending:
			stats.PendingCount = stat.Count
		}
		stats.TotalCount += stat.Count
	}

	// 计算完成率
	if stats.TotalCount > 0 {
		stats.CompletionRate = float64(stats.CompletedCount) / float64(stats.TotalCount) * 100
	}

	// 获取超时工单数
	var overdueCount int64
	err := db.Where("due_date < ? AND status NOT IN ?", time.Now(),
		[]int8{model.InstanceStatusCompleted, model.InstanceStatusCancelled, model.InstanceStatusRejected}).
		Count(&overdueCount).Error
	if err != nil {
		d.logger.Error("获取超时工单数失败", zap.Error(err))
		return nil, fmt.Errorf("获取超时工单数失败: %w", err)
	}
	stats.OverdueCount = overdueCount

	// 获取今日创建和完成数
	today := time.Now().Truncate(24 * time.Hour)
	tomorrow := today.Add(24 * time.Hour)

	var todayCreated, todayCompleted int64
	db.Where("created_at >= ? AND created_at < ?", today, tomorrow).Count(&todayCreated)
	db.Where("completed_at >= ? AND completed_at < ?", today, tomorrow).Count(&todayCompleted)

	stats.TodayCreated = todayCreated
	stats.TodayCompleted = todayCompleted

	return &stats, nil
}

// GetInstanceTrend 获取工单趋势
func (d *instanceDAO) GetInstanceTrend(ctx context.Context, req *model.TrendStatsReq) (*model.TrendStatsResp, error) {
	var result model.TrendStatsResp

	db := d.db.WithContext(ctx).Model(&model.Instance{}).
		Where("created_at BETWEEN ? AND ?", req.StartDate, req.EndDate)

	// 应用分类过滤
	if req.CategoryID != nil {
		db = db.Where("category_id = ?", *req.CategoryID)
	}

	// 根据维度构建查询
	var dateFormat string
	switch req.Dimension {
	case "day":
		dateFormat = "%Y-%m-%d"
	case "week":
		dateFormat = "%Y-%u"
	case "month":
		dateFormat = "%Y-%m"
	default:
		dateFormat = "%Y-%m-%d"
	}

	var trendData []struct {
		Date            string `json:"date"`
		CreatedCount    int64  `json:"created_count"`
		CompletedCount  int64  `json:"completed_count"`
		ProcessingCount int64  `json:"processing_count"`
	}

	err := db.Select(fmt.Sprintf(`
		 DATE_FORMAT(created_at, '%s') as date,
		 COUNT(*) as created_count,
		 SUM(CASE WHEN status = ? THEN 1 ELSE 0 END) as completed_count,
		 SUM(CASE WHEN status = ? THEN 1 ELSE 0 END) as processing_count
	 `, dateFormat), model.InstanceStatusCompleted, model.InstanceStatusProcessing).
		Group("date").
		Order("date ASC").
		Find(&trendData).Error

	if err != nil {
		d.logger.Error("获取工单趋势失败", zap.Error(err))
		return nil, fmt.Errorf("获取工单趋势失败: %w", err)
	}

	// 构建返回结果
	for _, data := range trendData {
		result.Dates = append(result.Dates, data.Date)
		result.CreatedCounts = append(result.CreatedCounts, data.CreatedCount)
		result.CompletedCounts = append(result.CompletedCounts, data.CompletedCount)
		result.ProcessingCounts = append(result.ProcessingCounts, data.ProcessingCount)
	}

	return &result, nil
}

// GetCategoryStatistics 获取分类统计
func (d *instanceDAO) GetCategoryStatistics(ctx context.Context, req *model.CategoryStatsReq) (*model.CategoryStatsResp, error) {
	db := d.db.WithContext(ctx).Model(&model.Instance{}).
		Joins("LEFT JOIN category ON instance.category_id = category.id")

	// 应用时间范围过滤
	if req.StartDate != nil && req.EndDate != nil {
		db = db.Where("instance.created_at BETWEEN ? AND ?", req.StartDate, req.EndDate)
	}

	top := req.Top
	if top == 0 {
		top = 10
	}

	var categoryStats []struct {
		CategoryID   int    `json:"category_id"`
		CategoryName string `json:"category_name"`
		Count        int64  `json:"count"`
	}

	err := db.Select("instance.category_id, COALESCE(category.name, '未分类') as category_name, count(*) as count").
		Group("instance.category_id, category.name").
		Order("count DESC").
		Limit(top).
		Find(&categoryStats).Error

	if err != nil {
		d.logger.Error("获取分类统计失败", zap.Error(err))
		return nil, fmt.Errorf("获取分类统计失败: %w", err)
	}

	// 计算总数用于计算百分比
	var total int64
	for _, stat := range categoryStats {
		total += stat.Count
	}

	result := &model.CategoryStatsResp{
		Items: make([]model.CategoryStatsItem, len(categoryStats)),
	}

	for i, stat := range categoryStats {
		percentage := float64(0)
		if total > 0 {
			percentage = float64(stat.Count) / float64(total) * 100
		}

		result.Items[i] = model.CategoryStatsItem{
			CategoryID:   stat.CategoryID,
			CategoryName: stat.CategoryName,
			Count:        stat.Count,
			Percentage:   percentage,
		}
	}

	return result, nil
}

// GetUserPerformanceStatistics 获取用户绩效统计
func (d *instanceDAO) GetUserPerformanceStatistics(ctx context.Context, req *model.PerformanceStatsReq) (*model.PerformanceStatsResp, error) {
	db := d.db.WithContext(ctx).Model(&model.Instance{})

	// 应用时间范围过滤
	if req.StartDate != nil && req.EndDate != nil {
		db = db.Where("created_at BETWEEN ? AND ?", req.StartDate, req.EndDate)
	}

	// 应用用户过滤
	if req.UserID != nil {
		db = db.Where("assignee_id = ?", *req.UserID)
	}

	top := req.Top
	if top == 0 {
		top = 20
	}

	var userStats []struct {
		UserID            int     `json:"user_id"`
		UserName          string  `json:"user_name"`
		AssignedCount     int64   `json:"assigned_count"`
		CompletedCount    int64   `json:"completed_count"`
		OverdueCount      int64   `json:"overdue_count"`
		AvgResponseTime   float64 `json:"avg_response_time"`
		AvgProcessingTime float64 `json:"avg_processing_time"`
	}

	err := db.Select(`
		 assignee_id as user_id,
		 assignee_name as user_name,
		 COUNT(*) as assigned_count,
		 SUM(CASE WHEN status = ? THEN 1 ELSE 0 END) as completed_count,
		 SUM(CASE WHEN due_date < NOW() AND status NOT IN (?, ?, ?) THEN 1 ELSE 0 END) as overdue_count,
		 AVG(CASE WHEN completed_at IS NOT NULL THEN TIMESTAMPDIFF(HOUR, created_at, completed_at) END) as avg_processing_time
	 `, model.InstanceStatusCompleted,
		model.InstanceStatusCompleted, model.InstanceStatusCancelled, model.InstanceStatusRejected).
		Where("assignee_id IS NOT NULL").
		Group("assignee_id, assignee_name").
		Order("completed_count DESC").
		Limit(top).
		Find(&userStats).Error

	if err != nil {
		d.logger.Error("获取用户绩效统计失败", zap.Error(err))
		return nil, fmt.Errorf("获取用户绩效统计失败: %w", err)
	}

	result := &model.PerformanceStatsResp{
		Items: make([]model.PerformanceStatsItem, len(userStats)),
	}

	for i, stat := range userStats {
		completionRate := float64(0)
		if stat.AssignedCount > 0 {
			completionRate = float64(stat.CompletedCount) / float64(stat.AssignedCount) * 100
		}

		result.Items[i] = model.PerformanceStatsItem{
			UserID:            stat.UserID,
			UserName:          stat.UserName,
			AssignedCount:     stat.AssignedCount,
			CompletedCount:    stat.CompletedCount,
			CompletionRate:    completionRate,
			AvgResponseTime:   stat.AvgResponseTime,
			AvgProcessingTime: stat.AvgProcessingTime,
			OverdueCount:      stat.OverdueCount,
		}
	}

	return result, nil
}

// GetMyInstances 获取我的工单
func (d *instanceDAO) GetMyInstances(ctx context.Context, userID int, req *model.MyInstanceReq) (*model.ListResp[model.Instance], error) {
	var instances []model.Instance
	var total int64

	db := d.db.WithContext(ctx).Model(&model.Instance{})

	// 根据类型过滤
	switch req.Type {
	case "created":
		db = db.Where("creator_id = ?", userID)
	case "assigned":
		db = db.Where("assignee_id = ?", userID)
	default:
		db = db.Where("creator_id = ? OR assignee_id = ?", userID, userID)
	}

	// 构建其他查询条件
	db = d.buildMyInstanceQuery(db, req)

	// 获取总数
	if err := db.Count(&total).Error; err != nil {
		d.logger.Error("获取我的工单总数失败", zap.Error(err), zap.Int("userID", userID))
		return nil, fmt.Errorf("获取我的工单总数失败: %w", err)
	}

	// 分页查询
	offset := (req.Page - 1) * req.Size
	err := db.Preload("Template").
		Preload("Process").
		Preload("Category").
		Order("created_at DESC").
		Offset(offset).
		Limit(req.Size).
		Find(&instances).Error

	if err != nil {
		d.logger.Error("获取我的工单列表失败", zap.Error(err), zap.Int("userID", userID))
		return nil, fmt.Errorf("获取我的工单列表失败: %w", err)
	}

	result := &model.ListResp[model.Instance]{
		Items: instances,
		Total: total,
	}

	return result, nil
}

// GetOverdueInstances 获取超时工单
func (d *instanceDAO) GetOverdueInstances(ctx context.Context) ([]model.Instance, error) {
	var instances []model.Instance

	err := d.db.WithContext(ctx).
		Where("due_date < ? AND status NOT IN ?", time.Now(),
			[]int8{model.InstanceStatusCompleted, model.InstanceStatusCancelled, model.InstanceStatusRejected}).
		Preload("Template").
		Preload("Process").
		Preload("Category").
		Find(&instances).Error

	if err != nil {
		d.logger.Error("获取超时工单失败", zap.Error(err))
		return nil, fmt.Errorf("获取超时工单失败: %w", err)
	}

	return instances, nil
}

// TransferInstance 转移工单
func (d *instanceDAO) TransferInstance(ctx context.Context, instanceID int, fromUserID int, toUserID int, comment string) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 更新工单处理人
		result := tx.Model(&model.Instance{}).
			Where("id = ? AND assignee_id = ?", instanceID, fromUserID).
			Updates(map[string]interface{}{
				"assignee_id": toUserID,
				"updated_at":  time.Now(),
			})

		if result.Error != nil {
			return fmt.Errorf("更新工单处理人失败: %w", result.Error)
		}

		if result.RowsAffected == 0 {
			return fmt.Errorf("工单不存在或不属于当前用户")
		}

		// 创建转移记录
		flow := &model.InstanceFlow{
			InstanceID: instanceID,
			StepID:     "transfer",
			StepName:   "工单转移",
			Action:     "transfer",
			OperatorID: fromUserID,
			Comment:    comment,
			FromStepID: "current",
			ToStepID:   "current",
		}

		if err := tx.Create(flow).Error; err != nil {
			return fmt.Errorf("创建转移记录失败: %w", err)
		}

		return nil
	})
}

// 辅助方法

// buildInstanceListQuery 构建工单列表查询条件
func (d *instanceDAO) buildInstanceListQuery(db *gorm.DB, req *model.ListInstanceReq) *gorm.DB {
	// 搜索条件
	if req.Search != "" {
		searchPattern := "%" + strings.TrimSpace(req.Search) + "%"
		db = db.Where("title LIKE ? OR description LIKE ?", searchPattern, searchPattern)
	}

	// 状态过滤
	if req.Status != nil {
		db = db.Where("status = ?", *req.Status)
	}

	// 优先级过滤
	if req.Priority != nil {
		db = db.Where("priority = ?", *req.Priority)
	}

	// 分类过滤
	if req.CategoryID != nil {
		db = db.Where("category_id = ?", *req.CategoryID)
	}

	// 创建人过滤
	if req.CreatorID != nil {
		db = db.Where("creator_id = ?", *req.CreatorID)
	}

	// 处理人过滤
	if req.AssigneeID != nil {
		db = db.Where("assignee_id = ?", *req.AssigneeID)
	}

	// 流程过滤
	if req.ProcessID != nil {
		db = db.Where("process_id = ?", *req.ProcessID)
	}

	// 模板过滤
	if req.TemplateID != nil {
		db = db.Where("template_id = ?", *req.TemplateID)
	}

	// 时间范围过滤
	if req.StartDate != nil && req.EndDate != nil {
		db = db.Where("created_at BETWEEN ? AND ?", req.StartDate, req.EndDate)
	} else if req.StartDate != nil {
		db = db.Where("created_at >= ?", req.StartDate)
	} else if req.EndDate != nil {
		db = db.Where("created_at <= ?", req.EndDate)
	}

	// 标签过滤
	if len(req.Tags) > 0 {
		for _, tag := range req.Tags {
			db = db.Where("tags LIKE ?", "%"+tag+"%")
		}
	}

	// 超时过滤
	if req.Overdue != nil {
		if *req.Overdue {
			db = db.Where("due_date < ? AND status NOT IN ?", time.Now(),
				[]int8{model.InstanceStatusCompleted, model.InstanceStatusCancelled, model.InstanceStatusRejected})
		} else {
			db = db.Where("due_date >= ? OR status IN ?", time.Now(),
				[]int8{model.InstanceStatusCompleted, model.InstanceStatusCancelled, model.InstanceStatusRejected})
		}
	}

	return db
}

// buildMyInstanceQuery 构建我的工单查询条件
func (d *instanceDAO) buildMyInstanceQuery(db *gorm.DB, req *model.MyInstanceReq) *gorm.DB {
	// 搜索条件
	if req.Search != "" {
		searchPattern := "%" + strings.TrimSpace(req.Search) + "%"
		db = db.Where("title LIKE ? OR description LIKE ?", searchPattern, searchPattern)
	}

	// 状态过滤
	if req.Status != nil {
		db = db.Where("status = ?", *req.Status)
	}

	// 优先级过滤
	if req.Priority != nil {
		db = db.Where("priority = ?", *req.Priority)
	}

	// 分类过滤
	if req.CategoryID != nil {
		db = db.Where("category_id = ?", *req.CategoryID)
	}

	// 流程过滤
	if req.ProcessID != nil {
		db = db.Where("process_id = ?", *req.ProcessID)
	}

	// 时间范围过滤
	if req.StartDate != nil && req.EndDate != nil {
		db = db.Where("created_at BETWEEN ? AND ?", req.StartDate, req.EndDate)
	} else if req.StartDate != nil {
		db = db.Where("created_at >= ?", req.StartDate)
	} else if req.EndDate != nil {
		db = db.Where("created_at <= ?", req.EndDate)
	}

	return db
}

// normalizeTimeFields 处理零值时间字段
func (d *instanceDAO) normalizeTimeFields(instance *model.Instance) {
	if instance.CompletedAt != nil && instance.CompletedAt.IsZero() {
		instance.CompletedAt = nil
	}
	if instance.DueDate != nil && instance.DueDate.IsZero() {
		instance.DueDate = nil
	}
}

// ConvertToResp 转换为响应结构
func (d *instanceDAO) ConvertFlowToResp(flow *model.InstanceFlow) *model.InstanceFlowResp {
	return &model.InstanceFlowResp{
		ID:           flow.ID,
		InstanceID:   flow.InstanceID,
		StepID:       flow.StepID,
		StepName:     flow.StepName,
		Action:       flow.Action,
		OperatorID:   flow.OperatorID,
		OperatorName: flow.OperatorName,
		Comment:      flow.Comment,
		FormData:     map[string]interface{}(flow.FormData), // 直接转换
		Duration:     flow.Duration,
		FromStepID:   flow.FromStepID,
		ToStepID:     flow.ToStepID,
		CreatedAt:    flow.CreatedAt,
	}
}

// isDuplicateKeyError 判断是否为重复键错误
func (d *instanceDAO) isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	return err == gorm.ErrDuplicatedKey ||
		strings.Contains(err.Error(), "UNIQUE constraint failed") ||
		strings.Contains(err.Error(), "Duplicate entry") ||
		strings.Contains(err.Error(), "duplicate key")
}
