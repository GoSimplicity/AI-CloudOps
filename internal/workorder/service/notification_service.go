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
	"crypto/sha256"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	userDao "github.com/GoSimplicity/AI-CloudOps/internal/system/dao"
	workorderDao "github.com/GoSimplicity/AI-CloudOps/internal/workorder/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/notification"
	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/utils"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type WorkorderNotificationService interface {
	CreateNotification(ctx context.Context, req *model.CreateWorkorderNotificationReq) error
	UpdateNotification(ctx context.Context, req *model.UpdateWorkorderNotificationReq) error
	DeleteNotification(ctx context.Context, req *model.DeleteWorkorderNotificationReq) error
	ListNotification(ctx context.Context, req *model.ListWorkorderNotificationReq) (*model.ListResp[*model.WorkorderNotification], error)
	DetailNotification(ctx context.Context, req *model.DetailWorkorderNotificationReq) (*model.WorkorderNotification, error)
	GetSendLogs(ctx context.Context, req *model.ListWorkorderNotificationLogReq) (*model.ListResp[*model.WorkorderNotificationLog], error)
	ListNotificationEvents(ctx context.Context, req *model.ListWorkorderNotificationEventsReq) (*model.ListResp[*model.WorkorderNotificationEventStatus], error)
	ListNotificationEventDeliveries(ctx context.Context, id uint64) ([]*model.WorkorderNotificationDeliveryStatus, error)
	ResolveNotificationDelivery(ctx context.Context, eventID uint64, req *model.ResolveWorkorderNotificationDeliveryReq, operatorID int) error
	ListNotificationEventResolutions(ctx context.Context, eventID uint64) ([]*model.WorkorderNotificationResolution, error)
	RetryNotificationEvent(ctx context.Context, eventID uint64, reason string, operatorID int) error
	TestSendNotification(ctx context.Context, req *model.TestSendWorkorderNotificationReq) error
	SendWorkorderNotification(ctx context.Context, instanceID int, eventType string, customContent ...string) error
	SendNotificationByChannels(ctx context.Context, channels []string, recipient, subject, content string) error
	GetAvailableChannels() *model.ListResp[*model.WorkorderNotificationChannel]
}

// notificationSender 保持公开构造函数不变，便于在业务层验证请求参数而不触发外发
type notificationSender interface {
	SendNotification(context.Context, *notification.SendRequest) (*notification.SendResponse, error)
	GetAvailableChannels() []string
}

type workorderNotificationService struct {
	dao             workorderDao.WorkorderNotificationDAO
	logger          *zap.Logger
	notificationMgr notificationSender
	instanceDAO     workorderDao.WorkorderInstanceDAO
	userDAO         userDao.UserDAO
}

func NewWorkorderNotificationService(dao workorderDao.WorkorderNotificationDAO, notificationMgr *notification.Manager, logger *zap.Logger, instanceDAO workorderDao.WorkorderInstanceDAO, userDAO userDao.UserDAO) WorkorderNotificationService {
	var sender notificationSender
	if notificationMgr != nil {
		sender = notificationMgr
	}
	return &workorderNotificationService{
		logger:          logger,
		dao:             dao,
		notificationMgr: sender,
		instanceDAO:     instanceDAO,
		userDAO:         userDAO,
	}
}

func (s *workorderNotificationService) CreateNotification(ctx context.Context, req *model.CreateWorkorderNotificationReq) error {
	if err := validateImmediateNotification(req.TriggerType, req.ScheduledTime, req.RepeatInterval, req.TriggerCondition, false); err != nil {
		return err
	}
	if err := validateNotificationDestinations(req.Channels, req.RecipientTypes, req.RecipientUsers, req.RecipientRoles, req.RecipientDepts, req.RecipientAddresses); err != nil {
		return err
	}
	return s.dao.CreateNotification(ctx, req)
}

func (s *workorderNotificationService) UpdateNotification(ctx context.Context, req *model.UpdateWorkorderNotificationReq) error {
	if err := validateImmediateNotification(req.TriggerType, req.ScheduledTime, req.RepeatInterval, req.TriggerCondition, true); err != nil {
		return err
	}
	existing, err := s.dao.GetNotificationByID(ctx, req.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("通知配置不存在")
		}
		return fmt.Errorf("查询通知配置失败: %w", err)
	}

	if err := validateNotificationActivation(existing, req); err != nil {
		return err
	}
	channels, types, users, roles, depts, addresses := existing.Channels, existing.RecipientTypes, existing.RecipientUsers, existing.RecipientRoles, existing.RecipientDepts, existing.RecipientAddresses
	if req.Channels != nil {
		channels = req.Channels
	}
	if req.RecipientTypes != nil {
		types = req.RecipientTypes
	}
	if req.RecipientUsers != nil {
		users = req.RecipientUsers
	}
	if req.RecipientRoles != nil {
		roles = req.RecipientRoles
	}
	if req.RecipientDepts != nil {
		depts = req.RecipientDepts
	}
	if req.RecipientAddresses != nil {
		addresses = req.RecipientAddresses
	}
	if err := validateNotificationDestinations(channels, types, users, roles, depts, addresses); err != nil {
		return err
	}
	return s.dao.UpdateNotification(ctx, req)
}

func (s *workorderNotificationService) DeleteNotification(ctx context.Context, req *model.DeleteWorkorderNotificationReq) error {
	_, err := s.dao.GetNotificationByID(ctx, req.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("通知配置不存在")
		}
		return fmt.Errorf("查询通知配置失败: %w", err)
	}

	return s.dao.DeleteNotification(ctx, req)
}

func (s *workorderNotificationService) ListNotification(ctx context.Context, req *model.ListWorkorderNotificationReq) (*model.ListResp[*model.WorkorderNotification], error) {
	result, err := s.dao.ListNotification(ctx, req)
	if err != nil {
		s.logger.Error("获取通知配置列表失败", zap.Error(err))
		return nil, fmt.Errorf("获取通知配置列表失败: %w", err)
	}
	return result, nil
}

func (s *workorderNotificationService) DetailNotification(ctx context.Context, req *model.DetailWorkorderNotificationReq) (*model.WorkorderNotification, error) {
	return s.dao.DetailNotification(ctx, req)
}

func (s *workorderNotificationService) GetSendLogs(ctx context.Context, req *model.ListWorkorderNotificationLogReq) (*model.ListResp[*model.WorkorderNotificationLog], error) {
	result, err := s.dao.GetSendLogs(ctx, req)
	if err != nil {
		s.logger.Error("获取发送日志失败", zap.Error(err))
		return nil, fmt.Errorf("获取发送日志失败: %w", err)
	}
	return result, nil
}

func (s *workorderNotificationService) TestSendNotification(ctx context.Context, req *model.TestSendWorkorderNotificationReq) error {
	notificationConfig, err := s.dao.GetNotificationByID(ctx, req.NotificationID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("通知配置不存在")
		}
		return fmt.Errorf("查询通知配置失败: %w", err)
	}

	if notificationConfig.Status != 1 {
		return errors.New("通知配置已禁用，无法发送")
	}

	var senderID int
	if uid := ctx.Value("user_id"); uid != nil {
		if id, ok := uid.(int); ok {
			senderID = id
		}
	}

	selected := req.Channel
	if selected == "" {
		if len(notificationConfig.Channels) != 1 {
			return fmt.Errorf("多渠道测试必须指定channel")
		}
		selected = notificationConfig.Channels[0]
	}
	found := false
	for _, channel := range notificationConfig.Channels {
		if selected == channel {
			found = true
		}
	}
	if !found {
		return fmt.Errorf("测试channel不在通知配置中")
	}
	if _, err := notification.NormalizeAddress(selected, req.Recipient); err != nil {
		return err
	}
	for _, channel := range []string{selected} {
		recipientAddr := strings.TrimSpace(req.Recipient)
		if recipientAddr == "" {
			return fmt.Errorf("测试通知必须提供明确的接收人地址")
		}

		// 创建模拟的实例ID用于测试
		testInstanceID := 999999

		sendRequest := &notification.SendRequest{
			MaxRetries:           &notificationConfig.MaxRetries,
			RetryIntervalMinutes: &notificationConfig.RetryInterval,
			Subject:              notificationConfig.SubjectTemplate,
			Content:              notificationConfig.MessageTemplate,
			Priority:             notificationConfig.Priority,
			Channel:              channel,
			RecipientType:        channel,
			RecipientID:          "test_user",
			RecipientAddr:        recipientAddr,
			RecipientName:        "测试用户",
			EventType:            "test",
			InstanceID:           &testInstanceID,
			Templates:            make(map[string]string),
			Metadata: map[string]interface{}{
				"notification_id": notificationConfig.ID,
				"sender_id":       senderID,
			},
		}

		sendRequest.Templates["workorder_id"] = fmt.Sprintf("%d", testInstanceID)
		sendRequest.Templates["serial_number"] = fmt.Sprintf("WO-%d", testInstanceID)
		sendRequest.Templates["title"] = "系统测试工单 - 通知功能验证"
		sendRequest.Templates["description"] = "系统测试工单，验证工单通知功能是否正常工作。"
		sendRequest.Templates["operator_name"] = "系统管理员"
		sendRequest.Templates["assignee_name"] = "运维工程师"
		sendRequest.Templates["priority_level"] = fmt.Sprintf("%d", int(notificationConfig.Priority))
		sendRequest.Templates["priority_text"] = notification.FormatPriority(notificationConfig.Priority)
		sendRequest.Templates["status"] = "测试进行中"
		sendRequest.Templates["created_time"] = time.Now().Format("2006-01-02 15:04:05")
		sendRequest.Templates["updated_time"] = time.Now().Format("2006-01-02 15:04:05")
		sendRequest.Templates["event_type"] = notification.GetEventTypeText("test")
		sendRequest.Templates["notification_time"] = time.Now().Format("2006-01-02 15:04:05")
		sendRequest.Templates["company_name"] = "AI-CloudOps"
		sendRequest.Templates["platform_name"] = "运维管理平台"
		sendRequest.Templates["department"] = "技术运维部"
		sendRequest.Templates["test_content"] = "本次测试验证了系统通知功能的完整性，包括邮件发送、飞书消息推送等多个渠道的有效性。"
		response, err := s.notificationMgr.SendNotification(ctx, sendRequest)

		log := &model.WorkorderNotificationLog{
			NotificationID: notificationConfig.ID,
			EventType:      "test",
			Channel:        channel,
			RecipientType:  "test",
			RecipientID:    "test_user",
			RecipientName:  "测试用户",
			RecipientAddr:  recipientAddr,
			Subject:        notificationConfig.SubjectTemplate,
			Content:        notificationConfig.MessageTemplate,
			Status:         2,
			SendAt:         time.Now(),
			SenderID:       senderID,
		}

		if err != nil {
			log.Status = 4
			log.ErrorMessage = err.Error()
			_ = s.dao.AddSendLog(ctx, log)
			return fmt.Errorf("测试通知发送失败: %w", err)
		} else if response != nil {
			log.Status = 3
			if response.ExternalID != "" {
				log.ResponseData = map[string]interface{}{
					"external_id": response.ExternalID,
				}
			}
			if response.Cost != nil {
				log.Cost = response.Cost
			}
		}

		if err := s.dao.AddSendLog(ctx, log); err != nil {
			s.logger.Error("记录发送日志失败", zap.Error(err))
		}
	}

	return s.dao.IncrementSentCount(ctx, notificationConfig.ID)
}

// SendWorkorderNotification 发送工单相关通知
func (s *workorderNotificationService) SendWorkorderNotification(ctx context.Context, instanceID int, eventType string, customContent ...string) error {
	instance, err := s.instanceDAO.GetInstanceByID(ctx, instanceID)
	if err != nil {
		s.logger.Error("获取工单实例失败",
			zap.Int("instance_id", instanceID),
			zap.Error(err))
		return fmt.Errorf("获取工单实例失败: %w", err)
	}

	return s.sendInstanceNotification(ctx, instance, eventType, 0, customContent...)
}

// sendInstanceNotification 支持删除工单后的不可变事件快照
func (s *workorderNotificationService) sendInstanceNotification(ctx context.Context, instance *model.WorkorderInstance, eventType string, senderID int, customContent ...string) error {
	instanceID := instance.ID
	if uid := ctx.Value("user_id"); senderID == 0 && uid != nil {
		if id, ok := uid.(int); ok {
			senderID = id
		}
	}

	notifications, err := s.dao.GetActiveNotificationsByEventType(ctx, eventType, instance.ProcessID)
	if err != nil {
		s.logger.Error("获取通知配置失败",
			zap.String("event_type", eventType),
			zap.Int("process_id", instance.ProcessID),
			zap.Error(err))
		return fmt.Errorf("获取通知配置失败: %w", err)
	}

	if len(notifications) == 0 {
		s.logger.Info("没有找到匹配的通知配置",
			zap.String("event_type", eventType),
			zap.Int("process_id", instance.ProcessID))
		return nil
	}

	var failures []error
	for _, notification := range notifications {
		if err := s.processNotification(ctx, notification, instance, eventType, senderID, customContent...); err != nil {
			s.logger.Error("处理通知配置失败",
				zap.Int("notification_id", notification.ID),
				zap.Int("instance_id", instanceID),
				zap.String("error_type", fmt.Sprintf("%T", err)))
			failures = append(failures, err)
			continue
		}
	}

	if len(failures) > 0 {
		return errors.Join(failures...)
	}

	s.logger.Info("工单通知发送完成",
		zap.Int("instance_id", instanceID),
		zap.String("event_type", eventType),
		zap.Int("notification_count", len(notifications)))

	return nil
}

func (s *workorderNotificationService) processNotification(ctx context.Context, notification *model.WorkorderNotification,
	instance *model.WorkorderInstance, eventType string, senderID int, customContent ...string) error {
	if !notificationMatchesInstanceScope(notification, instance) {
		return nil
	}
	if err := validateAutomaticNotification(notification); err != nil {
		return err
	}

	recipients, err := s.getRecipients(ctx, notification, instance)
	if err != nil {
		return fmt.Errorf("获取接收人失败: %w", err)
	}

	if len(recipients) == 0 {
		s.logger.Info("没有找到接收人",
			zap.Int("notification_id", notification.ID),
			zap.Int("instance_id", instance.ID))
		return nil
	}

	var wg sync.WaitGroup
	channelErrors := make(chan error, len(notification.Channels))

	for _, channel := range notification.Channels {
		wg.Add(1)
		go func(ch string) {
			defer wg.Done()

			channelCtx := ctx

			if err := s.sendChannelNotification(channelCtx, notification, instance, ch, recipients, eventType, senderID, customContent...); err != nil {
				s.logger.Error("发送渠道通知失败",
					zap.String("channel", ch),
					zap.Int("notification_id", notification.ID),
					zap.String("error_type", fmt.Sprintf("%T", err)))
				channelErrors <- fmt.Errorf("渠道 %s 发送失败: %w", ch, err)
			} else {
				s.logger.Info("渠道通知发送成功",
					zap.String("channel", ch),
					zap.Int("notification_id", notification.ID))
			}
		}(channel)
	}

	wg.Wait()
	close(channelErrors)

	var errors []string
	for err := range channelErrors {
		errors = append(errors, err.Error())
	}

	if len(errors) > 0 {
		return fmt.Errorf("通知渠道失败: %s", strings.Join(errors, "; "))
	}

	return nil
}

func (s *workorderNotificationService) getRecipients(ctx context.Context, notification *model.WorkorderNotification,
	instance *model.WorkorderInstance) ([]RecipientInfo, error) {

	var recipients []RecipientInfo

	for _, recipientType := range notification.RecipientTypes {
		switch recipientType {
		case model.RecipientTypeCreator:
			recipients = append(recipients, RecipientInfo{
				ID:   fmt.Sprintf("%d", instance.OperatorID),
				Name: instance.OperatorName,
				Type: recipientType,
			})
		case model.RecipientTypeAssignee:
			if instance.AssigneeID != nil {
				assigneeName := "处理人"
				if user, err := s.userDAO.GetByID(ctx, *instance.AssigneeID); err == nil {
					assigneeName = user.RealName
				}

				recipients = append(recipients, RecipientInfo{
					ID:   fmt.Sprintf("%d", *instance.AssigneeID),
					Name: assigneeName,
					Type: recipientType,
				})
			}
		case model.RecipientTypeUser:
			for _, userIDStr := range notification.RecipientUsers {
				userID, err := strconv.Atoi(userIDStr)
				if err != nil || userID <= 0 {
					return nil, fmt.Errorf("无效的用户ID")
				}

				userName := "指定用户"
				if user, err := s.userDAO.GetByID(ctx, userID); err == nil {
					userName = user.RealName
				}

				recipients = append(recipients, RecipientInfo{
					ID:   userIDStr,
					Name: userName,
					Type: recipientType,
				})
			}
		case model.RecipientTypeRole:
			ids := make([]int, 0, len(notification.RecipientRoles))
			for _, raw := range notification.RecipientRoles {
				id, err := strconv.Atoi(raw)
				if err != nil || id <= 0 {
					return nil, fmt.Errorf("无效角色ID: %s", raw)
				}
				ids = append(ids, id)
			}
			users, err := s.userDAO.GetByRoleIDs(ctx, ids)
			if err != nil {
				return nil, fmt.Errorf("查询角色接收人失败: %w", err)
			}
			for _, user := range users {
				recipients = append(recipients, RecipientInfo{ID: strconv.Itoa(user.ID), Name: user.RealName, Type: recipientType})
			}
		case model.RecipientTypeDept:
			ids, err := recipientIDs(notification.RecipientDepts)
			if err != nil {
				return nil, err
			}
			users, err := s.userDAO.GetByDepartmentIDs(ctx, ids)
			if err != nil {
				return nil, fmt.Errorf("查询部门接收人失败: %w", err)
			}
			for _, user := range users {
				recipients = append(recipients, RecipientInfo{ID: strconv.Itoa(user.ID), Name: user.RealName, Type: recipientType})
			}
		case model.RecipientTypeCustom:
			if len(notification.RecipientAddresses) == 0 {
				return nil, fmt.Errorf("自定义接收地址不能为空")
			}
			for _, destination := range notification.RecipientAddresses {
				normalized, err := normalizeDestination(destination.Channel, destination.Address)
				if err != nil {
					return nil, err
				}
				recipients = append(recipients, RecipientInfo{ID: fmt.Sprintf("custom:%s:%x", destination.Channel, sha256.Sum256([]byte(normalized))), Name: destination.Name, Type: recipientType, Channel: destination.Channel, Address: normalized})
			}
		default:
			return nil, fmt.Errorf("未知接收人类型: %s", recipientType)
		}
	}

	unique := make([]RecipientInfo, 0, len(recipients))
	seen := make(map[string]bool)
	for _, recipient := range recipients {
		if !seen[recipient.ID] {
			seen[recipient.ID] = true
			unique = append(unique, recipient)
		}
	}
	if len(unique) == 0 {
		return nil, fmt.Errorf("没有可用的通知接收人")
	}
	return unique, nil
}

// sendChannelNotification 通过指定渠道发送通知
func (s *workorderNotificationService) sendChannelNotification(ctx context.Context, notificationConfig *model.WorkorderNotification,
	instance *model.WorkorderInstance, channel string, recipients []RecipientInfo, eventType string, senderID int, customContent ...string) error {

	var subject, content string
	var destinations []RecipientInfo
	if planned, ok := ctx.Value(eventPlanChannelKey{}).(*eventPlanChannel); ok {
		subject, content, destinations = planned.Subject, planned.Content, planned.Recipients
	} else {
		subject, content = s.buildMessageContent(ctx, notificationConfig, instance, eventType, customContent...)
		var err error
		destinations, err = s.resolveChannelRecipients(ctx, channel, recipients)
		if err != nil {
			return err
		}
	}
	var wg sync.WaitGroup
	recipientErrors := make(chan error, 2*len(destinations)+1)
	sem := make(chan struct{}, 8)
	for _, recipient := range destinations {
		select {
		case sem <- struct{}{}:
		case <-ctx.Done():
			recipientErrors <- ctx.Err()
			break
		}
		if ctx.Err() != nil {
			break
		}
		wg.Add(1)
		go func(rec RecipientInfo) {
			defer wg.Done()
			defer func() { <-sem }()
			recipientCtx := ctx
			recipientAddr := rec.Address

			recipientType := s.getRecipientTypeForChannel(channel)

			sendRequest := &notification.SendRequest{
				MaxRetries:           &notificationConfig.MaxRetries,
				RetryIntervalMinutes: &notificationConfig.RetryInterval,
				Subject:              subject,
				Content:              content,
				Priority:             notificationConfig.Priority,
				Channel:              channel,
				RecipientType:        recipientType,
				RecipientID:          rec.ID,
				RecipientAddr:        recipientAddr,
				RecipientName:        rec.Name,
				InstanceID:           &instance.ID,
				EventType:            eventType,
				Metadata: map[string]interface{}{
					"notification_id": notificationConfig.ID,
					"instance_id":     instance.ID,
					"sender_id":       senderID,
					"recipient_type":  rec.Type,
				},
			}

			ledger, _ := recipientCtx.Value(eventDeliveryKey{}).(*eventDeliveryLedger)
			var deliveryKey string
			if ledger != nil {
				deliveryKey = fmt.Sprintf("%x", sha256.Sum256([]byte(recipientAddr)))
				already, checkErr := ledger.delivered(recipientCtx, notificationConfig.ID, channel, deliveryKey)
				if checkErr != nil {
					recipientErrors <- checkErr
					return
				}
				if already {
					return
				}
				if markErr := ledger.markDelivery(notificationConfig.ID, channel, deliveryKey, "sending", notificationConfig.MaxRetries, notificationConfig.RetryInterval); markErr != nil {
					recipientErrors <- markErr
					return
				}
				// 每次工作器领取仅调用渠道一次，重试由持久化预算控制
				noRetry := 0
				sendRequest.MaxRetries = &noRetry
				sendRequest.MessageID = fmt.Sprintf("workorder-%d-%d-%s-%s", ledger.eventID, notificationConfig.ID, channel, deliveryKey)
			}
			response, err := s.notificationMgr.SendNotification(recipientCtx, sendRequest)
			if ledger != nil {
				status := "failed"
				if err == nil {
					status = "success"
				} else if isDeliveryUncertain(recipientCtx, err) {
					// SMTP底层可能在上下文取消后仍执行；不能自动重复发送
					status = "uncertain"
				}
				if saveErr := ledger.markDelivery(notificationConfig.ID, channel, deliveryKey, status, notificationConfig.MaxRetries, notificationConfig.RetryInterval); saveErr != nil {
					recipientErrors <- saveErr
					return
				}
			}

			log := &model.WorkorderNotificationLog{
				NotificationID: notificationConfig.ID,
				InstanceID:     &instance.ID,
				EventType:      eventType,
				Channel:        channel,
				RecipientType:  rec.Type,
				RecipientID:    rec.ID,
				RecipientName:  rec.Name,
				RecipientAddr:  recipientAddr,
				Subject:        subject,
				Content:        content,
				Status:         2,
				SendAt:         time.Now(),
				SenderID:       senderID,
			}

			if err != nil {
				log.Status = 4
				log.ErrorMessage = err.Error()
				s.logger.Error("发送通知失败",
					zap.String("channel", channel),
					zap.String("error_type", fmt.Sprintf("%T", err)))
				recipientErrors <- fmt.Errorf("接收人 %s 发送失败: %w", recipientAddr, err)
			} else if response != nil {
				log.Status = 3
				if response.ExternalID != "" {
					log.ResponseData = map[string]interface{}{
						"external_id": response.ExternalID,
					}
				}
				if response.Cost != nil {
					log.Cost = response.Cost
				}
				s.logger.Info("通知发送成功", zap.String("channel", channel))
			}

			if err := s.dao.AddSendLog(recipientCtx, log); err != nil {
				recipientErrors <- fmt.Errorf("记录发送结果失败: %w", err)
			}
		}(recipient)
	}

	wg.Wait()
	close(recipientErrors)

	var errors []string
	for err := range recipientErrors {
		errors = append(errors, err.Error())
	}

	if len(errors) > 0 {
		return fmt.Errorf("通知发送失败: %s", strings.Join(errors, "; "))
	}

	return nil
}

func (s *workorderNotificationService) buildMessageContent(ctx context.Context, notificationConfig *model.WorkorderNotification,
	instance *model.WorkorderInstance, eventType string, customContent ...string) (string, string) {

	// 创建发送请求对象，用于模板渲染
	sendRequest := &notification.SendRequest{
		Subject:    notificationConfig.SubjectTemplate,
		Content:    notificationConfig.MessageTemplate,
		Priority:   notificationConfig.Priority,
		EventType:  eventType,
		InstanceID: &instance.ID,
		Templates:  make(map[string]string),
		Metadata:   make(map[string]interface{}),
	}

	sendRequest.Templates["workorder_id"] = fmt.Sprintf("%d", instance.ID)
	sendRequest.Templates["serial_number"] = instance.SerialNumber
	sendRequest.Templates["title"] = instance.Title
	sendRequest.Templates["description"] = instance.Description
	sendRequest.Templates["operator_name"] = instance.OperatorName
	sendRequest.Templates["priority_level"] = fmt.Sprintf("%d", int(instance.Priority))
	sendRequest.Templates["priority_text"] = notification.FormatPriority(instance.Priority)
	sendRequest.Templates["status"] = utils.GetInstanceStatusName(instance.Status)
	sendRequest.Templates["created_time"] = instance.CreatedAt.Format("2006-01-02 15:04:05")
	sendRequest.Templates["event_type"] = notification.GetEventTypeText(eventType)
	sendRequest.Templates["event_type_text"] = notification.GetEventTypeText(eventType)
	sendRequest.Templates["notification_time"] = time.Now().Format("2006-01-02 15:04:05")
	sendRequest.Templates["company_name"] = "AI-CloudOps"
	sendRequest.Templates["platform_name"] = "运维管理平台"
	sendRequest.Templates["department"] = "技术运维部"

	assigneeName := "待分配"
	if instance.AssigneeID != nil {
		if user, err := s.userDAO.GetByID(ctx, *instance.AssigneeID); err == nil && user != nil {
			assigneeName = user.RealName
		}
	}
	sendRequest.Templates["assignee_name"] = assigneeName

	// 如果有更新时间，添加更新时间
	if !instance.UpdatedAt.IsZero() {
		sendRequest.Templates["updated_time"] = instance.UpdatedAt.Format("2006-01-02 15:04:05")
	} else {
		sendRequest.Templates["updated_time"] = sendRequest.Templates["created_time"]
	}

	// 如果有自定义内容，添加到变量中
	if len(customContent) > 0 && customContent[0] != "" {
		sendRequest.Templates["custom_content"] = customContent[0]
	} else {
		sendRequest.Templates["custom_content"] = ""
	}

	// 渲染主题
	subject := notificationConfig.SubjectTemplate
	if subject == "" {
		subject = fmt.Sprintf("【AI-CloudOps】工单通知 - %s", instance.Title)
	} else {
		renderedSubject, _ := notification.RenderTemplate(subject, sendRequest)
		subject = renderedSubject
	}

	content := notificationConfig.MessageTemplate
	if content == "" {
		content = fmt.Sprintf(`尊敬的用户，您好！

您收到一条来自AI-CloudOps运维管理平台的工单通知：

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📋 工单基本信息
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
工单编号：%s
工单标题：%s
当前状态：%s
优先级别：%s
操作人员：%s
处理人员：%s
事件类型：%s
创建时间：%s

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📝 工单详情
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
%s

%s

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

此消息由AI-CloudOps运维管理平台发送，请及时处理相关工单。
如有疑问，请联系技术运维部门。

AI-CloudOps 技术运维部
发送时间：%s`,
			instance.SerialNumber,
			instance.Title,
			utils.GetInstanceStatusName(instance.Status),
			notification.FormatPriority(instance.Priority),
			instance.OperatorName,
			assigneeName,
			notification.GetEventTypeText(eventType),
			instance.CreatedAt.Format("2006-01-02 15:04:05"),
			instance.Description,
			sendRequest.Templates["custom_content"],
			time.Now().Format("2006-01-02 15:04:05"))
	} else {
		renderedContent, _ := notification.RenderTemplate(content, sendRequest)
		content = renderedContent
	}

	s.logger.Debug("消息内容构建完成", zap.Int("instance_id", instance.ID))

	return subject, content
}

// getRecipientAddress 根据渠道类型获取接收人地址
func (s *workorderNotificationService) getRecipientAddress(ctx context.Context, recipient RecipientInfo, channel string) string {
	if recipient.Channel != "" {
		if recipient.Channel == channel {
			return recipient.Address
		}
		return ""
	}
	userID, err := strconv.Atoi(recipient.ID)
	if err != nil {
		s.logger.Error("无效的用户ID",
			zap.String("recipient_id", recipient.ID),
			zap.Error(err))
		return ""
	}

	user, err := s.userDAO.GetByID(ctx, userID)
	if err != nil {
		s.logger.Error("获取用户信息失败",
			zap.Int("user_id", userID),
			zap.Error(err))
		return ""
	}

	if user == nil || user.Enable != 1 {
		return ""
	}
	switch channel {
	case model.NotificationChannelEmail:
		if user.Email == "" {
			s.logger.Warn("用户没有配置邮箱",
				zap.Int("user_id", userID),
				zap.String("user_name", user.RealName))
			return ""
		}
		return user.Email
	case model.NotificationChannelFeishu:
		if user.FeiShuUserId == "" {
			s.logger.Warn("用户没有配置飞书用户ID",
				zap.Int("user_id", userID),
				zap.String("user_name", user.RealName))
			return ""
		}
		return user.FeiShuUserId
	case model.NotificationChannelSMS:
		if user.Mobile == "" {
			s.logger.Warn("用户没有配置手机号",
				zap.Int("user_id", userID),
				zap.String("user_name", user.RealName))
			return ""
		}
		return user.Mobile
	case model.NotificationChannelWebhook:
		s.logger.Warn("Webhook地址需要配置",
			zap.Int("user_id", userID),
			zap.String("user_name", user.RealName))
		return ""
	default:
		s.logger.Warn("不支持的通知渠道",
			zap.String("channel", channel))
		return ""
	}
}

func (s *workorderNotificationService) getRecipientTypeForChannel(channel string) string {
	switch channel {
	case model.NotificationChannelEmail:
		return "email"
	case model.NotificationChannelFeishu:
		return "feishu_user"
	case model.NotificationChannelSMS:
		return "sms"
	case model.NotificationChannelWebhook:
		return "webhook"
	default:
		return channel
	}
}

// SendNotificationByChannels 通过多个渠道发送通知
func (s *workorderNotificationService) SendNotificationByChannels(ctx context.Context, channels []string, recipient, subject, content string) error {
	if len(channels) != 1 {
		return fmt.Errorf("直接发送必须指定单一渠道及对应地址")
	}
	if s.notificationMgr == nil {
		return errors.New("通知管理器未初始化")
	}

	for _, channel := range channels {
		sendRequest := &notification.SendRequest{
			Subject:       subject,
			Content:       content,
			Priority:      2,
			Channel:       channel,
			RecipientType: channel,
			RecipientAddr: recipient,
			EventType:     "manual",
		}

		_, err := s.notificationMgr.SendNotification(ctx, sendRequest)
		if err != nil {
			s.logger.Error("发送通知失败", zap.String("channel", channel), zap.String("error_type", fmt.Sprintf("%T", err)))
			return err
		}
	}

	return nil
}

// GetAvailableChannels 获取当前可用的通知渠道列表
func (s *workorderNotificationService) GetAvailableChannels() *model.ListResp[*model.WorkorderNotificationChannel] {
	if s.notificationMgr == nil {
		return &model.ListResp[*model.WorkorderNotificationChannel]{
			Items: []*model.WorkorderNotificationChannel{},
			Total: 0,
		}
	}

	availableChannels := s.notificationMgr.GetAvailableChannels()
	channels := make([]*model.WorkorderNotificationChannel, 0, len(availableChannels))

	for _, channel := range availableChannels {
		channels = append(channels, &model.WorkorderNotificationChannel{
			Channels: model.StringList{channel},
		})
	}

	return &model.ListResp[*model.WorkorderNotificationChannel]{
		Items: channels,
		Total: int64(len(channels)),
	}
}

type RecipientInfo struct {
	Channel string `json:"channel,omitempty"`
	Address string `json:"address,omitempty"`
	ID      string `json:"id"`
	Name    string `json:"name"`
	Type    string `json:"type"`
}

// ListNotificationEvents 返回已授权用户可查看的事件状态，不回显快照和正文
func (s *workorderNotificationService) ListNotificationEvents(ctx context.Context, req *model.ListWorkorderNotificationEventsReq) (*model.ListResp[*model.WorkorderNotificationEventStatus], error) {
	store, ok := s.instanceDAO.(interface {
		ListNotificationEvents(context.Context, int, int, string) (*model.ListResp[*model.WorkorderNotificationEventStatus], error)
	})
	if !ok || req == nil {
		return nil, errors.New("事件状态存储不可用")
	}
	return store.ListNotificationEvents(ctx, req.Page, req.Size, req.Status)
}
func (s *workorderNotificationService) ListNotificationEventDeliveries(ctx context.Context, id uint64) ([]*model.WorkorderNotificationDeliveryStatus, error) {
	store, ok := s.instanceDAO.(interface {
		ListNotificationEventDeliveries(context.Context, uint64) ([]*model.WorkorderNotificationDeliveryStatus, error)
	})
	if !ok || id == 0 {
		return nil, errors.New("事件状态存储不可用或事件ID无效")
	}
	return store.ListNotificationEventDeliveries(ctx, id)
}

func (s *workorderNotificationService) ResolveNotificationDelivery(ctx context.Context, eventID uint64, req *model.ResolveWorkorderNotificationDeliveryReq, operatorID int) error {
	store, ok := s.instanceDAO.(interface {
		ResolveNotificationDelivery(context.Context, uint64, *model.ResolveWorkorderNotificationDeliveryReq, int) error
	})
	if !ok {
		return errors.New("事件状态存储不可用")
	}
	return store.ResolveNotificationDelivery(ctx, eventID, req, operatorID)
}
func (s *workorderNotificationService) ListNotificationEventResolutions(ctx context.Context, eventID uint64) ([]*model.WorkorderNotificationResolution, error) {
	store, ok := s.instanceDAO.(interface {
		ListNotificationEventResolutions(context.Context, uint64) ([]*model.WorkorderNotificationResolution, error)
	})
	if !ok || eventID == 0 {
		return nil, errors.New("事件状态存储不可用或事件ID无效")
	}
	return store.ListNotificationEventResolutions(ctx, eventID)
}

func (s *workorderNotificationService) RetryNotificationEvent(ctx context.Context, eventID uint64, reason string, operatorID int) error {
	store, ok := s.instanceDAO.(interface {
		RetryNotificationEvent(context.Context, uint64, string, int) error
	})
	if !ok {
		return errors.New("事件状态存储不可用")
	}
	return store.RetryNotificationEvent(ctx, eventID, reason, operatorID)
}
