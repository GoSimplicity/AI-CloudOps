package cache

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	configdao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/config"
	"gorm.io/gorm"
)

func TestAlertRuleProjectionMemberOnlyLocatorCannotAuthorizeDeletion(t *testing.T) {
	runProjectionMemberOnlyLocatorCannotAuthorizeDeletion(t, newAlertRuleReuseFixture)
}
func runProjectionMemberOnlyLocatorCannotAuthorizeDeletion(t *testing.T, newFixture func(*testing.T) *recordReuseFixture) {
	for _, mode := range []string{"empty targets", "off"} {
		t.Run(mode, func(t *testing.T) {
			f := newFixture(t)
			const ip = "127.0.0.27"
			f.pools[7].PrometheusInstances = nil
			if mode == "off" {
				f.setDesiredEnabled(7, 2)
			}
			f.store.values[f.rawKey(ip)] = "preserve unknown legacy bytes"
			f.store.sets[f.memberKey(7)] = map[string]bool{ip: true}
			before := copyAlertRedis(f.store.values)
			if err := f.cache(7).GenerateMainConfig(context.Background()); err == nil {
				t.Fatal("unproved member acquired cleanup permission")
			}
			if !reflect.DeepEqual(before, f.store.values) || !f.store.sets[f.memberKey(7)][ip] {
				t.Fatal("unknown legacy bytes/member were modified")
			}
			var operations int64
			if f.sql.db.Model(&model.MonitorConfigPoolOperation{}).Where("family=?", f.family).Count(&operations).Error != nil || operations != 0 {
				t.Fatal("unproved member acquired a journal operation")
			}
		})
	}
}
func TestAlertRuleProjectionHistoricalOwnersRequireExactNamespaceProof(t *testing.T) {
	runProjectionHistoricalOwnersRequireExactNamespaceProof(t, newAlertRuleReuseFixture)
}
func runProjectionHistoricalOwnersRequireExactNamespaceProof(t *testing.T, newFixture func(*testing.T) *recordReuseFixture) {
	for _, indexed := range []bool{false, true} {
		for _, foreign := range []bool{false, true} {
			t.Run(fmt.Sprintf("indexed%v/foreign%v", indexed, foreign), func(t *testing.T) {
				f := newFixture(t)
				ctx := context.Background()
				const ip = "127.0.0.27"
				f.setDesiredEnabled(7, 2)
				own := &model.MonitorConfig{Model: model.Model{ID: 91, DeletedAt: 19}, PoolID: 7, ConfigType: f.family, InstanceIP: ip, Name: f.configName(7, ip), ConfigContent: "groups: [] # historical\n", Status: 2}
				if f.sql.db.Session(&gorm.Session{SkipHooks: true}).Unscoped().Create(own).Error != nil {
					t.Fatal("SQL tombstone fixture failed")
				}
				if foreign {
					row := *own
					row.ID = 92
					row.PoolID = 99
					row.Name = "custom historical foreign"
					if f.sql.db.Session(&gorm.Session{SkipHooks: true}).Unscoped().Create(&row).Error != nil {
						t.Fatal("foreign tombstone fixture failed")
					}
				}
				f.store.values[f.rawKey(ip)] = own.ConfigContent
				if indexed {
					f.store.sets[f.memberKey(7)] = map[string]bool{ip: true}
				}
				before := copyAlertRedis(f.store.values)
				err := f.cache(7).GenerateMainConfig(ctx)
				if foreign {
					if err == nil || !reflect.DeepEqual(before, f.store.values) {
						t.Fatal("ambiguous history was guessed into cleanup")
					}
					var ops int64
					if f.sql.db.Model(&model.MonitorConfigPoolOperation{}).Where("family=?", f.family).Count(&ops).Error != nil || ops != 0 {
						t.Fatal("ambiguous history committed cleanup authority")
					}
				} else {
					if err != nil || f.store.values[f.rawKey(ip)] != "" || len(f.store.sets[f.memberKey(7)]) != 0 {
						t.Fatal("exact SQL tombstone was not discovered and retired", err)
					}
					receipt, err := f.sql.journal.LoadLastProjectedProjection(ctx, f.family, 7)
					if err != nil || receipt == nil || len(receipt.Artifacts) != 1 || receipt.Artifacts[0].ConfigType != f.family || receipt.Artifacts[0].Action != model.ConfigProjectionRetire {
						t.Fatal("historical cleanup lacks its actual projected artifact receipt")
					}
				}
				var audit model.MonitorConfig
				if f.sql.db.Unscoped().First(&audit, own.ID).Error != nil || audit.DeletedAt != own.DeletedAt || audit.Name != own.Name {
					t.Fatal("historical audit owner was modified")
				}
			})
		}
	}
}

type alertRuleFixtureCommitHook struct {
	configdao.AlertRuleProjectionJournalDAO
	before func()
}

func (h *alertRuleFixtureCommitHook) CommitAlertRuleProjectionPlan(ctx context.Context, req configdao.ScrapeRuleProjectionPlanRequest) (configdao.ProjectionReceipt, error) {
	if h.before != nil {
		h.before()
	}
	return h.AlertRuleProjectionJournalDAO.CommitAlertRuleProjectionPlan(ctx, req)
}
func TestAlertRuleProjectionCommitRechecksHistoryAfterForeignABA(t *testing.T) {
	runProjectionCommitRechecksHistoryAfterForeignABA(t, newAlertRuleReuseFixture)
}
func runProjectionCommitRechecksHistoryAfterForeignABA(t *testing.T, newFixture func(*testing.T) *recordReuseFixture) {
	for _, state := range []string{"tombstone", "active deleted after preflight"} {
		t.Run(state, func(t *testing.T) {
			f := newFixture(t)
			ctx := context.Background()
			const ip = "127.0.0.27"
			var own model.MonitorConfig
			if state == "tombstone" {
				own = model.MonitorConfig{Model: model.Model{ID: 91, DeletedAt: 19}, PoolID: 7, ConfigType: f.family, InstanceIP: ip, Name: f.configName(7, ip), ConfigContent: "groups: [] # historical\n", Status: 2}
				if f.sql.db.Session(&gorm.Session{SkipHooks: true}).Create(&own).Error != nil {
					t.Fatal("tombstone fixture failed")
				}
				f.store.sets[f.memberKey(7)] = map[string]bool{ip: true}
				f.store.values[f.rawKey(ip)] = own.ConfigContent
			} else {
				if err := f.cache(7).GenerateMainConfig(ctx); err != nil {
					t.Fatal(err)
				}
				rows, err := f.sql.MonitorConfigDAO.GetMonitorConfigsByInstances(ctx, []string{ip}, f.family)
				if err != nil || len(rows) != 1 {
					t.Fatal("current owner fixture missing")
				}
				own = *rows[0]
			}
			beforeSHA, beforeACK := f.store.values[f.legacyKey(7)], f.store.values[projectionPoolKey(f.family, 7, "ack")]
			previous, err := f.sql.journal.LoadLastCommittedProjection(ctx, f.family, 7)
			if err != nil {
				t.Fatal(err)
			}
			f.setDesiredEnabled(7, 2)
			called := false
			before := func() {
				if called {
					t.Fatal("commit hook repeated")
				}
				called = true
				if state != "tombstone" {
					if err := f.sql.MonitorConfigDAO.DeleteMonitorConfig(ctx, own.ID); err != nil {
						t.Fatal(err)
					}
				}
				row := &model.MonitorConfig{PoolID: 99, ConfigType: f.family, InstanceIP: ip, Name: "foreign transient", ConfigContent: "foreign transient body", Status: 2}
				if err := f.sql.MonitorConfigDAO.CreateMonitorConfig(ctx, row); err != nil {
					t.Fatal(err)
				}
				if err := f.sql.MonitorConfigDAO.DeleteMonitorConfig(ctx, row.ID); err != nil {
					t.Fatal(err)
				}
				f.store.values[f.rawKey(ip)] = row.ConfigContent
			}
			a := f.cacheWithCommitHook(7, before)
			err = a.GenerateMainConfig(ctx)
			if !called {
				t.Fatal("actual typed commit interleaving was not reached")
			}
			if err == nil {
				t.Fatal("foreign create/delete ABA preserved stale history permission")
			}
			if f.store.values[f.rawKey(ip)] != "foreign transient body" || f.store.values[f.legacyKey(7)] != beforeSHA || f.store.values[projectionPoolKey(f.family, 7, "ack")] != beforeACK {
				t.Fatal("rejected history guard changed raw/ACK")
			}
			current, err := f.sql.journal.LoadLastCommittedProjection(ctx, f.family, 7)
			if err != nil {
				t.Fatal(err)
			}
			if previous == nil {
				if current != nil {
					t.Fatal("rejected guard issued a new manifest")
				}
			} else if current == nil || current.Reservation != previous.Reservation {
				t.Fatal("rejected guard replaced last-good operation")
			}
			var rows []model.MonitorConfig
			if f.sql.db.Unscoped().Where("config_type=? AND instance_ip=?", f.family, ip).Find(&rows).Error != nil || len(rows) != 2 {
				t.Fatal("ABA audit history was removed")
			}
			for _, row := range rows {
				if row.DeletedAt == 0 {
					t.Fatal("ABA fixture was not vacant")
				}
			}
		})
	}
}
