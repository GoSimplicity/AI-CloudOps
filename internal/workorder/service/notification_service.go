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
	CreateNotification(ctx context.Context, req *model.CreateNotificationReq) error
	UpdateNotification(ctx context.Context, req *model.UpdateNotificationReq) error
	DeleteNotification(ctx context.Context, req *model.DeleteNotificationReq) error
	ListNotification(ctx context.Context, req *model.ListNotificationReq) (model.ListResp[*model.Notification], error)
	DetailNotification(ctx context.Context, req *model.DetailNotificationReq) (*model.Notification, error)
	UpdateStatus(ctx context.Context, req *model.UpdateStatusReq) error
	GetStatistics(ctx context.Context) (*model.NotificationStats, error)
	GetSendLogs(ctx context.Context, req *model.ListSendLogReq) (model.ListResp[*model.NotificationLog], error)
	TestSendNotification(ctx context.Context, req *model.TestSendNotificationReq) error
	DuplicateNotification(ctx context.Context, req *model.DuplicateNotificationReq) error
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
func (n *notificationService) CreateNotification(ctx context.Context, req *model.CreateNotificationReq) error {
	// 校验触发类型和定时时间的一致性
	if req.TriggerType == model.NotificationTriggerScheduled && req.ScheduledTime == nil {
		return errors.New("定时发送必须设置定时时间")
	}

	if req.TriggerType != model.NotificationTriggerScheduled && req.ScheduledTime != nil {
		req.ScheduledTime = nil // 非定时发送，清空定时时间
	}

	return n.dao.CreateNotification(ctx, req)
}

// UpdateNotification 更新通知配置
func (n *notificationService) UpdateNotification(ctx context.Context, req *model.UpdateNotificationReq) error {
	// 校验通知是否存在
	notification, err := n.dao.GetNotificationByID(ctx, req.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("通知配置不存在")
		}
		return fmt.Errorf("查询通知配置失败: %w", err)
	}

	// 校验触发类型和定时时间的一致性
	if req.TriggerType == model.NotificationTriggerScheduled && req.ScheduledTime == nil {
		return errors.New("定时发送必须设置定时时间")
	}

	if req.TriggerType != model.NotificationTriggerScheduled && req.ScheduledTime != nil {
		req.ScheduledTime = nil // 非定时发送，清空定时时间
	}

	// 保持一致性：状态应该从请求中获取，如果请求没有指定，则使用现有状态
	if req.Status != model.NotificationStatusEnabled && req.Status != model.NotificationStatusDisabled {
		req.Status = notification.Status
	}

	return n.dao.UpdateNotification(ctx, req)
}

// DeleteNotification 删除通知配置
func (n *notificationService) DeleteNotification(ctx context.Context, req *model.DeleteNotificationReq) error {
	// 校验通知是否存在
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
func (n *notificationService) ListNotification(ctx context.Context, req *model.ListNotificationReq) (model.ListResp[*model.Notification], error) {
	return n.dao.ListNotification(ctx, req)
}

// DetailNotification 获取通知配置详情
func (n *notificationService) DetailNotification(ctx context.Context, req *model.DetailNotificationReq) (*model.Notification, error) {
	return n.dao.DetailNotification(ctx, req)
}

// UpdateStatus 更新通知配置状态
func (n *notificationService) UpdateStatus(ctx context.Context, req *model.UpdateStatusReq) error {
	// 校验通知是否存在
	_, err := n.dao.GetNotificationByID(ctx, req.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("通知配置不存在")
		}
		return fmt.Errorf("查询通知配置失败: %w", err)
	}

	return n.dao.UpdateStatus(ctx, req)
}

// GetStatistics 获取通知统计信息
func (n *notificationService) GetStatistics(ctx context.Context) (*model.NotificationStats, error) {
	return n.dao.GetStatistics(ctx)
}

// GetSendLogs 获取发送日志
func (n *notificationService) GetSendLogs(ctx context.Context, req *model.ListSendLogReq) (model.ListResp[*model.NotificationLog], error) {
	return n.dao.GetSendLogs(ctx, req)
}

// TestSendNotification 测试发送通知
func (n *notificationService) TestSendNotification(ctx context.Context, req *model.TestSendNotificationReq) error {
	// 获取通知配置
	notification, err := n.dao.GetNotificationByID(ctx, req.NotificationID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("通知配置不存在")
		}
		return fmt.Errorf("查询通知配置失败: %w", err)
	}

	// 判断通知是否启用
	if notification.Status != model.NotificationStatusEnabled {
		return errors.New("通知配置已禁用，无法发送")
	}

	// 从上下文获取用户ID
	var senderID int
	if uid := ctx.Value("user_id"); uid != nil {
		if id, ok := uid.(int); ok {
			senderID = id
		}
	}

	// 循环发送各个渠道的通知
	for _, channel := range notification.Channels {
		for _, recipient := range notification.Recipients {
			// 创建发送日志
			log := &model.NotificationLog{
				NotificationID: notification.ID,
				Channel:        channel,
				Recipient:      recipient,
				Content:        notification.MessageTemplate,
				SenderID:       senderID,
			}

			// 实际发送通知
			err := n.sendNotification(channel, recipient, notification.MessageTemplate)
			if err != nil {
				// 发送失败，记录错误
				log.Status = "failed"
				log.Error = err.Error()
				n.logger.Error("发送通知失败",
					zap.String("channel", channel),
					zap.String("recipient", recipient),
					zap.Error(err))
			} else {
				// 发送成功
				log.Status = "success"
			}

			// 记录发送日志
			if err := n.dao.AddSendLog(ctx, log); err != nil {
				n.logger.Error("记录发送日志失败", zap.Error(err))
			}
		}
	}

	// 更新发送次数和最后发送时间
	return n.dao.IncrementSentCount(ctx, notification.ID)
}

// DuplicateNotification 复制通知配置
func (n *notificationService) DuplicateNotification(ctx context.Context, req *model.DuplicateNotificationReq) error {
	// 获取源通知配置
	source, err := n.dao.GetNotificationByID(ctx, req.SourceID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("源通知配置不存在")
		}
		return fmt.Errorf("查询源通知配置失败: %w", err)
	}

	// 创建新的通知配置
	newReq := &model.CreateNotificationReq{
		FormID:          source.FormID,
		Channels:        []string(source.Channels),
		Recipients:      []string(source.Recipients),
		MessageTemplate: source.MessageTemplate,
		TriggerType:     source.TriggerType,
		FormUrl:         source.FormUrl,
	}

	// 如果是定时发送，复制定时时间
	if source.TriggerType == model.NotificationTriggerScheduled && source.ScheduledTime != nil {
		scheduledTime := *source.ScheduledTime
		newReq.ScheduledTime = &scheduledTime
	}

	// 创建新通知配置
	return n.dao.CreateNotification(ctx, newReq)
}

// sendNotification 根据不同的通道发送通知
func (n *notificationService) sendNotification(channel, recipient, content string) error {
	switch channel {
	case model.NotificationChannelFeishu:
		return n.sendFeishuNotification(recipient, content)
	case model.NotificationChannelEmail:
		return n.sendEmailNotification(recipient, content)
	case model.NotificationChannelDingtalk:
		return n.sendDingtalkNotification(recipient, content)
	case model.NotificationChannelWechat:
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
