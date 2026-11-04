//go:build integration

package cache

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	configdao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/config"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type rootProjectionWireTotals struct {
	mu                                                sync.Mutex
	calls, frameBytes, bodyBytes, respBytes, failures int64
	elapsed                                           time.Duration
	invalid                                           bool
}

func (h *rootProjectionWireTotals) DialHook(next redis.DialHook) redis.DialHook { return next }
func (h *rootProjectionWireTotals) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		start := time.Now()
		err := next(ctx, cmd)
		h.observe(cmd, time.Since(start), err)
		return err
	}
}
func (h *rootProjectionWireTotals) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []redis.Cmder) error {
		start := time.Now()
		err := next(ctx, cmds)
		elapsed := time.Since(start)
		for _, cmd := range cmds {
			if cmd.Name() != "eval" && cmd.Name() != "evalsha" {
				continue
			}
			h.observe(cmd, elapsed, cmd.Err())
			elapsed = 0
		}
		return err
	}
}
func (h *rootProjectionWireTotals) observe(cmd redis.Cmder, elapsed time.Duration, err error) {
	if cmd.Name() != "eval" && cmd.Name() != "evalsha" {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.calls++
	h.elapsed += elapsed
	if err != nil {
		h.failures++
	}
	args := cmd.Args()
	if len(args) < 5 {
		h.invalid = true
		return
	}
	keys, ok := args[2].(int)
	if !ok || keys < 0 || len(args) != keys+5 {
		h.invalid = true
		return
	}
	frame, frameOK := args[keys+3].(string)
	body, bodyOK := args[keys+4].(string)
	if !frameOK || !bodyOK {
		h.invalid = true
		return
	}
	h.frameBytes += int64(len(frame))
	h.bodyBytes += int64(len(body))
	// The actual command arguments are encoded as a RESP array of bulk strings.
	// Count lengths only; do not retain or log script, keys, bodies or credentials.
	bytes := 1 + len(strconv.Itoa(len(args))) + 2
	for _, arg := range args {
		var value string
		switch arg := arg.(type) {
		case string:
			value = arg
		case int:
			value = strconv.Itoa(arg)
		default:
			h.invalid = true
			return
		}
		bytes += 1 + len(strconv.Itoa(len(value))) + 2 + len(value) + 2
	}
	h.respBytes += int64(bytes)
}

func rootProjectionEvalStats(t *testing.T, ctx context.Context, client *redis.Client) (int64, int64) {
	t.Helper()
	info, err := client.Info(ctx, "commandstats").Result()
	if err != nil {
		t.Fatal("owned Redis command statistics unavailable")
	}
	for _, line := range strings.Split(info, "\n") {
		if !strings.HasPrefix(line, "cmdstat_eval:") {
			continue
		}
		values := map[string]int64{}
		for _, pair := range strings.Split(strings.TrimSpace(strings.TrimPrefix(line, "cmdstat_eval:")), ",") {
			p := strings.SplitN(pair, "=", 2)
			if len(p) != 2 {
				continue
			}
			if n, err := strconv.ParseInt(p[1], 10, 64); err == nil {
				values[p[0]] = n
			}
		}
		return values["calls"], values["usec"]
	}
	return 0, 0
}

// This adds actual Redis/Lua/ACK and SQL completion to the separate microbench.
// Timings are observations, never host-dependent pass/fail thresholds.
func TestProjectionStoreNativeScaleCost(t *testing.T) {
	client := projectionRedisNative(t)
	for _, size := range []int{1, 4, 16, 64, 128} {
		t.Run(fmt.Sprintf("N%d", size), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			journal, _, db := projectionStoreSQL(t)
			ref, err := journal.ReservePoolOperation(ctx, configdao.PoolOperationRequest{Family: 4, PoolID: 22000 + size, Nonce: [16]byte{1}})
			if err != nil {
				t.Fatal("scale reservation failed")
			}
			items := make([]configdao.GeneratedPlanItem, 0, size)
			for i := 0; i < size; i++ {
				items = append(items, projectionRecordItem(ref.PoolID, uint64(i+1), fmt.Sprintf("10.21.%d.%d", i/250, i%250+1), "groups: []\n"))
			}
			receipt, err := journal.CommitGeneratedProjectionPlan(ctx, projectionFullRequest(ref, items...))
			if err != nil {
				t.Fatal("scale plan commit failed")
			}
			store, err := NewProjectionStore(client)
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			wire := &rootProjectionWireTotals{}
			store.client.AddHook(wire)
			selects := &atomic.Int64{}
			observe := func(tx *gorm.DB) {
				if strings.HasPrefix(strings.ToUpper(strings.TrimSpace(tx.Statement.SQL.String())), "SELECT ") {
					selects.Add(1)
				}
			}
			if db.Callback().Query().After("gorm:query").Register("native_scale:query", observe) != nil || db.Callback().Row().After("gorm:row").Register("native_scale:row", observe) != nil {
				t.Fatal("scale SQL observer unavailable")
			}
			beforeCalls, beforeUsec := rootProjectionEvalStats(t, ctx, client)
			started := time.Now()
			err = journal.WithPoolProjectionLease(ctx, receipt, func(lease configdao.ProjectionLease) error {
				snapshot, err := lease.Snapshot()
				if err != nil {
					return err
				}
				handle, err := compileProjection(store, lease, snapshot)
				if err != nil {
					return err
				}
				projectionTrack(t, client, handle)
				return NewRecordProjectionRunner(store).Project(ctx, lease)
			})
			elapsed := time.Since(started)
			timedSelects := selects.Load()
			if err != nil {
				t.Fatal("real scale projection failed")
			}
			afterCalls, afterUsec := rootProjectionEvalStats(t, ctx, client)
			projected, err := journal.LoadLastProjectedProjection(ctx, 4, ref.PoolID)
			if err != nil || projected == nil || projected.Reservation != ref {
				t.Fatal("scale projection lacks durable SQL completion")
			}
			for _, item := range items {
				if body, err := client.Get(ctx, buildRedisKeyRecordRule(item.Desired.InstanceIP)).Result(); err != nil || body != item.Desired.ConfigContent {
					t.Fatal("scale raw artifact differs from accepted source")
				}
			}
			if n, err := client.SCard(ctx, buildRedisSetKeyRecordRulePoolIPs(ref.PoolID)).Result(); err != nil || n != int64(size) {
				t.Fatal("scale membership is incomplete")
			}
			wire.mu.Lock()
			defer wire.mu.Unlock()
			if wire.invalid || wire.failures != 0 || wire.calls < 1 || afterCalls-beforeCalls != wire.calls {
				t.Fatal("actual EVAL accounting differs from owned Redis")
			}
			t.Logf("scale size=%d full_sql_redis_ns=%d sql_selects=%d eval_calls=%d eval_client_ns=%d eval_server_us=%d frame_bytes=%d body_bytes=%d eval_resp_bytes=%d", size, elapsed.Nanoseconds(), timedSelects, wire.calls, wire.elapsed.Nanoseconds(), afterUsec-beforeUsec, wire.frameBytes, wire.bodyBytes, wire.respBytes)
		})
	}
}
