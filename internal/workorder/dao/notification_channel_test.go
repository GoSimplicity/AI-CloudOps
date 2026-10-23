package dao

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	databaseLogger "github.com/GoSimplicity/AI-CloudOps/pkg/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestNotificationChannelFilterSQLite(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "channel.db")), &gorm.Config{Logger: databaseLogger.NewGORMLogger(zap.NewNop())})
	if err != nil {
		t.Fatal(err)
	}
	pool, _ := db.DB()
	defer pool.Close()
	assertNotificationChannelPagination(t, db)
}
func assertNotificationChannelPagination(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.AutoMigrate(&model.WorkorderNotification{}); err != nil {
		t.Fatal("notification fixture migration failed")
	}
	var expected []int
	rows := []struct {
		name     string
		channels model.StringList
		raw      string
		status   int8
	}{
		{name: "email-first", channels: model.StringList{"email"}, status: 1},
		{name: "feishu-only", channels: model.StringList{"feishu"}, status: 1},
		{name: "email-second", channels: model.StringList{"email", "feishu"}, status: 2},
		{name: "prefix-only", channels: model.StringList{"email_backup"}, status: 1},
		{name: "email-third", channels: model.StringList{"sms", "email"}, status: 1},
		{name: "case-only", channels: model.StringList{"EMAIL"}, status: 1},
		{name: "webhook-only", channels: model.StringList{"webhook"}, status: 1},
		{name: "empty", channels: model.StringList{}, status: 1},
		{name: "scalar", raw: `"email"`, status: 1},
		{name: "legacy-invalid", raw: `email`, status: 1},
		{name: "nested", raw: `[["email"]]`, status: 1},
	}
	for _, row := range rows {
		value := &model.WorkorderNotification{Name: row.name, Channels: row.channels, EventTypes: model.StringList{"instance_created"}, RecipientTypes: model.StringList{"creator"}, MessageTemplate: "fixture", Status: row.status}
		if err := db.Create(value).Error; err != nil {
			t.Fatal("notification fixture create failed")
		}
		if row.raw != "" {
			if err := db.Model(value).Update("channels", row.raw).Error; err != nil {
				t.Fatal("legacy fixture setup failed")
			}
		}
		if row.name == "email-first" || row.name == "email-second" || row.name == "email-third" {
			expected = append(expected, value.ID)
		}
	}
	store := NewNotificationDAO(db, zap.NewNop())
	ctx := context.Background()
	request := func(payload string) *model.ListWorkorderNotificationReq {
		t.Helper()
		var req model.ListWorkorderNotificationReq
		if err := json.Unmarshal([]byte(payload), &req); err != nil {
			t.Fatal(err)
		}
		return &req
	}
	for page := 1; page <= 4; page++ {
		req := request(`{"channel":"email","page_size":1}`)
		req.Page = page
		result, err := store.ListNotification(ctx, req)
		if err != nil {
			t.Fatal("channel query failed")
		}
		if result.Total != 3 {
			t.Fatal("channel filter was not applied before count")
		}
		if page <= 3 {
			if len(result.Items) != 1 || result.Items[0].ID != expected[3-page] {
				t.Fatal("exact channel match was filtered only within current page or order changed")
			}
		} else if len(result.Items) != 0 {
			t.Fatal("pagination past last filtered row was not empty")
		}
	}
	result, err := store.ListNotification(ctx, request(`{"page":1,"page_size":20}`))
	if err != nil || result.Total != int64(len(rows)) {
		t.Fatal("omitted channel no longer returns all configurations")
	}
	for _, tt := range []struct {
		payload string
		total   int64
	}{
		{`{"channel":"feishu","page_size":20}`, 2}, {`{"channel":"sms","page_size":20}`, 1}, {`{"channel":"webhook","page_size":20}`, 1},
		{`{"channel":"email","status":2,"page_size":20}`, 1},
	} {
		result, err := store.ListNotification(ctx, request(tt.payload))
		if err != nil || result.Total != tt.total {
			t.Fatal("channel membership or combined status filter changed")
		}
	}
	if _, err := store.ListNotification(ctx, request(`{"channel":"email_backup"}`)); err == nil {
		t.Fatal("unsupported channel accepted by direct DAO")
	}
}
func TestNotificationChannelQueryBinding(t *testing.T) {
	for _, channel := range []string{"", "email", "feishu", "sms", "webhook", "EMAIL", "email_backup"} {
		ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
		ctx.Request = httptest.NewRequest("GET", "/?page=2&page_size=3&channel="+channel, nil)
		var req model.ListWorkorderNotificationReq
		err := ctx.ShouldBindQuery(&req)
		invalid := channel == "EMAIL" || channel == "email_backup"
		if (err != nil) != invalid || (!invalid && (req.Page != 2 || req.PageSize != 3)) {
			t.Fatal("notification channel enum/pagination binding failed")
		}
	}
}
