package service

import (
	"errors"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
)

// 来源只在创建时从已验证对象取快照，避免后续修改源分类影响历史工单路由
func notificationScopeID(value *int) *int {
	if value == nil {
		return nil
	}
	copied := *value
	return &copied
}

func notificationMatchesInstanceScope(config *model.WorkorderNotification, instance *model.WorkorderInstance) bool {
	if config == nil || instance == nil {
		return false
	}
	if config.ProcessID != nil && (*config.ProcessID <= 0 || *config.ProcessID != instance.ProcessID) {
		return false
	}
	return notificationScopeMatches(config.CategoryID, instance.CategoryID) && notificationScopeMatches(config.TemplateID, instance.TemplateID)
}

func notificationScopeMatches(configured, actual *int) bool {
	if configured == nil {
		return true
	}
	return *configured > 0 && actual != nil && *actual > 0 && *configured == *actual
}

func validateAutomaticNotification(config *model.WorkorderNotification) error {
	if config == nil {
		return errors.New("通知配置不存在")
	}
	if (config.TriggerType != model.TriggerTypeImmediate || config.RepeatCount != nil) && config.TriggerVersion != 1 {
		return errors.New("历史高级触发需重新验证并显式启用")
	}
	return validateNotificationTrigger(config.TriggerType, config.ScheduledTime, config.DelayMinutes, config.RepeatInterval, config.RepeatCount, config.TriggerCondition)
}

// 只在显式启用时验证最终配置；禁用及历史普通元数据编辑仍可执行
func validateNotificationActivation(existing *model.WorkorderNotification, req *model.UpdateWorkorderNotificationReq) error {
	if req == nil {
		return errors.New("通知更新请求不能为空")
	}
	if req.Status != model.NotificationStatusEnabled {
		return nil
	}
	if existing == nil {
		return errors.New("通知配置不存在")
	}
	trigger, scheduled, delay, repeat, count, condition := existing.TriggerType, existing.ScheduledTime, existing.DelayMinutes, existing.RepeatInterval, existing.RepeatCount, existing.TriggerCondition
	if req.ClearTrigger {
		scheduled, delay, repeat, count, condition = nil, nil, nil, nil, nil
	}
	if req.TriggerType != "" {
		trigger = req.TriggerType
	}
	if req.ScheduledTime != nil {
		scheduled = req.ScheduledTime
	}
	if req.RepeatInterval != nil {
		repeat = req.RepeatInterval
	}
	if req.DelayMinutes != nil {
		delay = req.DelayMinutes
	}
	if req.RepeatCount != nil {
		count = req.RepeatCount
	}
	if req.TriggerCondition != nil {
		condition = req.TriggerCondition
	}
	if err := validateNotificationTrigger(trigger, scheduled, delay, repeat, count, condition); err != nil {
		return err
	}
	if trigger == model.TriggerTypeScheduled {
		return validateNewScheduleTime(scheduled)
	}
	return nil
}
