package cache

import (
	"context"
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	configdao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/config"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func projectionStoreSQL(t *testing.T) (configdao.ProjectionPlanJournalDAO, configdao.MonitorConfigDAO, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "store.sqlite")), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatal(err)
	}
	pool, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	pool.SetMaxOpenConns(1)
	t.Cleanup(func() { pool.Close() })
	if err := db.AutoMigrate(&model.MonitorConfig{}, &model.MonitorConfigNamespaceGuard{}, &model.MonitorConfigPoolControl{}, &model.MonitorConfigPoolOperation{}, &model.MonitorConfigArtifactEvent{}); err != nil {
		t.Fatal(err)
	}
	return configdao.NewProjectionPlanJournalDAO(zap.NewNop(), db), configdao.NewMonitorConfigDAO(zap.NewNop(), db), db
}
func projectionRecordItem(pool int, source uint64, ip, body string) configdao.GeneratedPlanItem {
	return configdao.GeneratedPlanItem{SourceID: source, Action: model.ConfigProjectionRender, Desired: &model.MonitorConfig{PoolID: pool, ConfigType: 4, InstanceIP: ip, Name: fmt.Sprintf(ConfigNameRecordRule, pool, ip), ConfigContent: body, Status: 1}, Binding: &configdao.ProjectionBinding{Version: 1, Role: "artifact", ConfigType: 4, InstanceIP: ip}}
}
func projectionFullRequest(ref configdao.PoolReservation, items ...configdao.GeneratedPlanItem) configdao.GeneratedProjectionPlanRequest {
	hash := sha256.Sum256([]byte(fmt.Sprint(ref.Nonce)))
	sources := []uint64{}
	ips := []string{}
	for _, item := range items {
		sources = append(sources, item.SourceID)
		if item.Action == model.ConfigProjectionRender && item.Desired.ConfigType == ref.Family {
			ips = append(ips, item.Binding.InstanceIP)
		}
	}
	return configdao.GeneratedProjectionPlanRequest{GeneratedPlanRequest: configdao.GeneratedPlanRequest{Reservation: ref, SourceHash: hash, Items: items}, Controls: []configdao.ProjectionControl{{ControlID: 1, Kind: configdao.ProjectionMembersFull, Members: &configdao.FullMembersControl{DesiredIPs: ips}}, {ControlID: 2, Kind: configdao.ProjectionACKFull, Ack: &configdao.ProjectionAckControl{LegacySHA: &hash, ArtifactSourceIDs: sources, RequiredControlIDs: []uint64{1}}}}}
}
func projectionReserve(t *testing.T, journal configdao.ProjectionPlanJournalDAO, family int8, pool int, n byte) configdao.PoolReservation {
	t.Helper()
	ref, err := journal.ReservePoolOperation(context.Background(), configdao.PoolOperationRequest{Family: family, PoolID: pool, Nonce: [16]byte{n}})
	if err != nil {
		t.Fatal(err)
	}
	return ref
}
func TestProjectionStoreTypedCompilationAndNoRetryPool(t *testing.T) {
	journal, _, _ := projectionStoreSQL(t)
	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1", MaxRetries: 7})
	defer client.Close()
	store, err := NewProjectionStore(client)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if store.client == client || store.client.Options().MaxRetries != 0 || client.Options().MaxRetries != 7 {
		t.Fatal("Store did not isolate retry ownership")
	}
	ref := projectionReserve(t, journal, 4, 7, 1)
	item := projectionRecordItem(7, 11, "2001:DB8::1", "native synthetic bytes")
	receipt, err := journal.CommitGeneratedProjectionPlan(context.Background(), projectionFullRequest(ref, item))
	if err != nil {
		t.Fatal(err)
	}
	err = journal.WithPoolProjectionLease(context.Background(), receipt, func(lease configdao.ProjectionLease) error {
		snapshot, err := lease.Snapshot()
		if err != nil {
			return err
		}
		handle, err := compileProjection(store, lease, snapshot)
		if err != nil {
			return err
		}
		rawKey, _ := projectionRawKey(snapshot.Artifacts[0].Binding)
		if rawKey != buildRedisKeyRecordRule("2001:DB8::1") {
			t.Fatal("logical binding was normalized")
		}
		if handle.wire.Legacy == 0 || handle.proof == nil || handle.wire.PoolFence == 0 {
			t.Fatal("full controls omitted")
		}
		projected := *snapshot.Pool
		projected.Phase = model.ConfigPublicationProjected
		if projectionPoolDigest(projected) != projectionPoolDigest(*snapshot.Pool) {
			t.Fatal("mutable phase changed fence signature")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if projectionVersion(9007199254740992) >= projectionVersion(9007199254740993) || projectionVersion(^uint64(0)) != "18446744073709551615" {
		t.Fatal("uint64 ordering lost precision")
	}
	for _, binding := range []configdao.ProjectionBinding{{Version: 1, Role: "artifact", ConfigType: 4, InstanceIP: "pool:7:ips"}, {Version: 1, Role: "intent", ConfigType: 5, InstanceIP: "127.0.0.1"}} {
		if _, err := projectionRawKey(binding); err == nil {
			t.Fatal("SQL key acquired arbitrary Redis role")
		}
	}
}

func TestProjectionStoreCompilesEveryFamilyWithoutFreePhysicalKeys(t *testing.T) {
	patterns := map[int8]string{1: ConfigNamePrometheus, 2: ConfigNameAlertManager, 3: ConfigNameAlertRule, 4: ConfigNameRecordRule}
	for family := int8(1); family <= 5; family++ {
		t.Run(fmt.Sprint(family), func(t *testing.T) {
			journal, _, _ := projectionStoreSQL(t)
			base := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1"})
			defer base.Close()
			store, _ := NewProjectionStore(base)
			defer store.Close()
			ref := projectionReserve(t, journal, family, 800+int(family), byte(family))
			ip := fmt.Sprintf("127.0.8.%d", family)
			row := &model.MonitorConfig{PoolID: ref.PoolID, ConfigType: family, InstanceIP: ip, Status: 1, ConfigContent: "synthetic"}
			binding := &configdao.ProjectionBinding{Version: 1, Role: "artifact", ConfigType: family, InstanceIP: ip}
			if family == 5 {
				group := 9
				binding.GroupID = &group
				row.InstanceIP += "_9"
				row.Name = "webhook_family_9_" + ip
			} else {
				row.Name = fmt.Sprintf(patterns[family], ref.PoolID, ip)
			}
			item := configdao.GeneratedPlanItem{SourceID: 11, Action: model.ConfigProjectionRender, Desired: row, Binding: binding}
			request := projectionFullRequest(ref, item)
			if family == 2 {
				request.Controls[1].Ack.RequiredControlIDs = []uint64{1, 3}
				request.Controls = append(request.Controls, configdao.ProjectionControl{ControlID: 3, Kind: configdao.ProjectionWebhookIntents, Intents: &configdao.WebhookIntentsControl{RequireEmpty: true}})
			}
			if family == 5 {
				request.Controls = []configdao.ProjectionControl{{ControlID: 9, Kind: configdao.ProjectionACKWebhookSubset, Ack: &configdao.ProjectionAckControl{ArtifactSourceIDs: []uint64{11}}}}
			}
			receipt, err := journal.CommitGeneratedProjectionPlan(context.Background(), request)
			if err != nil {
				t.Fatal(err)
			}
			err = journal.WithPoolProjectionLease(context.Background(), receipt, func(lease configdao.ProjectionLease) error {
				snapshot, err := lease.Snapshot()
				if err != nil {
					return err
				}
				handle, err := compileProjection(store, lease, snapshot)
				if err != nil {
					return err
				}
				if family == 5 {
					if handle.wire.Legacy != 0 || len(handle.wire.Controls) != 1 {
						t.Fatal("subset gained full controls")
					}
				} else if handle.wire.Legacy == 0 {
					t.Fatal("full family lost legacy SHA")
				}
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}
