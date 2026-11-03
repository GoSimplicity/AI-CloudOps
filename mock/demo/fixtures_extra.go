package demo

import (
	"fmt"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
)

func extraFixtures() []fixture {
	makeRow := func(key, module, field string, build func(map[string]int, model.User) any) fixture {
		return fixture{Item: Item{Key: key, Module: module, Name: Prefix + key}, field: field, build: build}
	}
	var rows []fixture
	for i, team := range []string{"平台运维", "业务研发", "安全保障"} {
		label := team
		key := fmt.Sprintf("department-%d", i+1)
		rows = append(rows, makeRow(key, "system", "name", func(_ map[string]int, _ model.User) any {
			return &model.Department{Name: Prefix + key, Description: label + "，" + marker, ParentID: 0}
		}))
	}
	for i := 1; i <= 2; i++ {
		key := fmt.Sprintf("role-%d", i)
		rows = append(rows, makeRow(key, "system", "name", func(_ map[string]int, _ model.User) any {
			return &model.Role{Name: Prefix + key, Code: Prefix + key, Description: "演示角色，无API授权", Status: 1, IsSystem: 0}
		}))
	}
	for i := 1; i <= 6; i++ {
		index := i
		key := fmt.Sprintf("user-%d", i)
		rows = append(rows, makeRow(key, "system", "username", func(ids map[string]int, _ model.User) any {
			return &model.User{Username: Prefix + key, RealName: fmt.Sprintf("演示成员%d", index), DepartmentID: ids[fmt.Sprintf("department-%d", (index-1)%3+1)], Desc: marker, Enable: 2, AccountType: 1, Email: fmt.Sprintf("demo-%d@example.invalid", index)}
		}))
	}
	for i, env := range []string{"development", "staging", "production-demo"} {
		index := i
		environment := env
		key := fmt.Sprintf("tree-%d", i+1)
		rows = append(rows, makeRow(key, "cmdb", "name", func(ids map[string]int, u model.User) any {
			return &model.TreeNode{Name: Prefix + key, ParentID: ids["tree"], Level: 2, IsLeaf: 1, Status: 1, Description: environment + "，" + marker, CreateUserID: u.ID, CreateUserName: u.Username}
		}))
		for j := 1; j <= 2; j++ {
			hostIndex := index*2 + j + 3
			hostKey := fmt.Sprintf("host-%d", hostIndex)
			rows = append(rows, makeRow(hostKey, "cmdb", "name", func(_ map[string]int, u model.User) any {
				status := model.STOPPED
				if hostIndex%2 == 0 {
					status = model.ERROR
				}
				return &model.TreeLocalResource{Name: Prefix + hostKey, Description: marker, Environment: environment, Status: status, IpAddr: fmt.Sprintf("192.0.2.%d", hostIndex+10), Port: 22, Username: "demo", AuthMode: model.AuthModePassword, OSType: "linux", OSName: "Ubuntu demo", Cpu: hostIndex, Memory: hostIndex * 2, Disk: 80, Tags: model.KeyValueList{{Key: "dataset", Value: Prefix}, {Key: "environment", Value: environment}}, CreateUserID: u.ID, CreateUserName: u.Username}
			}))
			rows[len(rows)-1].treeKey = key
		}
	}
	for i := 2; i <= 3; i++ {
		key := fmt.Sprintf("duty-%d", i)
		days := i
		rows = append(rows, makeRow(key, "on-duty", "name", func(_ map[string]int, u model.User) any {
			return &model.MonitorOnDutyGroup{Name: Prefix + key, Description: marker, UserID: u.ID, CreateUserName: u.Username, ShiftDays: days, Enable: 2}
		}))
	}
	for i := 2; i <= 4; i++ {
		key := fmt.Sprintf("scrape-job-%d", i)
		rows = append(rows, makeRow(key, "monitoring", "name", func(ids map[string]int, u model.User) any {
			return &model.MonitorScrapeJob{Name: Prefix + key, PoolID: ids["scrape-pool"], Enable: 2, UserID: u.ID, CreateUserName: u.Username, ServiceDiscoveryType: model.ServiceDiscoveryTypeStatic, IpAddress: "192.0.2.11", Tags: model.StringList{"dataset=" + Prefix}}
		}))
	}
	for i := 2; i <= 4; i++ {
		key := fmt.Sprintf("alert-rule-%d", i)
		severity := model.AlertRuleSeverity((i-1)%3 + 1)
		rows = append(rows, makeRow(key, "monitoring", "name", func(ids map[string]int, u model.User) any {
			return &model.MonitorAlertRule{Name: Prefix + key, PoolID: ids["scrape-pool"], SendGroupID: ids["sender"], Enable: 2, Expr: "up == 0", ForTime: "5m", Severity: severity, UserID: u.ID, CreateUserName: u.Username, Labels: model.StringList{"dataset=" + Prefix}, Annotations: model.StringList{"summary=本地禁用演示"}}
		}))
	}
	for i := 2; i <= 3; i++ {
		key := fmt.Sprintf("record-rule-%d", i)
		rows = append(rows, makeRow(key, "monitoring", "name", func(ids map[string]int, u model.User) any {
			return &model.MonitorRecordRule{Name: Prefix + key, PoolID: ids["scrape-pool"], Enable: 2, Expr: "sum(up)", UserID: u.ID, CreateUserName: u.Username, Labels: model.StringList{"dataset=" + Prefix}}
		}))
	}
	for i := 1; i <= 5; i++ {
		key := fmt.Sprintf("event-%d", i)
		index := i
		rows = append(rows, makeRow(key, "monitoring", "fingerprint", func(ids map[string]int, _ model.User) any {
			return &model.MonitorAlertEvent{AlertName: fmt.Sprintf("本地演示历史告警%d（已恢复）", index), Fingerprint: Prefix + key, Status: model.MonitorAlertEventStatusResolved, RuleID: ids["alert-rule"], SendGroupID: ids["sender"], EventTimes: index, Labels: model.StringList{"dataset=" + Prefix, "environment=local-demo", "severity=warning"}}
		}))
	}
	for i := 2; i <= 3; i++ {
		key := fmt.Sprintf("category-%d", i)
		rows = append(rows, makeRow(key, "workorder", "name", func(_ map[string]int, u model.User) any {
			return &model.WorkorderCategory{Name: Prefix + key, Description: marker, Status: 1, OperatorID: u.ID, OperatorName: u.Username}
		}))
	}
	states := []int8{model.InstanceStatusPending, model.InstanceStatusProcessing, model.InstanceStatusRejected, model.InstanceStatusCancelled}
	for i, state := range states {
		index := i + 3
		status := state
		key := fmt.Sprintf("ticket-%d", index)
		rows = append(rows, makeRow(key, "workorder", "title", func(ids map[string]int, u model.User) any {
			category := ids["category"]
			template := ids["template"]
			step := "approval"
			due := time.Date(2026, 11, 10+index, 8, 0, 0, 0, time.UTC)
			assignee := ids[fmt.Sprintf("user-%d", index)]
			return &model.WorkorderInstance{Title: Prefix + key, SerialNumber: fmt.Sprintf("DEMO-LOCAL-V1-%03d", index), Description: marker, ProcessID: ids["process"], TemplateID: &template, CategoryID: &category, CurrentStepID: &step, Status: status, Priority: int8((index-1)%3 + 1), OperatorID: u.ID, OperatorName: u.Username, AssigneeID: &assignee, DueDate: &due, FormData: model.JSONMap{"summary": "服务器变更演示，无外部操作"}, Tags: model.StringList{Prefix}}
		}))
	}
	for i := 1; i <= 6; i++ {
		ticket := i
		for _, action := range []string{model.TimelineActionCreate, model.TimelineActionUpdate} {
			kind := action
			key := fmt.Sprintf("timeline-%d-%s", i, action)
			rows = append(rows, makeRow(key, "workorder", "comment", func(ids map[string]int, u model.User) any {
				return &model.WorkorderInstanceTimeline{InstanceID: ids[fmt.Sprintf("ticket-%d", ticket)], Action: kind, OperatorID: u.ID, OperatorName: u.Username, Comment: Prefix + key, ActionDetail: `{"source":"local-demo","note":"仅演示审计时间线，无外部事件"}`}
			}))
		}
		key := fmt.Sprintf("comment-%d", i)
		rows = append(rows, makeRow(key, "workorder", "content", func(ids map[string]int, u model.User) any {
			return &model.WorkorderInstanceComment{InstanceID: ids[fmt.Sprintf("ticket-%d", ticket)], OperatorID: u.ID, OperatorName: u.Username, Content: Prefix + key, Type: model.CommentTypeNormal, Status: 1, IsSystem: 2}
		}))
	}
	for i := 2; i <= 3; i++ {
		index := i
		key := fmt.Sprintf("cron-%d", i)
		rows = append(rows, makeRow(key, "cron", "name", func(_ map[string]int, u model.User) any {
			return &model.CronJob{Name: Prefix + key, Description: marker, JobType: model.CronJobTypeCommand, Status: model.CronJobStatusDisabled, IsBuiltIn: 2, Schedule: "0 0 0 1 1 *", Command: "true", Timeout: 10, LastRunStatus: int8(index - 1), RunCount: 3, SuccessCount: 4 - index, FailureCount: index - 1, LastRunOutput: "历史展示样例，未执行", CreatedBy: u.ID, CreatedByName: u.Username}
		}))
	}
	return rows
}
