//go:build integration

package config

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	mysqlDriver "github.com/go-sql-driver/mysql"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func recoveryNativeRequest(t *testing.T, family int8, pool int) PoolOperationRequest {
	t.Helper()
	req := PoolOperationRequest{Family: family, PoolID: pool}
	if _, err := rand.Read(req.Nonce[:]); err != nil {
		t.Fatal("native nonce unavailable")
	}
	return req
}
func recoveryNativeCommitEmpty(t *testing.T, db *gorm.DB, family int8, pool int) ProjectionReceipt {
	t.Helper()
	journal := NewProjectionPlanJournalDAO(zap.NewNop(), db)
	ref, err := journal.ReservePoolOperation(context.Background(), recoveryNativeRequest(t, family, pool))
	if err != nil {
		t.Fatal("native empty reserve failed")
	}
	result, err := journal.CommitGeneratedProjectionPlan(context.Background(), v2RecordPlan(ref))
	if err != nil {
		t.Fatal("native empty commit failed")
	}
	return result
}
func recoveryNativeAMPlan(ref PoolReservation) GeneratedProjectionPlanRequest {
	if ref.Family == 2 {
		return v2AMPlan(ref)
	}
	return GeneratedProjectionPlanRequest{GeneratedPlanRequest: GeneratedPlanRequest{Reservation: ref, Items: []GeneratedPlanItem{}}, Controls: []ProjectionControl{{ControlID: 3, Kind: ProjectionACKWebhookSubset, Ack: &ProjectionAckControl{ArtifactSourceIDs: []uint64{}, RequiredControlIDs: []uint64{}}}}}
}
func recoveryNativeAMCommit(t *testing.T, db *gorm.DB, family int8, pool int) ProjectionReceipt {
	t.Helper()
	ctx := context.Background()
	dao := NewAMProjectionJournalDAO(zap.NewNop(), db)
	source, err := NewProjectionRecoveryJournalDAO(zap.NewNop(), db).ReadAMPoolSourceVersion(ctx, pool)
	if err != nil {
		t.Fatal("native AM source read failed")
	}
	ref, err := dao.ReserveAMPoolOperation(ctx, recoveryNativeRequest(t, family, pool), source)
	if err != nil {
		t.Fatal("native AM reserve failed")
	}
	result, err := dao.CommitAMProjectionPlan(ctx, AMProjectionPlanRequest{GeneratedProjectionPlanRequest: recoveryNativeAMPlan(ref), SourceVersion: source})
	if err != nil {
		t.Fatal("native AM commit failed")
	}
	return result
}
func runRecoveryMySQLCases(t *testing.T, f *recoveryOwnedMySQL) {
	t.Run("real keyset highwater poison manifest and reserved last-good", func(t *testing.T) {
		ctx := context.Background()
		ja := NewProjectionPlanJournalDAO(zap.NewNop(), f.a)
		reader := NewProjectionRecoveryJournalDAO(zap.NewNop(), f.b)
		old := recoveryNativeCommitEmpty(t, f.a, 4, 1001)
		current := recoveryNativeCommitEmpty(t, f.a, 4, 1001)
		projected := recoveryNativeCommitEmpty(t, f.a, 3, 1002)
		proof, err := projectionAckProof(projected)
		if err != nil {
			t.Fatal("native SQL proof unavailable")
		}
		if marked, err := ja.MarkProjectionCompleteIfCurrent(ctx, projected, proof); err != nil || !marked {
			t.Fatal("SQL state-only marker failed")
		}
		t.Log("SQL projected marker tested without Redis; this is not external projection evidence")
		if _, err := ja.ReservePoolOperation(ctx, recoveryNativeRequest(t, 4, 1001)); err != nil {
			t.Fatal("native failed-attempt reserve failed")
		}
		high, err := reader.CurrentProjectionRecoveryHighWater(ctx)
		if err != nil {
			t.Fatal("native highwater unavailable")
		}
		late := recoveryNativeCommitEmpty(t, f.a, 1, 1003)
		page, err := reader.ListCurrentUnprojectedPoolOperations(ctx, 0, high, 1)
		if err != nil || len(page) != 1 || page[0] != current.Reservation {
			t.Fatal("current last-good was hidden by history, projected or reserved state")
		}
		page, err = reader.ListCurrentUnprojectedPoolOperations(ctx, current.Reservation.OperationID, high, 1)
		if err != nil || len(page) != 0 {
			t.Fatal("round exceeded its real SQL highwater")
		}
		// JSON itself is valid for MySQL, while the persisted journal envelope is not.
		if f.a.Model(&model.MonitorConfigPoolOperation{}).Where("operation_id=?", current.Reservation.OperationID).UpdateColumn("manifest", `{"invalid":"synthetic-poison"}`).Error != nil {
			t.Fatal("cannot prepare valid-JSON corrupt envelope")
		}
		page, err = reader.ListCurrentUnprojectedPoolOperations(ctx, 0, late.Reservation.OperationID, 1)
		if err != nil || len(page) != 1 || page[0] != current.Reservation {
			t.Fatal("poison locator prevented bounded enumeration")
		}
		if _, err := ja.LoadLastCommittedProjection(ctx, 4, 1001); err == nil {
			t.Fatal("poison payload acquired usable authority")
		}
		next, err := reader.ListCurrentUnprojectedPoolOperations(ctx, page[0].OperationID, late.Reservation.OperationID, 1)
		if err != nil || len(next) != 1 || next[0] != late.Reservation {
			t.Fatal("poison pool starved later healthy candidate")
		}
		if old.Reservation.OperationID >= current.Reservation.OperationID {
			t.Fatal("native fixture did not establish monotone IDs")
		}
	})
	for _, family := range []int8{1, 3, 4} {
		t.Run(fmt.Sprintf("source counter unchanged but last-committed replaced family %d", family), func(t *testing.T) {
			ctx := context.Background()
			pool := 2000 + int(family)
			ja, jb := NewProjectionPlanJournalDAO(zap.NewNop(), f.a), NewProjectionPlanJournalDAO(zap.NewNop(), f.b)
			first := recoveryNativeCommitEmpty(t, f.a, family, pool)
			bRef, err := jb.ReservePoolOperation(ctx, recoveryNativeRequest(t, family, pool))
			if err != nil {
				t.Fatal("B reservation failed")
			}
			counter, err := NewProjectionRecoveryJournalDAO(zap.NewNop(), f.a).ReadPoolProjectionSourceVersion(ctx, family, pool)
			if err != nil || counter != bRef.Generation {
				t.Fatal("A did not read pre-reserved B counter")
			}
			started, release := make(chan struct{}), make(chan struct{})
			done := make(chan error, 1)
			request := recoveryNativeRequest(t, family, pool)
			guard := first.Reservation.OperationID
			request.ExpectedLastCommittedOperationID = &guard
			request.RecoveryOfOperationID = &guard
			go func() {
				close(started)
				<-release
				var err error
				switch family {
				case 1:
					_, err = NewPromMainProjectionJournalDAO(zap.NewNop(), f.a).ReservePromMainPoolOperation(ctx, request, counter)
				case 3:
					_, err = NewAlertRuleProjectionJournalDAO(zap.NewNop(), f.a).ReserveAlertRulePoolOperation(ctx, request, counter)
				case 4:
					_, err = NewRecordProjectionJournalDAO(zap.NewNop(), f.a).ReserveRecordPoolOperation(ctx, request, counter)
				}
				done <- err
			}()
			<-started
			second, err := jb.CommitGeneratedProjectionPlan(ctx, v2RecordPlan(bRef))
			close(release)
			reserveErr := <-done
			if err != nil || !errors.Is(reserveErr, ErrJournalConflict) {
				t.Fatal("old recovery re-reserved after B committed without a counter increment")
			}
			current, err := ja.LoadLastCommittedProjection(ctx, family, pool)
			after, readErr := NewProjectionRecoveryJournalDAO(zap.NewNop(), f.a).ReadPoolProjectionSourceVersion(ctx, family, pool)
			if err != nil || current == nil || current.Reservation != second.Reservation || readErr != nil || after != counter {
				t.Fatal("rejected reservation changed B or its permanent counter")
			}
		})
	}
	t.Run("commit guard and known committed retry", func(t *testing.T) {
		ctx := context.Background()
		ja := NewProjectionPlanJournalDAO(zap.NewNop(), f.a)
		first := recoveryNativeCommitEmpty(t, f.a, 4, 2100)
		second := recoveryNativeCommitEmpty(t, f.b, 4, 2100)
		req := recoveryNativeRequest(t, 4, 2100)
		expected := second.Reservation.OperationID
		req.ExpectedLastCommittedOperationID = &expected
		req.RecoveryOfOperationID = &expected
		ref, err := ja.ReservePoolOperation(ctx, req)
		if err != nil {
			t.Fatal("guarded reserve failed")
		}
		plan := v2RecordPlan(ref)
		plan.ExpectedLastCommittedOperationID = &expected
		// Deliberately restore another valid committed pointer while preserving this
		// exact reserved attempt: isolate Commit's guard from the independent counter.
		if f.b.Model(&model.MonitorConfigPoolControl{}).Where("family=4 AND pool_id=2100").UpdateColumn("last_committed_operation_id", first.Reservation.OperationID).Error != nil {
			t.Fatal("guard fixture pointer change failed")
		}
		if _, err := ja.CommitGeneratedProjectionPlan(ctx, plan); !errors.Is(err, ErrJournalConflict) {
			t.Fatal("commit ignored changed last-committed pointer")
		}
		if f.b.Model(&model.MonitorConfigPoolControl{}).Where("family=4 AND pool_id=2100").UpdateColumn("last_committed_operation_id", second.Reservation.OperationID).Error != nil {
			t.Fatal("guard fixture pointer restore failed")
		}
		receipt, err := ja.CommitGeneratedProjectionPlan(ctx, plan)
		if err != nil {
			t.Fatal("restored guarded commit failed")
		}
		again, err := ja.CommitGeneratedProjectionPlan(ctx, plan)
		if err != nil || !equalProjectionPayload(receipt, again) {
			t.Fatal("own known committed retry was not idempotent")
		}
	})
	for _, family := range []int8{2, 5} {
		t.Run(fmt.Sprintf("AM peer source and failed-reserved last-good family %d", family), func(t *testing.T) {
			ctx := context.Background()
			pool := 3000 + int(family)
			peer := int8(7 - family)
			a, b := NewAMProjectionJournalDAO(zap.NewNop(), f.a), NewAMProjectionJournalDAO(zap.NewNop(), f.b)
			own := recoveryNativeAMCommit(t, f.a, family, pool)
			source, err := NewProjectionRecoveryJournalDAO(zap.NewNop(), f.a).ReadAMPoolSourceVersion(ctx, pool)
			if err != nil {
				t.Fatal("AM source read failed")
			}
			peerRef, err := b.ReserveAMPoolOperation(ctx, recoveryNativeRequest(t, peer, pool), source)
			if err != nil {
				t.Fatal("AM peer reserve failed")
			}
			if err := a.WithPoolProjectionLease(ctx, own, func(lease ProjectionLease) error { return lease.RecheckCurrent(ctx) }); err != nil {
				t.Fatal("reserved peer starved committed last-good")
			}
			peerReceipt, err := b.CommitAMProjectionPlan(ctx, AMProjectionPlanRequest{GeneratedProjectionPlanRequest: recoveryNativeAMPlan(peerRef), SourceVersion: source})
			if err != nil {
				t.Fatal("AM peer commit failed")
			}
			entered := false
			if err := a.WithPoolProjectionLease(ctx, own, func(ProjectionLease) error { entered = true; return nil }); err == nil || entered {
				t.Fatal("old AM pair acquired authority after peer committed")
			}
			if _, err := a.ReserveAMPoolOperation(ctx, recoveryNativeRequest(t, family, pool), source); !errors.Is(err, ErrJournalConflict) {
				t.Fatal("old AM pair source reserved new work")
			}
			if err := b.WithPoolProjectionLease(ctx, peerReceipt, func(lease ProjectionLease) error { return lease.RecheckCurrent(ctx) }); err != nil {
				t.Fatal("new peer receipt was unusable")
			}
		})
		t.Run(fmt.Sprintf("AM 2 then 5 control locks deadline and retry family %d", family), func(t *testing.T) { runRecoveryMySQLAMLock(t, f, family) })
	}
	t.Run("KILL original lease seals material and completion after another SQL writer", func(t *testing.T) { runRecoveryMySQLKilledLease(t, f) })
}

func recoveryMySQLSafeErrorClass(err error) string {
	switch {
	case err == nil:
		return "nil"
	case errors.Is(err, context.DeadlineExceeded):
		return "deadline"
	case errors.Is(err, context.Canceled):
		return "canceled"
	case errors.Is(err, ErrJournalConflict):
		return "journal_conflict"
	case errors.Is(err, ErrJournalInvalid):
		return "journal_invalid"
	}
	var server *mysqlDriver.MySQLError
	if errors.As(err, &server) {
		return "mysql_code_" + strconv.Itoa(int(server.Number))
	}
	return fmt.Sprintf("type_%T", err)
}

func runRecoveryMySQLAMLock(t *testing.T, f *recoveryOwnedMySQL, family int8) {
	t.Helper()
	ctx := context.Background()
	pool := 4000 + int(family)
	peer := int8(7 - family)
	a, b := NewAMProjectionJournalDAO(zap.NewNop(), f.a), NewAMProjectionJournalDAO(zap.NewNop(), f.b)
	receipt := recoveryNativeAMCommit(t, f.a, family, pool)
	source, err := NewProjectionRecoveryJournalDAO(zap.NewNop(), f.b).ReadAMPoolSourceVersion(ctx, pool)
	if err != nil {
		t.Fatal("AM lock source unavailable")
	}
	var bid, aid uint64
	if f.b.Raw("SELECT CONNECTION_ID()").Scan(&bid).Error != nil {
		t.Fatal("waiting runtime session unavailable")
	}
	order := []string{}
	callback := "owned_recovery:lock_order_" + strconv.Itoa(int(family))
	if f.a.Callback().Query().After("gorm:query").Register(callback, func(tx *gorm.DB) {
		if tx.Statement.Table == "cl_monitor_config_pool_controls" && strings.Contains(tx.Statement.SQL.String(), "FOR UPDATE") && len(tx.Statement.Vars) >= 2 {
			order = append(order, fmt.Sprint(tx.Statement.Vars[0]))
		}
	}) != nil {
		t.Fatal("lock-order observer unavailable")
	}
	defer f.a.Callback().Query().Remove(callback)
	stage := "lease_entry"
	innodbCount, edgeCount, polls := 0, 0, 0
	innodbClass, edgeClass, reserveClass := "not_queried", "not_queried", "not_returned"
	threadState := "not_queried"
	err = a.WithPoolProjectionLease(ctx, receipt, func(lease ProjectionLease) error {
		stage = "order"
		if len(order) < 2 || !reflect.DeepEqual(order[:2], []string{"2", "5"}) {
			return errors.New("AM SQL control lock order was not 2 then 5")
		}
		if lease.(*sqlProjectionLease).tx.Raw("SELECT CONNECTION_ID()").Scan(&aid).Error != nil {
			return errors.New("original lease session unavailable")
		}
		waiting, waitingCancel := context.WithTimeout(ctx, 1200*time.Millisecond)
		defer waitingCancel()
		started := make(chan struct{})
		done := make(chan error, 1)
		request := recoveryNativeRequest(t, peer, pool)
		go func() { close(started); _, err := b.ReserveAMPoolOperation(waiting, request, source); done <- err }()
		<-started
		// A positive Performance Schema edge must identify both exact runtime
		// sessions and the WAITING/GRANTED records in this owned control table.
		// INNODB_TRX is diagnostic only: its snapshot can disagree with P_S.
		poll := time.NewTicker(10 * time.Millisecond)
		defer poll.Stop()
		stage = "lock_wait_observation"
		observed := false
		for !observed {
			polls++
			queryErr := f.admin.QueryRowContext(waiting, "SELECT COUNT(*) FROM information_schema.innodb_trx WHERE trx_mysql_thread_id=? AND trx_state='LOCK WAIT'", bid).Scan(&innodbCount)
			innodbClass = recoveryMySQLSafeErrorClass(queryErr)
			const exactControlWait = `SELECT COUNT(*)
FROM performance_schema.data_lock_waits AS w
JOIN performance_schema.data_locks AS requested
  ON requested.ENGINE=w.ENGINE
 AND requested.ENGINE_LOCK_ID=w.REQUESTING_ENGINE_LOCK_ID
JOIN performance_schema.data_locks AS blocker
  ON blocker.ENGINE=w.ENGINE
 AND blocker.ENGINE_LOCK_ID=w.BLOCKING_ENGINE_LOCK_ID
JOIN performance_schema.threads AS requesting_thread
  ON requesting_thread.THREAD_ID=w.REQUESTING_THREAD_ID
 AND requesting_thread.THREAD_ID=requested.THREAD_ID
JOIN performance_schema.threads AS blocking_thread
  ON blocking_thread.THREAD_ID=w.BLOCKING_THREAD_ID
 AND blocking_thread.THREAD_ID=blocker.THREAD_ID
WHERE requesting_thread.PROCESSLIST_ID=?
  AND blocking_thread.PROCESSLIST_ID=?
  AND requested.OBJECT_SCHEMA=? AND blocker.OBJECT_SCHEMA=?
  AND requested.OBJECT_NAME='cl_monitor_config_pool_controls'
  AND blocker.OBJECT_NAME='cl_monitor_config_pool_controls'
  AND requested.LOCK_TYPE='RECORD' AND blocker.LOCK_TYPE='RECORD'
  AND requested.LOCK_STATUS='WAITING' AND blocker.LOCK_STATUS='GRANTED'`
			queryErr = f.admin.QueryRowContext(waiting, exactControlWait, bid, aid, f.schema, f.schema).Scan(&edgeCount)
			edgeClass = recoveryMySQLSafeErrorClass(queryErr)
			if queryErr == nil && edgeCount > 0 {
				observed = true
				break
			}
			var state string
			queryErr = f.admin.QueryRowContext(waiting, "SELECT COALESCE(PROCESSLIST_STATE,'') FROM performance_schema.threads WHERE PROCESSLIST_ID=?", bid).Scan(&state)
			if queryErr == nil {
				threadState = state
			} else {
				threadState = recoveryMySQLSafeErrorClass(queryErr)
			}
			select {
			case reserveErr := <-done:
				reserveClass = recoveryMySQLSafeErrorClass(reserveErr)
				return errors.New("reserve returned before a server LOCK WAIT was observed")
			case <-waiting.Done():
				reserveClass = recoveryMySQLSafeErrorClass(<-done)
				return errors.New("native runtime did not establish an observed SQL lock wait")
			case <-poll.C:
			}
		}
		stage = "reserve_deadline"
		reserveErr := <-done
		reserveClass = recoveryMySQLSafeErrorClass(reserveErr)
		if !errors.Is(reserveErr, context.DeadlineExceeded) {
			return errors.New("blocked AM reserve did not honor its deadline")
		}
		stage = "lease_recheck"
		return lease.RecheckCurrent(ctx)
	})
	if err != nil {
		t.Fatalf("native AM ordered lock/deadline failed: stage=%s order=%v lease_id=%d peer_id=%d polls=%d innodb_waits=%d innodb_error=%s matched_edges=%d edge_error=%s peer_state=%q reserve=%s outer=%s", stage, order, aid, bid, polls, innodbCount, innodbClass, edgeCount, edgeClass, threadState, reserveClass, recoveryMySQLSafeErrorClass(err))
	}
	refreshed, err := NewProjectionRecoveryJournalDAO(zap.NewNop(), f.b).ReadAMPoolSourceVersion(ctx, pool)
	if err != nil {
		t.Fatal("post-cancel peer source read failed")
	}
	if _, err := b.ReserveAMPoolOperation(ctx, recoveryNativeRequest(t, peer, pool), refreshed); err != nil {
		t.Fatal("peer failed to recover after real lock release")
	}
}

func runRecoveryMySQLKilledLease(t *testing.T, f *recoveryOwnedMySQL) {
	t.Helper()
	ctx := context.Background()
	a := NewProjectionPlanJournalDAO(zap.NewNop(), f.a)
	ref, err := a.ReservePoolOperation(ctx, recoveryNativeRequest(t, 4, 7))
	if err != nil {
		t.Fatal("KILL fixture reserve failed")
	}
	item := journalBehaviorRender(1, "127.0.0.91")
	receipt, err := a.CommitGeneratedProjectionPlan(ctx, v2RecordPlan(ref, *item))
	if err != nil {
		t.Fatal("KILL fixture commit failed")
	}
	entered := false
	err = a.WithPoolProjectionLease(ctx, receipt, func(handle ProjectionLease) error {
		entered = true
		lease := handle.(*sqlProjectionLease)
		var target, adminID uint64
		if lease.tx.Raw("SELECT CONNECTION_ID()").Scan(&target).Error != nil || f.admin.QueryRowContext(ctx, "SELECT CONNECTION_ID()").Scan(&adminID) != nil || target == 0 || target == adminID {
			return errors.New("dedicated original lease identity unavailable")
		}
		var database, user string
		if f.admin.QueryRowContext(ctx, "SELECT DB, USER FROM information_schema.PROCESSLIST WHERE ID=?", target).Scan(&database, &user) != nil || database != f.schema || user != f.user {
			return errors.New("refusing KILL outside the exact owned schema")
		}
		if _, err := f.admin.ExecContext(ctx, "KILL CONNECTION "+strconv.FormatUint(target, 10)); err != nil {
			return errors.New("cannot kill the captured owned lease connection")
		}
		ordinary := NewMonitorConfigDAO(zap.NewNop(), f.b)
		changed, err := ordinary.GetMonitorConfigByID(ctx, item.Desired.ID)
		if err != nil {
			return errors.New("replacement writer could not read live row")
		}
		changed.ConfigContent = "groups: [] # independent writer after KILL\n"
		if ordinary.UpdateMonitorConfig(ctx, changed) != nil {
			return errors.New("independent runtime could not commit after lease KILL")
		}
		detected := handle.RecheckCurrent(context.Background())
		if detected == nil {
			return errors.New("killed lease reconnected silently")
		}
		if _, err := handle.Snapshot(); err == nil {
			return errors.New("known-dead lease retained snapshot authority")
		}
		proof, err := projectionAckProof(receipt)
		if err != nil {
			return err
		}
		if ok, err := handle.MarkComplete(proof); ok || err == nil {
			return errors.New("known-dead lease marked complete")
		}
		fresh, freshCancel := context.WithTimeout(context.Background(), time.Second)
		defer freshCancel()
		if handle.RecheckCurrent(fresh) == nil {
			return errors.New("fresh context renewed a dead original lease")
		}
		return detected
	})
	if !entered || err == nil {
		t.Fatal("original lease connection loss was not rejected")
	}
	renewed := false
	if err := a.WithPoolProjectionLease(ctx, receipt, func(ProjectionLease) error { renewed = true; return nil }); err == nil || renewed {
		t.Fatal("a fresh SQL connection renewed the obsolete pre-KILL receipt")
	}
	current, err := NewMonitorConfigDAO(zap.NewNop(), f.b).GetMonitorConfigByID(ctx, item.Desired.ID)
	if err != nil || current.ConfigContent != "groups: [] # independent writer after KILL\n" {
		t.Fatal("dead lease changed the new SQL writer content")
	}
	loaded, err := NewProjectionPlanJournalDAO(zap.NewNop(), f.b).LoadLastCommittedProjection(ctx, 4, 7)
	if err != nil || loaded == nil || loaded.Phase != model.ConfigPublicationCommitted {
		t.Fatal("KILL falsely projected or discarded the committed operation")
	}
}
