package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/dao"
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
	// 新增方法
	SendWorkorderNotification(ctx context.Context, instanceID int, eventType string, customContent ...string) error
	SendNotificationByChannels(ctx context.Context, channels []string, recipient, subject, content string) error
	GetAvailableChannels() *model.ListResp[*model.WorkorderNotificationChannel]
}

type workorderNotificationService struct {
	dao             dao.WorkorderNotificationDAO
	logger          *zap.Logger
	notificationMgr *notification.Manager
}

func NewWorkorderNotificationService(dao dao.WorkorderNotificationDAO, notificationMgr *notification.Manager, logger *zap.Logger) WorkorderNotificationService {
	return &workorderNotificationService{
		logger:          logger,
		dao:             dao,
		notificationMgr: notificationMgr,
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

// DetailNotification 获取通知配置详情
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

	// 使用新的通知管理器发送通知
	for _, channel := range notificationConfig.Channels {
		sendRequest := &notification.SendRequest{
			Subject:       notificationConfig.SubjectTemplate,
			Content:       notificationConfig.MessageTemplate,
			Priority:      notificationConfig.Priority,
			RecipientType: channel,
			RecipientID:   "test_user",
			RecipientAddr: req.Recipient,
			RecipientName: "测试用户",
			EventType:     "test",
			Metadata: map[string]interface{}{
				"notification_id": notificationConfig.ID,
				"sender_id":       senderID,
			},
		}

		// 发送通知
		response, err := n.notificationMgr.SendNotification(ctx, sendRequest)

		// 记录日志
		log := &model.WorkorderNotificationLog{
			NotificationID: notificationConfig.ID,
			EventType:      "test",
			Channel:        channel,
			RecipientType:  "test",
			RecipientID:    "test_user",
			RecipientName:  "测试用户",
			RecipientAddr:  req.Recipient,
			Subject:        notificationConfig.SubjectTemplate,
			Content:        notificationConfig.MessageTemplate,
			Status:         2, // 2-发送中
			SendAt:         time.Now(),
			SenderID:       senderID,
		}

		if err != nil {
			log.Status = 4 // 4-发送失败
			log.ErrorMessage = err.Error()
		} else if response != nil {
			log.Status = 3 // 3-发送成功
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

// SendWorkorderNotification 发送工单通知
func (n *workorderNotificationService) SendWorkorderNotification(ctx context.Context, instanceID int, eventType string, customContent ...string) error {
	// 这里可以根据工单ID获取相关的通知配置并发送
	// 实现逻辑取决于具体的业务需求
	n.logger.Info("发送工单通知",
		zap.Int("instance_id", instanceID),
		zap.String("event_type", eventType))
	return nil
}

// SendNotificationByChannels 通过指定渠道发送通知
func (n *workorderNotificationService) SendNotificationByChannels(ctx context.Context, channels []string, recipient, subject, content string) error {
	if n.notificationMgr == nil {
		return errors.New("通知管理器未初始化")
	}

	for _, channel := range channels {
		sendRequest := &notification.SendRequest{
			Subject:       subject,
			Content:       content,
			Priority:      2, // 默认中等优先级
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
