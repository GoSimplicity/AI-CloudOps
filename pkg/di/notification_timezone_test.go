package di

import (
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"testing"
	"time"
)

func TestNotificationTimeZoneRequiresExplicitFixedZone(t *testing.T) {
	for _, tc := range []struct {
		dsn, want string
		valid     bool
	}{
		{"u:p@tcp(localhost)/db?parseTime=true&loc=UTC", "UTC", true},
		{"u:p@tcp(localhost)/db?loc=Asia%2FShanghai", "Asia/Shanghai", true},
		{"u:p@tcp(localhost)/db?loc=Local", "", false},
		{"u:p@tcp(localhost)/db", "", false},
		{"u:p@tcp(localhost)/db?loc=UTC&loc=Asia%2FShanghai", "", false},
		{"u:p@tcp(localhost)/db?loc=Invalid%2FZone", "", false},
	} {
		zone, err := notificationTimeZone(tc.dsn)
		if (err == nil) != tc.valid || zone != tc.want {
			t.Fatalf("zone=%q valid=%v error=%v", zone, tc.valid, err)
		}
	}
}

func TestNotificationTimeZoneRejectsLegacyScheduledConfiguration(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:legacy-schedule?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	pool, _ := db.DB()
	defer pool.Close()
	if err := db.AutoMigrate(&model.WorkorderNotification{}); err != nil {
		t.Fatal(err)
	}
	scheduled := time.Now().Add(time.Hour)
	row := model.WorkorderNotification{Name: "fixture", TriggerType: "scheduled", ScheduledTime: &scheduled, EventTypes: model.StringList{"created"}, Channels: model.StringList{"email"}, RecipientTypes: model.StringList{"creator"}, MessageTemplate: "fixture"}
	if err := db.Create(&row).Error; err != nil {
		t.Fatal(err)
	}
	if err := CheckNotificationTimeZone(db, "UTC"); err == nil {
		t.Fatal("legacy scheduled time was silently accepted")
	}
}

func TestNotificationTimeZoneDatabaseGuard(t *testing.T) {
	for _, tc := range []struct {
		name             string
		legacy           bool
		saved, requested string
		want             bool
	}{
		{"fresh UTC", false, "", "UTC", true},
		{"same zone", false, "Asia/Shanghai", "Asia/Shanghai", true},
		{"different zone", false, "UTC", "Asia/Shanghai", false},
		{"legacy without confirmation", true, "", "UTC", false},
		{"legacy explicitly confirmed", true, "Asia/Shanghai", "Asia/Shanghai", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
			if err != nil {
				t.Fatal(err)
			}
			pool, _ := db.DB()
			defer pool.Close()
			if tc.legacy {
				if err := db.AutoMigrate(&model.WorkorderNotificationEvent{}); err != nil {
					t.Fatal(err)
				}
				if err := db.Create(&model.WorkorderNotificationEvent{InstanceID: 1, EventType: "fixture", Status: "pending", Snapshot: []byte("{}")}).Error; err != nil {
					t.Fatal(err)
				}
			}
			if tc.saved != "" {
				if err := db.AutoMigrate(&model.WorkorderNotificationClock{}); err != nil {
					t.Fatal(err)
				}
				if err := db.Create(&model.WorkorderNotificationClock{ID: 1, TimeZone: tc.saved}).Error; err != nil {
					t.Fatal(err)
				}
			}
			err = CheckNotificationTimeZone(db, tc.requested)
			if (err == nil) != tc.want {
				t.Fatalf("guard error=%v", err)
			}
			if !tc.want && tc.saved == "" {
				var count int64
				db.Model(&model.WorkorderNotificationClock{}).Count(&count)
				if count != 0 {
					t.Fatal("unverified history was marked")
				}
			}
		})
	}
}
