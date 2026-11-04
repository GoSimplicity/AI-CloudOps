package config

import (
	"context"
	"crypto/sha256"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func v2RecordPlan(ref PoolReservation, items ...GeneratedPlanItem) GeneratedProjectionPlanRequest {
	hash := sha256.Sum256([]byte("v2 synthetic source"))
	ips := []string{}
	sources := []uint64{}
	for _, item := range items {
		sources = append(sources, item.SourceID)
		if item.Action == model.ConfigProjectionRender {
			ips = append(ips, item.Binding.InstanceIP)
		}
	}
	return GeneratedProjectionPlanRequest{GeneratedPlanRequest: GeneratedPlanRequest{Reservation: ref, SourceHash: hash, Items: items}, Controls: []ProjectionControl{{ControlID: 1, Kind: ProjectionMembersFull, Members: &FullMembersControl{DesiredIPs: ips}}, {ControlID: 2, Kind: ProjectionACKFull, Ack: &ProjectionAckControl{LegacySHA: &hash, ArtifactSourceIDs: sources, RequiredControlIDs: []uint64{1}}}}}
}
func TestProjectionV2NonemptyReceiptAndSealedSingleConnectionLease(t *testing.T) {
	_, _, db := journalBehaviorFixture(t)
	dao := NewProjectionPlanJournalDAO(zap.NewNop(), db)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	ref, err := dao.ReservePoolOperation(ctx, journalBehaviorRequest(1))
	if err != nil {
		t.Fatal(err)
	}
	item := journalBehaviorRender(11, "127.0.0.1")
	item.Desired.ConfigContent = "groups: [] # native unredacted synthetic bytes\n"
	req := v2RecordPlan(ref, *item)
	receipt, err := dao.CommitGeneratedProjectionPlan(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if item.Desired.ID == 0 || len(receipt.Artifacts) != 1 {
		t.Fatal("missing committed row/event")
	}
	again, err := dao.CommitGeneratedProjectionPlan(ctx, req)
	if err != nil || !reflect.DeepEqual(again, receipt) {
		t.Fatal("idempotent controls changed", err)
	}
	var saved ProjectionLease
	err = dao.WithPoolProjectionLease(ctx, receipt, func(lease ProjectionLease) error {
		saved = lease
		snapshot, err := lease.Snapshot()
		if err != nil {
			return err
		}
		if snapshot.Pool == nil || len(snapshot.Artifacts) != 1 || string(snapshot.Artifacts[0].Body) != item.Desired.ConfigContent {
			t.Fatal("lease lost committed native bytes")
		}
		snapshot.Artifacts[0].Body[0] = 'X'
		snapshot.Pool.Controls[0].Members.DesiredIPs[0] = "changed"
		second, err := lease.Snapshot()
		if err != nil {
			return err
		}
		if string(second.Artifacts[0].Body) != item.Desired.ConfigContent || second.Pool.Controls[0].Members.DesiredIPs[0] != "127.0.0.1" {
			t.Fatal("snapshot aliases lease material")
		}
		if err := lease.RecheckCurrent(ctx); err != nil {
			return err
		}
		proof, _ := projectionAckProof(receipt)
		ok, err := lease.MarkComplete(proof)
		if !ok && err == nil {
			t.Fatal("complete rejected current receipt")
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := saved.Snapshot(); err == nil {
		t.Fatal("escaped lease retained authority")
	}
	if saved.RecheckCurrent(ctx) == nil {
		t.Fatal("escaped recheck opened new transaction")
	}
	loaded, err := dao.LoadLastProjectedProjection(ctx, 4, 7)
	if err != nil || loaded == nil || loaded.Reservation != ref {
		t.Fatal("complete not durable", err)
	}
	changed := req
	changed.Controls = cloneProjectionControls(req.Controls)
	changed.Controls[0].ControlID = 9
	changed.Controls[1].Ack.RequiredControlIDs = []uint64{9}
	if _, err := dao.CommitGeneratedProjectionPlan(ctx, changed); err == nil {
		t.Fatal("nonce retry overwrote signed controls")
	}
}
func TestProjectionV2LastGoodEmptyAndFailedAttempt(t *testing.T) {
	_, _, db := journalBehaviorFixture(t)
	dao := NewProjectionPlanJournalDAO(zap.NewNop(), db)
	ctx := context.Background()
	first, _ := dao.ReservePoolOperation(ctx, journalBehaviorRequest(1))
	receipt, err := dao.CommitGeneratedProjectionPlan(ctx, v2RecordPlan(first))
	if err != nil {
		t.Fatal(err)
	}
	next, _ := dao.ReservePoolOperation(ctx, journalBehaviorRequest(2))
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := dao.CommitGeneratedProjectionPlan(canceled, v2RecordPlan(next)); !errors.Is(err, context.Canceled) {
		t.Fatal("canceled operation committed", err)
	}
	proof, _ := projectionAckProof(receipt)
	ok, err := dao.MarkProjectionCompleteIfCurrent(ctx, receipt, proof)
	if err != nil || !ok {
		t.Fatal("failed attempt starved lastGood", err)
	}
	recovery := journalBehaviorRequest(3)
	recovery.RecoveryOfOperationID = &first.OperationID
	third, err := dao.ReservePoolOperation(ctx, recovery)
	if err != nil || third.Generation != 3 {
		t.Fatal("recovery lost counter", err)
	}
	loaded, err := NewProjectionPlanJournalDAO(zap.NewNop(), db).LoadLastProjectedProjection(ctx, 4, 7)
	if err != nil || loaded == nil || !loaded.CompleteControls || loaded.Reservation != first {
		t.Fatal("recovery lost durable empty lastGood", err)
	}
}
func TestProjectionV2StaleAndOrdinaryLeaseRestrictions(t *testing.T) {
	for _, mode := range []string{"ordinary edit", "ordinary retire", "owner move", "cancel callback"} {
		t.Run(mode, func(t *testing.T) {
			ordinary, _, db := journalBehaviorFixture(t)
			dao := NewProjectionPlanJournalDAO(zap.NewNop(), db)
			ctx := context.Background()
			ref, _ := dao.ReservePoolOperation(ctx, journalBehaviorRequest(1))
			item := journalBehaviorRender(11, "127.0.0.1")
			receipt, err := dao.CommitGeneratedProjectionPlan(ctx, v2RecordPlan(ref, *item))
			if err != nil {
				t.Fatal(err)
			}
			if mode == "cancel callback" {
				child, cancel := context.WithCancel(ctx)
				var saved ProjectionLease
				err := dao.WithPoolProjectionLease(child, receipt, func(lease ProjectionLease) error { saved = lease; cancel(); return lease.RecheckCurrent(child) })
				if !errors.Is(err, context.Canceled) {
					t.Fatal("cancellation lost", err)
				}
				if _, err := saved.Snapshot(); err == nil {
					t.Fatal("canceled handle stayed valid")
				}
				return
			}
			row := *item.Desired
			if mode == "ordinary retire" {
				err = ordinary.DeleteMonitorConfig(ctx, row.ID)
			} else {
				row.Name = "ordinary renamed"
				if mode == "owner move" {
					row.PoolID = 99
				}
				err = ordinary.UpdateMonitorConfig(ctx, &row)
			}
			if err != nil {
				t.Fatal(err)
			}
			called := false
			if err := dao.WithPoolProjectionLease(ctx, receipt, func(ProjectionLease) error { called = true; return nil }); err == nil || called {
				t.Fatal("stale pool projection acquired authority")
			}
			var event model.MonitorConfigArtifactEvent
			if err := db.Order("event_id DESC").First(&event).Error; err != nil {
				t.Fatal(err)
			}
			artifact, err := artifactReceipt(event, 0)
			if err != nil {
				t.Fatal(err)
			}

			err = dao.WithArtifactProjectionLease(ctx, artifact, func(lease ProjectionLease) error {
				snapshot, err := lease.Snapshot()
				if err != nil {
					return err
				}
				if snapshot.Pool != nil || len(snapshot.Artifacts) != 1 || len(snapshot.Artifacts[0].Body) != 0 {
					t.Fatal("ordinary lease gained render/pool authority")
				}
				return lease.RecheckCurrent(ctx)
			})
			if err != nil {
				t.Fatal(err)
			}
			err = dao.WithArtifactProjectionLease(ctx, artifact, func(lease ProjectionLease) error {
				if _, err := lease.MarkComplete(ProjectionAckProof{}); err == nil {
					t.Fatal("ordinary lease gained full ACK")
				}
				if _, err := lease.Snapshot(); err == nil {
					t.Fatal("failed completion left ordinary handle active")
				}
				return nil
			})
			if err == nil {
				t.Fatal("failed completion escaped as successful lease")
			}
		})
	}
}
func TestProjectionV2ControlValidationPreservesReservedOperation(t *testing.T) {
	for _, mode := range []string{"duplicate control", "extra member", "wrong vector", "extra payload", "forged intent family", "wrong SHA"} {
		t.Run(mode, func(t *testing.T) {
			_, _, db := journalBehaviorFixture(t)
			dao := NewProjectionPlanJournalDAO(zap.NewNop(), db)
			ctx := context.Background()
			ref, _ := dao.ReservePoolOperation(ctx, journalBehaviorRequest(1))
			request := v2RecordPlan(ref)
			switch mode {
			case "duplicate control":
				request.Controls[1].ControlID = 1
			case "extra member":
				request.Controls[0].Members.DesiredIPs = []string{"127.0.0.1"}
			case "wrong vector":
				request.Controls[1].Ack.ArtifactSourceIDs = []uint64{9}
			case "extra payload":
				request.Controls[0].Ack = &ProjectionAckControl{}
			case "forged intent family":
				request.Controls[0].Kind = ProjectionWebhookIntents
				request.Controls[0].Members = nil
				request.Controls[0].Intents = &WebhookIntentsControl{RequireEmpty: true}
			case "wrong SHA":
				wrong := [32]byte{1}
				request.Controls[1].Ack.LegacySHA = &wrong
			}
			if _, err := dao.CommitGeneratedProjectionPlan(ctx, request); err == nil {
				t.Fatal("invalid controls committed")
			}
			snapshot, err := dao.LoadProjectionByNonce(ctx, ref.Nonce)
			if err != nil || snapshot.Projection != nil || snapshot.SourceHash != nil {
				t.Fatal("rejected controls mutated operation", err)
			}
			var count int64
			if db.Model(&model.MonitorConfigArtifactEvent{}).Count(&count).Error != nil || count != 0 {
				t.Fatal("rejected controls leaked events")
			}
		})
	}
}

func v2AMPlan(ref PoolReservation, items ...GeneratedPlanItem) GeneratedProjectionPlanRequest {
	request := v2RecordPlan(ref, items...)
	request.Controls[0].Members.DesiredIPs = []string{}
	for _, item := range items {
		if item.Action == model.ConfigProjectionRender && item.Desired.ConfigType == 2 {
			request.Controls[0].Members.DesiredIPs = append(request.Controls[0].Members.DesiredIPs, item.Binding.InstanceIP)
		}
	}
	request.Controls = append(request.Controls, ProjectionControl{ControlID: 3, Kind: ProjectionWebhookIntents, Intents: &WebhookIntentsControl{RequireEmpty: true}})
	request.Controls[1].Ack.RequiredControlIDs = []uint64{1, 3}
	return request
}
func webhookPlanItem() GeneratedPlanItem {
	group := 3
	return GeneratedPlanItem{SourceID: 31, Action: model.ConfigProjectionRender, Desired: &model.MonitorConfig{PoolID: 7, ConfigType: 5, InstanceIP: "127.0.0.1_3", Name: "webhook_original_3_127.0.0.1", ConfigContent: "synthetic webhook native bytes", Status: 1}, Binding: &ProjectionBinding{Version: 1, Role: "artifact", ConfigType: 5, InstanceIP: "127.0.0.1", GroupID: &group}}
}
func TestProjectionV2IntentProofAndHistoricalTakeover(t *testing.T) {
	for _, mode := range []string{"tombstone proof", "fabricated owner alias", "N/A replacement", "cross pool history"} {
		t.Run(mode, func(t *testing.T) {
			ordinary, _, db := journalBehaviorFixture(t)
			dao := NewProjectionPlanJournalDAO(zap.NewNop(), db)
			ctx := context.Background()
			reserve := journalBehaviorRequest(1)
			reserve.Family = 2
			ref, _ := dao.ReservePoolOperation(ctx, reserve)
			item := webhookPlanItem()
			first, err := dao.CommitGeneratedProjectionPlan(ctx, v2AMPlan(ref, item))
			if err != nil {
				t.Fatal(err)
			}
			owner := GeneratedConfigOwnerOf(item.Desired)
			if err := ordinary.DeleteMonitorConfig(ctx, owner.ID); err != nil {
				t.Fatal(err)
			}
			if mode == "N/A replacement" {
				replacement := &model.MonitorConfig{PoolID: 99, ConfigType: 5, InstanceIP: owner.InstanceIP, Name: "custom", Status: 2}
				if err := ordinary.CreateMonitorConfig(ctx, replacement); err != nil {
					t.Fatal(err)
				}
			}
			reserve.Nonce = [16]byte{2}
			if mode == "cross pool history" {
				reserve.PoolID = 99
			}
			next, _ := dao.ReservePoolOperation(ctx, reserve)
			request := v2AMPlan(next)
			if mode == "fabricated owner alias" {
				owner.Name = "webhook_invented_3_127.0.0.1"
			}
			request.Controls[2].Intents.Add = []IntentOwnerChange{{Owner: owner, Proof: ProjectionProofRef{EventID: first.Artifacts[0].EventID}}}
			second, err := dao.CommitGeneratedProjectionPlan(ctx, request)
			if mode != "tombstone proof" {
				if err == nil {
					t.Fatal("unproven historic owner acquired retirement control")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			err = dao.WithPoolProjectionLease(ctx, second, func(lease ProjectionLease) error {
				snapshot, err := lease.Snapshot()
				if err != nil {
					return err
				}
				if len(snapshot.Pool.Controls) != 3 {
					t.Fatal("intent not persisted")
				}
				return lease.RecheckCurrent(ctx)
			})
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestProjectionV2CommittedLostACKLoadsExactPersistedControls(t *testing.T) {
	ordinary, _, db := journalBehaviorFixture(t)
	dao := NewProjectionPlanJournalDAO(zap.NewNop(), db)
	ctx := context.Background()
	ref, err := dao.ReservePoolOperation(ctx, journalBehaviorRequest(1))
	if err != nil {
		t.Fatal(err)
	}
	pool, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	wrapped := db.Session(&gorm.Session{NewDB: true})
	fault := &journalAckLossPool{DB: pool}
	fault.lose.Store(true)
	wrapped.ConnPool = fault
	wrapped.Statement.ConnPool = fault
	uncertain := NewProjectionPlanJournalDAO(zap.NewNop(), wrapped)
	item := journalBehaviorRender(11, "127.0.0.1")
	request := v2RecordPlan(ref, *item)
	if _, err := uncertain.CommitGeneratedProjectionPlan(ctx, request); !errors.Is(err, ErrJournalUncertain) {
		t.Fatal("lost commit ACK did not report uncertainty", err)
	}
	if item.Desired.ID != 0 {
		t.Fatal("uncertain commit leaked input ID")
	}
	snapshot, err := dao.LoadProjectionByNonce(ctx, ref.Nonce)
	if err != nil || snapshot.Projection == nil || !snapshot.Projection.CompleteControls || len(snapshot.Projection.Artifacts) != 1 {
		t.Fatal("lost ACK lost immutable controls", err)
	}
	recovered, err := dao.CommitGeneratedProjectionPlan(ctx, request)
	if err != nil || !equalProjectionPayload(recovered, *snapshot.Projection) || item.Desired.ID == 0 {
		t.Fatal("nonce recovery did not preserve receipt", err)
	}
	proof, _ := projectionAckProof(recovered)
	proof.RequiredControls[0].Digest[0] ^= 0xff
	if ok, err := dao.MarkProjectionCompleteIfCurrent(ctx, recovered, proof); ok || err == nil {
		t.Fatal("forged ACK proof was silently accepted")
	}
	var count int64
	if ordinary.db.Model(&model.MonitorConfig{}).Count(&count).Error != nil || count != 1 {
		t.Fatal("recovery duplicated metadata")
	}
}
