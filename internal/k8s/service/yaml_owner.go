package service

import (
	"github.com/GoSimplicity/AI-CloudOps/internal/constants"
	"github.com/GoSimplicity/AI-CloudOps/pkg/base"
	"go.uber.org/zap"
)

const (
	yamlObjectTask     = "yaml_task"
	yamlObjectTemplate = "yaml_template"
)

// authorizeYamlOwner 是 YAML 任务和模板的唯一归属判断。
// super 必须来自 permission.IsSuperAdmin（内置 super_admin 角色），调用方不能采信请求体。
// 超管可以操作他人资源，但更新语句仍必须按原所有者的 user_id 写入，不能改所有者。
func authorizeYamlOwner(logger *zap.Logger, actorID, ownerID, objectID int, objectType string, super bool) error {
	if actorID <= 0 {
		return base.NewBusinessError(constants.ErrInvalidParam, "用户ID不能为空")
	}
	if actorID == ownerID {
		return nil
	}
	if super {
		logger.Info("超级管理员操作他人YAML资源",
			zap.Int("operator_user_id", actorID),
			zap.Int("owner_user_id", ownerID),
			zap.String("object_type", objectType),
			zap.Int("object_id", objectID))
		return nil
	}
	message := "当前用户无权限操作该YAML任务"
	if objectType == yamlObjectTemplate {
		message = "当前用户无权限操作该YAML模板"
	}
	return base.NewBusinessError(constants.ErrNotResourceOwner, message)
}
