package config

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type initialReviewFixture struct {
	dao           *monitorConfigDAO
	journal       ProjectionPlanJournalDAO
	db            *gorm.DB
	receipt       ProjectionReceipt
	first, second GeneratedPlanItem
}

func newInitialReviewFixture(t *testing.T) *initialReviewFixture {
	t.Helper()
	dao, _, db := journalBehaviorFixture(t)
	journal := NewProjectionPlanJournalDAO(zap.NewNop(), db)
	ref, err := journal.ReservePoolOperation(context.Background(), journalBehaviorRequest(101))
	if err != nil {
		t.Fatal(err)
	}
	first, second := *journalBehaviorRender(11, "127.0.0.181"), *journalBehaviorRender(22, "127.0.0.182")
	receipt, err := journal.CommitGeneratedProjectionPlan(context.Background(), v2RecordPlan(ref, first, second))
	if err != nil {
		t.Fatal(err)
	}
	return &initialReviewFixture{dao: dao, journal: journal, db: db, receipt: receipt, first: first, second: second}
}
func initialReviewEvent(t *testing.T, db *gorm.DB, id uint64) model.MonitorConfigArtifactEvent {
	t.Helper()
	var event model.MonitorConfigArtifactEvent
	if db.First(&event, "event_id=?", id).Error != nil {
		t.Fatal("review event unavailable")
	}
	return event
}
func initialReviewCorruptOtherDigest(t *testing.T, db *gorm.DB, operationID uint64) {
	t.Helper()
	var op model.MonitorConfigPoolOperation
	if db.First(&op, "operation_id=?", operationID).Error != nil || op.Manifest == nil {
		t.Fatal("review manifest unavailable")
	}
	var envelope map[string]json.RawMessage
	if json.Unmarshal([]byte(*op.Manifest), &envelope) != nil {
		t.Fatal("manifest decode fixture failed")
	}
	var artifacts []map[string]json.RawMessage
	if json.Unmarshal(envelope["artifacts"], &artifacts) != nil || len(artifacts) != 2 {
		t.Fatal("two-entry fixture unavailable")
	}
	artifacts[1]["plan_digest"], _ = json.Marshal(strings.Repeat("0", 64))
	raw, err := json.Marshal(artifacts)
	if err != nil {
		t.Fatal(err)
	}
	envelope["artifacts"] = raw
	raw, err = json.Marshal(envelope)
	if err != nil {
		t.Fatal(err)
	}
	if db.Model(&model.MonitorConfigPoolOperation{}).Where("operation_id=?", operationID).UpdateColumn("manifest", string(raw)).Error != nil {
		t.Fatal("cannot seed corrupt persisted manifest")
	}
}

func TestInitialFactsReviewWarmOperationDoesNotAuthorizeBadSecondNamespace(t *testing.T) {
	for _, mode := range []string{"new revision", "body drift", "foreign owner", "duplicate custom"} {
		t.Run(mode, func(t *testing.T) {
			f := newInitialReviewFixture(t)
			switch mode {
			case "new revision":
				row, err := f.dao.GetMonitorConfigByID(context.Background(), f.second.Desired.ID)
				if err != nil {
					t.Fatal(err)
				}
				row.ConfigContent = "groups: [] # subsequent ordinary revision\n"
				if f.dao.UpdateMonitorConfig(context.Background(), row) != nil {
					t.Fatal("ordinary revision failed")
				}
			case "body drift":
				if f.db.Model(&model.MonitorConfig{}).Where("id=?", f.second.Desired.ID).UpdateColumn("config_content", "groups: [] # persisted drift\n").Error != nil {
					t.Fatal("cannot seed body drift")
				}
			case "foreign owner":
				if f.db.Model(&model.MonitorConfig{}).Where("id=?", f.second.Desired.ID).UpdateColumn("pool_id", 99).Error != nil {
					t.Fatal("cannot seed foreign owner")
				}
			case "duplicate custom":
				row := *f.second.Desired
				row.Model = model.Model{}
				row.Name = "custom persisted duplicate"
				row.Status = 2
				if f.db.Create(&row).Error != nil {
					t.Fatal("cannot seed historical namespace duplicate")
				}
			}
			warm := false
			// Exercise the actual helper with both namespace locks and a single facts
			// reader: A fully succeeds before B is asked to reuse the same operation.
			err := f.db.Transaction(func(tx *gorm.DB) error {
				if _, err := lockExistingPoolControl(tx, 4, 7); err != nil {
					return err
				}
				keys := []configNamespaceKey{{4, f.first.Desired.InstanceIP}, {4, f.second.Desired.InstanceIP}}
				if err := lockConfigNamespacesTx(tx, keys); err != nil {
					return err
				}
				operations := newLeaseInitialOperations(tx)
				if _, err := leaseMaterialWithOperations(tx, f.receipt.Artifacts[0], false, operations); err != nil {
					return err
				}
				warm = true
				if _, err := leaseMaterialWithOperations(tx, f.receipt.Artifacts[1], false, operations); err == nil {
					t.Error("warm immutable membership bypassed B's current event/namespace/owner/body checks")
				}
				return nil
			})
			if err != nil || !warm {
				t.Fatal("valid first namespace did not warm the actual operation facts", err)
			}
			entered := false
			if err := f.journal.WithPoolProjectionLease(context.Background(), f.receipt, func(ProjectionLease) error { entered = true; return nil }); err == nil || entered {
				t.Fatal("whole lease initial validation authorized the corrupt second namespace")
			}
		})
	}
}

func TestInitialFactsReviewRequestedMemberCannotHideBadOtherManifestEntry(t *testing.T) {
	f := newInitialReviewFixture(t)
	first := initialReviewEvent(t, f.db, f.receipt.Artifacts[0].EventID)
	initialReviewCorruptOtherDigest(t, f.db, f.receipt.Reservation.OperationID)
	if err := f.db.Transaction(func(tx *gorm.DB) error {
		operations := newLeaseInitialOperations(tx)
		root, err := trustedBindingRootWithOperations(tx, &first, configNamespaceKey{4, f.first.Desired.InstanceIP}, operations)
		if err != nil {
			return err
		}
		if root != nil {
			t.Error("requested valid member bypassed the corrupt unrelated manifest entry")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	entered := false
	if err := f.journal.WithPoolProjectionLease(context.Background(), f.receipt, func(ProjectionLease) error { entered = true; return nil }); err == nil || entered {
		t.Fatal("partial manifest validation granted initial lease")
	}
}

func TestInitialFactsReviewAMParsedMembershipDoesNotReplaceFullCardinality(t *testing.T) {
	_, _, db := journalBehaviorFixture(t)
	journal := NewAMProjectionJournalDAO(zap.NewNop(), db)
	ctx := context.Background()
	request := journalBehaviorRequest(102)
	request.Family = 2
	source := AMSourceVersion{PoolID: 7}
	ref, err := journal.ReserveAMPoolOperation(ctx, request, source)
	if err != nil {
		t.Fatal(err)
	}
	item := webhookPlanItem()
	receipt, err := journal.CommitAMProjectionPlan(ctx, AMProjectionPlanRequest{GeneratedProjectionPlanRequest: v2AMPlan(ref, item), SourceVersion: source})
	if err != nil {
		t.Fatal(err)
	}
	event := initialReviewEvent(t, db, receipt.Artifacts[0].EventID)
	// An extra persisted event is not represented in the issued manifest. It has
	// another key, so latest-event validation for the real artifact cannot mask
	// a missing full-receipt cardinality check.
	extra := event
	extra.EventID = 0
	extra.InstanceKey = []byte("127.0.0.199_3")
	if db.Create(&extra).Error != nil {
		t.Fatal("cannot seed orphan operation event")
	}
	dao := journal.(*monitorConfigDAO)
	err = db.Transaction(func(tx *gorm.DB) error {
		operations := newLeaseInitialOperations(tx)
		parsed, err := operations.operation(ref.OperationID)
		if err != nil {
			return err
		}
		if parsed == nil || !parsed.containsEvent(event.EventID) {
			return errors.New("valid parsed membership fixture was not accepted")
		}
		var op model.MonitorConfigPoolOperation
		if err := tx.First(&op, "operation_id=?", ref.OperationID).Error; err != nil {
			return err
		}
		if _, err := dao.loadReceipt(tx, op); err == nil {
			t.Error("parsed membership was mistaken for full AM event/cardinality validation")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	entered := false
	if err := journal.WithPoolProjectionLease(ctx, receipt, func(ProjectionLease) error { entered = true; return nil }); err == nil || entered {
		t.Fatal("orphan AM event passed the actual full initial loader")
	}
}

func TestInitialFactsReviewWarmRootsPreserveNamespaceNAAndDepthBoundaries(t *testing.T) {
	for _, mode := range []string{"wrong namespace", "wrong raw key", "N/A latest", "self parent", "depth"} {
		t.Run(mode, func(t *testing.T) {
			f := newInitialReviewFixture(t)
			root := initialReviewEvent(t, f.db, f.receipt.Artifacts[0].EventID)
			key := configNamespaceKey{4, f.first.Desired.InstanceIP}
			err := f.db.Transaction(func(tx *gorm.DB) error {
				if err := lockConfigNamespacesTx(tx, []configNamespaceKey{key}); err != nil {
					return err
				}
				operations := newLeaseInitialOperations(tx)
				if got, err := trustedBindingRootWithOperations(tx, &root, key, operations); err != nil || got == nil {
					return errors.New("valid generated root did not warm membership")
				}
				checkReject := func(event *model.MonitorConfigArtifactEvent, target configNamespaceKey) error {
					got, err := trustedBindingRootWithOperations(tx, event, target, operations)
					if err != nil {
						return err
					}
					if got != nil {
						t.Error("warm historical fact bypassed current chain boundary")
					}
					return nil
				}
				if mode == "wrong namespace" {
					return checkReject(&root, configNamespaceKey{3, key.raw})
				}
				if mode == "wrong raw key" {
					return checkReject(&root, configNamespaceKey{4, f.second.Desired.InstanceIP})
				}
				makeChild := func(parent uint64, policy string) (model.MonitorConfigArtifactEvent, error) {
					child := model.MonitorConfigArtifactEvent{ConfigType: 4, InstanceKey: []byte(key.raw), Action: model.ConfigProjectionInvalidate, ProjectionPolicy: policy, DeliveryState: model.ConfigProjectionApplied}
					if policy == model.ConfigProjectionLineageOnly {
						child.BindingEventID = &parent
					}
					err := tx.Create(&child).Error
					return child, err
				}
				if mode == "N/A latest" {
					child, err := makeChild(root.EventID, model.ConfigProjectionNotApplicable)
					if err != nil {
						return err
					}
					return checkReject(&child, key)
				}
				if mode == "self parent" {
					child, err := makeChild(root.EventID, model.ConfigProjectionLineageOnly)
					if err != nil {
						return err
					}
					child.BindingEventID = &child.EventID
					if err := tx.Model(&child).UpdateColumn("binding_event_id", child.EventID).Error; err != nil {
						return err
					}
					return checkReject(&child, key)
				}
				current := root
				for links := 1; links <= 128; links++ {
					child, err := makeChild(current.EventID, model.ConfigProjectionLineageOnly)
					if err != nil {
						return err
					}
					current = child
					if links == 127 {
						got, err := trustedBindingRootWithOperations(tx, &current, key, operations)
						if err != nil || got == nil {
							return errors.New("existing 128-node valid boundary was rejected")
						}
					}
				}
				return checkReject(&current, key)
			})
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestInitialFactsReviewOperationFactsDoNotEscapeInitialScope(t *testing.T) {
	f := newInitialReviewFixture(t)
	id := f.receipt.Reservation.OperationID
	if err := f.db.Transaction(func(tx *gorm.DB) error {
		facts, err := newLeaseInitialOperations(tx).operation(id)
		if err != nil {
			return err
		}
		if facts == nil || !facts.containsEvent(f.receipt.Artifacts[0].EventID) {
			return errors.New("valid first initial-scope facts missing")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	initialReviewCorruptOtherDigest(t, f.db, id)
	if err := f.db.Transaction(func(tx *gorm.DB) error {
		facts, err := newLeaseInitialOperations(tx).operation(id)
		if err != nil {
			return err
		}
		if facts != nil {
			t.Error("a new initial transaction inherited old operation facts instead of rejecting persisted corruption")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}
