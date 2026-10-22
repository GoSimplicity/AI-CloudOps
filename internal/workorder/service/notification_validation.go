package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/notification"
)

func recipientIDs(raw []string) ([]int, error) {
	ids := []int{}
	seen := map[int]bool{}
	for _, value := range raw {
		id, err := strconv.Atoi(value)
		if err != nil || id <= 0 {
			return nil, fmt.Errorf("接收人ID必须为正整数")
		}
		if !seen[id] {
			ids = append(ids, id)
			seen[id] = true
		}
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("接收人ID不能为空")
	}
	return ids, nil
}
func validateNotificationDestinations(channels, types, users, roles, depts model.StringList, addresses model.NotificationAddresses) error {
	if len(channels) == 0 || len(types) == 0 {
		return fmt.Errorf("通知渠道和接收人类型不能为空")
	}
	allowed := map[string]bool{}
	for _, channel := range channels {
		if channel != "email" && channel != "feishu" {
			return fmt.Errorf("不支持的通知渠道: %s", channel)
		}
		if allowed[channel] {
			return fmt.Errorf("通知渠道不能重复")
		}
		allowed[channel] = true
	}
	custom := false
	for _, kind := range types {
		var ids []string
		switch kind {
		case "creator", "assignee":
			continue
		case "user":
			ids = users
		case "role":
			ids = roles
		case "dept":
			ids = depts
		case "custom":
			custom = true
			continue
		default:
			return fmt.Errorf("未知接收人类型: %s", kind)
		}
		if _, err := recipientIDs(ids); err != nil {
			return err
		}
	}
	if custom && len(addresses) == 0 {
		return fmt.Errorf("自定义接收人必须提供recipient_addresses")
	}
	if !custom && len(addresses) > 0 {
		return fmt.Errorf("recipient_addresses需要custom接收人类型")
	}
	if len(addresses) > 100 {
		return fmt.Errorf("自定义接收人不能超过100个")
	}
	for i := range addresses {
		if !allowed[addresses[i].Channel] {
			return fmt.Errorf("自定义地址渠道不在通知channels中")
		}
		normalized, err := notification.NormalizeAddress(addresses[i].Channel, addresses[i].Address)
		if err != nil {
			return err
		}
		addresses[i].Address = normalized
		addresses[i].Name = strings.TrimSpace(addresses[i].Name)
		if len([]rune(addresses[i].Name)) > 100 {
			return fmt.Errorf("接收人名称不能超过100字符")
		}
	}
	return nil
}

func normalizeDestination(channel, address string) (string, error) {
	return notification.NormalizeAddress(channel, address)
}
func (s *workorderNotificationService) resolveChannelRecipients(ctx context.Context, channel string, recipients []RecipientInfo) ([]RecipientInfo, error) {
	result := []RecipientInfo{}
	seen := map[string]bool{}
	for _, recipient := range recipients {
		if recipient.Channel != "" && recipient.Channel != channel {
			continue
		}
		address := s.getRecipientAddress(ctx, recipient, channel)
		normalized, err := notification.NormalizeAddress(channel, address)
		if err != nil {
			return nil, fmt.Errorf("接收人未配置有效的%s地址", channel)
		}
		if seen[normalized] {
			continue
		}
		seen[normalized] = true
		recipient.Address = normalized
		recipient.Channel = channel
		result = append(result, recipient)
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("渠道%s没有可用接收人", channel)
	}
	return result, nil
}
