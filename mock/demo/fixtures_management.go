package demo

import (
	"fmt"
	"strconv"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"gorm.io/datatypes"
)

// These rows are passive metadata, not notification events or deliveries.
// Real API catalog entries already exist in the local bootstrap; no duplicate
// wildcard or permission grants are added here.
func managementFixtures() []fixture {
	rows := make([]fixture, 0, 9)
	for i, code := range []int{200, 200, 400, 403, 500, 200} {
		index := i + 1
		status := code
		key := fmt.Sprintf("audit-%d", index)
		rows = append(rows, fixture{
			Item: Item{Key: key, Module: "administration", Name: Prefix + key}, field: "trace_id",
			build: func(ids map[string]int, u model.User) any {
				return &model.AuditLog{UserID: u.ID, TraceID: Prefix + key, IPAddress: "127.0.0.1", UserAgent: "local-demo-fixture", HttpMethod: "GET", Endpoint: "/api/tree/local/list", Description: "本地演示审计，不代表真实请求；" + marker, OperationType: "read", TargetType: "tree_local", TargetID: strconv.Itoa(ids["host-1"]), StatusCode: status, Duration: int64(index * 1200), RequestBody: datatypes.JSON(`{}`), ResponseBody: datatypes.JSON(`{}`)}
			},
		})
	}
	for i, event := range []string{model.EventTypeInstanceCreated, model.EventTypeInstanceCompleted, model.EventTypeInstanceCommented} {
		index := i + 1
		eventType := event
		key := fmt.Sprintf("notification-%d", index)
		rows = append(rows, fixture{
			Item: Item{Key: key, Module: "notification-config", Name: Prefix + key}, field: "name",
			build: func(ids map[string]int, u model.User) any {
				process := ids["process"]
				return &model.WorkorderNotification{Name: Prefix + key, Description: marker, ProcessID: &process, EventTypes: model.StringList{eventType}, TriggerType: model.TriggerTypeImmediate, TriggerVersion: 1, Channels: model.StringList{"in_app"}, RecipientTypes: model.StringList{model.RecipientTypeUser}, RecipientUsers: model.StringList{strconv.Itoa(u.ID)}, MessageTemplate: "仅本地演示通知配置，不产生投递", SubjectTemplate: "本地演示通知（禁用）", Status: model.NotificationStatusDisabled, Priority: int8(index), OperatorID: u.ID, IsDefault: model.IsDefaultNo, Settings: model.JSONMap{"demo_only": true}}
			},
		})
	}
	return rows
}
