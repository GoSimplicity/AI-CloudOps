package cache

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"testing"

	configdao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/config"
	"github.com/redis/go-redis/v9"
)

// This hook is only a Go send-boundary failure. It neither executes Lua nor
// acknowledges anything, and it prevents all network access in these unit tests.
type batchGoBoundaryHook struct {
	calls   int
	args    []interface{}
	failure error
}

func (h *batchGoBoundaryHook) DialHook(redis.DialHook) redis.DialHook {
	return func(context.Context, string, string) (net.Conn, error) {
		return nil, errors.New("unit test forbids network")
	}
}
func (h *batchGoBoundaryHook) ProcessHook(redis.ProcessHook) redis.ProcessHook {
	return func(_ context.Context, cmd redis.Cmder) error {
		if cmd.Name() == "eval" {
			h.calls++
			h.args = append([]interface{}(nil), cmd.Args()...)
		}
		return h.failure
	}
}
func (h *batchGoBoundaryHook) ProcessPipelineHook(redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(context.Context, []redis.Cmder) error { return h.failure }
}

func batchGoBoundaryFixture(t *testing.T, bodies []string, callback func(*PreparedProjection, []uint64, *batchGoBoundaryHook) error) error {
	t.Helper()
	journal, _, _ := projectionStoreSQL(t)
	base := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1"})
	defer base.Close()
	store, err := NewProjectionStore(base)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	hook := &batchGoBoundaryHook{failure: errors.New("synthetic EVAL reply unavailable")}
	store.client.AddHook(hook)
	ref, err := journal.ReservePoolOperation(context.Background(), configdao.PoolOperationRequest{Family: 4, PoolID: 7, Nonce: [16]byte{1}})
	if err != nil {
		t.Fatal(err)
	}
	items := make([]configdao.GeneratedPlanItem, 0, len(bodies))
	for i, body := range bodies {
		items = append(items, projectionRecordItem(7, uint64(i+1), fmt.Sprintf("10.30.0.%d", i+1), body))
	}
	receipt, err := journal.CommitGeneratedProjectionPlan(context.Background(), projectionFullRequest(ref, items...))
	if err != nil {
		t.Fatal(err)
	}
	return journal.WithPoolProjectionLease(context.Background(), receipt, func(lease configdao.ProjectionLease) error {
		snapshot, err := lease.Snapshot()
		if err != nil {
			return err
		}
		prepared, err := compileProjection(store, lease, snapshot)
		if err != nil {
			return err
		}
		ids := make([]uint64, 0, len(receipt.Artifacts))
		for _, artifact := range receipt.Artifacts {
			ids = append(ids, artifact.EventID)
		}
		return callback(prepared, ids, hook)
	})
}
func TestProjectionBatchGoValidatesEntireSelectionBeforeAnySend(t *testing.T) {
	for _, mode := range []string{"late unknown", "late duplicate", "late missing material", "late unsupported action", "late body mismatch"} {
		t.Run(mode, func(t *testing.T) {
			bodies := make([]string, 17)
			for i := range bodies {
				bodies[i] = "groups: []\n"
			}
			err := batchGoBoundaryFixture(t, bodies, func(handle *PreparedProjection, ids []uint64, hook *batchGoBoundaryHook) error {
				switch mode {
				case "late unknown":
					ids[len(ids)-1] = ^uint64(0)
				case "late duplicate":
					ids[len(ids)-1] = ids[0]
				case "late missing material":
					delete(handle.materials, strconv.FormatUint(ids[len(ids)-1], 10))
				case "late unsupported action":
					handle.wire.Artifacts[len(handle.wire.Artifacts)-1].Action = "unsupported"
				case "late body mismatch":
					handle.materials[strconv.FormatUint(ids[len(ids)-1], 10)] = []byte("wrong synthetic body")
				}
				err := handle.ApplyArtifacts(context.Background(), ids)
				if !errors.Is(err, ErrProjectionStore) || hook.calls != 0 {
					t.Fatal("bad later selection was validated after an earlier command", err, hook.calls)
				}
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}
func TestProjectionBatchGoErrorNeverFallsBackAndBodiesStayRaw(t *testing.T) {
	body := string([]byte{'#', ' ', 0xff, 0, 0xfe, '\n'})
	bodies := make([]string, 17)
	for i := range bodies {
		bodies[i] = body
	}
	err := batchGoBoundaryFixture(t, bodies, func(handle *PreparedProjection, ids []uint64, hook *batchGoBoundaryHook) error {
		err := handle.ApplyArtifacts(context.Background(), ids)
		if !errors.Is(err, ErrProjectionStore) || hook.calls != 1 {
			t.Fatal("EVAL error retried, fell back or sent the next command", err, hook.calls)
		}
		keyCount, ok := hook.args[2].(int)
		if !ok {
			t.Fatal("missing EVAL key count")
		}
		keys := make([]string, keyCount)
		for i := range keys {
			keys[i] = hook.args[i+3].(string)
		}
		wire, err := decodeProjectionTestWire(hook.args[keyCount+3].(string), keys)
		if err != nil {
			t.Fatal(err)
		}
		if wire.Mode != "artifact_batch" || wire.Target != "" {
			t.Fatal("eligible full batch did not use new mode")
		}
		var selected []string
		if err := json.Unmarshal([]byte(hook.args[keyCount+4].(string)), &selected); err != nil {
			t.Fatal(err)
		}
		if len(selected) != 16 || len(hook.args) != keyCount+5+16 {
			t.Fatal("batch ARGV/count contract changed")
		}
		for i := 0; i < 16; i++ {
			if !bytes.Equal([]byte(hook.args[keyCount+5+i].(string)), []byte(body)) {
				t.Fatal("body passed through JSON/UTF8 replacement")
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
func TestProjectionBatchPlannerUsesCountAndCompleteRequestBudget(t *testing.T) {
	for _, mode := range []string{"count", "large body", "large metadata"} {
		t.Run(mode, func(t *testing.T) {
			bodies := []string{"groups: []\n", "groups: []\n"}
			if mode == "count" {
				bodies = make([]string, 33)
				for i := range bodies {
					bodies[i] = "groups: []\n"
				}
			}
			if mode == "large body" {
				bodies[0] = "groups: []\n#" + strings.Repeat("x", int(projectionArtifactBatchMaxRequestBytes))
			}
			err := batchGoBoundaryFixture(t, bodies, func(handle *PreparedProjection, ids []uint64, hook *batchGoBoundaryHook) error {
				selected, err := handle.selectArtifacts(ids)
				if err != nil {
					return err
				}
				wire := handle.wire
				wire.Mode = "artifact_batch"
				wire.Target = ""
				if mode == "large metadata" {
					wire.AckReceipt = strings.Repeat("m", int(projectionArtifactBatchMaxRequestBytes))
				}
				encoded, err := json.Marshal(wire)
				if err != nil {
					return err
				}
				commands, err := handle.artifactCommands(string(encoded), selected)
				if err != nil {
					return err
				}
				if mode == "count" {
					if len(commands) != 3 || len(commands[0].selections) != 16 || len(commands[1].selections) != 16 || !commands[2].single {
						t.Fatal("count-based partition lost order or cap")
					}
				} else if len(commands) != 2 || !commands[0].single || !commands[1].single {
					t.Fatal("oversized request was rejected or batched instead of legacy fallback")
				}
				for _, command := range commands {
					if command.single {
						continue
					}
					args := []string{string(encoded), command.selectionJSON}
					for _, item := range command.selections {
						args = append(args, string(item.body))
					}
					exact := batchIndependentRESPBytes(handle.keys, args)
					if projectionEvalRESPBytes(handle.keys, args) != exact || exact > projectionArtifactBatchMaxRequestBytes {
						t.Fatal("complete EVAL RESP accounting ignored an argument or framing", exact)
					}
				}
				if hook.calls != 0 {
					t.Fatal("planning sent a command")
				}
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}
func batchIndependentRESPBytes(keys []string, args []string) int64 {
	all := []string{"eval", projectionFenceLua, strconv.Itoa(len(keys))}
	all = append(all, keys...)
	all = append(all, args...)
	var encoded bytes.Buffer
	fmt.Fprintf(&encoded, "*%d\r\n", len(all))
	for _, value := range all {
		fmt.Fprintf(&encoded, "$%d\r\n", len(value))
		encoded.WriteString(value)
		encoded.WriteString("\r\n")
	}
	return int64(encoded.Len())
}
