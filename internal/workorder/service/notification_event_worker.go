package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/notification"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const eventLease = 4 * time.Minute
const eventTimeout = 3 * time.Minute
const eventMaxAttempts = 10

var errNoNotificationEvent = errors.New("没有可处理通知事件")

type eventDeliveryKey struct{}
type eventDeliveryLedger struct {
	db      *gorm.DB
	eventID uint64
	token   string
}

func (l *eventDeliveryLedger) delivered(ctx context.Context, notificationID int, channel, hash string) (bool, error) {
	var delivery model.WorkorderNotificationDelivery
	err := l.db.WithContext(ctx).Where("event_id = ? AND notification_id = ? AND channel = ? AND recipient_hash = ?", l.eventID, notificationID, channel, hash).First(&delivery).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	switch delivery.Status {
	case "success":
		return true, nil
	case "sending", "uncertain":
		return false, errors.New("前次发送结果未知，需人工核对后处理")
	case "failed":
		if delivery.Attempts > delivery.MaxRetries {
			return false, errors.New("重试次数已用尽")
		}
		if delivery.NextAttemptAt != nil && delivery.NextAttemptAt.After(time.Now().UTC()) {
			return false, errors.New("尚未到达重试时间")
		}
	}
	return false, nil
}

// 更新收件人状态前锁住当前事件行，过期消费者无法写入新租约的数据
func (l *eventDeliveryLedger) markDelivery(notificationID int, channel, hash, status string, maxRetries, interval int) error {
	if maxRetries < 0 || maxRetries > 100 || interval < 0 || uint64(interval) > uint64((1<<63-1)/time.Minute) {
		return errors.New("通知重试配置无效")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return l.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var event model.WorkorderNotificationEvent
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND status = ? AND lease_token = ? AND lease_until > ?", l.eventID, "processing", l.token, time.Now().UTC()).First(&event).Error; err != nil {
			return errors.New("通知事件租约已失效")
		}
		var delivery model.WorkorderNotificationDelivery
		result := tx.Where("event_id = ? AND notification_id = ? AND channel = ? AND recipient_hash = ?", l.eventID, notificationID, channel, hash).First(&delivery)
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			if status != "sending" {
				return errors.New("缺少发送记录")
			}
			return tx.Create(&model.WorkorderNotificationDelivery{EventID: l.eventID, NotificationID: notificationID, Channel: channel, RecipientHash: hash, Status: "sending", Attempts: 1, MaxRetries: maxRetries, RetryIntervalMinutes: interval}).Error
		}
		if result.Error != nil {
			return result.Error
		}
		if status == "sending" {
			if delivery.Status != "failed" || delivery.Attempts > delivery.MaxRetries || (delivery.NextAttemptAt != nil && delivery.NextAttemptAt.After(time.Now().UTC())) {
				return errors.New("收件人当前不可重发")
			}
			return tx.Model(&delivery).Updates(map[string]interface{}{"status": "sending", "attempts": gorm.Expr("attempts + 1"), "next_attempt_at": nil}).Error
		}
		if delivery.Status != "sending" {
			return errors.New("收件人状态已变更")
		}
		values := map[string]interface{}{"status": status}
		if status == "failed" {
			values["next_attempt_at"] = time.Now().UTC().Add(time.Duration(delivery.RetryIntervalMinutes) * time.Minute)
		}
		return tx.Model(&delivery).Updates(values).Error
	})
}
func isDeliveryUncertain(ctx context.Context, err error) bool {
	return ctx.Err() != nil || !errors.Is(err, notification.ErrDeliveryRejected)
}

type eventPlanChannelKey struct{}
type eventPlanChannel struct {
	Config     model.WorkorderNotification `json:"config"`
	Channel    string                      `json:"channel"`
	Recipients []RecipientInfo             `json:"recipients"`
	Subject    string                      `json:"subject"`
	Content    string                      `json:"content"`
}

// preparePlan 先解析全部配置和接收人；任何读取失败都不会发送
func (w *NotificationEventWorker) preparePlan(ctx context.Context, event *model.WorkorderNotificationEvent, instance *model.WorkorderInstance) ([]eventPlanChannel, error) {
	configs, err := w.service.dao.GetActiveNotificationsByEventType(ctx, event.EventType, instance.ProcessID)
	if err != nil {
		return nil, err
	}
	plan := make([]eventPlanChannel, 0)
	for _, config := range configs {
		if !notificationMatchesInstanceScope(config, instance) {
			continue
		}
		if err = validateAutomaticNotification(config); err != nil {
			return nil, err
		}
		recipients, err := w.service.getRecipients(ctx, config, instance)
		if err != nil {
			return nil, err
		}
		subject, content := w.service.buildMessageContent(ctx, config, instance, event.EventType, event.Content)
		reduced := model.WorkorderNotification{Model: model.Model{ID: config.ID}, Priority: config.Priority, MaxRetries: config.MaxRetries, RetryInterval: config.RetryInterval}
		for _, channel := range config.Channels {
			addresses, err := w.service.resolveChannelRecipients(ctx, channel, recipients)
			if err != nil {
				return nil, err
			}
			plan = append(plan, eventPlanChannel{Config: reduced, Channel: channel, Recipients: addresses, Subject: subject, Content: content})
		}
	}
	return plan, nil
}
func (w *NotificationEventWorker) deliverPlan(ctx context.Context, event *model.WorkorderNotificationEvent, instance *model.WorkorderInstance, plan []eventPlanChannel) error {
	var failures []error
	for i := range plan {
		channel := &plan[i]
		channelCtx := context.WithValue(ctx, eventPlanChannelKey{}, channel)
		if err := w.service.sendChannelNotification(channelCtx, &channel.Config, instance, channel.Channel, channel.Recipients, event.EventType, event.SenderID, event.Content); err != nil {
			failures = append(failures, err)
		}
	}
	return errors.Join(failures...)
}

type NotificationEventWorker struct {
	db            *gorm.DB
	service       *workorderNotificationService
	logger        *zap.Logger
	retentionDays int
}

func NewNotificationEventWorker(db *gorm.DB, svc WorkorderNotificationService, logger *zap.Logger) (*NotificationEventWorker, error) {
	service, ok := svc.(*workorderNotificationService)
	if !ok || service == nil || db == nil || logger == nil {
		return nil, errors.New("工单通知事件工作器依赖不可用")
	}
	days, err := notificationRetentionDays(os.Getenv("WORKORDER_EVENT_RETENTION_DAYS"))
	if err != nil {
		return nil, err
	}
	return &NotificationEventWorker{db: db, service: service, logger: logger, retentionDays: days}, nil
}
func notificationRetentionDays(value string) (int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 90, nil
	}
	days, err := strconv.Atoi(value)
	if err != nil || days < 1 || days > 3650 {
		return 0, errors.New("WORKORDER_EVENT_RETENTION_DAYS 必须为1至3650的整数")
	}
	return days, nil
}

// 保留人工审计与任何未完成记录，每批最多清理100个无人工处置的完成事件
func (w *NotificationEventWorker) cleanupCompleted(ctx context.Context, now time.Time) (int, error) {
	if w.retentionDays <= 0 {
		return 0, nil
	}
	cutoff := now.UTC().AddDate(0, 0, -w.retentionDays)
	deleted := 0
	err := w.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var events []model.WorkorderNotificationEvent
		query := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Select("id").Where("status IN ? AND updated_at < ?", []string{"success", "skipped"}, cutoff).Where("NOT EXISTS (SELECT 1 FROM cl_workorder_notification_resolution r WHERE r.event_id = cl_workorder_notification_event.id)").Order("id").Limit(100).Find(&events)
		if query.Error != nil {
			return query.Error
		}
		if len(events) == 0 {
			return nil
		}
		ids := make([]uint64, 0, len(events))
		for _, event := range events {
			ids = append(ids, event.ID)
		}
		if err := tx.Where("event_id IN ?", ids).Delete(&model.WorkorderNotificationDelivery{}).Error; err != nil {
			return err
		}
		result := tx.Where("id IN ? AND status IN ? AND updated_at < ?", ids, []string{"success", "skipped"}, cutoff).Delete(&model.WorkorderNotificationEvent{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != int64(len(ids)) {
			return errors.New("通知事件清理时状态发生变化")
		}
		deleted = len(ids)
		return nil
	})
	return deleted, err
}
func (w *NotificationEventWorker) Run(ctx context.Context) error {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	nextCleanup := time.Time{}
	for {
		if w.retentionDays > 0 && !time.Now().Before(nextCleanup) {
			count, err := w.cleanupCompleted(ctx, time.Now())
			if err != nil && ctx.Err() == nil {
				w.logger.Warn("通知事件清理失败", zap.String("error_type", fmt.Sprintf("%T", err)))
			}
			if err != nil || count < 100 {
				nextCleanup = time.Now().Add(time.Hour)
			}
		}
		err := w.processOne(ctx)
		if err == nil {
			if ctx.Err() != nil {
				return nil
			}
			continue
		}
		if !errors.Is(err, errNoNotificationEvent) && ctx.Err() == nil {
			w.logger.Error("工单通知事件处理失败", zap.String("error_type", fmt.Sprintf("%T", err)))
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}
func (w *NotificationEventWorker) processOne(ctx context.Context) error {
	now := time.Now().UTC()
	var candidates []model.WorkorderNotificationEvent
	if err := w.db.WithContext(ctx).Where("(status = ? AND next_attempt_at <= ?) OR (status = ? AND lease_until < ?)", "pending", now, "processing", now).Order("id").Limit(10).Find(&candidates).Error; err != nil {
		return err
	}
	for _, event := range candidates {
		var tokenBytes [16]byte
		if _, err := rand.Read(tokenBytes[:]); err != nil {
			return err
		}
		token := hex.EncodeToString(tokenBytes[:])
		now = time.Now().UTC()
		until := now.Add(eventLease)
		claim := w.db.WithContext(ctx).Model(&model.WorkorderNotificationEvent{}).Where("id = ? AND ((status = ? AND next_attempt_at <= ?) OR (status = ? AND lease_until < ?))", event.ID, "pending", now, "processing", now).Updates(map[string]interface{}{"status": "processing", "lease_token": token, "lease_until": until, "attempts": gorm.Expr("attempts + 1")})
		if claim.Error != nil {
			return claim.Error
		}
		if claim.RowsAffected != 1 {
			continue
		}
		event.Attempts++
		workCtx, cancel := context.WithTimeout(ctx, eventTimeout)
		workCtx = context.WithValue(workCtx, eventDeliveryKey{}, &eventDeliveryLedger{db: w.db, eventID: event.ID, token: token})
		var instance model.WorkorderInstance
		err := json.Unmarshal(event.Snapshot, &instance)
		if err == nil && instance.ID != event.InstanceID {
			err = errors.New("事件快照工单ID不匹配")
		}
		var plan []eventPlanChannel
		if err == nil && len(event.Plan) == 0 {
			plan, err = w.preparePlan(workCtx, &event, &instance)
			if err == nil {
				var encoded []byte
				encoded, err = json.Marshal(plan)
				if err == nil {
					update := w.db.WithContext(workCtx).Model(&model.WorkorderNotificationEvent{}).Where("id = ? AND status = ? AND lease_token = ? AND lease_until > ? AND plan IS NULL", event.ID, "processing", token, time.Now().UTC()).Update("plan", encoded)
					if update.Error != nil {
						err = update.Error
					} else if update.RowsAffected != 1 {
						err = errors.New("事件计划保存失败或租约已失效")
					}
				}
			}
		} else if err == nil {
			err = json.Unmarshal(event.Plan, &plan)
		}
		if err == nil {
			err = w.deliverPlan(workCtx, &event, &instance, plan)
		}
		cancel()
		finishCtx, finishCancel := context.WithTimeout(context.Background(), 10*time.Second)
		updates := map[string]interface{}{"lease_token": "", "lease_until": nil}
		var successful int64
		if err == nil {
			if queryErr := w.db.WithContext(finishCtx).Model(&model.WorkorderNotificationDelivery{}).Where("event_id = ? AND status = ?", event.ID, "success").Count(&successful).Error; queryErr != nil {
				finishCancel()
				return queryErr
			}
		}
		var uncertain int64
		queryErr := w.db.WithContext(finishCtx).Model(&model.WorkorderNotificationDelivery{}).Where("event_id = ? AND status IN ?", event.ID, []string{"sending", "uncertain"}).Count(&uncertain).Error
		if queryErr != nil {
			finishCancel()
			return queryErr
		}
		var failed []model.WorkorderNotificationDelivery
		if uncertain == 0 {
			if queryErr := w.db.WithContext(finishCtx).Where("event_id = ? AND status = ?", event.ID, "failed").Find(&failed).Error; queryErr != nil {
				finishCancel()
				return queryErr
			}
		}
		var nextRetry *time.Time
		for _, delivery := range failed {
			if delivery.Attempts <= delivery.MaxRetries && delivery.NextAttemptAt != nil && (nextRetry == nil || delivery.NextAttemptAt.Before(*nextRetry)) {
				nextRetry = delivery.NextAttemptAt
			}
		}
		switch {
		case uncertain > 0:
			updates["status"] = "uncertain"
			updates["last_error"] = "上次发送结果不确定，需人工核对；已停止自动重试"
		case err == nil && len(failed) > 0:
			updates["status"] = "failed"
			updates["last_error"] = "已有未成功投递，配置或收件人不再匹配，需人工核对"
		case err == nil && successful == 0:
			updates["snapshot"] = []byte("{}")
			updates["plan"] = nil
			updates["content"] = ""
			updates["status"] = "skipped"
			updates["last_error"] = "没有匹配的启用通知配置或收件人"
		case err == nil:
			updates["snapshot"] = []byte("{}")
			updates["plan"] = nil
			updates["content"] = ""
			updates["status"] = "success"
			updates["last_error"] = ""
		case nextRetry != nil:
			updates["status"] = "pending"
			updates["last_error"] = "已确认拒绝，等待重试"
			updates["next_attempt_at"] = *nextRetry
		case len(failed) > 0:
			updates["status"] = "failed"
			updates["last_error"] = "已确认拒绝且重试次数已用尽"
		case event.Attempts >= eventMaxAttempts:
			updates["status"] = "failed"
			updates["last_error"] = "通知处理失败，需人工检查发送日志"
		default:
			updates["status"] = "pending"
			updates["last_error"] = "通知处理失败，等待重试"
			updates["next_attempt_at"] = time.Now().UTC().Add(time.Duration(event.Attempts) * time.Minute)
		}
		finish := w.db.WithContext(finishCtx).Model(&model.WorkorderNotificationEvent{}).Where("id = ? AND status = ? AND lease_token = ? AND lease_until > ?", event.ID, "processing", token, time.Now().UTC()).Updates(updates)
		finishCancel()
		if finish.Error != nil {
			return finish.Error
		}
		if finish.RowsAffected != 1 {
			return errors.New("通知事件租约已失效")
		}
		return nil
	}
	return errNoNotificationEvent
}
