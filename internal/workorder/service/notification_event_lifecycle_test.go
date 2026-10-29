package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	workdao "github.com/GoSimplicity/AI-CloudOps/internal/workorder/dao"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// 工单状态与事件必须同时提交或回滚，且事件快照必须反映提交后的状态
func TestNotificationLifecycleAtomicity(t *testing.T) {
	type operation func(*instanceService, int) error
	cases := []struct {
		name, step, event string
		before, after     int8
		assigned          bool
		run               operation
	}{
		{"submit", "start", model.EventTypeInstanceSubmitted, model.InstanceStatusDraft, model.InstanceStatusPending, false, func(s *instanceService, id int) error { return s.SubmitInstance(context.Background(), id, 7, "tester") }},
		{"assign", "approval", model.EventTypeInstanceAssigned, model.InstanceStatusPending, model.InstanceStatusProcessing, false, func(s *instanceService, id int) error {
			return s.AssignInstance(context.Background(), id, 7, 7, "tester")
		}},
		{"approve-next", "approval", model.EventTypeInstanceApproved, model.InstanceStatusPending, model.InstanceStatusProcessing, false, func(s *instanceService, id int) error {
			return s.ApproveInstance(context.Background(), id, 7, "tester", "approved")
		}},
		{"approve-end", "last", model.EventTypeInstanceCompleted, model.InstanceStatusPending, model.InstanceStatusCompleted, false, func(s *instanceService, id int) error {
			return s.ApproveInstance(context.Background(), id, 7, "tester", "approved")
		}},
		{"reject", "approval", model.EventTypeInstanceRejected, model.InstanceStatusPending, model.InstanceStatusRejected, false, func(s *instanceService, id int) error {
			return s.RejectInstance(context.Background(), id, 7, "tester", "reject reason")
		}},
		{"cancel", "approval", model.EventTypeInstanceCancelled, model.InstanceStatusPending, model.InstanceStatusCancelled, false, func(s *instanceService, id int) error {
			return s.CancelInstance(context.Background(), id, 7, "tester", "cancel reason")
		}},
		{"complete", "task", model.EventTypeInstanceCompleted, model.InstanceStatusProcessing, model.InstanceStatusCompleted, true, func(s *instanceService, id int) error {
			return s.CompleteInstance(context.Background(), id, 7, "tester", "complete reason")
		}},
		{"return", "task", model.EventTypeInstanceReturned, model.InstanceStatusProcessing, model.InstanceStatusDraft, true, func(s *instanceService, id int) error {
			return s.ReturnInstance(context.Background(), id, 7, "tester", "return reason")
		}},
	}
	for _, tc := range cases {
		for _, failEvent := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/rollback=%t", tc.name, failEvent), func(t *testing.T) {
				db := eventTestDB(t)
				form := model.WorkorderFormDesign{Name: "test-form", Schema: model.JSONMap{"fields": []any{}}, Status: model.FormDesignStatusPublished, OperatorID: 7, OperatorName: "tester"}
				if err := db.Create(&form).Error; err != nil {
					t.Fatal(err)
				}
				definition := model.JSONMap{
					"steps": []any{
						map[string]any{"id": "start", "type": "start"},
						map[string]any{"id": "approval", "type": "approval", "assignee_type": "user", "assignee_ids": []int{7}},
						map[string]any{"id": "task", "type": "task", "assignee_type": "user", "assignee_ids": []int{7}},
						map[string]any{"id": "last", "type": "approval", "assignee_type": "user", "assignee_ids": []int{7}},
						map[string]any{"id": "end", "type": "end"},
					},
					"connections": []any{map[string]any{"from": "start", "to": "approval"}, map[string]any{"from": "approval", "to": "task"}, map[string]any{"from": "last", "to": "end"}},
				}
				process := model.WorkorderProcess{Name: "test-process", FormDesignID: form.ID, Definition: definition, Status: model.ProcessStatusPublished, OperatorID: 7, OperatorName: "tester"}
				if err := db.Create(&process).Error; err != nil {
					t.Fatal(err)
				}
				instance := model.WorkorderInstance{Title: "fixture", SerialNumber: "test-unique", ProcessID: process.ID, CurrentStepID: &tc.step, Status: tc.before, OperatorID: 7, OperatorName: "tester"}
				if tc.assigned {
					id := 7
					instance.AssigneeID = &id
				}
				if err := db.Create(&instance).Error; err != nil {
					t.Fatal(err)
				}
				svc := &instanceService{dao: workdao.NewWorkorderInstanceDAO(db, zap.NewNop()), processDao: workdao.NewProcessDAO(db, zap.NewNop()), formDesignDao: workdao.NewWorkorderFormDesignDAO(db, zap.NewNop()), flowDao: workdao.NewInstanceFlowDAO(db, zap.NewNop()), timelineDao: workdao.NewInstanceTimeLineDAO(db, zap.NewNop()), commentDao: workdao.NewWorkorderInstanceCommentDAO(db, zap.NewNop()), notificationService: &workorderNotificationService{}, logger: zap.NewNop()}
				if failEvent {
					if err := db.Callback().Create().Before("gorm:create").Register("fail-owned-event", func(tx *gorm.DB) {
						if tx.Statement.Table == "cl_workorder_notification_event" {
							tx.AddError(errors.New("owned event failure"))
						}
					}); err != nil {
						t.Fatal(err)
					}
				}
				err := tc.run(svc, instance.ID)
				if failEvent && err == nil {
					t.Fatal("事件写入失败必须回滚")
				}
				if !failEvent && err != nil {
					t.Fatalf("工单操作失败: %v", err)
				}
				var got model.WorkorderInstance
				if err := db.First(&got, instance.ID).Error; err != nil {
					t.Fatal(err)
				}
				var events []model.WorkorderNotificationEvent
				if err := db.Where("instance_id = ?", instance.ID).Find(&events).Error; err != nil {
					t.Fatal(err)
				}
				if failEvent {
					if got.Status != tc.before || got.CurrentStepID == nil || *got.CurrentStepID != tc.step || (got.AssigneeID != nil) != tc.assigned || got.CompletedAt != nil || len(events) != 0 {
						t.Fatalf("事务回滚不完整: instance=%+v events=%d", got, len(events))
					}
					for _, table := range []any{&model.WorkorderInstanceFlow{}, &model.WorkorderInstanceTimeline{}, &model.WorkorderInstanceComment{}} {
						var count int64
						if err := db.Model(table).Where("instance_id = ?", instance.ID).Count(&count).Error; err != nil || count != 0 {
							t.Fatalf("流转记录未回滚: count=%d err=%v", count, err)
						}
					}
					return
				}
				if got.Status != tc.after || len(events) != 1 || events[0].EventType != tc.event || events[0].SenderID != 7 {
					t.Fatalf("状态或事件错误: status=%d events=%+v", got.Status, events)
				}
				var snapshot model.WorkorderInstance
				if err := json.Unmarshal(events[0].Snapshot, &snapshot); err != nil {
					t.Fatal(err)
				}
				if snapshot.ID != instance.ID || snapshot.Status != got.Status || snapshot.Title != got.Title {
					t.Fatalf("事件快照与提交状态不符: %+v", snapshot)
				}
				if tc.name == "return" && (got.AssigneeID != nil || snapshot.AssigneeID != nil) {
					t.Fatal("退回工单必须清除受理人")
				}
				if tc.name == "assign" && (got.AssigneeID == nil || *got.AssigneeID != 7 || snapshot.AssigneeID == nil || *snapshot.AssigneeID != 7) {
					t.Fatal("指派状态及快照不符")
				}
			})
		}
	}
}

func TestTemplateCreationEventAtomicity(t *testing.T) {
	for _, failEvent := range []bool{false, true} {
		t.Run(fmt.Sprintf("rollback=%t", failEvent), func(t *testing.T) {
			db := eventTestDB(t)
			if err := db.AutoMigrate(&model.WorkorderTemplate{}); err != nil {
				t.Fatal(err)
			}
			category := model.WorkorderCategory{Name: "template-category", Status: 1}
			if err := db.Create(&category).Error; err != nil {
				t.Fatal(err)
			}
			form := model.WorkorderFormDesign{Name: "template-form", Schema: model.JSONMap{"fields": []any{}}, Status: model.FormDesignStatusPublished}
			if err := db.Create(&form).Error; err != nil {
				t.Fatal(err)
			}
			process := model.WorkorderProcess{Name: "template-process", FormDesignID: form.ID, Definition: model.JSONMap{"steps": []any{}, "connections": []any{}}, Status: model.ProcessStatusPublished}
			if err := db.Create(&process).Error; err != nil {
				t.Fatal(err)
			}
			template := model.WorkorderTemplate{Name: "owned-template", FormDesignID: form.ID, ProcessID: process.ID, CategoryID: &category.ID, Status: model.TemplateStatusEnabled}
			if err := db.Create(&template).Error; err != nil {
				t.Fatal(err)
			}
			svc := &instanceService{dao: workdao.NewWorkorderInstanceDAO(db, zap.NewNop()), templateDao: workdao.NewTemplateDAO(db, zap.NewNop()), processDao: workdao.NewProcessDAO(db, zap.NewNop()), formDesignDao: workdao.NewWorkorderFormDesignDAO(db, zap.NewNop()), flowDao: workdao.NewInstanceFlowDAO(db, zap.NewNop()), timelineDao: workdao.NewInstanceTimeLineDAO(db, zap.NewNop()), notificationService: &workorderNotificationService{}, logger: zap.NewNop()}
			if failEvent {
				if err := db.Callback().Create().Before("gorm:create").Register("fail-owned-template-event", func(tx *gorm.DB) {
					if tx.Statement.Table == "cl_workorder_notification_event" {
						tx.AddError(errors.New("owned event failure"))
					}
				}); err != nil {
					t.Fatal(err)
				}
			}
			req := &model.CreateWorkorderInstanceFromTemplateReq{Title: "created-from-template", Priority: model.PriorityNormal, OperatorID: 7, OperatorName: "tester"}
			err := svc.CreateInstanceFromTemplate(context.Background(), template.ID, req)
			if failEvent && err == nil {
				t.Fatal("事件写入失败必须回滚模板创建")
			}
			if !failEvent && err != nil {
				t.Fatal(err)
			}
			var instances []model.WorkorderInstance
			if err := db.Where("title = ?", req.Title).Find(&instances).Error; err != nil {
				t.Fatal(err)
			}
			var events []model.WorkorderNotificationEvent
			if err := db.Find(&events).Error; err != nil {
				t.Fatal(err)
			}
			if failEvent {
				if len(instances) != 0 || len(events) != 0 {
					t.Fatal("模板实例或事件未回滚")
				}
				for _, table := range []any{&model.WorkorderInstanceFlow{}, &model.WorkorderInstanceTimeline{}} {
					var count int64
					if err := db.Model(table).Count(&count).Error; err != nil || count != 0 {
						t.Fatalf("模板流转记录未回滚: count=%d err=%v", count, err)
					}
				}
				return
			}
			if len(instances) != 1 || len(events) != 1 || events[0].EventType != model.EventTypeInstanceCreated || events[0].SenderID != 7 {
				t.Fatal("模板实例事件未提交")
			}
			var snapshot model.WorkorderInstance
			if err := json.Unmarshal(events[0].Snapshot, &snapshot); err != nil {
				t.Fatal(err)
			}
			if snapshot.ID != instances[0].ID || snapshot.TemplateID == nil || *snapshot.TemplateID != template.ID || snapshot.CategoryID == nil || *snapshot.CategoryID != category.ID {
				t.Fatal("模板来源快照错误")
			}
		})
	}
}
