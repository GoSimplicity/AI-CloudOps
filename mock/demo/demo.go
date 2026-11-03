// Package demo builds an explicitly local, disabled demonstration dataset.
// It is never called by application startup and never starts business services.
package demo

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const Prefix = "demo-local-v1-"
const marker = "Local demonstration only; external execution disabled."

type Item struct {
	Key    string `json:"key"`
	Module string `json:"module"`
	Name   string `json:"name"`
}
type Report struct {
	Created  int            `json:"created"`
	Existing int            `json:"existing"`
	Modules  map[string]int `json:"modules"`
}
type fixture struct {
	Item
	field   string
	treeKey string
	build   func(map[string]int, model.User) any
}

func Plan() []Item {
	result := make([]Item, 0)
	for _, row := range fixtures() {
		result = append(result, row.Item)
	}
	return result
}
func fixtures() []fixture {
	f := func(key, module string, build func(map[string]int, model.User) any) fixture {
		return fixture{Item: Item{Key: key, Module: module, Name: Prefix + key}, field: "name", build: build}
	}
	rows := []fixture{
		f("tree", "cmdb", func(_ map[string]int, u model.User) any {
			return &model.TreeNode{Name: Prefix + "tree", Description: marker, Level: 1, IsLeaf: 2, Status: 1, CreateUserID: u.ID, CreateUserName: u.Username}
		}),
	}
	for i := 1; i <= 3; i++ {
		index := i
		key := fmt.Sprintf("host-%d", i)
		rows = append(rows, f(key, "cmdb", func(_ map[string]int, u model.User) any {
			return &model.TreeLocalResource{Name: Prefix + key, Description: marker, Environment: "local-demo", Status: model.STOPPED, IpAddr: fmt.Sprintf("192.0.2.%d", index+10), Port: 22, Username: "demo", AuthMode: model.AuthModePassword, OSType: "linux", OSName: "Linux demo", Cpu: 2 * index, Memory: 4 * index, Disk: 40 * index, Tags: model.KeyValueList{{Key: "dataset", Value: Prefix}}, CreateUserID: u.ID, CreateUserName: u.Username}
		}))
	}
	rows = append(rows,
		f("scrape-pool", "monitoring", func(_ map[string]int, u model.User) any {
			return &model.MonitorScrapePool{Name: Prefix + "scrape-pool", UserID: u.ID, CreateUserName: u.Username, SupportAlert: 2, SupportRecord: 2, PrometheusInstances: model.StringList{}, Tags: model.StringList{"dataset=" + Prefix}}
		}),
		f("alert-pool", "monitoring", func(_ map[string]int, u model.User) any {
			return &model.MonitorAlertManagerPool{Name: Prefix + "alert-pool", UserID: u.ID, CreateUserName: u.Username, AlertManagerInstances: model.StringList{}, GroupBy: model.StringList{"service,api"}, Receiver: "demo-disabled"}
		}),
		f("duty", "on-duty", func(_ map[string]int, u model.User) any {
			return &model.MonitorOnDutyGroup{Name: Prefix + "duty", Description: marker, UserID: u.ID, CreateUserName: u.Username, ShiftDays: 7, Enable: 2}
		}),
		f("sender", "monitoring", func(ids map[string]int, u model.User) any {
			return &model.MonitorSendGroup{Name: Prefix + "sender", NameZh: "本地演示通知组（禁用）", UserID: u.ID, CreateUserName: u.Username, PoolID: ids["alert-pool"], OnDutyGroupID: ids["duty"], Enable: 2, SendResolved: 2, NeedUpgrade: 2, NotifyMethods: model.StringList{}}
		}),
		f("scrape-job", "monitoring", func(ids map[string]int, u model.User) any {
			return &model.MonitorScrapeJob{Name: Prefix + "scrape-job", UserID: u.ID, CreateUserName: u.Username, PoolID: ids["scrape-pool"], Enable: 2, ServiceDiscoveryType: model.ServiceDiscoveryTypeStatic, IpAddress: "192.0.2.11", Tags: model.StringList{"dataset=" + Prefix}}
		}),
		f("alert-rule", "monitoring", func(ids map[string]int, u model.User) any {
			return &model.MonitorAlertRule{Name: Prefix + "alert-rule", UserID: u.ID, CreateUserName: u.Username, PoolID: ids["scrape-pool"], SendGroupID: ids["sender"], Enable: 2, Expr: "up == 0", Severity: model.AlertRuleSeverityWarning, Labels: model.StringList{"dataset=" + Prefix}, Annotations: model.StringList{"summary=本地演示规则，不执行"}}
		}),
		f("record-rule", "monitoring", func(ids map[string]int, u model.User) any {
			return &model.MonitorRecordRule{Name: Prefix + "record-rule", UserID: u.ID, CreateUserName: u.Username, PoolID: ids["scrape-pool"], Enable: 2, Expr: "sum(up)", Labels: model.StringList{"dataset=" + Prefix}}
		}),
		f("category", "workorder", func(_ map[string]int, u model.User) any {
			return &model.WorkorderCategory{Name: Prefix + "category", Description: marker, Status: 1, OperatorID: u.ID, OperatorName: u.Username}
		}),
		f("form", "workorder", func(ids map[string]int, u model.User) any {
			category := ids["category"]
			return &model.WorkorderFormDesign{Name: Prefix + "form", Description: marker, CategoryID: &category, Status: 2, IsTemplate: 2, OperatorID: u.ID, OperatorName: u.Username, Tags: model.StringList{Prefix}, Schema: model.JSONMap{"fields": []map[string]any{{"id": "summary", "name": "summary", "label": "申请说明", "type": "textarea", "required": 1}}}}
		}),
		f("process", "workorder", func(ids map[string]int, u model.User) any {
			category := ids["category"]
			return &model.WorkorderProcess{Name: Prefix + "process", Description: marker, CategoryID: &category, FormDesignID: ids["form"], Status: 1, IsDefault: 2, OperatorID: u.ID, OperatorName: u.Username, Tags: model.StringList{Prefix}, Definition: model.JSONMap{"steps": []map[string]any{{"id": "start", "type": "start", "name": "开始", "sort_order": 1}, {"id": "approval", "type": "approval", "name": "人工审批（演示）", "assignee_type": "user", "assignee_ids": []int{u.ID}, "actions": []string{"approve", "reject"}, "sort_order": 2}, {"id": "end", "type": "end", "name": "结束", "sort_order": 3}}, "connections": []map[string]any{{"from": "start", "to": "approval"}, {"from": "approval", "to": "end"}}}}
		}),
		f("template", "workorder", func(ids map[string]int, u model.User) any {
			category := ids["category"]
			return &model.WorkorderTemplate{Name: Prefix + "template", Description: marker, ProcessID: ids["process"], FormDesignID: ids["form"], CategoryID: &category, Status: 2, OperatorID: u.ID, OperatorName: u.Username, Tags: model.StringList{Prefix}, DefaultValues: model.JSONMap{"summary": "本地演示，无外部操作"}}
		}),
	)
	for i, status := range []int8{model.InstanceStatusDraft, model.InstanceStatusCompleted} {
		index := i
		state := status
		key := fmt.Sprintf("ticket-%d", i+1)
		row := f(key, "workorder", func(ids map[string]int, u model.User) any {
			category := ids["category"]
			template := ids["template"]
			return &model.WorkorderInstance{Title: Prefix + key, SerialNumber: fmt.Sprintf("DEMO-LOCAL-V1-%03d", index+1), Description: marker, ProcessID: ids["process"], TemplateID: &template, CategoryID: &category, Status: state, Priority: 2, OperatorID: u.ID, OperatorName: u.Username, FormData: model.JSONMap{"summary": "仅本地演示工单"}, Tags: model.StringList{Prefix}}
		})
		row.field = "title"
		rows = append(rows, row)
	}
	rows = append(rows, f("cron", "cron", func(_ map[string]int, u model.User) any {
		return &model.CronJob{Name: Prefix + "cron", Description: marker, JobType: model.CronJobTypeCommand, Status: model.CronJobStatusDisabled, IsBuiltIn: 2, Schedule: "0 0 0 1 1 *", Command: "true", Timeout: 10, MaxRetry: 0, CreatedBy: u.ID, CreatedByName: u.Username}
	}))
	return append(append(rows, extraFixtures()...), managementFixtures()...)
}

// Apply only inserts missing names. Existing records and associations are never
// updated, restored, deleted, or re-enabled. Schema migration belongs to cmd/schema.
// The caller must exclusively own the local connection and serialize seed runs.
func Apply(ctx context.Context, db *gorm.DB, ownerID int) (Report, error) {
	report := Report{Modules: map[string]int{}}
	if ownerID <= 0 {
		return report, errors.New("existing owner ID is required")
	}
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var owner model.User
		if err := tx.First(&owner, ownerID).Error; err != nil {
			return errors.New("demo owner does not exist")
		}
		ids := map[string]int{}
		for _, row := range fixtures() {
			value := row.build(ids, owner)
			result := tx.Unscoped().Where(row.field+" = ?", row.Name).First(value)
			created := errors.Is(result.Error, gorm.ErrRecordNotFound)
			if result.Error != nil && !created {
				return fmt.Errorf("inspect demo %s failed", row.Key)
			}
			if !created {
				record := reflect.ValueOf(value).Elem()
				if record.FieldByName("DeletedAt").Uint() != 0 {
					return fmt.Errorf("demo %s was deleted; not restoring user data", row.Key)
				}
				for _, field := range []string{"UserID", "CreateUserID", "OperatorID", "CreatedBy"} {
					if v := record.FieldByName(field); v.IsValid() && v.Int() != int64(ownerID) {
						return fmt.Errorf("demo %s belongs to another owner", row.Key)
					}
				}
				if err := validateSafe(value); err != nil {
					return fmt.Errorf("existing demo %s is not inert", row.Key)
				}
				report.Existing++
			} else {
				if err := validateSafe(value); err != nil {
					return fmt.Errorf("unsafe demo %s rejected", row.Key)
				}
				if err := tx.Omit(clause.Associations).Create(value).Error; err != nil {
					return fmt.Errorf("create demo %s failed", row.Key)
				}
				report.Created++
			}
			ids[row.Key] = int(reflect.ValueOf(value).Elem().FieldByName("ID").Int())
			report.Modules[row.Module]++
			// Only a freshly inserted record receives its initial relation. Reruns
			// deliberately preserve a user's later unbind or membership edits.
			if created && strings.HasPrefix(row.Key, "duty") {
				if err := tx.Model(value).Omit("Users.*").Association("Users").Append(&owner); err != nil {
					return errors.New("demo duty member relation failed")
				}
			}
			if created && strings.HasPrefix(row.Key, "user-") {
				role := model.Role{Model: model.Model{ID: ids["role-1"]}}
				if err := tx.Model(&role).Omit("Users.*").Association("Users").Append(value); err != nil {
					return errors.New("demo user role relation failed")
				}
			}
			if created && strings.HasPrefix(row.Key, "host-") {
				parentKey := row.treeKey
				if parentKey == "" {
					parentKey = "tree"
				}
				treeID := ids[parentKey]
				if treeID <= 0 {
					return errors.New("demo tree relation unresolved")
				}

				node := model.TreeNode{Model: model.Model{ID: treeID}}
				if err := tx.Model(value).Omit("TreeNodes.*").Association("TreeNodes").Append(&node); err != nil {
					return errors.New("demo CMDB relation failed")
				}
			}
		}
		return nil
	})
	if err != nil {
		return Report{}, err
	}
	return report, nil
}
func validateSafe(value any) error {
	unsafe := false
	switch row := value.(type) {
	case *model.WorkorderNotification:
		unsafe = row.Status != model.NotificationStatusDisabled || row.IsDefault != model.IsDefaultNo || row.TriggerType != model.TriggerTypeImmediate || row.ScheduledTime != nil || row.DelayMinutes != nil || row.RepeatInterval != nil || row.RepeatCount != nil || len(row.RecipientAddresses) != 0 || len(row.Channels) != 1 || row.Channels[0] != "in_app"
	case *model.User:
		unsafe = row.Enable != 2 || row.Password != ""
	case *model.TreeLocalResource:
		unsafe = row.Password != "" || row.Key != "" || (row.Status != model.STOPPED && row.Status != model.ERROR) || !strings.HasPrefix(row.IpAddr, "192.0.2.")
	case *model.MonitorScrapePool:
		unsafe = len(row.PrometheusInstances) != 0 || row.RemoteReadUrl != "" || row.RemoteWriteUrl != "" || row.AlertManagerUrl != "" || row.SupportAlert != 2 || row.SupportRecord != 2
	case *model.MonitorAlertManagerPool:
		unsafe = len(row.AlertManagerInstances) != 0
	case *model.MonitorScrapeJob:
		unsafe = row.Enable != 2 || row.BearerToken != "" || row.KubeConfigFilePath != ""
	case *model.MonitorSendGroup:
		unsafe = row.Enable != 2 || row.FeiShuQunRobotToken != "" || len(row.NotifyMethods) != 0
	case *model.MonitorAlertRule:
		unsafe = row.Enable != 2
	case *model.MonitorRecordRule:
		unsafe = row.Enable != 2
	case *model.MonitorOnDutyGroup:
		unsafe = row.Enable != 2
	case *model.CronJob:
		unsafe = row.Status != model.CronJobStatusDisabled || row.SSHResourceID != nil || row.HTTPUrl != ""
	}
	if unsafe {
		return errors.New("external execution or credentials present")
	}
	return nil
}
