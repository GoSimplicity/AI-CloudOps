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
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	userDao "github.com/GoSimplicity/AI-CloudOps/internal/user/dao"
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
	TestSendNotification(ctx context.Context, req *model.TestSendWorkorderNotificationReq) error
	SendWorkorderNotification(ctx context.Context, instanceID int, eventType string, customContent ...string) error
	SendNotificationByChannels(ctx context.Context, channels []string, recipient, subject, content string) error
	GetAvailableChannels() *model.ListResp[*model.WorkorderNotificationChannel]
}

type workorderNotificationService struct {
	dao             workorderDao.WorkorderNotificationDAO
	logger          *zap.Logger
	notificationMgr *notification.Manager
	instanceDAO     workorderDao.WorkorderInstanceDAO
	userDAO         userDao.UserDAO
}

func NewWorkorderNotificationService(dao workorderDao.WorkorderNotificationDAO, notificationMgr *notification.Manager, logger *zap.Logger, instanceDAO workorderDao.WorkorderInstanceDAO, userDAO userDao.UserDAO) WorkorderNotificationService {
	return &workorderNotificationService{
		logger:          logger,
		dao:             dao,
		notificationMgr: notificationMgr,
		instanceDAO:     instanceDAO,
		userDAO:         userDAO,
	}
}

// CreateNotification 创建通知配置
func (n *workorderNotificationService) CreateNotification(ctx context.Context, req *model.CreateWorkorderNotificationReq) error {
	return n.dao.CreateNotification(ctx, req)
}

// UpdateNotification 更新通知配置
func (n *workorderNotificationService) UpdateNotification(ctx context.Context, req *model.UpdateWorkorderNotificationReq) error {
	_, err := n.dao.GetNotificationByID(ctx, req.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("通知配置不存在")
		}
		return fmt.Errorf("查询通知配置失败: %w", err)
	}

	return n.dao.UpdateNotification(ctx, req)
}

// DeleteNotification 删除通知配置
func (n *workorderNotificationService) DeleteNotification(ctx context.Context, req *model.DeleteWorkorderNotificationReq) error {
	_, err := n.dao.GetNotificationByID(ctx, req.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("通知配置不存在")
		}
		return fmt.Errorf("查询通知配置失败: %w", err)
	}

	return n.dao.DeleteNotification(ctx, req)
}

// ListNotification 获取通知配置列表
func (n *workorderNotificationService) ListNotification(ctx context.Context, req *model.ListWorkorderNotificationReq) (*model.ListResp[*model.WorkorderNotification], error) {
	result, err := n.dao.ListNotification(ctx, req)
	if err != nil {
		n.logger.Error("获取通知配置列表失败", zap.Error(err))
		return nil, fmt.Errorf("获取通知配置列表失败: %w", err)
	}
	return result, nil
}

// DetailNotification 获取通知配置
func (n *workorderNotificationService) DetailNotification(ctx context.Context, req *model.DetailWorkorderNotificationReq) (*model.WorkorderNotification, error) {
	return n.dao.DetailNotification(ctx, req)
}

// GetSendLogs 获取发送日志
func (n *workorderNotificationService) GetSendLogs(ctx context.Context, req *model.ListWorkorderNotificationLogReq) (*model.ListResp[*model.WorkorderNotificationLog], error) {
	result, err := n.dao.GetSendLogs(ctx, req)
	if err != nil {
		n.logger.Error("获取发送日志失败", zap.Error(err))
		return nil, fmt.Errorf("获取发送日志失败: %w", err)
	}
	return result, nil
}

// TestSendNotification 测试发送指定通知配置
func (n *workorderNotificationService) TestSendNotification(ctx context.Context, req *model.TestSendWorkorderNotificationReq) error {
	notificationConfig, err := n.dao.GetNotificationByID(ctx, req.NotificationID)
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

	for _, channel := range notificationConfig.Channels {
		var recipientAddr string
		if req.Recipient != "" {
			recipientAddr = req.Recipient
		} else {
			switch channel {
			case model.NotificationChannelEmail:
				recipientAddr = "xxx@163.com"
			case model.NotificationChannelFeishu:
				recipientAddr = "xxx"
			case model.NotificationChannelSMS:
				recipientAddr = "13800138000"
			case model.NotificationChannelWebhook:
				recipientAddr = "https://webhook.site/test"
			default:
				recipientAddr = "test_recipient"
			}
		}

		// 创建模拟的实例ID用于测试
		testInstanceID := 999999

		sendRequest := &notification.SendRequest{
			Subject:       notificationConfig.SubjectTemplate,
			Content:       notificationConfig.MessageTemplate,
			Priority:      notificationConfig.Priority,
			RecipientType: channel,
			RecipientID:   "test_user",
			RecipientAddr: recipientAddr,
			RecipientName: "测试用户",
			EventType:     "test",
			InstanceID:    &testInstanceID,
			Templates:     make(map[string]string),
			Metadata: map[string]interface{}{
				"notification_id": notificationConfig.ID,
				"sender_id":       senderID,
			},
		}

		// 统一商务化模板变量设置
		sendRequest.Templates["workorder_id"] = fmt.Sprintf("%d", testInstanceID)
		sendRequest.Templates["serial_number"] = fmt.Sprintf("WO-%d", testInstanceID)
		sendRequest.Templates["title"] = "AI-CloudOps 测试工单 - 系统功能验证"
		sendRequest.Templates["description"] = "这是一个AI-CloudOps运维管理平台的系统测试工单，用于验证通知功能的完整性和可靠性。"
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
		response, err := n.notificationMgr.SendNotification(ctx, sendRequest)

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

		if err := n.dao.AddSendLog(ctx, log); err != nil {
			n.logger.Error("记录发送日志失败", zap.Error(err))
		}
	}

	return n.dao.IncrementSentCount(ctx, notificationConfig.ID)
}

// SendWorkorderNotification 发送工单相关通知
func (n *workorderNotificationService) SendWorkorderNotification(ctx context.Context, instanceID int, eventType string, customContent ...string) error {
	instance, err := n.instanceDAO.GetInstanceByID(ctx, instanceID)
	if err != nil {
		n.logger.Error("获取工单实例失败",
			zap.Int("instance_id", instanceID),
			zap.Error(err))
		return fmt.Errorf("获取工单实例失败: %w", err)
	}

	var senderID int
	if uid := ctx.Value("user_id"); uid != nil {
		if id, ok := uid.(int); ok {
			senderID = id
		}
	}

	notifications, err := n.dao.GetActiveNotificationsByEventType(ctx, eventType, instance.ProcessID)
	if err != nil {
		n.logger.Error("获取通知配置失败",
			zap.String("event_type", eventType),
			zap.Int("process_id", instance.ProcessID),
			zap.Error(err))
		return fmt.Errorf("获取通知配置失败: %w", err)
	}

	if len(notifications) == 0 {
		n.logger.Info("没有找到匹配的通知配置",
			zap.String("event_type", eventType),
			zap.Int("process_id", instance.ProcessID))
		return nil
	}

	for _, notification := range notifications {
		if err := n.processNotification(ctx, notification, instance, eventType, senderID, customContent...); err != nil {
			n.logger.Error("处理通知配置失败",
				zap.Int("notification_id", notification.ID),
				zap.Int("instance_id", instanceID),
				zap.Error(err))
			continue
		}
	}

	n.logger.Info("工单通知发送完成",
		zap.Int("instance_id", instanceID),
		zap.String("event_type", eventType),
		zap.Int("notification_count", len(notifications)))

	return nil
}

// processNotification 处理并发送单个通知配置
func (n *workorderNotificationService) processNotification(ctx context.Context, notification *model.WorkorderNotification,
	instance *model.WorkorderInstance, eventType string, senderID int, customContent ...string) error {

	recipients, err := n.getRecipients(ctx, notification, instance)
	if err != nil {
		return fmt.Errorf("获取接收人失败: %w", err)
	}

	if len(recipients) == 0 {
		n.logger.Info("没有找到接收人",
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

			channelCtx := context.Background()
			if deadline, ok := ctx.Deadline(); ok {
				var cancel context.CancelFunc
				channelCtx, cancel = context.WithDeadline(context.Background(), deadline)
				defer cancel()
			}

			if err := n.sendChannelNotification(channelCtx, notification, instance, ch, recipients, eventType, senderID, customContent...); err != nil {
				n.logger.Error("发送渠道通知失败",
					zap.String("channel", ch),
					zap.Int("notification_id", notification.ID),
					zap.Error(err))
				channelErrors <- fmt.Errorf("渠道 %s 发送失败: %w", ch, err)
			} else {
				n.logger.Info("渠道通知发送成功",
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
		n.logger.Warn("部分渠道发送失败，但其他渠道已成功发送",
			zap.Strings("errors", errors),
			zap.Int("notification_id", notification.ID))
	}

	return nil
}

// getRecipients 根据配置获取接收人列表
func (n *workorderNotificationService) getRecipients(ctx context.Context, notification *model.WorkorderNotification,
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
				if user, err := n.userDAO.GetUserByID(ctx, *instance.AssigneeID); err == nil {
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
				if err != nil {
					n.logger.Warn("无效的用户ID",
						zap.String("user_id", userIDStr))
					continue
				}

				userName := "指定用户"
				if user, err := n.userDAO.GetUserByID(ctx, userID); err == nil {
					userName = user.RealName
				}

				recipients = append(recipients, RecipientInfo{
					ID:   userIDStr,
					Name: userName,
					Type: recipientType,
				})
			}
		case model.RecipientTypeRole:
			n.logger.Info("角色用户通知暂未实现",
				zap.Strings("roles", notification.RecipientRoles))
		case model.RecipientTypeDept:
			n.logger.Info("部门用户通知暂未实现",
				zap.Strings("depts", notification.RecipientDepts))
		case model.RecipientTypeCustom:
			n.logger.Info("自定义用户通知暂未实现")
		}
	}

	return recipients, nil
}

// sendChannelNotification 通过指定渠道发送通知
func (n *workorderNotificationService) sendChannelNotification(ctx context.Context, notificationConfig *model.WorkorderNotification,
	instance *model.WorkorderInstance, channel string, recipients []RecipientInfo, eventType string, senderID int, customContent ...string) error {

	subject, content := n.buildMessageContent(notificationConfig, instance, eventType, customContent...)

	var wg sync.WaitGroup
	recipientErrors := make(chan error, len(recipients))

	for _, recipient := range recipients {
		wg.Add(1)
		go func(rec RecipientInfo) {
			defer wg.Done()

			recipientCtx := context.Background()
			if deadline, ok := ctx.Deadline(); ok {
				var cancel context.CancelFunc
				recipientCtx, cancel = context.WithDeadline(context.Background(), deadline)
				defer cancel()
			}

			recipientAddr := n.getRecipientAddress(rec, channel)
			if recipientAddr == "" {
				n.logger.Warn("无法获取接收人地址",
					zap.String("recipient_id", rec.ID),
					zap.String("channel", channel))
				return
			}

			recipientType := n.getRecipientTypeForChannel(channel)

			sendRequest := &notification.SendRequest{
				Subject:       subject,
				Content:       content,
				Priority:      notificationConfig.Priority,
				RecipientType: recipientType,
				RecipientID:   rec.ID,
				RecipientAddr: recipientAddr,
				RecipientName: rec.Name,
				InstanceID:    &instance.ID,
				EventType:     eventType,
				Metadata: map[string]interface{}{
					"notification_id": notificationConfig.ID,
					"instance_id":     instance.ID,
					"sender_id":       senderID,
					"recipient_type":  rec.Type,
				},
			}

			response, err := n.notificationMgr.SendNotification(recipientCtx, sendRequest)

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
				n.logger.Error("发送通知失败",
					zap.String("channel", channel),
					zap.String("recipient", recipientAddr),
					zap.Error(err))
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
				n.logger.Info("通知发送成功",
					zap.String("channel", channel),
					zap.String("recipient", recipientAddr))
			}

			if err := n.dao.AddSendLog(recipientCtx, log); err != nil {
				n.logger.Error("记录发送日志失败", zap.Error(err))
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
		n.logger.Warn("部分接收人发送失败，但其他接收人已成功发送",
			zap.Strings("errors", errors),
			zap.String("channel", channel))
	}

	return nil
}

// buildMessageContent 根据模板构建消息内容
func (n *workorderNotificationService) buildMessageContent(notificationConfig *model.WorkorderNotification,
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

	// 统一商务化模板变量设置
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

	// 处理处理人名称
	assigneeName := "待分配"
	if instance.AssigneeID != nil {
		if user, err := n.userDAO.GetUserByID(context.Background(), *instance.AssigneeID); err == nil && user != nil {
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

	// 渲染内容
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

	n.logger.Debug("消息内容构建完成",
		zap.String("subject", subject),
		zap.String("content", content))

	return subject, content
}

// getRecipientAddress 根据渠道类型获取接收人地址
func (n *workorderNotificationService) getRecipientAddress(recipient RecipientInfo, channel string) string {
	userID, err := strconv.Atoi(recipient.ID)
	if err != nil {
		n.logger.Error("无效的用户ID",
			zap.String("recipient_id", recipient.ID),
			zap.Error(err))
		return ""
	}

	user, err := n.userDAO.GetUserByID(context.Background(), userID)
	if err != nil {
		n.logger.Error("获取用户信息失败",
			zap.Int("user_id", userID),
			zap.Error(err))
		return ""
	}

	switch channel {
	case model.NotificationChannelEmail:
		if user.Email == "" {
			n.logger.Warn("用户没有配置邮箱",
				zap.Int("user_id", userID),
				zap.String("user_name", user.RealName))
			return ""
		}
		return user.Email
	case model.NotificationChannelFeishu:
		if user.FeiShuUserId == "" {
			n.logger.Warn("用户没有配置飞书用户ID",
				zap.Int("user_id", userID),
				zap.String("user_name", user.RealName))
			return ""
		}
		return user.FeiShuUserId
	case model.NotificationChannelSMS:
		if user.Mobile == "" {
			n.logger.Warn("用户没有配置手机号",
				zap.Int("user_id", userID),
				zap.String("user_name", user.RealName))
			return ""
		}
		return user.Mobile
	case model.NotificationChannelWebhook:
		n.logger.Warn("Webhook地址需要配置",
			zap.Int("user_id", userID),
			zap.String("user_name", user.RealName))
		return ""
	default:
		n.logger.Warn("不支持的通知渠道",
			zap.String("channel", channel))
		return ""
	}
}

// getRecipientTypeForChannel 获取渠道对应的接收人类型
func (n *workorderNotificationService) getRecipientTypeForChannel(channel string) string {
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
func (n *workorderNotificationService) SendNotificationByChannels(ctx context.Context, channels []string, recipient, subject, content string) error {
	if n.notificationMgr == nil {
		return errors.New("通知管理器未初始化")
	}

	for _, channel := range channels {
		sendRequest := &notification.SendRequest{
			Subject:       subject,
			Content:       content,
			Priority:      2,
			RecipientType: channel,
			RecipientAddr: recipient,
			EventType:     "manual",
		}

		_, err := n.notificationMgr.SendNotification(ctx, sendRequest)
		if err != nil {
			n.logger.Error("发送通知失败",
				zap.String("channel", channel),
				zap.String("recipient", recipient),
				zap.Error(err))
			return err
		}
	}

	return nil
}

// GetAvailableChannels 获取当前可用的通知渠道列表
func (n *workorderNotificationService) GetAvailableChannels() *model.ListResp[*model.WorkorderNotificationChannel] {
	if n.notificationMgr == nil {
		return &model.ListResp[*model.WorkorderNotificationChannel]{
			Items: []*model.WorkorderNotificationChannel{},
			Total: 0,
		}
	}

	availableChannels := n.notificationMgr.GetAvailableChannels()
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

// RecipientInfo 接收人信息
type RecipientInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}
