package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/notification"
)

// Only bounded, explicit triggers are accepted. Conditions inspect immutable event snapshots.
func validateNotificationTrigger(trigger string, scheduled *time.Time, delay, interval, count *int, condition model.JSONMap) error {
	switch trigger {
	case model.TriggerTypeImmediate:
		if scheduled != nil || delay != nil || len(condition) != 0 {
			return fmt.Errorf("即时通知不能设置排期或条件")
		}
	case model.TriggerTypeDelayed:
		if delay == nil || *delay < 1 || *delay > 10080 || scheduled != nil || len(condition) != 0 {
			return fmt.Errorf("延迟通知需要1至10080分钟延迟")
		}
	case model.TriggerTypeScheduled:
		if scheduled == nil || scheduled.IsZero() || delay != nil || len(condition) != 0 {
			return fmt.Errorf("定时通知需要明确的UTC时刻")
		}
	case model.TriggerTypeConditional:
		if scheduled != nil || delay != nil || len(condition) == 0 {
			return fmt.Errorf("条件通知需要工单快照条件")
		}
		for key, raw := range condition {
			number, ok := conditionNumber(raw)
			if !ok || (key == "status" && (number < 1 || number > 6)) || (key == "priority" && (number < 1 || number > 3)) || (key != "status" && key != "priority") {
				return fmt.Errorf("不支持的通知条件")
			}
		}
	default:
		return fmt.Errorf("不支持的通知触发类型")
	}
	if (interval == nil) != (count == nil) {
		return fmt.Errorf("重复间隔与次数必须同时设置")
	}
	if interval != nil && (*interval < 1 || *interval > 10080 || *count < 2 || *count > 100) {
		return fmt.Errorf("重复间隔或次数超出范围")
	}
	return nil
}

func conditionNumber(raw any) (int, bool) {
	switch value := raw.(type) {
	case float64:
		if value != float64(int(value)) {
			return 0, false
		}
		return int(value), true
	case int:
		return value, true
	case int8:
		return int(value), true
	case json.Number:
		n, err := strconv.Atoi(string(value))
		return n, err == nil
	default:
		return 0, false
	}
}

func validateNewScheduleTime(scheduled *time.Time) error {
	if scheduled == nil {
		return fmt.Errorf("定时时刻不能为空")
	}
	now := time.Now().UTC()
	if !scheduled.After(now) || scheduled.After(now.Add(366*24*time.Hour)) {
		return fmt.Errorf("定时时刻必须在未来366天内")
	}
	return nil
}

func validateInputScheduleTime(scheduled *time.Time) error {
	if scheduled == nil {
		return fmt.Errorf("定时时刻不能为空")
	}
	_, offset := scheduled.Zone()
	if offset != 0 {
		return fmt.Errorf("定时时刻必须使用UTC")
	}
	return validateNewScheduleTime(scheduled)
}

func validateImmediateNotification(trigger string, scheduled *time.Time, repeat *int, condition model.JSONMap, partial bool) error {
	if trigger != model.TriggerTypeImmediate && !(partial && trigger == "") {
		return fmt.Errorf("1.0仅支持immediate即时通知，尚不支持延迟、定时或条件触发")
	}
	if scheduled != nil || repeat != nil || len(condition) != 0 {
		return fmt.Errorf("1.0即时通知不能设置尚未实现的定时、重复或条件参数")
	}
	return nil
}

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
		if channel != "email" && channel != "feishu" && channel != "in_app" {
			return fmt.Errorf("不支持的通知渠道: %s", channel)
		}
		if allowed[channel] {
			return fmt.Errorf("通知渠道不能重复")
		}
		allowed[channel] = true
	}
	custom := false
	inAppRecipient := false
	for _, kind := range types {
		var ids []string
		switch kind {
		case "creator", "assignee":
			inAppRecipient = true
			continue
		case "user":
			inAppRecipient = true
			ids = users
		case "role":
			inAppRecipient = true
			ids = roles
		case "dept":
			inAppRecipient = true
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
	if allowed["in_app"] && !inAppRecipient {
		return fmt.Errorf("站内通知必须配置系统用户、创建人、处理人、角色或部门")
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
		if channel == "in_app" {
			id, err := strconv.Atoi(recipient.ID)
			if err != nil || id <= 0 || s.userDAO == nil {
				return nil, fmt.Errorf("站内通知用户身份无效")
			}
			user, err := s.userDAO.GetByID(ctx, id)
			if err != nil || user == nil || user.Enable != 1 {
				return nil, fmt.Errorf("站内通知用户不存在或已禁用")
			}
			address := strconv.Itoa(id)
			if seen[address] {
				continue
			}
			seen[address] = true
			recipient.ID, recipient.Address, recipient.Channel = address, address, channel
			result = append(result, recipient)
			if len(result) > 1000 {
				return nil, fmt.Errorf("站内通知接收人数量超限")
			}
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
