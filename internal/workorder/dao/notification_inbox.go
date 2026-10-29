package dao

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// DeliverInboxMessage 在一个事务中写站内信和成功投递账本；提交不明可安全重试。
func (d *workorderInstanceDAO) DeliverInboxMessage(ctx context.Context, eventID uint64, token string, notificationID, userID, instanceID int, eventType, subject, content string) error {
	if eventID == 0 || token == "" || notificationID <= 0 || userID <= 0 || instanceID <= 0 || len([]rune(subject)) > 500 || len(content) > 60000 {
		return errors.New("站内通知数据无效")
	}
	address := strconv.Itoa(userID)
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(address)))
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var event model.WorkorderNotificationEvent
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND status = ? AND lease_token = ? AND lease_until > ?", eventID, "processing", token, time.Now().UTC()).First(&event).Error; err != nil {
			return errors.New("通知事件租约已失效")
		}
		if event.InstanceID != instanceID || event.EventType != eventType {
			return errors.New("站内通知事件不匹配")
		}
		var delivery model.WorkorderNotificationDelivery
		err := tx.Where("event_id = ? AND notification_id = ? AND channel = ? AND recipient_hash = ?", eventID, notificationID, "in_app", hash).First(&delivery).Error
		if err == nil {
			if delivery.Status != "success" {
				return errors.New("站内投递账本状态异常")
			}
			var count int64
			if err := tx.Model(&model.WorkorderInboxMessage{}).Where("event_id = ? AND notification_id = ? AND user_id = ?", eventID, notificationID, userID).Count(&count).Error; err != nil {
				return err
			}
			if count != 1 {
				return errors.New("站内投递账本与消息不一致")
			}
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		var user model.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND enable = ?", userID, 1).First(&user).Error; err != nil {
			return errors.New("站内通知用户不存在或已禁用")
		}
		message := model.WorkorderInboxMessage{EventID: eventID, NotificationID: notificationID, UserID: userID, InstanceID: instanceID, EventType: eventType, Subject: subject, Content: content}
		if err := tx.Create(&message).Error; err != nil {
			return err
		}
		return tx.Create(&model.WorkorderNotificationDelivery{EventID: eventID, NotificationID: notificationID, Channel: "in_app", RecipientHash: hash, Status: "success", Attempts: 1}).Error
	})
}

func (d *workorderInstanceDAO) ListInboxMessages(ctx context.Context, userID, page, size int, unread bool) (*model.ListResp[*model.WorkorderInboxMessage], error) {
	if userID <= 0 {
		return nil, errors.New("用户身份无效")
	}
	page, size = ValidatePagination(page, size)
	query := DBWithContext(ctx, d.db).Model(&model.WorkorderInboxMessage{}).Where("user_id = ? AND deleted_at IS NULL", userID)
	if unread {
		query = query.Where("read_at IS NULL")
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}
	rows := make([]*model.WorkorderInboxMessage, 0)
	if err := query.Order("id DESC").Limit(size).Offset((page - 1) * size).Find(&rows).Error; err != nil {
		return nil, err
	}
	return &model.ListResp[*model.WorkorderInboxMessage]{Items: rows, Total: total}, nil
}

func (d *workorderInstanceDAO) CountUnreadInboxMessages(ctx context.Context, userID int) (int64, error) {
	if userID <= 0 {
		return 0, errors.New("用户身份无效")
	}
	var count int64
	err := DBWithContext(ctx, d.db).Model(&model.WorkorderInboxMessage{}).Where("user_id = ? AND deleted_at IS NULL AND read_at IS NULL", userID).Count(&count).Error
	return count, err
}

func (d *workorderInstanceDAO) MarkInboxRead(ctx context.Context, userID int, id uint64) error {
	if userID <= 0 || id == 0 {
		return errors.New("请求无效")
	}
	now := time.Now().UTC()
	result := DBWithContext(ctx, d.db).Model(&model.WorkorderInboxMessage{}).Where("id = ? AND user_id = ? AND deleted_at IS NULL", id, userID).Updates(map[string]interface{}{"read_at": gorm.Expr("COALESCE(read_at, ?)", now)})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		var count int64
		if err := DBWithContext(ctx, d.db).Model(&model.WorkorderInboxMessage{}).Where("id = ? AND user_id = ? AND deleted_at IS NULL", id, userID).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			return errors.New("消息不存在")
		}
	}
	return nil
}

func (d *workorderInstanceDAO) MarkAllInboxRead(ctx context.Context, userID int) error {
	if userID <= 0 {
		return errors.New("用户身份无效")
	}
	return DBWithContext(ctx, d.db).Model(&model.WorkorderInboxMessage{}).Where("user_id = ? AND deleted_at IS NULL AND read_at IS NULL", userID).Update("read_at", time.Now().UTC()).Error
}

func (d *workorderInstanceDAO) DeleteInboxMessage(ctx context.Context, userID int, id uint64) error {
	if userID <= 0 || id == 0 {
		return errors.New("请求无效")
	}
	result := DBWithContext(ctx, d.db).Model(&model.WorkorderInboxMessage{}).Where("id = ? AND user_id = ? AND deleted_at IS NULL", id, userID).Updates(map[string]interface{}{"deleted_at": time.Now().UTC(), "subject": "", "content": ""})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("消息不存在")
	}
	return nil
}
