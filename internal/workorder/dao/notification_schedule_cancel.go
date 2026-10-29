package dao

import (
	"context"
	"errors"
	"strings"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// CancelScheduledNotification 只撤销尚未开始或仅明确拒绝过的排期子事件。
func (d *workorderInstanceDAO) CancelScheduledNotification(ctx context.Context, eventID uint64, reason string, operatorID int) error {
	reason = strings.TrimSpace(reason)
	if eventID == 0 || operatorID <= 0 || len(reason) < 5 || len(reason) > 500 {
		return errors.New("无效的排期取消请求")
	}
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var event model.WorkorderNotificationEvent
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND status = ? AND parent_event_id > 0 AND parent_event_id <> id", eventID, "pending").First(&event).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("事件不存在或不是可取消的排期")
			}
			return err
		}
		var unsafe int64
		if err := tx.Model(&model.WorkorderNotificationDelivery{}).Where("event_id = ? AND status <> ?", eventID, "failed").Count(&unsafe).Error; err != nil {
			return err
		}
		if unsafe > 0 {
			return errors.New("事件含已发送、发送中或结果未知的投递记录，不可取消")
		}
		update := tx.Model(&model.WorkorderNotificationEvent{}).Where("id = ? AND status = ? AND parent_event_id > 0 AND parent_event_id <> id", eventID, "pending").Updates(map[string]interface{}{
			"status": "cancelled", "plan": nil, "snapshot": []byte{}, "content": "", "last_error": "", "lease_token": "", "lease_until": nil,
		})
		if update.Error != nil {
			return update.Error
		}
		if update.RowsAffected != 1 {
			return errors.New("事件状态已变化")
		}
		return tx.Create(&model.WorkorderNotificationResolution{EventID: eventID, DeliveryID: 0, Action: "cancel_scheduled", OperatorID: operatorID, Reason: reason}).Error
	})
}
