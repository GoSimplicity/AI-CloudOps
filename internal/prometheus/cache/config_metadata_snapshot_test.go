package cache

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	configdao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/config"
)

type configSnapshotPages struct {
	configdao.MonitorConfigDAO
	calls int
	read  func(context.Context, *model.GetMonitorConfigListReq) ([]*model.MonitorConfig, int64, error)
}

func (d *configSnapshotPages) GetMonitorConfigList(ctx context.Context, req *model.GetMonitorConfigListReq) ([]*model.MonitorConfig, int64, error) {
	d.calls++
	return d.read(ctx, req)
}
func snapshotRows(count int) []*model.MonitorConfig {
	rows := make([]*model.MonitorConfig, count)
	for i := range rows {
		rows[i] = &model.MonitorConfig{Model: model.Model{ID: i + 1}, ConfigType: model.ConfigTypePrometheus, Name: fmt.Sprint("row-", i+1)}
	}
	return rows
}

func TestConfigMetadataSnapshotRejectsIncompleteOrAmbiguousInventory(t *testing.T) {
	for _, mode := range []string{"nil row", "zero ID", "negative ID", "duplicate ID", "negative total", "excess rows", "short page", "early empty", "total shrink", "total grow", "duplicate across pages", "first read error", "later read error"} {
		t.Run(mode, func(t *testing.T) {
			first := snapshotRows(100)
			dao := &configSnapshotPages{read: func(_ context.Context, req *model.GetMonitorConfigListReq) ([]*model.MonitorConfig, int64, error) {
				if req.Page == 1 {
					switch mode {
					case "nil row":
						return []*model.MonitorConfig{nil}, 1, nil
					case "zero ID":
						return []*model.MonitorConfig{{}}, 1, nil
					case "negative ID":
						return []*model.MonitorConfig{{Model: model.Model{ID: -1}}}, 1, nil
					case "duplicate ID":
						return []*model.MonitorConfig{first[0], first[0]}, 2, nil
					case "negative total":
						return nil, -1, nil
					case "excess rows":
						return first, 99, nil
					case "short page":
						return first[:99], 101, nil
					case "first read error":
						return nil, 0, errors.New("PRIVATE_DB_ERROR")
					}
					return first, 101, nil
				}
				switch mode {
				case "early empty":
					return nil, 101, nil
				case "total shrink":
					return []*model.MonitorConfig{{Model: model.Model{ID: 101}}}, 100, nil
				case "total grow":
					return []*model.MonitorConfig{{Model: model.Model{ID: 101}}}, 102, nil
				case "duplicate across pages":
					return []*model.MonitorConfig{first[0]}, 101, nil
				default:
					return nil, 0, errors.New("PRIVATE_DB_ERROR")
				}
			}}
			rows, err := readConfigMetadataSnapshot(context.Background(), dao, model.ConfigTypePrometheus)
			if err == nil || rows != nil {
				t.Fatal("invalid inventory returned a partial authoritative snapshot")
			}
			if strings.Contains(err.Error(), "PRIVATE_DB_ERROR") {
				t.Fatal("dependency details leaked")
			}
		})
	}
}
func TestConfigMetadataSnapshotAllowsEmptyAndExactPagination(t *testing.T) {
	for _, total := range []int{0, 1, 100, 101, 200} {
		t.Run(fmt.Sprint(total), func(t *testing.T) {
			source := snapshotRows(total)
			dao := &configSnapshotPages{read: func(_ context.Context, req *model.GetMonitorConfigListReq) ([]*model.MonitorConfig, int64, error) {
				if req.Size != 100 || req.ConfigType == nil || *req.ConfigType != model.ConfigTypePrometheus {
					t.Fatal("reader changed requested kind/page size")
				}
				start := (req.Page - 1) * req.Size
				end := start + req.Size
				if end > total {
					end = total
				}
				return source[start:end], int64(total), nil
			}}
			rows, err := readConfigMetadataSnapshot(context.Background(), dao, model.ConfigTypePrometheus)
			if err != nil || len(rows) != total {
				t.Fatal("valid complete inventory rejected")
			}
			wantCalls := (total + 99) / 100
			if wantCalls == 0 {
				wantCalls = 1
			}
			if dao.calls != wantCalls {
				t.Fatal("reader made unexpected extra page reads")
			}
			if total > 0 && rows[0] == source[0] {
				t.Fatal("reader retained source row alias")
			}
		})
	}
}
func TestConfigMetadataSnapshotClonesEachPageBeforeBufferReuse(t *testing.T) {
	first := snapshotRows(100)
	first[0].Name = "before"
	first[0].ConfigContent = "original bytes"
	dao := &configSnapshotPages{read: func(_ context.Context, req *model.GetMonitorConfigListReq) ([]*model.MonitorConfig, int64, error) {
		if req.Page == 1 {
			return first, 101, nil
		}
		first[0].ID = 900
		first[0].Name = "reused"
		first[0].ConfigContent = "mutated"
		return []*model.MonitorConfig{{Model: model.Model{ID: 101}}}, 101, nil
	}}
	rows, err := readConfigMetadataSnapshot(context.Background(), dao, model.ConfigTypePrometheus)
	if err != nil || rows[0].ID != 1 || rows[0].Name != "before" || rows[0].ConfigContent != "original bytes" {
		t.Fatal("later read mutated an already captured page")
	}
}
func TestConfigMetadataSnapshotNeverCachesAcrossCalls(t *testing.T) {
	source := snapshotRows(1)
	source[0].Name = "first"
	dao := &configSnapshotPages{read: func(context.Context, *model.GetMonitorConfigListReq) ([]*model.MonitorConfig, int64, error) {
		return source, 1, nil
	}}
	first, err := readConfigMetadataSnapshot(context.Background(), dao, model.ConfigTypePrometheus)
	if err != nil {
		t.Fatal(err)
	}
	source[0].Name = "second"
	second, err := readConfigMetadataSnapshot(context.Background(), dao, model.ConfigTypePrometheus)
	if err != nil || dao.calls != 2 || first[0].Name != "first" || second[0].Name != "second" {
		t.Fatal("snapshot was reused across independent calls")
	}
}
func TestConfigMetadataSnapshotNilStorageAndCancellation(t *testing.T) {
	if rows, err := readConfigMetadataSnapshot(context.Background(), nil, 1); err == nil || rows != nil {
		t.Fatal("nil storage accepted")
	}
	for _, mode := range []string{"before read", "during successful read", "during failed read"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			dao := &configSnapshotPages{read: func(context.Context, *model.GetMonitorConfigListReq) ([]*model.MonitorConfig, int64, error) {
				cancel()
				if mode == "during failed read" {
					return nil, 0, ctx.Err()
				}
				return snapshotRows(1), 1, nil
			}}
			if mode == "before read" {
				cancel()
			}
			rows, err := readConfigMetadataSnapshot(ctx, dao, model.ConfigTypePrometheus)
			if !errors.Is(err, context.Canceled) || rows != nil {
				t.Fatal("cancelled inventory became authoritative or lost error identity")
			}
			if mode == "before read" && dao.calls != 0 {
				t.Fatal("cancelled request read storage")
			}
		})
	}
}
func TestConfigMetadataSnapshotLeavesUnexpectedKindPolicyToPublishers(t *testing.T) {
	for _, domain := range []string{"record", "prom"} {
		t.Run(domain, func(t *testing.T) {
			dao := &configSnapshotPages{read: func(context.Context, *model.GetMonitorConfigListReq) ([]*model.MonitorConfig, int64, error) {
				return []*model.MonitorConfig{{Model: model.Model{ID: 1}, PoolID: 7, ConfigType: model.ConfigTypeAlertManager, InstanceIP: "127.0.0.1", Name: "wrong-kind custom"}}, 1, nil
			}}
			desired := map[string]string{"127.0.0.1": "fixture bytes"}
			if domain == "record" {
				c, _, _, _, _ := newRecordReconcile(t)
				c.configDAO = dao
				plan, err := c.planRecordPublication(context.Background(), 7, desired, nil)
				if err != nil || len(plan.create) != 1 || len(plan.update) != 0 || len(plan.stale) != 0 || plan.create[0].ConfigType != model.ConfigTypeRecordRule {
					t.Fatal("record publisher no longer ignores validated unexpected kind")
				}
			} else {
				c, _, _, _, _ := newPromReconcile(t)
				c.configDAO = dao
				plan, err := c.planPromPublication(context.Background(), 7, desired, nil)
				if err != nil || len(plan.create) != 1 || len(plan.update) != 0 || len(plan.stale) != 0 || plan.create[0].ConfigType != model.ConfigTypePrometheus {
					t.Fatal("Prom publisher no longer ignores validated unexpected kind")
				}
			}
		})
	}
}
