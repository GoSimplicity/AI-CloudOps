package model

import (
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestNamespaceGuardModelKeepsExactByteCompositeIdentity(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatal(err)
	}
	conn, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	conn.SetMaxOpenConns(1)
	defer conn.Close()
	if db.AutoMigrate(&MonitorConfigNamespaceGuard{}) != nil {
		t.Fatal("guard model migration failed")
	}
	keys := []string{"2001:db8::1", "2001:DB8::1", "2001:db8::1 ", " 2001:db8::1", strings.Repeat("🌍", 45)}
	for _, key := range keys {
		if db.Create(&MonitorConfigNamespaceGuard{ConfigType: 1, InstanceKey: []byte(key)}).Error != nil {
			t.Fatal("byte identity or full UTF-8 key was rejected")
		}
	}
	if db.Create(&MonitorConfigNamespaceGuard{ConfigType: 1, InstanceKey: []byte(keys[0])}).Error == nil {
		t.Fatal("same kind/key duplicate was accepted")
	}
	if db.Create(&MonitorConfigNamespaceGuard{ConfigType: 2, InstanceKey: []byte(keys[0])}).Error != nil {
		t.Fatal("cross-type key was merged")
	}
	var rows []MonitorConfigNamespaceGuard
	if db.Find(&rows).Error != nil || len(rows) != 6 {
		t.Fatal("unexpected guard inventory")
	}
	for _, column := range []string{"id", "deleted_at", "config_content"} {
		if db.Migrator().HasColumn(&MonitorConfigNamespaceGuard{}, column) {
			t.Fatal("guard model acquired unrelated fields")
		}
	}
}
