package cache_test

import (
	"context"
	"strings"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	configdao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/config"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Schedule a real ordinary DAO insert after metadata preflight but before the
// generator's create. A successful publication must not claim that occupied key.
func TestGeneratedCreationRejectsOwnerArrivingAfterPreflight(t *testing.T) {
	runGeneratedCreationConflict(t, model.ConfigTypeRecordRule)
}
func TestAlertRuleCreationRejectsOwnerArrivingAfterPreflight(t *testing.T) {
	runGeneratedCreationConflict(t, model.ConfigTypeAlertRule)
}
func runGeneratedCreationConflict(t *testing.T, family int8) {
	for _, status := range []int8{model.ConfigStatusActive, model.ConfigStatusInactive} {
		t.Run(map[int8]string{model.ConfigStatusActive: "active", model.ConfigStatusInactive: "inactive"}[status], func(t *testing.T) {
			db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
			if err != nil {
				t.Fatal(err)
			}
			sqlDB, err := db.DB()
			if err != nil {
				t.Fatal(err)
			}
			sqlDB.SetMaxOpenConns(1)
			t.Cleanup(func() { _ = sqlDB.Close() })
			if err := db.AutoMigrate(&model.MonitorConfig{}, &model.MonitorConfigNamespaceGuard{}, &model.MonitorConfigPoolControl{}, &model.MonitorConfigPoolOperation{}, &model.MonitorConfigArtifactEvent{}); err != nil {
				t.Fatal(err)
			}
			dao := configdao.NewMonitorConfigDAO(zap.NewNop(), db)
			ctx := context.Background()
			pool := &model.MonitorScrapePool{Model: model.Model{ID: 7}, Name: "creation-race", SupportRecord: 1, PrometheusInstances: model.StringList{"127.0.0.1"}}
			store := &ownerHTTPRedis{values: map[string]string{}, sets: map[string]map[string]bool{}}
			publisher, _ := ownerHTTPPublisherForFamily(t, family, db, dao, pool, store)
			foreign := &model.MonitorConfig{PoolID: 99, InstanceIP: "127.0.0.1", ConfigType: family, Name: "operator-owned", ConfigContent: "preserve operator bytes", Status: status}
			injected := false
			store.pendingHook = func() {
				if injected {
					return
				}
				injected = true
				if err := dao.CreateMonitorConfig(ctx, foreign); err != nil {
					t.Fatal("ordinary writer injection failed", err)
				}
			}
			err = publisher.GenerateMainConfig(ctx)
			if !injected {
				t.Fatal("interleaving did not reach post-preflight boundary")
			}
			if err == nil {
				t.Fatal("generator acknowledged a namespace occupied after preflight")
			}
			rows, err := dao.GetMonitorConfigsByInstances(ctx, []string{"127.0.0.1"}, family)
			if err != nil || len(rows) != 1 || rows[0].ID != foreign.ID || rows[0].ConfigContent != foreign.ConfigContent || rows[0].Status != status {
				t.Fatal("generated create changed or duplicated the new owner")
			}
			for key, value := range store.values {
				if !strings.Contains(key, ":publication:") || !strings.HasPrefix(value, "pending:") {
					t.Fatal("failed create published an artifact or success acknowledgment")
				}
			}
		})
	}
}
