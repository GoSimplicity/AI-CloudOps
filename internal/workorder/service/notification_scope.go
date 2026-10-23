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
	return validateImmediateNotification(config.TriggerType, config.ScheduledTime, config.RepeatInterval, config.TriggerCondition, false)
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
	trigger, scheduled, repeat, condition := existing.TriggerType, existing.ScheduledTime, existing.RepeatInterval, existing.TriggerCondition
	if req.TriggerType != "" {
		trigger = req.TriggerType
	}
	if req.ScheduledTime != nil {
		scheduled = req.ScheduledTime
	}
	if req.RepeatInterval != nil {
		repeat = req.RepeatInterval
	}
	if req.TriggerCondition != nil {
		condition = req.TriggerCondition
	}
	return validateImmediateNotification(trigger, scheduled, repeat, condition, false)
}
