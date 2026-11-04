package cache

import (
	"context"
	"fmt"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	configdao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/config"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Exercise the publisher boundary with the real DAO: filtering to owned/active
// rows would incorrectly authorize these colliding desired or retired keys.
func TestGeneratedPublicationNamespaceConflictsWithRealSQLite(t *testing.T) {
	for _, kind := range []int8{model.ConfigTypePrometheus, model.ConfigTypeRecordRule} {
		for _, target := range []string{"desired", "stale"} {
			for _, mode := range []string{"foreign", "custom", "inactive foreign", "duplicate"} {
				t.Run(fmt.Sprintf("kind%d/%s/%s", kind, target, mode), func(t *testing.T) {
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
					if err := db.AutoMigrate(&model.MonitorConfig{}, &model.MonitorConfigNamespaceGuard{}); err != nil {
						t.Fatal(err)
					}
					dao := configdao.NewMonitorConfigDAO(zap.NewNop(), db)
					const ip = "127.0.0.9"
					pattern := ConfigNamePrometheus
					if kind == model.ConfigTypeRecordRule {
						pattern = ConfigNameRecordRule
					}
					row := model.MonitorConfig{PoolID: 7, ConfigType: kind, InstanceIP: ip, Name: fmt.Sprintf(pattern, 7, ip), Status: model.ConfigStatusActive, ConfigContent: "preserved"}
					switch mode {
					case "foreign", "inactive foreign":
						row.PoolID = 99
					case "custom":
						row.Name = "operator-custom"
					}
					if mode == "inactive foreign" {
						row.Status = model.ConfigStatusInactive
					}
					if err := db.Create(&row).Error; err != nil {
						t.Fatal(err)
					}
					if mode == "duplicate" {
						duplicate := row
						duplicate.ID = 0
						if err := db.Create(&duplicate).Error; err != nil {
							t.Fatal(err)
						}
					}
					desired := map[string]string{}
					var old []string
					if target == "desired" {
						desired[ip] = "replacement"
					} else {
						old = []string{ip}
					}
					if kind == model.ConfigTypePrometheus {
						cache := &prometheusConfigCache{configDAO: dao}
						if _, err := cache.planPromPublication(context.Background(), 7, desired, old); err == nil {
							t.Fatal("ambiguous namespace authorized publication")
						}
					} else {
						cache := &recordRuleConfigCache{configDAO: dao}
						if _, err := cache.planRecordPublication(context.Background(), 7, desired, old); err == nil {
							t.Fatal("ambiguous namespace authorized publication")
						}
					}
					var saved model.MonitorConfig
					if err := db.First(&saved, row.ID).Error; err != nil {
						t.Fatal(err)
					}
					if saved.ConfigContent != "preserved" || saved.Name != row.Name || saved.PoolID != row.PoolID || saved.Status != row.Status {
						t.Fatal("preflight changed conflicting metadata")
					}
				})
			}
		}
	}
}

// A canonical orphan has no Redis membership. It must still be discovered, and
// every row sharing its address must be considered before authorizing retirement.
func TestGeneratedPublicationUnindexedOrphanNamespaceWithRealSQLite(t *testing.T) {
	for _, kind := range []int8{model.ConfigTypePrometheus, model.ConfigTypeRecordRule} {
		for _, mode := range []string{"owned only", "foreign collision", "custom collision", "inactive collision"} {
			t.Run(fmt.Sprintf("kind%d/%s", kind, mode), func(t *testing.T) {
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
				if err := db.AutoMigrate(&model.MonitorConfig{}, &model.MonitorConfigNamespaceGuard{}); err != nil {
					t.Fatal(err)
				}
				dao := configdao.NewMonitorConfigDAO(zap.NewNop(), db)
				const ip = "127.0.0.9"
				pattern := ConfigNamePrometheus
				if kind == model.ConfigTypeRecordRule {
					pattern = ConfigNameRecordRule
				}
				row := model.MonitorConfig{PoolID: 7, ConfigType: kind, InstanceIP: ip, Name: fmt.Sprintf(pattern, 7, ip), Status: model.ConfigStatusActive, ConfigContent: "preserved orphan"}
				if err := db.Create(&row).Error; err != nil {
					t.Fatal(err)
				}
				if mode != "owned only" {
					collision := model.MonitorConfig{PoolID: 99, ConfigType: kind, InstanceIP: ip, Name: "foreign", Status: model.ConfigStatusActive}
					if mode == "custom collision" {
						collision.PoolID = 7
						collision.Name = "operator-custom"
					}
					if mode == "inactive collision" {
						collision.Status = model.ConfigStatusInactive
					}
					if err := db.Create(&collision).Error; err != nil {
						t.Fatal(err)
					}
				}
				var stale []string
				var retired map[string]configdao.GeneratedConfigOwner
				if kind == model.ConfigTypePrometheus {
					cache := &prometheusConfigCache{configDAO: dao}
					plan, planErr := cache.planPromPublication(context.Background(), 7, nil, nil)
					stale, retired, err = plan.stale, plan.retired, planErr
				} else {
					cache := &recordRuleConfigCache{configDAO: dao}
					plan, planErr := cache.planRecordPublication(context.Background(), 7, nil, nil)
					stale, retired, err = plan.stale, plan.retired, planErr
				}
				if mode == "owned only" {
					if err != nil || len(stale) != 1 || stale[0] != ip || retired[ip].ID != row.ID {
						t.Fatal("unindexed canonical orphan was not scheduled with its exact owner")
					}
				} else if err == nil {
					t.Fatal("orphan collision authorized retirement")
				}
				var saved model.MonitorConfig
				if err := db.First(&saved, row.ID).Error; err != nil {
					t.Fatal(err)
				}
				if saved.ConfigContent != row.ConfigContent || saved.DeletedAt != 0 {
					t.Fatal("planning changed orphan metadata")
				}
			})
		}
	}
}

type countedPublicationQueryDAO struct {
	configdao.MonitorConfigDAO
	poolCalls, namespaceCalls, returnedRows, fullListCalls int
}

func (d *countedPublicationQueryDAO) GetGeneratedConfigPoolCandidates(ctx context.Context, req configdao.GeneratedConfigPoolQuery) ([]*model.MonitorConfig, int64, error) {
	rows, total, err := d.MonitorConfigDAO.GetGeneratedConfigPoolCandidates(ctx, req)
	d.poolCalls++
	d.returnedRows += len(rows)
	return rows, total, err
}
func (d *countedPublicationQueryDAO) GetGeneratedConfigNamespacePage(ctx context.Context, req configdao.GeneratedConfigKeysQuery) ([]*model.MonitorConfig, int64, error) {
	rows, total, err := d.MonitorConfigDAO.GetGeneratedConfigNamespacePage(ctx, req)
	d.namespaceCalls++
	d.returnedRows += len(rows)
	return rows, total, err
}
func (d *countedPublicationQueryDAO) GetMonitorConfigList(ctx context.Context, req *model.GetMonitorConfigListReq) ([]*model.MonitorConfig, int64, error) {
	d.fullListCalls++
	return d.MonitorConfigDAO.GetMonitorConfigList(ctx, req)
}

// Count actual DAO results rather than assuming that a pool filter is applied.
// This measures transferred metadata, not physical DB scan cost or wall time.
func TestGeneratedPublicationReturnedRowsScaleWithRealSQLite(t *testing.T) {
	for _, kind := range []int8{model.ConfigTypePrometheus, model.ConfigTypeRecordRule} {
		for _, pools := range []int{5, 10} {
			t.Run(fmt.Sprintf("kind%d/pools%d", kind, pools), func(t *testing.T) {
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
				if err := db.AutoMigrate(&model.MonitorConfig{}, &model.MonitorConfigNamespaceGuard{}); err != nil {
					t.Fatal(err)
				}
				dao := &countedPublicationQueryDAO{MonitorConfigDAO: configdao.NewMonitorConfigDAO(zap.NewNop(), db)}
				pattern := ConfigNamePrometheus
				if kind == model.ConfigTypeRecordRule {
					pattern = ConfigNameRecordRule
				}
				for poolID := 1; poolID <= pools; poolID++ {
					ip := fmt.Sprintf("127.0.0.%d", poolID)
					row := model.MonitorConfig{PoolID: poolID, ConfigType: kind, InstanceIP: ip, Name: fmt.Sprintf(pattern, poolID, ip), Status: model.ConfigStatusActive, ConfigContent: "owned", ConfigHash: calculateConfigHash("owned")}
					if err := db.Create(&row).Error; err != nil {
						t.Fatal(err)
					}
				}
				for poolID := 1; poolID <= pools; poolID++ {
					ip := fmt.Sprintf("127.0.0.%d", poolID)
					if kind == model.ConfigTypePrometheus {
						cache := &prometheusConfigCache{configDAO: dao}
						plan, err := cache.planPromPublication(context.Background(), poolID, map[string]string{ip: "owned"}, []string{ip})
						if err != nil || len(plan.create)+len(plan.update)+len(plan.stale) != 0 {
							t.Fatal("stable Prom metadata did not produce an empty plan", err)
						}
					} else {
						cache := &recordRuleConfigCache{configDAO: dao}
						plan, err := cache.planRecordPublication(context.Background(), poolID, map[string]string{ip: "owned"}, []string{ip})
						if err != nil || len(plan.create)+len(plan.update)+len(plan.stale) != 0 {
							t.Fatal("stable Record metadata did not produce an empty plan", err)
						}
					}
				}
				if dao.poolCalls != pools || dao.namespaceCalls != pools || dao.returnedRows != 2*pools || dao.fullListCalls != 0 {
					t.Fatalf("unexpected query volume: pool=%d namespace=%d rows=%d full-list=%d", dao.poolCalls, dao.namespaceCalls, dao.returnedRows, dao.fullListCalls)
				}
				t.Logf("%d pools: %d metadata calls, %d returned rows, zero full-list reads", pools, dao.poolCalls+dao.namespaceCalls, dao.returnedRows)
			})
		}
	}
}
