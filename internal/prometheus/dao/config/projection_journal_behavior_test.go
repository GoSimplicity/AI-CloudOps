package config

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func journalBehaviorFixture(t *testing.T) (*monitorConfigDAO, ProjectionJournalDAO, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "journal.sqlite")), &gorm.Config{Logger: logger.Discard})
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
	dao := NewMonitorConfigDAO(zap.NewNop(), db).(*monitorConfigDAO)
	return dao, NewProjectionJournalDAO(zap.NewNop(), db), db
}
func journalBehaviorRequest(n byte) PoolOperationRequest {
	return PoolOperationRequest{Family: 4, PoolID: 7, Nonce: [16]byte{n}}
}
func journalBehaviorRender(sourceID uint64, ip string) *GeneratedPlanItem {
	return &GeneratedPlanItem{SourceID: sourceID, Action: model.ConfigProjectionRender, Desired: &model.MonitorConfig{PoolID: 7, ConfigType: 4, InstanceIP: ip, Name: "prometheus_record_rule_7_" + ip + ".yaml", ConfigContent: "groups: []\n", Status: 1}, Binding: &ProjectionBinding{Version: 1, Role: "artifact", ConfigType: 4, InstanceIP: ip}}
}
func journalBehaviorCommit(t *testing.T, dao ProjectionJournalDAO, n byte, item GeneratedPlanItem) PoolReceipt {
	t.Helper()
	ref, err := dao.ReservePoolOperation(context.Background(), journalBehaviorRequest(n))
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := dao.CommitGeneratedPlan(context.Background(), GeneratedPlanRequest{Reservation: ref, SourceHash: sha256.Sum256([]byte{n}), Items: []GeneratedPlanItem{item}})
	if err != nil {
		t.Fatal(err)
	}
	return receipt
}

func TestJournalBehaviorBindingRejectsPhysicalKeysAndInvalidLogicalRoles(t *testing.T) {
	for _, raw := range []string{
		`{"version":1,"role":"artifact","config_type":4,"instance_ip":"127.0.0.1","redis_key":"free"}`,
		`{"version":2,"role":"artifact","config_type":4,"instance_ip":"127.0.0.1"}`,
		`{"version":1,"role":"member","config_type":4,"instance_ip":"127.0.0.1"}`,
		`{"version":1,"role":"artifact","config_type":4,"instance_ip":"pool:7:ips"}`,
		`{"version":1,"role":"artifact","config_type":4,"instance_ip":"127.0.0.1 ","group_id":3}`,
		`{"version":1,"role":"artifact","config_type":5,"instance_ip":"127.0.0.1"}`,
		`{"version":1,"role":"artifact","config_type":5,"instance_ip":"127.0.0.1","group_id":0}`,
		`{"version":1,"role":"artifact","config_type":5,"instance_ip":"retirement:intent","group_id":3}`,
		`{"version":1,"role":"artifact","config_type":4,"instance_ip":"127.0.0.1"} null`,
	} {
		if _, err := ParseProjectionBinding(raw); err == nil {
			t.Fatal("invalid binding granted logical artifact authority")
		}
	}
	for _, ip := range []string{"2001:db8::1", "2001:DB8::1"} {
		binding, err := ParseProjectionBinding(`{"role":"artifact", "instance_ip":"` + ip + `", "config_type":4,"version":1}`)
		if err != nil || binding.InstanceIP != ip {
			t.Fatal("valid raw byte-distinct binding was normalized/rejected")
		}
	}
	group := 3
	binding := ProjectionBinding{Version: 1, Role: "artifact", ConfigType: 5, InstanceIP: "127.0.0.1", GroupID: &group}
	key, err := bindingNamespace(binding)
	if err != nil || key.raw != "127.0.0.1_3" {
		t.Fatal("webhook binding did not identify its exact compound namespace")
	}
}
func TestJournalBehaviorOrdinarySQLKeysNeverAcquireGeneratedBinding(t *testing.T) {
	for _, key := range []string{"pool:7:ips", strings.Repeat("🌍", 45), "retirement:intent"} {
		t.Run(key, func(t *testing.T) {
			ordinary, _, db := journalBehaviorFixture(t)
			row := &model.MonitorConfig{PoolID: 99, ConfigType: 5, InstanceIP: key, Name: "custom", ConfigContent: "SYNTHETIC_PRIVATE_BODY", Status: 2}
			if err := ordinary.CreateMonitorConfig(context.Background(), row); err != nil {
				t.Fatal(err)
			}
			var event model.MonitorConfigArtifactEvent
			if err := db.Order("event_id DESC").First(&event).Error; err != nil {
				t.Fatal(err)
			}
			if event.ProjectionPolicy != model.ConfigProjectionNotApplicable || event.Binding != nil || event.BindingEventID != nil || event.OperationID != nil || len(event.BodyDigest) != 32 || string(event.InstanceKey) != key {
				t.Fatal("SQL-only key gained projection authority or lost raw bytes")
			}
			var controlCount int64
			if db.Model(&model.MonitorConfigPoolControl{}).Count(&controlCount).Error != nil || controlCount != 0 {
				t.Fatal("ordinary event allocated pool control")
			}
			if err := ordinary.DeleteMonitorConfig(context.Background(), row.ID); err != nil {
				t.Fatal(err)
			}
			var count int64
			if db.Model(&model.MonitorConfigArtifactEvent{}).Count(&count).Error != nil || count != 2 {
				t.Fatal("ordinary delete did not append same-tx lineage record")
			}
		})
	}
}
func TestJournalBehaviorLineageMigrationAndRawOwnerBreak(t *testing.T) {
	for _, mode := range []string{"ordinary move", "raw owner break"} {
		t.Run(mode, func(t *testing.T) {
			ordinary, journal, db := journalBehaviorFixture(t)
			item := journalBehaviorRender(11, "127.0.0.1")
			receipt := journalBehaviorCommit(t, journal, 1, *item)
			if item.Desired.ID == 0 || len(receipt.Artifacts) != 1 {
				t.Fatal("committed source ID mapping missing")
			}
			if mode == "ordinary move" {
				if err := ordinary.UpdateMonitorConfig(context.Background(), &model.MonitorConfig{Model: model.Model{ID: item.Desired.ID}, InstanceIP: "127.0.0.2"}); err != nil {
					t.Fatal(err)
				}
				var events []model.MonitorConfigArtifactEvent
				if db.Order("event_id").Find(&events).Error != nil || len(events) != 3 {
					t.Fatal("move did not append old/new namespace events")
				}
				if events[1].ProjectionPolicy != model.ConfigProjectionLineageOnly || events[1].BindingEventID == nil || *events[1].BindingEventID != receipt.Artifacts[0].EventID || events[2].ProjectionPolicy != model.ConfigProjectionNotApplicable {
					t.Fatal("move granted authority to the new unbound namespace or lost old lineage")
				}
			} else {
				if err := db.Model(&model.MonitorConfig{}).Where("id = ?", item.Desired.ID).Update("name", "untracked owner").Error; err != nil {
					t.Fatal(err)
				}
				if err := ordinary.UpdateMonitorConfig(context.Background(), &model.MonitorConfig{Model: model.Model{ID: item.Desired.ID}, Name: "ordinary edit"}); err != nil {
					t.Fatal(err)
				}
				var event model.MonitorConfigArtifactEvent
				if db.Order("event_id DESC").First(&event).Error != nil || event.ProjectionPolicy != model.ConfigProjectionNotApplicable || event.BindingEventID != nil {
					t.Fatal("broken owner chain resurrected old binding")
				}
			}
			if ok, err := journal.MarkArtifactDeliveryIfCurrent(context.Background(), receipt.Artifacts[0], model.ConfigProjectionApplied); err != nil || ok {
				t.Fatal("stale issued event was marked applied")
			}
		})
	}
}
func TestJournalBehaviorNonemptyConditionalMarksAndPayloadCloning(t *testing.T) {
	_, journal, _ := journalBehaviorFixture(t)
	item := journalBehaviorRender(11, "127.0.0.1")
	receipt := journalBehaviorCommit(t, journal, 1, *item)
	forged := receipt.Artifacts[0]
	forged.BodyDigest = append([]byte(nil), forged.BodyDigest...)
	forged.BodyDigest[0] ^= 1
	if ok, err := journal.MarkArtifactDeliveryIfCurrent(context.Background(), forged, model.ConfigProjectionApplied); err == nil || ok {
		t.Fatal("altered digest was accepted")
	}
	// Mutating a returned receipt cannot mutate stored immutable payload.
	again, err := journal.LoadLastCommitted(context.Background(), 4, 7)
	if err != nil || again == nil || digestEqual(again.Artifacts[0].BodyDigest, forged.BodyDigest) {
		t.Fatal("receipt alias changed persisted proof")
	}
	if ok, err := journal.MarkPoolProjectedIfCurrent(context.Background(), receipt); err != nil || ok {
		t.Fatal("pool was marked before its artifact delivery")
	}
	if ok, err := journal.MarkArtifactDeliveryIfCurrent(context.Background(), receipt.Artifacts[0], model.ConfigProjectionApplied); err != nil || !ok {
		t.Fatal("current artifact ACK rejected", err)
	}
	if ok, err := journal.MarkPoolProjectedIfCurrent(context.Background(), receipt); err != nil || !ok {
		t.Fatal("complete current pool ACK rejected", err)
	}
	projected, err := journal.LoadLastProjected(context.Background(), 4, 7)
	if err != nil || projected == nil || projected.Reservation != receipt.Reservation {
		t.Fatal("lastProjected receipt lost")
	}
}

// A real SQLite commit succeeds, then this test-only pool drops its ACK. This
// verifies unknown outcome handling without claiming a failed commit rolled back.
type journalAckLossPool struct {
	*sql.DB
	lose atomic.Bool
}

func (p *journalAckLossPool) BeginTx(ctx context.Context, options *sql.TxOptions) (gorm.ConnPool, error) {
	tx, err := p.DB.BeginTx(ctx, options)
	if err != nil {
		return nil, err
	}
	return &journalAckLossTx{Tx: tx, pool: p}, nil
}

type journalAckLossTx struct {
	*sql.Tx
	pool *journalAckLossPool
}

func (tx *journalAckLossTx) Commit() error {
	if err := tx.Tx.Commit(); err != nil {
		return err
	}
	if tx.pool.lose.Swap(false) {
		return errors.New("PRIVATE_COMMIT_ACK_CANARY")
	}
	return nil
}
func TestJournalBehaviorUnknownCommitResolvesByNonceWithoutIDLeak(t *testing.T) {
	for _, phase := range []string{"reserve", "business"} {
		t.Run(phase, func(t *testing.T) {
			_, normal, db := journalBehaviorFixture(t)
			pool, err := db.DB()
			if err != nil {
				t.Fatal(err)
			}
			fault := &journalAckLossPool{DB: pool}
			wrapped := db.Session(&gorm.Session{NewDB: true, Initialized: true})
			wrapped.ConnPool = fault
			wrapped.Statement.ConnPool = fault
			journal := NewProjectionJournalDAO(zap.NewNop(), wrapped)
			request := journalBehaviorRequest(1)
			if phase == "reserve" {
				fault.lose.Store(true)
				ref, err := journal.ReservePoolOperation(context.Background(), request)
				if !errors.Is(err, ErrJournalUncertain) || ref.OperationID != 0 || strings.Contains(err.Error(), "PRIVATE_COMMIT_ACK_CANARY") {
					t.Fatal("lost reserve ACK was guessed or leaked")
				}
				snapshot, err := normal.LoadOperationByNonce(context.Background(), request.Nonce)
				if err != nil || snapshot.Phase != model.ConfigPublicationReserved || snapshot.Receipt != nil {
					t.Fatal("reserve fact could not be resolved")
				}
				repeated, err := normal.ReservePoolOperation(context.Background(), request)
				if err != nil || repeated != snapshot.Reservation {
					t.Fatal("unknown reserve retry allocated a new generation")
				}
			} else {
				ref, err := normal.ReservePoolOperation(context.Background(), request)
				if err != nil {
					t.Fatal(err)
				}
				item := journalBehaviorRender(11, "127.0.0.1")
				fault.lose.Store(true)
				receipt, err := journal.CommitGeneratedPlan(context.Background(), GeneratedPlanRequest{Reservation: ref, SourceHash: sha256.Sum256([]byte("source")), Items: []GeneratedPlanItem{*item}})
				if !errors.Is(err, ErrJournalUncertain) || receipt.Reservation.OperationID != 0 || item.Desired.ID != 0 || !item.Desired.CreatedAt.IsZero() {
					t.Fatal("unknown business commit leaked claimed success")
				}
				snapshot, err := normal.LoadOperationByNonce(context.Background(), request.Nonce)
				if err != nil || snapshot.Receipt == nil || len(snapshot.Receipt.Artifacts) != 1 || snapshot.Receipt.Artifacts[0].ConfigRowID == nil {
					t.Fatal("committed fact/source IDs could not be reloaded")
				}
			}
		})
	}
}

func TestJournalBehaviorRetireReceiptAndReplacementVector(t *testing.T) {
	for _, replacement := range []bool{false, true} {
		t.Run(map[bool]string{false: "current retire", true: "new owner supersedes retire"}[replacement], func(t *testing.T) {
			ordinary, journal, db := journalBehaviorFixture(t)
			item := journalBehaviorRender(11, "127.0.0.1")
			first := journalBehaviorCommit(t, journal, 1, *item)
			var row model.MonitorConfig
			if err := db.First(&row, item.Desired.ID).Error; err != nil {
				t.Fatal(err)
			}
			expected := GeneratedConfigOwnerOf(&row)
			ref, err := journal.ReservePoolOperation(context.Background(), journalBehaviorRequest(2))
			if err != nil {
				t.Fatal(err)
			}
			retired, err := journal.CommitGeneratedPlan(context.Background(), GeneratedPlanRequest{Reservation: ref, SourceHash: sha256.Sum256([]byte("retire")), Items: []GeneratedPlanItem{{SourceID: 12, Action: model.ConfigProjectionRetire, Expected: &expected}}})
			if err != nil || len(retired.Artifacts) != 1 || retired.Artifacts[0].ProjectionPolicy != model.ConfigProjectionLineageOnly || retired.Artifacts[0].BindingEventID == nil || *retired.Artifacts[0].BindingEventID != first.Artifacts[0].EventID {
				t.Fatal("retire did not inherit issued binding", err)
			}
			var count int64
			if db.Model(&model.MonitorConfig{}).Where("id=?", row.ID).Count(&count).Error != nil || count != 0 {
				t.Fatal("retire SQL did not soft-delete")
			}
			if replacement {
				newer := &model.MonitorConfig{PoolID: 99, ConfigType: 4, InstanceIP: "127.0.0.1", Name: "new custom owner", ConfigContent: "custom", Status: 2}
				if err := ordinary.CreateMonitorConfig(context.Background(), newer); err != nil {
					t.Fatal(err)
				}
				if ok, err := journal.MarkArtifactDeliveryIfCurrent(context.Background(), retired.Artifacts[0], model.ConfigProjectionApplied); err != nil || ok {
					t.Fatal("retire ACK ignored newer namespace owner")
				}
				if ok, err := journal.MarkArtifactDeliveryIfCurrent(context.Background(), retired.Artifacts[0], model.ConfigProjectionSuperseded); err != nil || !ok {
					t.Fatal("obsolete retire could not be conditionally superseded")
				}
			} else {
				if ok, err := journal.MarkArtifactDeliveryIfCurrent(context.Background(), retired.Artifacts[0], model.ConfigProjectionApplied); err != nil || !ok {
					t.Fatal("current retire ACK rejected", err)
				}
				if ok, err := journal.MarkPoolProjectedIfCurrent(context.Background(), retired); err != nil || !ok {
					t.Fatal("completed retire pool ACK rejected", err)
				}
				thirdRequest := journalBehaviorRequest(3)
				thirdRequest.RecoveryOfOperationID = &ref.OperationID
				third, err := journal.ReservePoolOperation(context.Background(), thirdRequest)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := journal.CommitGeneratedPlan(context.Background(), GeneratedPlanRequest{Reservation: third, SourceHash: sha256.Sum256([]byte("retry retire")), Items: []GeneratedPlanItem{{SourceID: 12, Action: model.ConfigProjectionRetire, Expected: &expected}}}); err != nil {
					t.Fatal("identical owner tombstone recovery rejected", err)
				}
			}
		})
	}
}
