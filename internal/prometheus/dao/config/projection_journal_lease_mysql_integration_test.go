//go:build integration

package config

import (
	"context"
	"errors"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	driver "github.com/go-sql-driver/mysql"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestProjectionLeaseNativeMySQL(t *testing.T) {
	path := os.Getenv("CLOUDOPS_PROJECTION_LEASE_MYSQL_DSN_FILE")
	if path == "" {
		t.Skip("private projection-lease fixture not configured")
	}
	before, err := os.Lstat(path)
	if err != nil || !before.Mode().IsRegular() || before.Mode().Perm()&0077 != 0 || before.Size() > 16384 {
		t.Fatal("private DSN must be a small owner-only regular file")
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatal("private DSN unavailable")
	}
	after, err := file.Stat()
	if err != nil || !os.SameFile(before, after) {
		file.Close()
		t.Fatal("private DSN changed")
	}
	raw, err := io.ReadAll(io.LimitReader(file, 16385))
	file.Close()
	if err != nil || len(raw) > 16384 {
		t.Fatal("private DSN unreadable")
	}
	cfg, err := driver.ParseDSN(strings.TrimSpace(string(raw)))
	if err != nil {
		t.Fatal("private DSN invalid")
	}
	if cfg.Net != "tcp" || cfg.Addr != "127.0.0.1:13306" || !regexp.MustCompile(`^cloudops_projection_lease_[a-z0-9_]+$`).MatchString(cfg.DBName) || cfg.DBName != os.Getenv("CLOUDOPS_PROJECTION_LEASE_MYSQL_DATABASE") {
		t.Fatal("fixture requires literal loopback and separately confirmed new private schema")
	}
	cfg.ParseTime = true
	open := func() *gorm.DB {
		t.Helper()
		db, err := gorm.Open(mysql.Open(cfg.FormatDSN()), &gorm.Config{Logger: logger.Discard})
		if err != nil {
			t.Fatal("private fixture connection failed")
		}
		pool, err := db.DB()
		if err != nil {
			t.Fatal("private fixture pool unavailable")
		}
		pool.SetMaxOpenConns(1)
		t.Cleanup(func() { pool.Close() })
		return db
	}
	a, b := open(), open()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	var firstID, secondID int64
	if a.WithContext(ctx).Raw("SELECT CONNECTION_ID()").Scan(&firstID).Error != nil || b.WithContext(ctx).Raw("SELECT CONNECTION_ID()").Scan(&secondID).Error != nil || firstID == secondID {
		t.Fatal("fixture requires two independent real sessions")
	}
	var objects int64
	if a.Raw("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE()").Scan(&objects).Error != nil || objects != 0 {
		t.Fatal("private schema must initially be empty")
	}
	tables := []interface{}{&model.MonitorConfig{}, &model.MonitorConfigNamespaceGuard{}, &model.MonitorConfigPoolControl{}, &model.MonitorConfigPoolOperation{}, &model.MonitorConfigArtifactEvent{}}
	if err := a.AutoMigrate(tables...); err != nil {
		t.Fatal("private fixture migration failed")
	}
	defer func() {
		if err := a.Migrator().DropTable(tables...); err != nil {
			t.Error("private fixture cleanup failed")
		}
	}()
	ja, jb := NewProjectionPlanJournalDAO(zap.NewNop(), a), NewProjectionPlanJournalDAO(zap.NewNop(), b)
	ordinaryB := NewMonitorConfigDAO(zap.NewNop(), b)
	commit := func(t *testing.T, n byte, ip string) (ProjectionReceipt, *model.MonitorConfig) {
		t.Helper()
		ref, err := ja.ReservePoolOperation(ctx, journalBehaviorRequest(n))
		if err != nil {
			t.Fatal("reserve failed", err)
		}
		item := journalBehaviorRender(uint64(n), ip)
		receipt, err := ja.CommitGeneratedProjectionPlan(ctx, v2RecordPlan(ref, *item))
		if err != nil {
			t.Fatal("commit failed", err)
		}
		return receipt, item.Desired
	}
	t.Run("pool lease holds namespace through callback and failed next attempt preserves lastGood", func(t *testing.T) {
		receipt, row := commit(t, 1, "127.0.0.1")
		var escaped ProjectionLease
		done := make(chan error, 1)
		started := make(chan struct{})
		err := ja.WithPoolProjectionLease(ctx, receipt, func(lease ProjectionLease) error {
			escaped = lease
			snapshot, err := lease.Snapshot()
			if err != nil {
				return err
			}
			if len(snapshot.Artifacts) != 1 || string(snapshot.Artifacts[0].Body) != row.ConfigContent {
				t.Fatal("native material changed")
			}
			go func() {
				changed := *row
				changed.ConfigContent = "ordinary after lease"
				close(started)
				done <- ordinaryB.UpdateMonitorConfig(ctx, &changed)
			}()
			<-started
			select {
			case err := <-done:
				t.Fatalf("namespace writer escaped lease: %v", err)
			case <-time.After(150 * time.Millisecond):
			}
			return lease.RecheckCurrent(ctx)
		})
		if err != nil {
			t.Fatal("lease failed", err)
		}
		select {
		case err := <-done:
			if err != nil {
				t.Fatal("writer did not recover after lease", err)
			}
		case <-ctx.Done():
			t.Fatal("writer stuck after lease")
		}
		if escaped.RecheckCurrent(ctx) == nil {
			t.Fatal("expired native lease reused")
		}
		called := false
		if err := ja.WithPoolProjectionLease(ctx, receipt, func(ProjectionLease) error { called = true; return nil }); err == nil || called {
			t.Fatal("stale native lease acquired")
		}
	})
	t.Run("pool control serializes new reservation and empty full ACK persists", func(t *testing.T) {
		ref, err := ja.ReservePoolOperation(ctx, journalBehaviorRequest(2))
		if err != nil {
			t.Fatal(err)
		}
		receipt, err := ja.CommitGeneratedProjectionPlan(ctx, v2RecordPlan(ref))
		if err != nil {
			t.Fatal(err)
		}
		done := make(chan error, 1)
		started := make(chan struct{})
		err = ja.WithPoolProjectionLease(ctx, receipt, func(lease ProjectionLease) error {
			go func() { close(started); _, err := jb.ReservePoolOperation(ctx, journalBehaviorRequest(3)); done <- err }()
			<-started
			select {
			case err := <-done:
				t.Fatalf("reserve escaped pool lease: %v", err)
			case <-time.After(150 * time.Millisecond):
			}
			proof, _ := projectionAckProof(receipt)
			ok, err := lease.MarkComplete(proof)
			if err != nil || !ok {
				t.Fatal("empty ACK rejected", err)
			}
			tx := lease.(*sqlProjectionLease).tx
			var first, second model.MonitorConfigPoolOperation
			if err := tx.First(&first, "operation_id = ?", ref.OperationID).Error; err != nil {
				return err
			}
			ok, err = lease.MarkComplete(proof)
			if err != nil || !ok {
				t.Fatal("same ACK retry lost idempotence", err)
			}
			if err := tx.First(&second, "operation_id = ?", ref.OperationID).Error; err != nil {
				return err
			}
			if first.ProjectedAt == nil || second.ProjectedAt == nil || !first.ProjectedAt.Equal(*second.ProjectedAt) {
				t.Fatal("same ACK retry rewrote projected timestamp")
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		if err := <-done; err != nil {
			t.Fatal(err)
		}
		loaded, err := jb.LoadLastProjectedProjection(ctx, 4, 7)
		if err != nil || loaded == nil || loaded.Reservation != ref || !loaded.CompleteControls {
			t.Fatal("native empty controls lost", err)
		}
	})
	t.Run("ordinary namespace-only lease cannot render or complete pool", func(t *testing.T) {
		_, row := commit(t, 4, "127.0.0.4")
		changed := *row
		changed.ConfigContent = "ordinary native"
		if err := ordinaryB.UpdateMonitorConfig(ctx, &changed); err != nil {
			t.Fatal(err)
		}
		var event model.MonitorConfigArtifactEvent
		if err := a.WithContext(ctx).Where("config_type = ? AND instance_key = ?", 4, []byte(row.InstanceIP)).Order("event_id DESC").First(&event).Error; err != nil {
			t.Fatal(err)
		}
		artifact, err := artifactReceipt(event, 0)
		if err != nil {
			t.Fatal(err)
		}
		err = ja.WithArtifactProjectionLease(ctx, artifact, func(lease ProjectionLease) error {
			snapshot, err := lease.Snapshot()
			if err != nil {
				return err
			}
			if snapshot.Pool != nil || len(snapshot.Artifacts) != 1 || len(snapshot.Artifacts[0].Body) != 0 {
				t.Fatal("ordinary native authority expanded")
			}
			return lease.RecheckCurrent(ctx)
		})
		if err != nil {
			t.Fatal(err)
		}
		err = ja.WithArtifactProjectionLease(ctx, artifact, func(lease ProjectionLease) error {
			if ok, err := lease.MarkComplete(ProjectionAckProof{}); ok || err == nil {
				t.Fatal("ordinary lease marked full pool")
			}
			if _, err := lease.Snapshot(); err == nil {
				t.Fatal("failed ordinary completion did not seal native handle")
			}
			return nil
		})
		if err == nil {
			t.Fatal("failed ordinary completion reported native lease success")
		}
	})
	t.Run("caller cancellation seals handle and preserves unprojected receipt", func(t *testing.T) {
		ref, err := ja.ReservePoolOperation(ctx, journalBehaviorRequest(5))
		if err != nil {
			t.Fatal(err)
		}
		receipt, err := ja.CommitGeneratedProjectionPlan(ctx, v2RecordPlan(ref))
		if err != nil {
			t.Fatal(err)
		}
		child, cancel := context.WithCancel(ctx)
		var escaped ProjectionLease
		err = ja.WithPoolProjectionLease(child, receipt, func(lease ProjectionLease) error { escaped = lease; cancel(); return lease.RecheckCurrent(child) })
		if !errors.Is(err, context.Canceled) {
			t.Fatal("cancel identity lost", err)
		}
		if _, err := escaped.Snapshot(); err == nil {
			t.Fatal("canceled handle stayed live")
		}
		loaded, err := jb.LoadLastCommittedProjection(ctx, 4, 7)
		if err != nil || loaded == nil || loaded.Phase != model.ConfigPublicationCommitted {
			t.Fatal("cancel changed committed business state", err)
		}
	})
	t.Run("detected connection loss seals original lease without reconnect", func(t *testing.T) {
		ref, err := ja.ReservePoolOperation(ctx, journalBehaviorRequest(6))
		if err != nil {
			t.Fatal(err)
		}
		receipt, err := ja.CommitGeneratedProjectionPlan(ctx, v2RecordPlan(ref))
		if err != nil {
			t.Fatal(err)
		}
		entered := false
		err = ja.WithPoolProjectionLease(ctx, receipt, func(handle ProjectionLease) error {
			entered = true
			lease := handle.(*sqlProjectionLease)
			var ownID, otherID int64
			if err := lease.tx.Raw("SELECT CONNECTION_ID()").Scan(&ownID).Error; err != nil {
				return err
			}
			if err := b.WithContext(ctx).Raw("SELECT CONNECTION_ID()").Scan(&otherID).Error; err != nil {
				return err
			}
			if ownID <= 0 || ownID == otherID {
				t.Fatal("kill target is not the dedicated lease session")
			}
			// Both sessions belong to this new empty-schema fixture's dedicated user.
			if err := b.WithContext(ctx).Exec("KILL CONNECTION " + strconv.FormatInt(ownID, 10)).Error; err != nil {
				t.Fatal("could not terminate own fixture session", err)
			}
			detected := handle.RecheckCurrent(ctx)
			if detected == nil {
				t.Fatal("dead lease silently reconnected")
			}
			if _, err := handle.Snapshot(); err == nil {
				t.Fatal("known-dead lease retained snapshot authority")
			}
			proof, _ := projectionAckProof(receipt)
			if ok, err := handle.MarkComplete(proof); ok || err == nil {
				t.Fatal("known-dead lease retained completion authority")
			}
			return detected
		})
		if !entered || err == nil {
			t.Fatal("connection loss did not fail original lease")
		}
		loaded, err := jb.LoadLastCommittedProjection(ctx, 4, 7)
		if err != nil || loaded == nil || !equalProjectionPayload(*loaded, receipt) || loaded.Phase != model.ConfigPublicationCommitted {
			t.Fatal("connection loss discarded committed business facts", err)
		}
	})

}
