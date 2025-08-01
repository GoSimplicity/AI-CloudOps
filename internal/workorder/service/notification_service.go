package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/dao"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type NotificationService interface {
	CreateNotification(ctx context.Context, req *model.CreateWorkorderNotificationReq) error
	UpdateNotification(ctx context.Context, req *model.UpdateWorkorderNotificationReq) error
	DeleteNotification(ctx context.Context, req *model.DeleteWorkorderNotificationReq) error
	ListNotification(ctx context.Context, req *model.ListWorkorderNotificationReq) (*model.ListResp[*model.WorkorderNotification], error)
	DetailNotification(ctx context.Context, req *model.DetailWorkorderNotificationReq) (*model.WorkorderNotification, error)
	GetSendLogs(ctx context.Context, req *model.ListWorkorderNotificationLogReq) (*model.ListResp[*model.WorkorderNotificationLog], error)
	TestSendNotification(ctx context.Context, req *model.TestSendWorkorderNotificationReq) error
	DuplicateNotification(ctx context.Context, req *model.DuplicateWorkorderNotificationReq) error
}

type notificationService struct {
	dao    dao.NotificationDAO
	logger *zap.Logger
}

func NewNotificationService(dao dao.NotificationDAO, logger *zap.Logger) NotificationService {
	return &notificationService{
		logger: logger,
		dao:    dao,
	}
}

// CreateNotification 创建通知配置
func (n *notificationService) CreateNotification(ctx context.Context, req *model.CreateWorkorderNotificationReq) error {
	return n.dao.CreateNotification(ctx, req)
}

// UpdateNotification 更新通知配置
func (n *notificationService) UpdateNotification(ctx context.Context, req *model.UpdateWorkorderNotificationReq) error {
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
func (n *notificationService) DeleteNotification(ctx context.Context, req *model.DeleteWorkorderNotificationReq) error {
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
func (n *notificationService) ListNotification(ctx context.Context, req *model.ListWorkorderNotificationReq) (*model.ListResp[*model.WorkorderNotification], error) {
	result, err := n.dao.ListNotification(ctx, req)
	if err != nil {
		n.logger.Error("获取通知配置列表失败", zap.Error(err))
		return nil, fmt.Errorf("获取通知配置列表失败: %w", err)
	}
	return result, nil
}

// DetailNotification 获取通知配置详情
func (n *notificationService) DetailNotification(ctx context.Context, req *model.DetailWorkorderNotificationReq) (*model.WorkorderNotification, error) {
	return n.dao.DetailNotification(ctx, req)
}

// GetSendLogs 获取发送日志
func (n *notificationService) GetSendLogs(ctx context.Context, req *model.ListWorkorderNotificationLogReq) (*model.ListResp[*model.WorkorderNotificationLog], error) {
	result, err := n.dao.GetSendLogs(ctx, req)
	if err != nil {
		n.logger.Error("获取发送日志失败", zap.Error(err))
		return nil, fmt.Errorf("获取发送日志失败: %w", err)
	}
	return result, nil
}

// TestSendNotification 测试发送通知
func (n *notificationService) TestSendNotification(ctx context.Context, req *model.TestSendWorkorderNotificationReq) error {
	notification, err := n.dao.GetNotificationByID(ctx, req.NotificationID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("通知配置不存在")
		}
		return fmt.Errorf("查询通知配置失败: %w", err)
	}

	if notification.Status != 1 {
		return errors.New("通知配置已禁用，无法发送")
	}

	var senderID int
	if uid := ctx.Value("user_id"); uid != nil {
		if id, ok := uid.(int); ok {
			senderID = id
		}
	}

	for _, channel := range notification.Channels {
		log := &model.WorkorderNotificationLog{
			NotificationID: notification.ID,
			EventType:      "test",
			Channel:        channel,
			RecipientType:  "test",
			RecipientID:    "test_user",
			RecipientName:  "测试用户",
			RecipientAddr:  req.Recipient,
			Subject:        "测试通知",
			Content:        notification.MessageTemplate,
			Status:         1,
			SendAt:         time.Now(),
			SenderID:       senderID,
		}

		err := n.sendNotification(channel, req.Recipient, notification.MessageTemplate)
		if err != nil {
			log.Status = 2
			log.ErrorMessage = err.Error()
			n.logger.Error("发送通知失败",
				zap.String("channel", channel),
				zap.String("recipient", req.Recipient),
				zap.Error(err))
		}

		if err := n.dao.AddSendLog(ctx, log); err != nil {
			n.logger.Error("记录发送日志失败", zap.Error(err))
		}
	}

	return n.dao.IncrementSentCount(ctx, notification.ID)
}

// DuplicateNotification 复制通知配置
func (n *notificationService) DuplicateNotification(ctx context.Context, req *model.DuplicateWorkorderNotificationReq) error {
	source, err := n.dao.GetNotificationByID(ctx, req.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("源通知配置不存在")
		}
		return fmt.Errorf("查询源通知配置失败: %w", err)
	}

	newReq := &model.CreateWorkorderNotificationReq{
		Name:             source.Name + "_副本",
		Description:      source.Description,
		ProcessID:        source.ProcessID,
		TemplateID:       source.TemplateID,
		CategoryID:       source.CategoryID,
		EventTypes:       source.EventTypes,
		TriggerType:      source.TriggerType,
		TriggerCondition: source.TriggerCondition,
		Channels:         source.Channels,
		RecipientTypes:   source.RecipientTypes,
		RecipientUsers:   source.RecipientUsers,
		RecipientRoles:   source.RecipientRoles,
		RecipientDepts:   source.RecipientDepts,
		MessageTemplate:  source.MessageTemplate,
		SubjectTemplate:  source.SubjectTemplate,
		ScheduledTime:    source.ScheduledTime,
		RepeatInterval:   source.RepeatInterval,
		MaxRetries:       source.MaxRetries,
		RetryInterval:    source.RetryInterval,
		Status:           source.Status,
		Priority:         source.Priority,
		IsDefault:        false,
		Settings:         source.Settings,
		UserID:           source.OperatorID,
	}

	return n.dao.CreateNotification(ctx, newReq)
}

// sendNotification 根据不同的通道发送通知
func (n *notificationService) sendNotification(channel, recipient, content string) error {
	switch channel {
	case "feishu":
		return n.sendFeishuNotification(recipient, content)
	case "email":
		return n.sendEmailNotification(recipient, content)
	case "dingtalk":
		return n.sendDingtalkNotification(recipient, content)
	case "wechat":
		return n.sendWechatNotification(recipient, content)
	default:
		return fmt.Errorf("不支持的通知渠道: %s", channel)
	}
}

// sendFeishuNotification 发送飞书通知
func (n *notificationService) sendFeishuNotification(recipient, content string) error {
	n.logger.Info("发送飞书通知",
		zap.String("recipient", recipient),
		zap.String("content", content))
	// 模拟发送延迟
	time.Sleep(100 * time.Millisecond)
	return nil
}

// sendEmailNotification 发送邮件通知
func (n *notificationService) sendEmailNotification(recipient, content string) error {
	n.logger.Info("发送邮件通知",
		zap.String("recipient", recipient),
		zap.String("content", content))
	// 模拟发送延迟
	time.Sleep(200 * time.Millisecond)
	return nil
}

// sendDingtalkNotification 发送钉钉通知
func (n *notificationService) sendDingtalkNotification(recipient, content string) error {
	n.logger.Info("发送钉钉通知",
		zap.String("recipient", recipient),
		zap.String("content", content))
	// 模拟发送延迟
	time.Sleep(150 * time.Millisecond)
	return nil
}

// sendWechatNotification 发送企业微信通知
func (n *notificationService) sendWechatNotification(recipient, content string) error {
	n.logger.Info("发送企业微信通知",
		zap.String("recipient", recipient),
		zap.String("content", content))
	// 模拟发送延迟
	time.Sleep(180 * time.Millisecond)
	return nil
}
