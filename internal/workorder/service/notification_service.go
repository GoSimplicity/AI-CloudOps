package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	userDao "github.com/GoSimplicity/AI-CloudOps/internal/user/dao"
	workorderDao "github.com/GoSimplicity/AI-CloudOps/internal/workorder/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/notification"
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

// TestSendNotification 测试发送通知
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
				recipientAddr = "123"
			case model.NotificationChannelSMS:
				recipientAddr = "13800138000"
			case model.NotificationChannelWebhook:
				recipientAddr = "https://webhook.site/test"
			default:
				recipientAddr = "test_recipient"
			}
		}

		sendRequest := &notification.SendRequest{
			Subject:       notificationConfig.SubjectTemplate,
			Content:       notificationConfig.MessageTemplate,
			Priority:      notificationConfig.Priority,
			RecipientType: channel,
			RecipientID:   "test_user",
			RecipientAddr: recipientAddr,
			RecipientName: "测试用户",
			EventType:     "test",
			Metadata: map[string]interface{}{
				"notification_id": notificationConfig.ID,
				"sender_id":       senderID,
			},
		}

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

// SendWorkorderNotification 发送通知
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

// processNotification 处理单个通知配置
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

// getRecipients 获取通知接收人列表
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

// sendChannelNotification 发送渠道
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

// buildMessageContent 构建消息内容
func (n *workorderNotificationService) buildMessageContent(notification *model.WorkorderNotification,
	instance *model.WorkorderInstance, eventType string, customContent ...string) (string, string) {

	// 添加空值检查，确保所有变量都有值
	safeString := func(s string) string {
		if s == "" {
			return "N/A"
		}
		return s
	}

	safeInt := func(i int) string {
		return fmt.Sprintf("%d", i)
	}

	// 处理处理人名称
	assigneeName := "未分配"
	if instance.AssigneeID != nil {
		if user, err := n.userDAO.GetUserByID(context.Background(), *instance.AssigneeID); err == nil && user != nil {
			assigneeName = safeString(user.RealName)
		}
	}

	// 基础变量 - 确保所有可能的变量名都被覆盖
	variables := map[string]string{
		// 基础变量
		"instance_id":   safeInt(instance.ID),
		"serial_number": safeString(instance.SerialNumber),
		"title":         safeString(instance.Title),
		"description":   safeString(instance.Description),
		"operator_name": safeString(instance.OperatorName),
		"assignee_name": assigneeName,
		"priority":      safeInt(int(instance.Priority)),
		"status":        model.GetInstanceStatusName(instance.Status),
		"event_type":    model.GetEventTypeName(eventType),
		"created_at":    instance.CreatedAt.Format("2006-01-02 15:04:05"),
	}

	// 如果有更新时间，添加更新时间
	if !instance.UpdatedAt.IsZero() {
		variables["updated_at"] = instance.UpdatedAt.Format("2006-01-02 15:04:05")
		variables["updatedAt"] = instance.UpdatedAt.Format("2006-01-02 15:04:05")
	}

	// 如果有自定义内容，添加到变量中
	if len(customContent) > 0 && customContent[0] != "" {
		variables["custom_content"] = customContent[0]
		variables["customContent"] = customContent[0]
		variables["content"] = customContent[0]
	} else {
		variables["custom_content"] = ""
		variables["customContent"] = ""
		variables["content"] = ""
	}

	// 添加调试日志
	n.logger.Debug("模板变量",
		zap.Any("variables", variables),
		zap.String("subject_template", notification.SubjectTemplate),
		zap.String("message_template", notification.MessageTemplate))

	// 处理主题模板
	subject := notification.SubjectTemplate
	if subject == "" {
		subject = fmt.Sprintf("工单通知 - %s", variables["title"])
	} else {
		subject = n.replaceTemplateVariables(subject, variables)
	}

	// 处理消息模板
	content := notification.MessageTemplate
	if content == "" {
		content = fmt.Sprintf(`工单通知

工单编号: %s
工单标题: %s
当前状态: %s
优先级: %s
操作人: %s
处理人: %s
事件类型: %s
创建时间: %s

%s
`, variables["serial_number"], variables["title"], variables["status"],
			variables["priority"], variables["operator_name"], variables["assignee_name"],
			variables["event_type"], variables["created_at"], variables["custom_content"])
	} else {
		content = n.replaceTemplateVariables(content, variables)
	}

	// 添加调试日志查看替换后的结果
	n.logger.Debug("模板替换结果",
		zap.String("subject", subject),
		zap.String("content", content))

	return subject, content
}

// replaceTemplateVariables 替换变量
func (n *workorderNotificationService) replaceTemplateVariables(template string, variables map[string]string) string {
	if template == "" {
		return ""
	}

	result := template

	// 记录替换前的模板
	n.logger.Debug("开始替换模板变量",
		zap.String("template", template),
		zap.Int("variables_count", len(variables)))

	// 支持多种格式的模板变量
	for key, value := range variables {
		// 替换 {{变量名}} 格式
		placeholder1 := fmt.Sprintf("{{%s}}", key)
		if strings.Contains(result, placeholder1) {
			result = strings.ReplaceAll(result, placeholder1, value)
			n.logger.Debug("替换变量",
				zap.String("placeholder", placeholder1),
				zap.String("value", value))
		}

		// 替换 {{ 变量名 }} 格式（带空格）
		placeholder2 := fmt.Sprintf("{{ %s }}", key)
		if strings.Contains(result, placeholder2) {
			result = strings.ReplaceAll(result, placeholder2, value)
			n.logger.Debug("替换变量",
				zap.String("placeholder", placeholder2),
				zap.String("value", value))
		}

		// 替换 {变量名} 格式
		placeholder3 := fmt.Sprintf("{%s}", key)
		if strings.Contains(result, placeholder3) {
			result = strings.ReplaceAll(result, placeholder3, value)
			n.logger.Debug("替换变量",
				zap.String("placeholder", placeholder3),
				zap.String("value", value))
		}

		// 替换 { 变量名 } 格式（带空格）
		placeholder4 := fmt.Sprintf("{ %s }", key)
		if strings.Contains(result, placeholder4) {
			result = strings.ReplaceAll(result, placeholder4, value)
			n.logger.Debug("替换变量",
				zap.String("placeholder", placeholder4),
				zap.String("value", value))
		}

		// 替换 ${变量名} 格式（类似shell变量）
		placeholder5 := fmt.Sprintf("${%s}", key)
		if strings.Contains(result, placeholder5) {
			result = strings.ReplaceAll(result, placeholder5, value)
			n.logger.Debug("替换变量",
				zap.String("placeholder", placeholder5),
				zap.String("value", value))
		}
	}

	// 检查是否还有未替换的变量
	remainingVars := n.findUnreplacedVariables(result)
	if len(remainingVars) > 0 {
		n.logger.Warn("存在未替换的模板变量",
			zap.Strings("remaining_vars", remainingVars),
			zap.String("template", result))
	}

	n.logger.Debug("模板变量替换完成",
		zap.String("result", result))

	return result
}

// findUnreplacedVariables 查找未替换变量
func (n *workorderNotificationService) findUnreplacedVariables(template string) []string {
	var unreplaced []string

	// 使用正则表达式查找所有可能的变量格式
	patterns := []string{
		`\{\{[^}]+\}\}`, // {{变量名}}
		`\{[^}]+\}`,     // {变量名}
		`\$\{[^}]+\}`,   // ${变量名}
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllString(template, -1)
		unreplaced = append(unreplaced, matches...)
	}

	return unreplaced
}

// getRecipientAddress 获取接收人地址
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

// getRecipientTypeForChannel 获取类型
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

// SendNotificationByChannels 发送通知
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

// GetAvailableChannels 获取可用的通知渠道
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
