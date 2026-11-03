package demo

import (
	"reflect"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
)

func TestPlan(t *testing.T) {
	rows := fixtures()
	if len(rows) != 88 {
		t.Fatalf("fixture count=%d", len(rows))
	}
	ids := map[string]int{}
	modules := map[string]int{}
	owner := model.User{Model: model.Model{ID: 37}, Username: "local-owner"}
	for i, row := range rows {
		t.Run(row.Key, func(t *testing.T) {
			if _, ok := ids[row.Key]; ok {
				t.Fatal("duplicate fixture key")
			}
			value := row.build(ids, owner)
			if err := validateSafe(value); err != nil {
				t.Fatal("fixture is not inert")
			}
			record := reflect.ValueOf(value).Elem()
			for _, field := range []string{"PoolID", "ProcessID", "FormDesignID", "SendGroupID", "OnDutyGroupID"} {
				id := record.FieldByName(field)
				if !id.IsValid() {
					continue
				}
				if id.Kind() == reflect.Pointer {
					if id.IsNil() {
						t.Fatalf("missing reference: %s", field)
					}
					id = id.Elem()
				}
				if id.Int() <= 0 {
					t.Fatalf("unresolved reference: %s", field)
				}
			}
			switch typed := value.(type) {
			case *model.WorkorderNotificationEvent, *model.WorkorderNotificationDelivery, *model.WorkorderNotificationQueue, *model.WorkorderInboxMessage:
				t.Fatal("seed must never create an outbox, delivery, queue, or orphan inbox row")
			case *model.WorkorderNotification:
				if typed.OperatorID != owner.ID || typed.ProcessID == nil || typed.ScheduledTime != nil {
					t.Fatal("invalid disabled notification metadata")
				}

			case *model.TreeNode:
				if row.Key == "tree" && typed.IsLeaf != 2 {
					t.Fatal("root with children must be a branch")
				}
			case *model.WorkorderProcess:
				steps, ok := typed.Definition["steps"].([]map[string]any)
				if !ok {
					t.Fatal("missing steps")
				}
				known := map[string]bool{}
				for _, step := range steps {
					key, ok := step["id"].(string)
					if !ok || known[key] {
						t.Fatal("invalid process step")
					}
					known[key] = true
				}
				if !known["approval"] {
					t.Fatal("demo tickets refer to missing approval step")
				}
			case *model.User:
				if typed.DepartmentID <= 0 {
					t.Fatal("missing department relation")
				}
			}
			if record.FieldByName("ID").Int() != 0 {
				t.Fatal("fixture must use database-assigned IDs")
			}
			ids[row.Key] = i + 1
			modules[row.Module]++
		})
	}
	for module, want := range map[string]int{"cmdb": 13, "monitoring": 19, "on-duty": 3, "workorder": 30, "cron": 3, "system": 11, "administration": 6, "notification-config": 3} {
		if modules[module] != want {
			t.Errorf("module %s count=%d want=%d", module, modules[module], want)
		}
	}
}
func TestRejectActiveOrCredentialFixtures(t *testing.T) {
	cases := []struct {
		name  string
		value any
	}{
		{"active notification", &model.WorkorderNotification{Status: model.NotificationStatusEnabled}},
		{"external notification channel", &model.WorkorderNotification{Status: 2, IsDefault: 2, TriggerType: model.TriggerTypeImmediate, Channels: model.StringList{"email"}}},
		{"host credential", &model.TreeLocalResource{Status: model.STOPPED, IpAddr: "192.0.2.11", Password: "fixture-only"}},
		{"host non-demo address", &model.TreeLocalResource{Status: model.STOPPED, IpAddr: "10.0.0.1"}},
		{"active cron", &model.CronJob{Status: model.CronJobStatusEnabled}},
		{"active send", &model.MonitorSendGroup{Enable: 1}},
		{"active scrape", &model.MonitorScrapeJob{Enable: 1}},
		{"active alert", &model.MonitorAlertRule{Enable: 1}},
		{"active record", &model.MonitorRecordRule{Enable: 1}},
		{"active duty", &model.MonitorOnDutyGroup{Enable: 1}},
		{"alert address", &model.MonitorAlertManagerPool{AlertManagerInstances: model.StringList{"https://demo.invalid"}}},
		{"remote read", &model.MonitorScrapePool{SupportAlert: 2, SupportRecord: 2, RemoteReadUrl: "https://demo.invalid"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if validateSafe(tc.value) == nil {
				t.Fatal("unsafe fixture accepted")
			}
		})
	}
}
