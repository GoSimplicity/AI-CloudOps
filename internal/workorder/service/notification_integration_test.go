package service

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	userdao "github.com/GoSimplicity/AI-CloudOps/internal/system/dao"
	workdao "github.com/GoSimplicity/AI-CloudOps/internal/workorder/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/notification"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type inboxFeishu struct{ url string }

func (inboxFeishu) IsEnabled() bool                   { return true }
func (inboxFeishu) GetMaxRetries() int                { return 0 }
func (inboxFeishu) GetRetryInterval() time.Duration   { return 0 }
func (inboxFeishu) GetTimeout() time.Duration         { return 3 * time.Second }
func (inboxFeishu) GetChannelName() string            { return "feishu" }
func (inboxFeishu) Validate() error                   { return nil }
func (inboxFeishu) GetAppID() string                  { return "local" }
func (inboxFeishu) GetAppSecret() string              { return "local-only" }
func (c inboxFeishu) GetWebhookURL() string           { return c.url + "/forbidden" }
func (c inboxFeishu) GetPrivateMessageAPI() string    { return c.url + "/messages" }
func (c inboxFeishu) GetTenantAccessTokenAPI() string { return c.url + "/token" }

type inboxConfig struct{ url string }

func (inboxConfig) GetEmail() notification.EmailConfig     { return nil }
func (c inboxConfig) GetFeishu() notification.FeishuConfig { return inboxFeishu{c.url} }

func TestDepartmentCustomNotificationRealLocalInbox(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "inbox.db")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err = db.AutoMigrate(&model.Department{}, &model.User{}, &model.WorkorderNotificationLog{}); err != nil {
		t.Fatal(err)
	}
	department := model.Department{Name: "Operations"}
	if err = db.Create(&department).Error; err != nil {
		t.Fatal(err)
	}
	members := []model.User{{Username: "active", DepartmentID: department.ID, Enable: 1, FeiShuUserId: "ou_member"}, {Username: "disabled", DepartmentID: department.ID, Enable: 2, FeiShuUserId: "ou_disabled"}}
	if err = db.Create(&members).Error; err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	var mu sync.Mutex
	delivered := map[string]int{}
	var fail atomic.Bool
	inbox := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/token" {
			_, _ = io.WriteString(w, `{"code":0,"tenant_access_token":"local-inbox-token"}`)
			return
		}
		if r.URL.Path != "/messages" {
			t.Error("unexpected destination")
			w.WriteHeader(404)
			return
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if fail.Load() {
			_, _ = io.WriteString(w, `{"code":999,"msg":"local rejection"}`)
			return
		}
		mu.Lock()
		delivered[body["receive_id"].(string)]++
		mu.Unlock()
		_, _ = io.WriteString(w, `{"code":0,"data":{"message_id":"local-delivery"}}`)
	}))
	defer inbox.Close()
	manager, err := notification.NewManager(inboxConfig{inbox.URL}, nil, zap.NewNop())
	if err != nil {
		t.Fatal(err)
	}
	svc := &workorderNotificationService{userDAO: userdao.NewUserDAO(db, zap.NewNop()), dao: workdao.NewNotificationDAO(db, zap.NewNop()), notificationMgr: manager, logger: zap.NewNop()}
	config := &model.WorkorderNotification{Model: model.Model{ID: 1}, TriggerType: model.TriggerTypeImmediate, Channels: []string{"feishu"}, RecipientTypes: []string{"dept", "custom"}, RecipientDepts: []string{strconv.Itoa(department.ID)}, RecipientAddresses: model.NotificationAddresses{{Channel: "feishu", Address: "ou_member"}, {Channel: "feishu", Address: "oc_team"}}, MessageTemplate: "local integration"}
	instance := &model.WorkorderInstance{Model: model.Model{ID: 1}, Title: "Local only"}
	if err = svc.processNotification(context.Background(), config, instance, "instance_created", 1); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	if len(delivered) != 2 || delivered["ou_member"] != 1 || delivered["oc_team"] != 1 {
		t.Error("mixed recipients were not correctly deduplicated or disabled user was included")
	}
	mu.Unlock()
	var successful int64
	db.Model(&model.WorkorderNotificationLog{}).Where("status = ?", 3).Count(&successful)
	if successful != 2 {
		t.Fatalf("success logs=%d", successful)
	}
	fail.Store(true)
	if err = svc.processNotification(context.Background(), config, instance, "instance_created", 1); err == nil {
		t.Fatal("provider rejection was swallowed")
	}
	var failed int64
	db.Model(&model.WorkorderNotificationLog{}).Where("status = ?", 4).Count(&failed)
	if failed != 2 {
		t.Fatalf("failure logs=%d", failed)
	}
}
