package dao

import (
	"context"
	"errors"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"strings"
	"time"
)

// ListNotificationEvents 仅读取可公开的状态字段，不返回事件快照或消息正文
func (d *workorderInstanceDAO) ListNotificationEvents(ctx context.Context, page, size int, status string) (*model.ListResp[*model.WorkorderNotificationEventStatus], error) {
	page, size = ValidatePagination(page, size)
	query := DBWithContext(ctx, d.db).Model(&model.WorkorderNotificationEvent{})
	if status != "" {
		query = query.Where("status = ?", status)
	}
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return nil, err
	}
	var rows []*model.WorkorderNotificationEventStatus
	if err := query.Select("id, instance_id, event_type, status, attempts, next_attempt_at, last_error, created_at, updated_at").Order("id DESC").Limit(size).Offset((page - 1) * size).Scan(&rows).Error; err != nil {
		return nil, err
	}
	return &model.ListResp[*model.WorkorderNotificationEventStatus]{Items: rows, Total: count}, nil
}
func (d *workorderInstanceDAO) ListNotificationEventDeliveries(ctx context.Context, id uint64) ([]*model.WorkorderNotificationDeliveryStatus, error) {
	var rows []*model.WorkorderNotificationDeliveryStatus
	err := DBWithContext(ctx, d.db).Model(&model.WorkorderNotificationDelivery{}).Select("id,event_id,notification_id,channel,recipient_hash,status,attempts,max_retries,retry_interval_minutes,next_attempt_at,created_at,updated_at").Where("event_id = ?", id).Order("id").Find(&rows).Error
	return rows, err
}

// ResolveNotificationDelivery 仅对终止/不确定事件进行人工处置，同事务记录操作者
func (d *workorderInstanceDAO) ResolveNotificationDelivery(ctx context.Context, eventID uint64, req *model.ResolveWorkorderNotificationDeliveryReq, operatorID int) error {
	if req == nil || eventID == 0 || req.DeliveryID == 0 || operatorID <= 0 || (req.Action != "delivered" && req.Action != "retry") || len(strings.TrimSpace(req.Reason)) < 5 || len(req.Reason) > 500 {
		return errors.New("无效的人工处置请求")
	}
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var event model.WorkorderNotificationEvent
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND status IN ?", eventID, []string{"failed", "uncertain"}).First(&event).Error; err != nil {
			return errors.New("事件不存在或不可处置")
		}
		var delivery model.WorkorderNotificationDelivery
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND event_id = ? AND status IN ?", req.DeliveryID, eventID, []string{"failed", "uncertain", "sending"}).First(&delivery).Error; err != nil {
			return errors.New("投递记录不存在或不可处置")
		}
		updates := map[string]interface{}{"status": "success", "next_attempt_at": nil}
		if req.Action == "retry" {
			updates = map[string]interface{}{"status": "failed", "attempts": 0, "next_attempt_at": time.Now().UTC()}
		}
		result := tx.Model(&model.WorkorderNotificationDelivery{}).Where("id = ? AND status = ?", delivery.ID, delivery.Status).Updates(updates)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errors.New("投递状态发生变化")
		}
		if err := tx.Create(&model.WorkorderNotificationResolution{EventID: eventID, DeliveryID: delivery.ID, Action: req.Action, OperatorID: operatorID, Reason: strings.TrimSpace(req.Reason)}).Error; err != nil {
			return err
		}
		var unresolved int64
		if err := tx.Model(&model.WorkorderNotificationDelivery{}).Where("event_id = ? AND status IN ?", eventID, []string{"sending", "uncertain"}).Count(&unresolved).Error; err != nil {
			return err
		}
		if unresolved == 0 {
			return tx.Model(&event).Updates(map[string]interface{}{"status": "pending", "next_attempt_at": time.Now().UTC(), "last_error": ""}).Error
		}
		return nil
	})
}
func (d *workorderInstanceDAO) ListNotificationEventResolutions(ctx context.Context, eventID uint64) ([]*model.WorkorderNotificationResolution, error) {
	var rows []*model.WorkorderNotificationResolution
	err := DBWithContext(ctx, d.db).Where("event_id = ?", eventID).Order("id DESC").Limit(100).Find(&rows).Error
	return rows, err
}

// RetryNotificationEvent 只恢复未生成投递记录的失败事件，避免批量重发已有收件人
func (d *workorderInstanceDAO) RetryNotificationEvent(ctx context.Context, eventID uint64, reason string, operatorID int) error {
	reason = strings.TrimSpace(reason)
	if eventID == 0 || operatorID <= 0 || len(reason) < 5 || len(reason) > 500 {
		return errors.New("无效的人工重试请求")
	}
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var event model.WorkorderNotificationEvent
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND status = ?", eventID, "failed").First(&event).Error; err != nil {
			return errors.New("事件不存在或不是失败状态")
		}
		var count int64
		if err := tx.Model(&model.WorkorderNotificationDelivery{}).Where("event_id = ?", eventID).Count(&count).Error; err != nil {
			return err
		}
		if count != 0 {
			return errors.New("事件已有投递记录，请逐个核对处置")
		}
		if err := tx.Create(&model.WorkorderNotificationResolution{EventID: eventID, DeliveryID: 0, Action: "retry_event", OperatorID: operatorID, Reason: reason}).Error; err != nil {
			return err
		}
		update := tx.Model(&model.WorkorderNotificationEvent{}).Where("id = ? AND status = ?", eventID, "failed").Updates(map[string]interface{}{"status": "pending", "attempts": 0, "next_attempt_at": time.Now().UTC(), "last_error": ""})
		if update.Error != nil {
			return update.Error
		}
		if update.RowsAffected != 1 {
			return errors.New("事件状态已变化")
		}
		return nil
	})
}
