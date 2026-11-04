//go:build integration

package cache

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	alertdao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/alert"
	configdao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/config"
	scrapedao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/scrape"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type rootNativeScrapeRulePublisher interface {
	GenerateMainConfig(context.Context) error
	GetConfigByIP(string) string
}

type rootAlertNativeRules struct {
	alertdao.AlertManagerRuleDAO
	pool      int
	expr      string
	forbidden bool
}

func (s rootAlertNativeRules) GetMonitorAlertRuleByPoolID(context.Context, int) ([]*model.MonitorAlertRule, int64, error) {
	if s.forbidden {
		return nil, 0, errors.New("off source must not read alert rules")
	}
	return []*model.MonitorAlertRule{{Model: model.Model{ID: 1}, PoolID: s.pool, Name: "native_alert", Enable: 1, Expr: s.expr, ForTime: "0s", Severity: model.AlertRuleSeverityWarning}}, 1, nil
}
func rootNativeScrapeRuleJournal(kind int8, db *gorm.DB) configdao.ProjectionPlanJournalDAO {
	switch kind {
	case 3:
		return configdao.NewAlertRuleProjectionJournalDAO(zap.NewNop(), db)
	case 4:
		return configdao.NewRecordProjectionJournalDAO(zap.NewNop(), db)
	default:
		panic("unsupported native scrape rule scope")
	}
}
func rootNativeScrapeRuleRaw(kind int8, ip string) string {
	switch kind {
	case 3:
		return buildRedisKeyAlertRule(ip)
	case 4:
		return buildRedisKeyRecordRule(ip)
	default:
		panic("unsupported native scrape rule scope")
	}
}
func rootNativeScrapeRuleLegacy(kind int8, pool int) string {
	switch kind {
	case 3:
		return alertRulePublicationKey(pool)
	case 4:
		return recordPublicationKey(pool)
	default:
		panic("unsupported native scrape rule scope")
	}
}
func rootNativeScrapeRuleMembers(kind int8, pool int) string {
	switch kind {
	case 3:
		return buildRedisSetKeyAlertRulePoolIPs(pool)
	case 4:
		return buildRedisSetKeyRecordRulePoolIPs(pool)
	default:
		panic("unsupported native scrape rule scope")
	}
}
func rootNativeScrapeRuleCache(kind int8, pool int, ip string, enabled int8, expr string, metadata configdao.MonitorConfigDAO, db *gorm.DB, client *redis.Client, store *ProjectionStore) rootNativeScrapeRulePublisher {
	logger := zap.NewNop()
	source := model.MonitorScrapePool{Model: model.Model{ID: pool}, Name: "native-pool-" + strconv.Itoa(pool), PrometheusInstances: model.StringList{ip}}
	switch kind {
	case 3:
		source.SupportAlert = enabled
		return NewAlertRuleConfigCache(logger, rootRecordNativePool{pool: source}, rootAlertNativeRules{pool: pool, expr: expr, forbidden: enabled == 2}, metadata, NewBatchConfigManager(metadata, logger), client, configdao.NewAlertRuleProjectionJournalDAO(logger, db), configdao.NewAlertRuleHistoryReader(logger, db), NewAlertRuleProjectionRunner(store))
	case 4:
		source.SupportRecord = enabled
		return NewRecordRuleConfigCache(logger, rootRecordNativePool{pool: source}, rootRecordNativeRules{pool: pool, expr: expr, forbidden: enabled == 2}, metadata, NewBatchConfigManager(metadata, logger), client, configdao.NewRecordProjectionJournalDAO(logger, db), configdao.NewGeneratedConfigHistoryReader(logger, db), NewRecordProjectionRunner(store))
	default:
		panic("unsupported native scrape rule scope")
	}
}

func TestAlertRuleProjectionNativeLatePacketsCannotDamageNewOwner(t *testing.T) {
	rootScrapeRuleNativeLatePackets(t, 3)
}

func rootScrapeRuleNativeLatePackets(t *testing.T, kind int8) {
	if kind != 3 && kind != 4 {
		t.Fatal("unsupported native test scope")
	}
	client := projectionRedisNative(t)
	for index, action := range []string{model.ConfigProjectionRender, model.ConfigProjectionRetire} {
		t.Run(action, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			_, metadata, db := projectionStoreSQL(t)
			journal := rootNativeScrapeRuleJournal(kind, db)
			normal, err := NewProjectionStore(client)
			if err != nil {
				t.Fatal(err)
			}
			defer normal.Close()
			const firstPool = 9091
			const nextPool = 9092
			ip := "127.0.90." + strconv.Itoa(index+1)
			makeRecord := func(pool int, enabled int8, expr string, store *ProjectionStore) rootNativeScrapeRulePublisher {
				return rootNativeScrapeRuleCache(kind, pool, ip, enabled, expr, metadata, db, client, store)
			}
			track := func(receipt configdao.ProjectionReceipt) {
				t.Helper()
				err := journal.WithPoolProjectionLease(ctx, receipt, func(lease configdao.ProjectionLease) error {
					snapshot, err := lease.Snapshot()
					if err != nil {
						return err
					}
					handle, err := compileProjection(normal, lease, snapshot)
					if err != nil {
						return err
					}
					projectionTrack(t, client, handle)
					return nil
				})
				if err != nil {
					t.Fatal("cannot track actual publisher fixture", err)
				}
			}
			initial := makeRecord(firstPool, 1, "sum(up)", normal)
			if err := initial.GenerateMainConfig(ctx); err != nil {
				t.Fatal("initial real publisher failed", err)
			}
			first, err := journal.LoadLastCommittedProjection(ctx, kind, firstPool)
			if err != nil || first == nil {
				t.Fatal("initial durable plan missing", err)
			}
			track(*first)
			firstRaw := initial.GetConfigByIP(ip)
			if firstRaw == "" {
				t.Fatal("initial real publisher did not serve raw bytes")
			}
			queue := rootNewRecordQueue(t, client.Options().Addr)
			options := *client.Options()
			options.Addr = queue.listener.Addr().String()
			options.Protocol = 2
			options.PoolSize = 1
			options.MinIdleConns = 0
			base := redis.NewClient(&options)
			defer base.Close()
			delayedStore, err := NewProjectionStore(base)
			if err != nil {
				t.Fatal(err)
			}
			defer delayedStore.Close()
			enabled := int8(1)
			if action == model.ConfigProjectionRetire {
				enabled = 2
			}
			old := makeRecord(firstPool, enabled, "sum(up) * 2", delayedStore)
			oldDone := make(chan error, 1)
			go func() { oldDone <- old.GenerateMainConfig(ctx) }()
			select {
			case <-queue.seen:
			case <-ctx.Done():
				t.Fatal("actual publisher never sent the held Redis artifact request")
			}
			select {
			case err := <-oldDone:
				if err == nil {
					t.Fatal("unknown Redis result falsely completed old publisher")
				}
			case <-ctx.Done():
				t.Fatal("old SQL lease did not finish after client EOF")
			}
			pending, err := journal.LoadLastCommittedProjection(ctx, kind, firstPool)
			if err != nil || pending == nil || pending.Phase != model.ConfigPublicationCommitted || len(pending.Artifacts) != 1 || pending.Artifacts[0].Action != action {
				t.Fatal("real old publisher did not leave its committed pending plan", err)
			}
			if action == model.ConfigProjectionRender {
				if pending.Artifacts[0].ConfigRowID == nil {
					t.Fatal("pending metadata identity missing")
				}
				if err := metadata.DeleteMonitorConfig(ctx, *pending.Artifacts[0].ConfigRowID); err != nil {
					t.Fatal(err)
				}
			}
			newer := makeRecord(nextPool, 1, "sum(up) * 3", normal)
			if err := newer.GenerateMainConfig(ctx); err != nil {
				t.Fatal("new owner's real publication failed", err)
			}
			good, err := journal.LoadLastProjectedProjection(ctx, kind, nextPool)
			if err != nil || good == nil {
				t.Fatal("new owner's projected SQL receipt missing", err)
			}
			track(*good)
			allKeys := []string{rootNativeScrapeRuleRaw(kind, ip), projectionNamespaceKey(kind, ip)}
			for _, pool := range []int{firstPool, nextPool} {
				allKeys = append(allKeys, projectionPoolKey(kind, pool, "fence"), projectionPoolKey(kind, pool, "ack"), rootNativeScrapeRuleLegacy(kind, pool), rootNativeScrapeRuleMembers(kind, pool))
			}
			before := rootProjectionValues(t, client, allKeys)
			newRaw := newer.GetConfigByIP(ip)
			if newRaw == "" || newRaw == firstRaw {
				t.Fatal("two real publications did not produce distinct bytes")
			}
			queue.unblock()
			select {
			case reply := <-queue.result:
				if reply.err != nil || reply.response != "STALE" {
					t.Fatalf("actual delayed %s packet did not get Redis STALE: %v", action, reply.err)
				}
			case <-ctx.Done():
				t.Fatal("delayed Redis packet was not actually evaluated")
			}
			if !reflect.DeepEqual(rootProjectionValues(t, client, allKeys), before) || newer.GetConfigByIP(ip) != newRaw {
				t.Fatal("delayed old publisher changed new body, members, ACK or fence")
			}
			current, err := journal.LoadLastProjectedProjection(ctx, kind, nextPool)
			if err != nil || current == nil || current.Reservation != good.Reservation {
				t.Fatal("new owner's durable completion changed", err)
			}
			{
				off := old
				if action == model.ConfigProjectionRender {
					if err := old.GenerateMainConfig(ctx); err == nil {
						t.Fatal("active old source reclaimed a foreign address")
					}
					off = makeRecord(firstPool, 2, "must not be read", normal)
				}
				if err := off.GenerateMainConfig(ctx); err != nil {
					t.Fatal("off publisher could not finish bookkeeping after a newer owner completed", err)
				}
				newOwnerKeys := []string{rootNativeScrapeRuleRaw(kind, ip), projectionNamespaceKey(kind, ip), projectionPoolKey(kind, nextPool, "fence"), projectionPoolKey(kind, nextPool, "ack"), rootNativeScrapeRuleLegacy(kind, nextPool), rootNativeScrapeRuleMembers(kind, nextPool)}
				for _, key := range newOwnerKeys {
					if rootProjectionValues(t, client, []string{key})[key] != before[key] {
						t.Fatal("off publisher changed the newer owner's raw bytes or completion proof")
					}
				}
				finished, err := journal.LoadLastProjectedProjection(ctx, kind, firstPool)
				if err != nil || finished == nil || finished.Phase != model.ConfigPublicationProjected || (action == model.ConfigProjectionRetire && finished.SourceHash != pending.SourceHash) || finished.Reservation.Generation <= pending.Reservation.Generation || len(finished.Artifacts) != 0 {
					t.Fatal("off publisher did not issue a fresh durable empty recovery", err)
				}
				if client.SCard(ctx, rootNativeScrapeRuleMembers(kind, firstPool)).Val() != 0 {
					t.Fatal("off publisher left its old member locator behind")
				}
				// A completion is historical evidence. A later legitimate B body
				// revision must not make A forget its already resolved retirement.
				updatedOwner := makeRecord(nextPool, 1, "sum(up) * 4", normal)
				if err := updatedOwner.GenerateMainConfig(ctx); err != nil {
					t.Fatal("new owner could not update after superseding A", err)
				}
				stable := rootProjectionValues(t, client, newOwnerKeys)
				for retry := 0; retry < 2; retry++ {
					rebuilt := makeRecord(firstPool, 2, "must not be read", normal)
					if err := rebuilt.GenerateMainConfig(ctx); err != nil {
						t.Fatal("rebuilt off publisher forgot durable retirement resolution", err)
					}
					if !reflect.DeepEqual(rootProjectionValues(t, client, newOwnerKeys), stable) {
						t.Fatal("rebuilt off publisher changed the current owner's projection")
					}
				}
				newRaw = updatedOwner.GetConfigByIP(ip)
			}
			rows, err := metadata.GetMonitorConfigsByInstances(ctx, []string{ip}, kind)
			if err != nil || len(rows) != 1 || rows[0].PoolID != nextPool || rows[0].ConfigContent != newRaw {
				t.Fatal("delayed command changed the current metadata owner", err)
			}
		})
	}
}

func rootScrapeRuleNativeLegacyOwner(t *testing.T, kind int8) {
	if kind != 3 && kind != 4 {
		t.Fatal("unsupported native test scope")
	}
	client := projectionRedisNative(t)
	_, metadata, db := projectionStoreSQL(t)
	journal := rootNativeScrapeRuleJournal(kind, db)
	store, err := NewProjectionStore(client)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	const oldPool = 9341
	const newPool = 9342
	const ip = "127.0.93.4"
	legacy := &model.MonitorConfig{PoolID: oldPool, ConfigType: kind, InstanceIP: ip, Name: rootNativeScrapeRuleName(kind, oldPool, ip), ConfigContent: "groups: [] # legacy before pool journal\n", Status: model.ConfigStatusActive}
	legacy.ConfigHash = calculateConfigHash(legacy.ConfigContent)
	if err := metadata.CreateMonitorConfig(ctx, legacy); err != nil {
		t.Fatal(err)
	}
	if err := metadata.DeleteMonitorConfig(ctx, legacy.ID); err != nil {
		t.Fatal(err)
	}
	if previous, err := journal.LoadLastCommittedProjection(ctx, kind, oldPool); err != nil || previous != nil {
		t.Fatal("legacy owner unexpectedly has an issued pool plan", err)
	}
	if client.Set(ctx, rootNativeScrapeRuleRaw(kind, ip), legacy.ConfigContent, 0).Err() != nil || client.SAdd(ctx, rootNativeScrapeRuleMembers(kind, oldPool), ip).Err() != nil {
		t.Fatal("could not seed owned legacy raw/member fixture")
	}
	t.Cleanup(func() {
		client.Del(context.Background(), rootNativeScrapeRuleRaw(kind, ip), rootNativeScrapeRuleMembers(kind, oldPool))
	})
	create := func(pool int, enabled int8) rootNativeScrapeRulePublisher {
		return rootNativeScrapeRuleCache(kind, pool, ip, enabled, "sum(up)", metadata, db, client, store)
	}
	track := func(receipt configdao.ProjectionReceipt) {
		t.Helper()
		if err := journal.WithPoolProjectionLease(ctx, receipt, func(lease configdao.ProjectionLease) error {
			snapshot, err := lease.Snapshot()
			if err != nil {
				return err
			}
			handle, err := compileProjection(store, lease, snapshot)
			if err != nil {
				return err
			}
			projectionTrack(t, client, handle)
			return nil
		}); err != nil {
			t.Fatal(err)
		}
	}
	newer := create(newPool, 1)
	if err := newer.GenerateMainConfig(ctx); err != nil {
		t.Fatal("new owner failed to establish a complete replacement", err)
	}
	complete, err := journal.LoadLastProjectedProjection(ctx, kind, newPool)
	if err != nil || complete == nil {
		t.Fatal("replacement completion missing", err)
	}
	track(*complete)
	protected := []string{rootNativeScrapeRuleRaw(kind, ip), projectionNamespaceKey(kind, ip), projectionPoolKey(kind, newPool, "fence"), projectionPoolKey(kind, newPool, "ack"), rootNativeScrapeRuleLegacy(kind, newPool), rootNativeScrapeRuleMembers(kind, newPool)}
	before := rootProjectionValues(t, client, protected)
	for retry := 0; retry < 2; retry++ {
		if err := create(oldPool, 2).GenerateMainConfig(ctx); err != nil {
			t.Fatal("legacy off owner could not finish despite a verified complete replacement", err)
		}
		recovered, err := journal.LoadLastProjectedProjection(ctx, kind, oldPool)
		if err != nil || recovered == nil || len(recovered.Artifacts) != 0 || recovered.Phase != model.ConfigPublicationProjected {
			t.Fatal("legacy owner did not durably confirm its empty state", err)
		}
		track(*recovered)
		if client.SCard(ctx, rootNativeScrapeRuleMembers(kind, oldPool)).Val() != 0 || !reflect.DeepEqual(rootProjectionValues(t, client, protected), before) {
			t.Fatal("legacy bookkeeping touched the replacement or retained obsolete members")
		}
	}
	// B is the current owner. A's already-deleted historical row must not
	// permanently prevent B from retiring its own configuration later.
	if err := create(newPool, 2).GenerateMainConfig(ctx); err != nil {
		t.Fatal("current owner could not retire because a foreign historical tombstone exists", err)
	}
	if client.Exists(ctx, rootNativeScrapeRuleRaw(kind, ip)).Val() != 0 || client.SCard(ctx, rootNativeScrapeRuleMembers(kind, newPool)).Val() != 0 {
		t.Fatal("current owner did not clear its own raw and member set")
	}
	if err := create(oldPool, 2).GenerateMainConfig(ctx); err != nil {
		t.Fatal("legacy resolved owner forgot its disposition after B retired", err)
	}
}

func TestAlertRuleProjectionNativeLegacyOwnerCanFinishAfterReplacement(t *testing.T) {
	rootScrapeRuleNativeLegacyOwner(t, 3)
}

func rootScrapeRuleNativeSameHash(t *testing.T, kind int8) {
	if kind != 3 && kind != 4 {
		t.Fatal("unsupported native test scope")
	}
	client := projectionRedisNative(t)
	_, metadata, db := projectionStoreSQL(t)
	journal := rootNativeScrapeRuleJournal(kind, db)
	store, err := NewProjectionStore(client)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	const poolID = 9551
	const desiredIP = "127.0.95.1"
	const obsoleteIP = "127.0.95.2"
	publisher := rootNativeScrapeRuleCache(kind, poolID, desiredIP, 1, "sum(up)", metadata, db, client, store)
	if err := publisher.GenerateMainConfig(ctx); err != nil {
		t.Fatal(err)
	}
	first, err := journal.LoadLastProjectedProjection(ctx, kind, poolID)
	if err != nil || first == nil {
		t.Fatal(err)
	}
	if err := journal.WithPoolProjectionLease(ctx, *first, func(lease configdao.ProjectionLease) error {
		snapshot, err := lease.Snapshot()
		if err != nil {
			return err
		}
		handle, err := compileProjection(store, lease, snapshot)
		if err != nil {
			return err
		}
		projectionTrack(t, client, handle)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	original := publisher.GetConfigByIP(desiredIP)
	if original == "" {
		t.Fatal("initial desired raw content missing")
	}
	obsolete := &model.MonitorConfig{PoolID: poolID, ConfigType: kind, InstanceIP: obsoleteIP, Name: rootNativeScrapeRuleName(kind, poolID, obsoleteIP), ConfigContent: "groups: [] # stale database-only owner\n", Status: model.ConfigStatusActive}
	obsolete.ConfigHash = calculateConfigHash(obsolete.ConfigContent)
	if err := metadata.CreateMonitorConfig(ctx, obsolete); err != nil {
		t.Fatal(err)
	}
	if client.Set(ctx, rootNativeScrapeRuleRaw(kind, obsoleteIP), obsolete.ConfigContent, 0).Err() != nil {
		t.Fatal("could not seed unindexed stale raw")
	}
	t.Cleanup(func() {
		client.Del(context.Background(), rootNativeScrapeRuleRaw(kind, obsoleteIP), projectionNamespaceKey(kind, obsoleteIP))
	})
	members, memberErr := client.SMembers(ctx, rootNativeScrapeRuleMembers(kind, poolID)).Result()
	if memberErr != nil {
		t.Fatal("cannot inspect the native member precondition", memberErr)
	}
	for _, ip := range members {
		if ip == obsoleteIP {
			t.Fatal("stale row must not be a pre-existing indexed target")
		}
	}
	if err := publisher.GenerateMainConfig(ctx); err != nil {
		t.Fatal("same-source reconciliation failed", err)
	}
	row, err := metadata.GetGeneratedMonitorConfigByID(ctx, obsolete.ID)
	if err != nil || row == nil || row.DeletedAt == 0 {
		t.Fatal("healthy old receipt hid a newly observed database-only stale owner", err)
	}
	if client.Exists(ctx, rootNativeScrapeRuleRaw(kind, obsoleteIP)).Val() != 0 || publisher.GetConfigByIP(desiredIP) != original {
		t.Fatal("same-source retirement did not preserve desired and clear obsolete raw")
	}
	current, err := journal.LoadLastProjectedProjection(ctx, kind, poolID)
	if err != nil || current == nil || current.SourceHash != first.SourceHash || current.Reservation.Generation <= first.Reservation.Generation {
		t.Fatal("same-source metadata change did not get a fresh durable plan", err)
	}
}

func TestAlertRuleProjectionNativeSameHashFindsNewDatabaseOnlyRetirement(t *testing.T) {
	rootScrapeRuleNativeSameHash(t, 3)
}

func rootNativeScrapeRuleName(kind int8, pool int, ip string) string {
	switch kind {
	case 3:
		return fmt.Sprintf(ConfigNameAlertRule, pool, ip)
	case 4:
		return fmt.Sprintf(ConfigNameRecordRule, pool, ip)
	default:
		panic("unsupported native test scope")
	}
}

type rootAlertRuleSnapshotHook struct {
	alertdao.AlertManagerRuleDAO
	afterSnapshot func()
	once          sync.Once
}

func (d *rootAlertRuleSnapshotHook) GetMonitorAlertRuleByPoolID(ctx context.Context, pool int) ([]*model.MonitorAlertRule, int64, error) {
	rows, total, err := d.AlertManagerRuleDAO.GetMonitorAlertRuleByPoolID(ctx, pool)
	if err == nil {
		d.once.Do(d.afterSnapshot)
	}
	return rows, total, err
}
func TestAlertRuleProjectionNativeOlderSourceCannotReserveAfterNewerPublication(t *testing.T) {
	rootScrapeRuleNativeOlderSource(t, 3)
}
func rootScrapeRuleNativeOlderSource(t *testing.T, kind int8) {
	if kind != 3 && kind != 4 {
		t.Fatal("unsupported native scope")
	}
	client := projectionRedisNative(t)
	for index, scope := range []string{"pool snapshot", "rule snapshot"} {
		t.Run(scope, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			_, metadata, db := projectionStoreSQL(t)
			if err := db.AutoMigrate(&model.MonitorScrapePool{}, &model.MonitorRecordRule{}, &model.MonitorAlertRule{}); err != nil {
				t.Fatal(err)
			}
			logger := zap.NewNop()
			pools := scrapedao.NewScrapePoolDAO(db, logger, nil)
			rules := alertdao.NewAlertManagerRecordDAO(db, logger, nil)
			alerts := alertdao.NewAlertManagerRuleDAO(db, logger, nil)
			pool := &model.MonitorScrapePool{Model: model.Model{ID: 9461 + index}, Name: "before-source-change", SupportRecord: 1, SupportAlert: 2, PrometheusInstances: model.StringList{"127.0.94." + strconv.Itoa(index+1)}}
			if kind == 3 {
				pool.SupportAlert, pool.SupportRecord = 1, 2
			}
			if err := pools.CreateMonitorScrapePool(ctx, pool); err != nil {
				t.Fatal(err)
			}
			rule := &model.MonitorRecordRule{PoolID: pool.ID, Name: "native_source_total", Expr: "sum(up)", Enable: 1}
			alert := &model.MonitorAlertRule{PoolID: pool.ID, Name: "native_source_alert", Expr: "sum(up)", Enable: 1, ForTime: "0s", Severity: model.AlertRuleSeverityWarning}
			if kind == 4 {
				if err := rules.CreateMonitorRecordRule(ctx, rule); err != nil {
					t.Fatal(err)
				}
			} else {
				if err := alerts.CreateMonitorAlertRule(ctx, alert); err != nil {
					t.Fatal(err)
				}
			}
			journal := rootNativeScrapeRuleJournal(kind, db)
			store, err := NewProjectionStore(client)
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			create := func(sourcePools scrapedao.ScrapePoolDAO, sourceRecords alertdao.AlertManagerRecordDAO, sourceAlerts alertdao.AlertManagerRuleDAO) rootNativeScrapeRulePublisher {
				if kind == 4 {
					return NewRecordRuleConfigCache(logger, sourcePools, sourceRecords, metadata, NewBatchConfigManager(metadata, logger), client, journal, configdao.NewGeneratedConfigHistoryReader(logger, db), NewRecordProjectionRunner(store))
				}
				return NewAlertRuleConfigCache(logger, sourcePools, sourceAlerts, metadata, NewBatchConfigManager(metadata, logger), client, journal, configdao.NewAlertRuleHistoryReader(logger, db), NewAlertRuleProjectionRunner(store))
			}
			newer := create(pools, rules, alerts)
			var completed *configdao.ProjectionReceipt
			var protected []string
			var baseline map[string]rootProjectionValue
			called := false
			publishNewSource := func() {
				called = true
				if scope == "pool snapshot" {
					req := &model.UpdateMonitorScrapePoolReq{ID: pool.ID, Name: "after-source-change", SupportRecord: 1, SupportAlert: 2, PrometheusInstances: append(model.StringList(nil), pool.PrometheusInstances...), ScrapeInterval: 30, ScrapeTimeout: 10, RemoteTimeoutSeconds: 5}
					if kind == 3 {
						req.SupportAlert, req.SupportRecord = 1, 2
					}
					if err := pools.UpdateMonitorScrapePool(ctx, req); err != nil {
						t.Fatal("real source pool update failed", err)
					}
				} else if kind == 4 {
					changed := *rule
					changed.Expr = "sum(up) * 3"
					if err := rules.UpdateMonitorRecordRule(ctx, &changed); err != nil {
						t.Fatal("real source rule update failed", err)
					}
				} else {
					req := &model.UpdateMonitorAlertRuleReq{ID: alert.ID, PoolID: pool.ID, Name: alert.Name, Expr: "sum(up) * 3", Enable: 1, ForTime: alert.ForTime, Severity: alert.Severity}
					if err := alerts.UpdateMonitorAlertRule(ctx, req); err != nil {
						t.Fatal("real source alert rule update failed", err)
					}
				}
				if err := newer.GenerateMainConfig(ctx); err != nil {
					t.Fatal("newer source publication failed", err)
				}
				var err error
				completed, err = journal.LoadLastProjectedProjection(ctx, kind, pool.ID)
				if err != nil || completed == nil {
					t.Fatal("new source was not fully acknowledged", err)
				}
				if err := journal.WithPoolProjectionLease(ctx, *completed, func(lease configdao.ProjectionLease) error {
					snapshot, err := lease.Snapshot()
					if err != nil {
						return err
					}
					handle, err := compileProjection(store, lease, snapshot)
					if err != nil {
						return err
					}
					projectionTrack(t, client, handle)
					protected = append([]string(nil), handle.keys...)
					baseline = rootProjectionValues(t, client, protected)
					return nil
				}); err != nil {
					t.Fatal(err)
				}
			}
			var older rootNativeScrapeRulePublisher
			if scope == "pool snapshot" {
				older = create(&rootRecordPoolSnapshotHook{ScrapePoolDAO: pools, afterSnapshot: publishNewSource}, rules, alerts)
			} else if kind == 4 {
				older = create(pools, &rootRecordRuleSnapshotHook{AlertManagerRecordDAO: rules, afterSnapshot: publishNewSource}, alerts)
			} else {
				older = create(pools, rules, &rootAlertRuleSnapshotHook{AlertManagerRuleDAO: alerts, afterSnapshot: publishNewSource})
			}
			err = older.GenerateMainConfig(ctx)
			if !called {
				t.Fatal("real source read barrier was not reached")
			}
			// A stale task may fail or coalesce into a safe no-op. It must never
			// replace the accepted source merely by reserving after a slow read.
			current, loadErr := journal.LoadLastProjectedProjection(ctx, kind, pool.ID)
			if loadErr != nil || current == nil {
				t.Fatal("cannot inspect publication after the delayed source read", loadErr)
			}
			if current.Reservation != completed.Reservation || current.SourceHash != completed.SourceHash {
				t.Fatalf("older source replaced generation %d with generation %d after newer completion (call failed=%v)", completed.Reservation.Generation, current.Reservation.Generation, err != nil)
			}
			accepted, loadErr := journal.LoadLastCommittedProjection(ctx, kind, pool.ID)
			if loadErr != nil || accepted == nil || accepted.Reservation != completed.Reservation {
				t.Fatal("older source changed the accepted metadata plan", loadErr)
			}
			if !reflect.DeepEqual(rootProjectionValues(t, client, protected), baseline) {
				t.Fatal("older source changed new raw, membership, SHA, ACK or fence")
			}
		})
	}
}

func TestScrapeRuleProjectionNativeFamiliesShareDatabaseWithoutCrossWrites(t *testing.T) {
	client := projectionRedisNative(t)
	_, metadata, db := projectionStoreSQL(t)
	store, err := NewProjectionStore(client)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	const pool = 9661
	const ip = "127.0.96.6"
	keys := func(kind int8) []string {
		return []string{rootNativeScrapeRuleRaw(kind, ip), projectionNamespaceKey(kind, ip), projectionPoolKey(kind, pool, "fence"), projectionPoolKey(kind, pool, "ack"), rootNativeScrapeRuleLegacy(kind, pool), rootNativeScrapeRuleMembers(kind, pool)}
	}
	t.Cleanup(func() {
		for _, kind := range []int8{3, 4} {
			client.Del(context.Background(), keys(kind)...)
		}
	})
	record := rootNativeScrapeRuleCache(4, pool, ip, 1, "sum(up)", metadata, db, client, store)
	alert := rootNativeScrapeRuleCache(3, pool, ip, 1, "sum(up)", metadata, db, client, store)
	if err := record.GenerateMainConfig(ctx); err != nil {
		t.Fatal("initial Record on shared SQL failed", err)
	}
	recordState := rootProjectionValues(t, client, keys(4))
	if err := alert.GenerateMainConfig(ctx); err != nil {
		t.Fatal("Alert on shared SQL failed", err)
	}
	if !reflect.DeepEqual(rootProjectionValues(t, client, keys(4)), recordState) {
		t.Fatal("Alert modified Record's data or fences")
	}
	alertOff := rootNativeScrapeRuleCache(3, pool, ip, 2, "must not be read", metadata, db, client, store)
	if err := alertOff.GenerateMainConfig(ctx); err != nil {
		t.Fatal("Alert off on shared SQL failed", err)
	}
	if client.Exists(ctx, rootNativeScrapeRuleRaw(3, ip)).Val() != 0 || record.GetConfigByIP(ip) == "" || !reflect.DeepEqual(rootProjectionValues(t, client, keys(4)), recordState) {
		t.Fatal("Alert off crossed family boundaries")
	}
	records, err := metadata.GetMonitorConfigsByInstances(ctx, []string{ip}, 4)
	if err != nil || len(records) != 1 {
		t.Fatal("Alert retired Record's SQL owner", err)
	}
	alertState := rootProjectionValues(t, client, keys(3))
	recordOff := rootNativeScrapeRuleCache(4, pool, ip, 2, "must not be read", metadata, db, client, store)
	if err := recordOff.GenerateMainConfig(ctx); err != nil {
		t.Fatal("Record off on shared SQL failed", err)
	}
	if !reflect.DeepEqual(rootProjectionValues(t, client, keys(3)), alertState) || client.Exists(ctx, rootNativeScrapeRuleRaw(4, ip)).Val() != 0 {
		t.Fatal("Record off crossed family boundaries")
	}
	for _, kind := range []int8{3, 4} {
		receipt, err := rootNativeScrapeRuleJournal(kind, db).LoadLastProjectedProjection(ctx, kind, pool)
		if err != nil || receipt == nil || receipt.Reservation.Family != kind {
			t.Fatal("separate family receipt lost", err)
		}
	}
}
