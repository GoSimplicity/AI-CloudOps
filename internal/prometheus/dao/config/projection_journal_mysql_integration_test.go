//go:build integration

package config

import (
	"context"
	"crypto/sha256"
	"errors"
	"io"
	"math"
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

func TestProjectionJournalNativeMySQL(t *testing.T) {
	path := os.Getenv("CLOUDOPS_PROJECTION_JOURNAL_MYSQL_DSN_FILE")
	if path == "" {
		t.Skip("private projection-journal fixture not configured")
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
	if cfg.Net != "tcp" || cfg.Addr != "127.0.0.1:13306" || !regexp.MustCompile(`^cloudops_projection_journal_[a-z0-9_]+$`).MatchString(cfg.DBName) || cfg.DBName != os.Getenv("CLOUDOPS_PROJECTION_JOURNAL_MYSQL_DATABASE") {
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
	ja, jb := NewProjectionJournalDAO(zap.NewNop(), a), NewProjectionJournalDAO(zap.NewNop(), b)
	request := func(pool int, n byte) PoolOperationRequest {
		return PoolOperationRequest{Family: 4, PoolID: pool, Nonce: [16]byte{n}}
	}
	render := func(pool int, id uint64, ip string) GeneratedPlanItem {
		return GeneratedPlanItem{SourceID: id, Action: model.ConfigProjectionRender, Desired: &model.MonitorConfig{PoolID: pool, ConfigType: 4, InstanceIP: ip, Name: "prometheus_record_rule_" + strconv.Itoa(pool) + "_" + ip + ".yaml", ConfigContent: "groups: []\n", Status: 1}, Binding: &ProjectionBinding{Version: 1, Role: "artifact", ConfigType: 4, InstanceIP: ip}}
	}

	t.Run("two sessions allocate unique generations and idempotent nonce", func(t *testing.T) {
		type result struct {
			ref PoolReservation
			err error
		}
		start := make(chan struct{})
		left, right := make(chan result, 1), make(chan result, 1)
		go func() { <-start; ref, err := ja.ReservePoolOperation(ctx, request(7, 1)); left <- result{ref, err} }()
		go func() { <-start; ref, err := jb.ReservePoolOperation(ctx, request(7, 2)); right <- result{ref, err} }()
		close(start)
		x, y := <-left, <-right
		if x.err != nil || y.err != nil || x.ref.Generation == y.ref.Generation || x.ref.Generation+y.ref.Generation != 3 {
			t.Fatal("permanent counter failed concurrent reservation")
		}
		same := request(9, 3)
		start = make(chan struct{})
		go func() { <-start; ref, err := ja.ReservePoolOperation(ctx, same); left <- result{ref, err} }()
		go func() { <-start; ref, err := jb.ReservePoolOperation(ctx, same); right <- result{ref, err} }()
		close(start)
		x, y = <-left, <-right
		if x.err != nil || y.err != nil || x.ref != y.ref || x.ref.Generation != 1 {
			t.Fatal("concurrent nonce allocated multiple slots")
		}
	})
	t.Run("nonempty JSON receipt and ordinary lineage", func(t *testing.T) {
		ref, err := ja.ReservePoolOperation(ctx, request(17, 4))
		if err != nil {
			t.Fatal(err)
		}
		item := render(17, 11, "2001:DB8::1")
		receipt, err := ja.CommitGeneratedPlan(ctx, GeneratedPlanRequest{Reservation: ref, SourceHash: sha256.Sum256([]byte("native source")), Items: []GeneratedPlanItem{item}})
		if err != nil || item.Desired.ID == 0 {
			t.Fatal("native committed receipt missing", err)
		}
		loaded, err := jb.LoadOperationByNonce(ctx, ref.Nonce)
		if err != nil || loaded.Receipt == nil || len(loaded.Receipt.Artifacts) != 1 || loaded.Receipt.Artifacts[0].SourceID != 11 || loaded.Receipt.Artifacts[0].Binding.InstanceIP != "2001:DB8::1" {
			t.Fatal("MySQL JSON normalization changed receipt")
		}
		ordinary := NewMonitorConfigDAO(zap.NewNop(), b)
		if err := ordinary.UpdateMonitorConfig(ctx, &model.MonitorConfig{Model: model.Model{ID: item.Desired.ID}, Name: "native custom owner"}); err != nil {
			t.Fatal(err)
		}
		var latest model.MonitorConfigArtifactEvent
		if b.Where("config_type=4 AND instance_key=?", []byte("2001:DB8::1")).Order("event_id DESC").First(&latest).Error != nil || latest.ProjectionPolicy != model.ConfigProjectionLineageOnly || latest.BindingEventID == nil || *latest.BindingEventID != receipt.Artifacts[0].EventID {
			t.Fatal("native ordinary lineage not recorded")
		}
		if ok, err := ja.MarkArtifactDeliveryIfCurrent(ctx, receipt.Artifacts[0], model.ConfigProjectionApplied); err != nil || ok {
			t.Fatal("native stale vector was acknowledged")
		}
	})
	t.Run("empty lastGood survives failed latest and new recovery", func(t *testing.T) {
		first, err := ja.ReservePoolOperation(ctx, request(31, 5))
		if err != nil {
			t.Fatal(err)
		}
		hash := sha256.Sum256([]byte("empty"))
		receipt, err := ja.CommitGeneratedPlan(ctx, GeneratedPlanRequest{Reservation: first, SourceHash: hash})
		if err != nil {
			t.Fatal(err)
		}
		if ok, err := ja.MarkPoolProjectedIfCurrent(ctx, receipt); err != nil || !ok {
			t.Fatal("empty receipt ACK rejected")
		}
		second, err := jb.ReservePoolOperation(ctx, request(31, 6))
		if err != nil {
			t.Fatal(err)
		}
		cancelled, stop := context.WithCancel(ctx)
		stop()
		if _, err := jb.CommitGeneratedPlan(cancelled, GeneratedPlanRequest{Reservation: second, SourceHash: hash}); !errors.Is(err, context.Canceled) {
			t.Fatal("native cancellation lost")
		}
		if ok, err := jb.MarkPoolOperationFailed(ctx, second, "cancelled"); err != nil || !ok {
			t.Fatal("reserved failure transition rejected")
		}
		last, err := jb.LoadLastCommitted(ctx, 4, 31)
		if err != nil || last == nil || last.Reservation != first {
			t.Fatal("failed attempt covered last committed empty operation")
		}
		recoverRequest := request(31, 7)
		recoverRequest.RecoveryOfOperationID = &first.OperationID
		third, err := jb.ReservePoolOperation(ctx, recoverRequest)
		if err != nil || third.Generation != 3 {
			t.Fatal("recovery counter did not advance")
		}
		recovered, err := jb.CommitGeneratedPlan(ctx, GeneratedPlanRequest{Reservation: third, SourceHash: hash})
		if err != nil {
			t.Fatal(err)
		}
		if ok, err := jb.MarkPoolProjectedIfCurrent(ctx, recovered); err != nil || !ok {
			t.Fatal("native empty recovery ACK rejected")
		}
		var notNull bool
		var length int
		if a.Raw("SELECT manifest IS NOT NULL,JSON_LENGTH(manifest) FROM cl_monitor_config_pool_operations WHERE operation_id=?", third.OperationID).Row().Scan(&notNull, &length) != nil || !notNull || length != 0 {
			t.Fatal("committed [] was confused with reserved NULL")
		}
	})
	t.Run("late actual SQL constraint rolls metadata events and receipt back", func(t *testing.T) {
		ref, err := ja.ReservePoolOperation(ctx, request(71, 8))
		if err != nil {
			t.Fatal(err)
		}
		if a.Exec("ALTER TABLE cl_monitor_config_pool_operations ADD CONSTRAINT journal_fixture_manifest CHECK (pool_id <> 71 OR manifest IS NULL)").Error != nil {
			t.Fatal("native late constraint fixture failed")
		}
		defer a.Exec("ALTER TABLE cl_monitor_config_pool_operations DROP CHECK journal_fixture_manifest")
		insertedRows, insertedEvents := 0, 0
		actualConstraint := false
		name := "fixture:journal-progress"
		a.Callback().Create().After("gorm:create").Register(name, func(tx *gorm.DB) {
			if tx.Error == nil && tx.Statement.Schema != nil && tx.RowsAffected == 1 {
				switch tx.Statement.Schema.Table {
				case "cl_monitor_configs":
					insertedRows++
				case "cl_monitor_config_artifact_events":
					insertedEvents++
				}
			}
		})
		a.Callback().Update().After("gorm:update").Register(name, func(tx *gorm.DB) {
			var sqlErr *driver.MySQLError
			if errors.As(tx.Error, &sqlErr) && sqlErr.Number == 3819 {
				actualConstraint = true
			}
		})
		defer a.Callback().Create().Remove(name)
		defer a.Callback().Update().Remove(name)
		item := render(71, 11, "127.0.0.71")
		receipt, err := ja.CommitGeneratedPlan(ctx, GeneratedPlanRequest{Reservation: ref, SourceHash: sha256.Sum256([]byte("rollback")), Items: []GeneratedPlanItem{item}})
		if err == nil || !actualConstraint || insertedRows != 1 || insertedEvents != 1 || receipt.Reservation.OperationID != 0 || item.Desired.ID != 0 || !item.Desired.CreatedAt.IsZero() {
			t.Fatal("native late SQL failure was not atomic or leaked IDs")
		}
		var metadata, events int64
		a.Model(&model.MonitorConfig{}).Where("pool_id=71").Count(&metadata)
		a.Model(&model.MonitorConfigArtifactEvent{}).Where("operation_id=?", ref.OperationID).Count(&events)
		var op model.MonitorConfigPoolOperation
		a.Where("operation_id=?", ref.OperationID).First(&op)
		if metadata != 0 || events != 0 || op.Manifest != nil || op.Phase != model.ConfigPublicationReserved {
			t.Fatal("business rollback altered reservation or retained partial rows")
		}
	})
	t.Run("committed unknown ACK resolves on independent session", func(t *testing.T) {
		ref, err := ja.ReservePoolOperation(ctx, request(81, 9))
		if err != nil {
			t.Fatal(err)
		}
		pool, err := a.DB()
		if err != nil {
			t.Fatal(err)
		}
		fault := &journalAckLossPool{DB: pool}
		wrapped := a.Session(&gorm.Session{NewDB: true, Initialized: true})
		wrapped.ConnPool = fault
		wrapped.Statement.ConnPool = fault
		faulty := NewProjectionJournalDAO(zap.NewNop(), wrapped)
		item := render(81, 11, "127.0.0.81")
		fault.lose.Store(true)
		receipt, err := faulty.CommitGeneratedPlan(ctx, GeneratedPlanRequest{Reservation: ref, SourceHash: sha256.Sum256([]byte("unknown")), Items: []GeneratedPlanItem{item}})
		if !errors.Is(err, ErrJournalUncertain) || receipt.Reservation.OperationID != 0 || item.Desired.ID != 0 {
			t.Fatal("unknown ACK was guessed as success/rollback")
		}
		observed, err := jb.LoadOperationByNonce(ctx, ref.Nonce)
		if err != nil || observed.Receipt == nil || len(observed.Receipt.Artifacts) != 1 || observed.Receipt.Artifacts[0].ConfigRowID == nil {
			t.Fatal("committed receipt could not be recovered by nonce")
		}
	})
	t.Run("generation unsigned exhaustion rejects without append", func(t *testing.T) {
		if a.Exec("INSERT INTO cl_monitor_config_pool_controls(family,pool_id,reserved_generation) VALUES(4,91,18446744073709551615)").Error != nil {
			t.Fatal("native maximum counter setup failed")
		}
		if ref, err := ja.ReservePoolOperation(ctx, request(91, 10)); !errors.Is(err, ErrJournalCounterExhausted) || ref.OperationID != 0 {
			t.Fatal("generation wrapped")
		}
		var control model.MonitorConfigPoolControl
		a.Where("family=4 AND pool_id=91").First(&control)
		if control.ReservedGeneration != math.MaxUint64 || control.CurrentAttemptOperationID != nil {
			t.Fatal("exhausted control was changed")
		}
	})
}
