package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	configdao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/config"
)

// This is layout qualification only. The plan/authority come from actual SQL
// and a sealed lease, but no Redis client is created, no command is sent, and no
// receipt is marked complete. Setup/lease timings are reported separately.
func BenchmarkProjectionCapacityLayout(b *testing.B) {
	for _, size := range []int{1024, 1280, 1536, 2048} {
		b.Run(fmt.Sprintf("N%d", size), func(b *testing.B) {
			for _, format := range []string{"IPv4Short", "IPv6Raw45"} {
				b.Run(format, func(b *testing.B) { benchmarkProjectionCapacityLayout(b, size, format) })
			}
		})
	}
}

type capacityLayoutMetrics struct {
	frameBytes         int
	keyCount           int
	keyBytes           int64
	batchCommands      int
	singleCommands     int
	batchMaxRESP       int64
	singleMaxRESP      int64
	artifactRESP       int64
	artifactFrameBytes int64
	bodyBytes          int64
	first16RESP        int64
	singleJSONVariants int
}

func benchmarkProjectionCapacityLayout(b *testing.B, size int, format string) {
	b.StopTimer()
	ctx, cancel := context.WithTimeout(b.Context(), 90*time.Second)
	defer cancel()
	setupStart := time.Now()
	journal, _, _ := projectionStoreSQL(b)
	ref, err := journal.ReservePoolOperation(ctx, configdao.PoolOperationRequest{Family: 4, PoolID: 23000 + size, Nonce: [16]byte{1}})
	if err != nil {
		b.Fatal(err)
	}
	items := make([]configdao.GeneratedPlanItem, 0, size)
	for i := 0; i < size; i++ {
		ip := fmt.Sprintf("10.42.%d.%d", i/250, i%250+1)
		if format == "IPv6Raw45" {
			// Keep every raw IPv4-mapped IPv6 literal at 45 bytes; no normalization.
			ip = fmt.Sprintf("0000:0000:0000:0000:0000:ffff:192.168.%d.%d", 100+i/155, 100+i%155)
			if len(ip) != 45 {
				b.Fatal("raw IPv6 fixture lost its fixed width")
			}
		}
		items = append(items, projectionRecordItem(ref.PoolID, uint64(i+1), ip, "groups: []\n"))
	}
	receipt, err := journal.CommitGeneratedProjectionPlan(ctx, projectionFullRequest(ref, items...))
	if err != nil {
		b.Fatalf("N%d/%s setup failed: %v", size, format, err)
	}
	setupElapsed := time.Since(setupStart)
	entryStart := time.Now()
	var entryElapsed, exitElapsed time.Duration
	var callbackEnd time.Time
	totals := capacityLayoutMetrics{}
	b.ReportAllocs()
	err = journal.WithPoolProjectionLease(ctx, receipt, func(lease configdao.ProjectionLease) error {
		entryElapsed = time.Since(entryStart)
		snapshot, err := lease.Snapshot()
		if err != nil {
			return err
		}
		handle, err := compileProjection(nil, lease, snapshot)
		if err != nil {
			return err
		}
		ids := make([]uint64, 0, len(receipt.Artifacts))
		for _, artifact := range receipt.Artifacts {
			ids = append(ids, artifact.EventID)
		}
		b.ResetTimer() // Timer remains stopped; exclude SQL, snapshot and compilation.
		for iteration := 0; iteration < b.N; iteration++ {
			if err := ctx.Err(); err != nil {
				return err
			}
			if err := lease.RecheckCurrent(ctx); err != nil {
				return err
			}
			b.StartTimer()
			metrics, err := calculateCapacityLayout(handle, ids)
			b.StopTimer()
			if err != nil {
				return err
			}
			totals = metrics
		}
		callbackEnd = time.Now()
		return nil
	})
	b.StopTimer()
	exitElapsed = time.Since(callbackEnd)
	if err != nil {
		b.Fatalf("N%d/%s exceeded its case budget or failed layout validation: %v", size, format, err)
	}
	// A qualified layout must never be confused with successful publication.
	last, err := journal.LoadLastCommittedProjection(ctx, 4, ref.PoolID)
	if err != nil || last == nil || last.Phase != model.ConfigPublicationCommitted {
		b.Fatal("layout unexpectedly changed committed state", err)
	}
	projected, err := journal.LoadLastProjectedProjection(ctx, 4, ref.PoolID)
	if err != nil || projected != nil {
		b.Fatal("layout unexpectedly issued projection completion", err)
	}
	b.ReportMetric(float64(setupElapsed.Nanoseconds()), "SQL_setup_ns")
	b.ReportMetric(float64(entryElapsed.Nanoseconds()), "lease_entry_ns")
	b.ReportMetric(float64(exitElapsed.Nanoseconds()), "lease_exit_ns")
	b.ReportMetric(float64(totals.frameBytes), "candidate_frame_B")
	b.ReportMetric(float64(totals.keyCount), "KEYS_count")
	b.ReportMetric(float64(totals.keyBytes), "KEYS_value_B")
	b.ReportMetric(float64(totals.batchCommands), "planned_batches")
	b.ReportMetric(float64(totals.singleCommands), "planned_singles")
	b.ReportMetric(float64(totals.batchCommands+totals.singleCommands), "planned_artifact_EVALs")
	b.ReportMetric(float64(totals.batchMaxRESP), "max_batch_RESP_B")
	b.ReportMetric(float64(totals.singleMaxRESP), "max_single_RESP_B")
	b.ReportMetric(float64(totals.artifactRESP), "total_artifact_RESP_B")
	b.ReportMetric(float64(totals.artifactFrameBytes), "total_artifact_JSON_B")
	b.ReportMetric(float64(totals.bodyBytes), "body_B")
	b.ReportMetric(float64(totals.first16RESP), "first16_candidate_RESP_B")
	b.ReportMetric(float64(totals.singleJSONVariants), "single_JSON_length_variants")
}

func calculateCapacityLayout(handle *PreparedProjection, ids []uint64) (capacityLayoutMetrics, error) {
	selected, err := handle.selectArtifacts(ids)
	if err != nil {
		return capacityLayoutMetrics{}, err
	}
	wire := handle.wire
	wire.Mode = "artifact_batch"
	wire.Target = ""
	encoded, err := json.Marshal(wire)
	if err != nil {
		return capacityLayoutMetrics{}, err
	}
	wireJSON := string(encoded)
	commands, err := handle.artifactCommands(wireJSON, selected)
	if err != nil {
		return capacityLayoutMetrics{}, err
	}
	result := capacityLayoutMetrics{frameBytes: len(encoded), keyCount: len(handle.keys)}
	for _, key := range handle.keys {
		result.keyBytes += int64(len(key))
	}
	// Full-frame single JSON changes only by a decimal ASCII target. Serialize a
	// real target for each encountered digit width and reuse only the byte length.
	// This avoids allocating gigabytes merely to calculate unsent RESP lengths;
	// its timing is NOT the cost of actually marshalling/sending all singles.
	singleFrameLengths := map[int]int{}
	for _, command := range commands {
		if command.single {
			result.singleCommands++
			item := command.selections[0]
			width := len(item.id)
			frameLength, known := singleFrameLengths[width]
			if !known {
				singleWire := handle.wire
				singleWire.Mode = "artifact"
				singleWire.Target = item.id
				raw, err := json.Marshal(singleWire)
				if err != nil {
					return capacityLayoutMetrics{}, err
				}
				frameLength = len(raw)
				singleFrameLengths[width] = frameLength
			}
			size := projectionEvalRESPBytesForLengths(handle.keys, []int{frameLength, len(item.body)})
			if size > result.singleMaxRESP {
				result.singleMaxRESP = size
			}
			result.artifactRESP += size
			result.artifactFrameBytes += int64(frameLength)
			result.bodyBytes += int64(len(item.body))
			continue
		}
		if len(command.selections) > projectionArtifactBatchMaxItems {
			return result, fmt.Errorf("layout exceeded batch item cap")
		}
		lengths := []int{len(wireJSON), len(command.selectionJSON)}
		for _, item := range command.selections {
			lengths = append(lengths, len(item.body))
			result.bodyBytes += int64(len(item.body))
		}
		size := projectionEvalRESPBytesForLengths(handle.keys, lengths)
		if size > projectionArtifactBatchMaxRequestBytes {
			return result, fmt.Errorf("layout exceeded batch RESP cap")
		}
		result.batchCommands++
		if size > result.batchMaxRESP {
			result.batchMaxRESP = size
		}
		result.artifactRESP += size
		result.artifactFrameBytes += int64(len(wireJSON))
	}
	result.singleJSONVariants = len(singleFrameLengths)
	first := selected
	if len(first) > projectionArtifactBatchMaxItems {
		first = first[:projectionArtifactBatchMaxItems]
	}
	selection, err := projectionSelectionJSON(first)
	if err != nil {
		return result, err
	}
	lengths := []int{len(wireJSON), len(selection)}
	for _, item := range first {
		lengths = append(lengths, len(item.body))
	}
	result.first16RESP = projectionEvalRESPBytesForLengths(handle.keys, lengths)
	// Both actual JSON lengths and real typed selections were used above. This
	// assertion also catches an accidental body-byte loss in the measurement.
	if result.bodyBytes != int64(len(ids)*len("groups: []\n")) {
		return result, fmt.Errorf("synthetic body accounting mismatch")
	}
	return result, nil
}
