package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	configdao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/config"
	"gorm.io/gorm"
)

// All bodies are the same tiny synthetic fixture. These benchmarks measure
// local SQLite/Go work, not MySQL latency, large YAML, Redis, Lua or delivery.
// No Prepare/EVAL/ACK is sent and no SQL operation is marked projected.
func BenchmarkProjectionScale(b *testing.B) {
	for _, size := range []int{1, 4, 16, 64, 128} {
		b.Run(fmt.Sprintf("N%d", size), func(b *testing.B) {
			for _, stage := range []string{"LeaseEntry", "SingleRecheck", "StoreCommandFrames"} {
				b.Run(stage, func(b *testing.B) { benchmarkProjectionScaleStage(b, size, stage) })
			}
		})
	}
}

type projectionScaleTotals struct {
	selects      int64
	entrySelects int64
	exitSelects  int64
	frames       int64
	frameBytes   int64
	rechecks     int64
	snapshots    int64
}

func benchmarkProjectionScaleStage(b *testing.B, size int, stage string) {
	b.StopTimer()
	journal, _, db := projectionStoreSQL(b)
	counter := &atomic.Int64{}
	observe := func(tx *gorm.DB) {
		// Observe actual generated SELECT executions without formatting arguments or
		// printing SQL, bodies, credentials or query results.
		if strings.HasPrefix(strings.ToUpper(strings.TrimSpace(tx.Statement.SQL.String())), "SELECT ") {
			counter.Add(1)
		}
	}
	if err := db.Callback().Query().After("gorm:query").Register("projection_scale:select_query", observe); err != nil {
		b.Fatal(err)
	}
	if err := db.Callback().Row().After("gorm:row").Register("projection_scale:select_row", observe); err != nil {
		b.Fatal(err)
	}
	ref, err := journal.ReservePoolOperation(context.Background(), configdao.PoolOperationRequest{Family: 4, PoolID: 7, Nonce: [16]byte{1}})
	if err != nil {
		b.Fatal(err)
	}
	items := make([]configdao.GeneratedPlanItem, 0, size)
	for i := 0; i < size; i++ {
		ip := fmt.Sprintf("10.20.%d.%d", i/250, (i%250)+1)
		items = append(items, projectionRecordItem(7, uint64(i+1), ip, "groups: []\n"))
	}
	receipt, err := journal.CommitGeneratedProjectionPlan(context.Background(), projectionFullRequest(ref, items...))
	if err != nil {
		b.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(b.Context(), 2*time.Minute)
	defer cancel()
	totals := projectionScaleTotals{}
	b.ReportAllocs()
	for iteration := 0; iteration < b.N; iteration++ {
		if err := ctx.Err(); err != nil {
			b.Fatalf("N%d/%s exceeded fixed case budget: %v", size, stage, err)
		}
		beforeEntry := counter.Load()
		afterCallback := beforeEntry
		if stage == "LeaseEntry" {
			b.StartTimer()
		}
		err := journal.WithPoolProjectionLease(ctx, receipt, func(lease configdao.ProjectionLease) error {
			entryEnd := counter.Load()
			totals.entrySelects += entryEnd - beforeEntry
			if stage == "LeaseEntry" {
				b.StopTimer()
				totals.selects += entryEnd - beforeEntry
				afterCallback = counter.Load()
				return nil
			}
			beforeMeasured := counter.Load()
			b.StartTimer()
			var measuredErr error
			switch stage {
			case "SingleRecheck":
				totals.rechecks++
				measuredErr = lease.RecheckCurrent(ctx)
			case "StoreCommandFrames":
				measuredErr = projectionScaleCommandFrames(ctx, lease, size, &totals)
			default:
				measuredErr = fmt.Errorf("unknown benchmark stage")
			}
			b.StopTimer()
			totals.selects += counter.Load() - beforeMeasured
			afterCallback = counter.Load()
			return measuredErr
		})
		b.StopTimer()
		totals.exitSelects += counter.Load() - afterCallback
		if err != nil {
			b.Fatalf("N%d/%s failed (no Redis command was sent): %v", size, stage, err)
		}
	}
	divisor := float64(b.N)
	b.ReportMetric(float64(totals.selects)/divisor, "SELECTs/op")
	b.ReportMetric(float64(totals.entrySelects)/divisor, "entry_SELECTs/op")
	b.ReportMetric(float64(totals.exitSelects)/divisor, "exit_SELECTs/op")
	b.ReportMetric(float64(totals.rechecks)/divisor, "explicit_rechecks/op")
	b.ReportMetric(float64(totals.snapshots)/divisor, "Snapshots/op")
	b.ReportMetric(float64(totals.frames)/divisor, "frames/op")
	b.ReportMetric(float64(totals.frameBytes)/divisor, "wire_bytes/op")
}

// Reproduce only the SQL and Go portions of the current Record Store path:
// runner Snapshot; Prepare's pre-Recheck, Snapshot and real compiler; then its
// N+3 command Rechecks and exact full-frame json.Marshal. No command is sent.
// Entry/exit checks are measured separately by the surrounding lease counters.
func projectionScaleCommandFrames(ctx context.Context, lease configdao.ProjectionLease, size int, totals *projectionScaleTotals) error {
	runnerSnapshot, err := lease.Snapshot()
	if err != nil {
		return err
	}
	totals.snapshots++
	if runnerSnapshot.Pool == nil || len(runnerSnapshot.Artifacts) != size {
		return fmt.Errorf("synthetic lease lost artifacts")
	}
	totals.rechecks++
	if err := lease.RecheckCurrent(ctx); err != nil {
		return err
	}
	prepareSnapshot, err := lease.Snapshot()
	if err != nil {
		return err
	}
	totals.snapshots++
	handle, err := compileProjection(nil, lease, prepareSnapshot)
	if err != nil {
		return err
	}
	marshalCommand := func(mode, target string) error {
		totals.rechecks++
		if err := lease.RecheckCurrent(ctx); err != nil {
			return err
		}
		wire := handle.wire
		wire.Mode, wire.Target = mode, target
		raw, err := json.Marshal(wire)
		if err != nil {
			return err
		}
		totals.frames++
		totals.frameBytes += int64(len(raw))
		return nil
	}
	if err := marshalCommand("prepare", ""); err != nil {
		return err
	}
	for _, artifact := range runnerSnapshot.Pool.Artifacts {
		id := strconv.FormatUint(artifact.EventID, 10)
		if _, ok := handle.materials[id]; !ok {
			return fmt.Errorf("synthetic compiler lost material")
		}
		if err := marshalCommand("artifact", id); err != nil {
			return err
		}
	}
	if err := marshalCommand("members", "1"); err != nil {
		return err
	}
	return marshalCommand("ack", "")
}
