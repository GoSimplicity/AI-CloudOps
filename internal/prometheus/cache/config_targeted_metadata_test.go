package cache

import (
	"context"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"reflect"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	configdao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/config"
)

type targetedMetadataStub struct {
	configdao.MonitorConfigDAO
	pool                                 func(context.Context, configdao.GeneratedConfigPoolQuery) ([]*model.MonitorConfig, int64, error)
	namespace                            func(context.Context, configdao.GeneratedConfigKeysQuery) ([]*model.MonitorConfig, int64, error)
	poolCalls, namespaceCalls, fullCalls int
}

func (d *targetedMetadataStub) GetGeneratedConfigPoolCandidates(ctx context.Context, req configdao.GeneratedConfigPoolQuery) ([]*model.MonitorConfig, int64, error) {
	d.poolCalls++
	if d.pool == nil {
		return nil, 0, errors.New("unexpected pool query")
	}
	return d.pool(ctx, req)
}
func (d *targetedMetadataStub) GetGeneratedConfigNamespacePage(ctx context.Context, req configdao.GeneratedConfigKeysQuery) ([]*model.MonitorConfig, int64, error) {
	d.namespaceCalls++
	if d.namespace == nil {
		return nil, 0, errors.New("unexpected namespace query")
	}
	return d.namespace(ctx, req)
}
func (d *targetedMetadataStub) GetMonitorConfigList(context.Context, *model.GetMonitorConfigListReq) ([]*model.MonitorConfig, int64, error) {
	d.fullCalls++
	return nil, 0, errors.New("full scan must not be used")
}
func typedSnapshotRows(n int, key string) []*model.MonitorConfig {
	rows := snapshotRows(n)
	for _, row := range rows {
		row.PoolID = 7
		row.ConfigType = model.ConfigTypeRecordRule
		row.InstanceIP = key
	}
	return rows
}

func TestTargetedMetadataRequiresCompletePoolAndNamespacePages(t *testing.T) {
	for _, path := range []string{"pool", "namespace"} {
		for _, mode := range []string{"negative total", "early empty", "short page", "total drift", "duplicate ID", "later error"} {
			t.Run(path+"/"+mode, func(t *testing.T) {
				first := typedSnapshotRows(100, "192.0.2.1")
				fetch := func(page int) ([]*model.MonitorConfig, int64, error) {
					if page == 1 {
						if mode == "negative total" {
							return nil, -1, nil
						}
						if mode == "short page" {
							return first[:99], 101, nil
						}
						return first, 101, nil
					}
					switch mode {
					case "early empty":
						return nil, 101, nil
					case "total drift":
						return []*model.MonitorConfig{{Model: model.Model{ID: 101}, PoolID: 7, ConfigType: 4, InstanceIP: "192.0.2.1"}}, 102, nil
					case "duplicate ID":
						return []*model.MonitorConfig{first[0]}, 101, nil
					default:
						return nil, 0, errors.New("fixture page failure")
					}
				}
				d := &targetedMetadataStub{pool: func(_ context.Context, r configdao.GeneratedConfigPoolQuery) ([]*model.MonitorConfig, int64, error) {
					return fetch(r.Page)
				}, namespace: func(_ context.Context, r configdao.GeneratedConfigKeysQuery) ([]*model.MonitorConfig, int64, error) {
					return fetch(r.Page)
				}}
				var rows []*model.MonitorConfig
				var err error
				if path == "pool" {
					rows, err = readPoolConfigCandidates(context.Background(), d, 7, 4)
				} else {
					rows, err = readConfigKeyNamespace(context.Background(), d, 4, []string{"192.0.2.1"})
				}
				if err == nil || rows != nil || d.fullCalls != 0 {
					t.Fatal("incomplete typed pages returned authority or fell back to full scan")
				}
			})
		}
	}
}
func TestTargetedMetadataPreservesFullNamespacePagination(t *testing.T) {
	all := typedSnapshotRows(101, "192.0.2.1")
	for i, row := range all {
		if i%2 == 0 {
			row.PoolID = 99
			row.Status = 2
			row.Name = "custom foreign"
		}
	}
	d := &targetedMetadataStub{namespace: func(_ context.Context, r configdao.GeneratedConfigKeysQuery) ([]*model.MonitorConfig, int64, error) {
		if r.Page == 1 {
			return all[:100], 101, nil
		}
		return all[100:], 101, nil
	}}
	rows, err := readConfigKeyNamespace(context.Background(), d, 4, []string{"192.0.2.1"})
	if err != nil || len(rows) != 101 || d.namespaceCalls != 2 || rows[0].PoolID != 99 || rows[0].Status != 2 {
		t.Fatal("namespace pagination filtered foreign/inactive/duplicate key rows")
	}
}
func TestTargetedMetadataSplitsKeysAndRejectsCrossChunkIdentityDrift(t *testing.T) {
	for _, mode := range []string{"valid", "duplicate ID", "out-of-chunk key"} {
		t.Run(mode, func(t *testing.T) {
			keys := make([]string, 101)
			ids := map[string]int{}
			for i := range keys {
				keys[i] = fmt.Sprintf("key-%03d", i)
				ids[keys[i]] = i + 1
			}
			d := &targetedMetadataStub{namespace: func(_ context.Context, r configdao.GeneratedConfigKeysQuery) ([]*model.MonitorConfig, int64, error) {
				if len(r.InstanceKeys) > 100 {
					t.Fatal("query exceeded DAO key cap")
				}
				var rows []*model.MonitorConfig
				for _, key := range r.InstanceKeys {
					rows = append(rows, &model.MonitorConfig{Model: model.Model{ID: ids[key]}, PoolID: 99, ConfigType: 4, InstanceIP: key})
				}
				if r.InstanceKeys[0] == "key-100" && mode == "duplicate ID" {
					rows[0].ID = 1
				}
				if r.InstanceKeys[0] == "key-000" && mode == "out-of-chunk key" {
					rows[0].InstanceIP = "key-100"
				}
				return rows, int64(len(rows)), nil
			}}
			rows, err := readConfigKeyNamespace(context.Background(), d, 4, append(keys, keys[0]))
			if mode == "valid" {
				if err != nil || len(rows) != 101 || d.namespaceCalls != 2 {
					t.Fatal("deduplicated chunk reads lost keys")
				}
			} else if err == nil || rows != nil {
				t.Fatal("cross-chunk identity/scope drift became authoritative")
			}
			if d.fullCalls != 0 {
				t.Fatal("typed reads fell back to full scan")
			}
		})
	}
}
func TestTargetedMetadataRejectsScopeViolations(t *testing.T) {
	for _, mode := range []string{"pool wrong owner", "pool wrong kind", "pool deleted", "namespace wrong kind", "namespace deleted", "namespace byte alias"} {
		t.Run(mode, func(t *testing.T) {
			row := typedSnapshotRows(1, "2001:db8::1")[0]
			switch mode {
			case "pool wrong owner":
				row.PoolID = 99
			case "pool wrong kind", "namespace wrong kind":
				row.ConfigType = 1
			case "pool deleted", "namespace deleted":
				row.DeletedAt = 1
			case "namespace byte alias":
				row.InstanceIP = "2001:DB8::1"
			}
			d := &targetedMetadataStub{pool: func(context.Context, configdao.GeneratedConfigPoolQuery) ([]*model.MonitorConfig, int64, error) {
				return []*model.MonitorConfig{row}, 1, nil
			}, namespace: func(context.Context, configdao.GeneratedConfigKeysQuery) ([]*model.MonitorConfig, int64, error) {
				return []*model.MonitorConfig{row}, 1, nil
			}}
			var err error
			if mode[:4] == "pool" {
				_, err = readPoolConfigCandidates(context.Background(), d, 7, 4)
			} else {
				_, err = readConfigKeyNamespace(context.Background(), d, 4, []string{"2001:db8::1"})
			}
			if err == nil {
				t.Fatal("typed DAO scope violation accepted")
			}
		})
	}
}
func TestTargetedMetadataClonesReusedPagesAndRespectsCancellation(t *testing.T) {
	for _, path := range []string{"pool", "namespace"} {
		t.Run(path, func(t *testing.T) {
			first := typedSnapshotRows(100, "192.0.2.1")
			first[0].Name = "captured"
			fetch := func(page int) ([]*model.MonitorConfig, int64, error) {
				if page == 1 {
					return first, 101, nil
				}
				first[0].Name = "reused"
				return []*model.MonitorConfig{{Model: model.Model{ID: 101}, PoolID: 7, ConfigType: 4, InstanceIP: "192.0.2.1"}}, 101, nil
			}
			d := &targetedMetadataStub{pool: func(_ context.Context, r configdao.GeneratedConfigPoolQuery) ([]*model.MonitorConfig, int64, error) {
				return fetch(r.Page)
			}, namespace: func(_ context.Context, r configdao.GeneratedConfigKeysQuery) ([]*model.MonitorConfig, int64, error) {
				return fetch(r.Page)
			}}
			var rows []*model.MonitorConfig
			var err error
			if path == "pool" {
				rows, err = readPoolConfigCandidates(context.Background(), d, 7, 4)
			} else {
				rows, err = readConfigKeyNamespace(context.Background(), d, 4, []string{"192.0.2.1"})
			}
			if err != nil || rows[0].Name != "captured" {
				t.Fatal("page reuse mutated typed snapshot")
			}
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			before := d.poolCalls + d.namespaceCalls
			if path == "pool" {
				_, err = readPoolConfigCandidates(ctx, d, 7, 4)
			} else {
				_, err = readConfigKeyNamespace(ctx, d, 4, []string{"192.0.2.1"})
			}
			if !errors.Is(err, context.Canceled) || before != d.poolCalls+d.namespaceCalls {
				t.Fatal("cancelled typed read consulted storage")
			}
		})
	}
	d := &targetedMetadataStub{}
	rows, err := readConfigKeyNamespace(context.Background(), d, 4, nil)
	if err != nil || len(rows) != 0 || d.namespaceCalls != 0 {
		t.Fatal("empty key set was treated as all namespace")
	}
}
func TestTargetedPublisherRejectsLocatorOwnerDriftButUsesFreshOwnedContent(t *testing.T) {
	for _, domain := range []string{"record", "prom", "alert"} {
		for _, change := range []string{"pool", "name", "kind", "IP", "missing", "body", "status"} {
			t.Run(domain+"/"+change, func(t *testing.T) {
				h := newRetirementFixture(t, domain)
				ctx := context.Background()
				if err := h.publisher.GenerateMainConfig(ctx); err != nil {
					t.Fatal(err)
				}
				var base configdao.MonitorConfigDAO
				switch c := h.publisher.(type) {
				case *recordRuleConfigCache:
					base = c.configDAO
				case *prometheusConfigCache:
					base = c.configDAO
				case *alertRuleConfigCache:
					base = c.configDAO
				}
				var target *model.MonitorConfig
				for _, row := range h.rows {
					if row.InstanceIP == "127.0.0.1" {
						target = row
					}
				}
				if target == nil {
					t.Fatal("baseline owned row absent")
				}
				changed := false
				d := &targetedMetadataStub{MonitorConfigDAO: base, pool: func(ctx context.Context, req configdao.GeneratedConfigPoolQuery) ([]*model.MonitorConfig, int64, error) {
					rows, total, err := base.GetGeneratedConfigPoolCandidates(ctx, req)
					if !changed {
						changed = true
						switch change {
						case "pool":
							target.PoolID = 99
						case "name":
							target.Name = "custom owner"
						case "kind":
							target.ConfigType = 5
						case "IP":
							target.InstanceIP = "127.0.0.9"
						case "missing":
							delete(h.rows, target.ID)
						case "body":
							target.ConfigContent = "drifted body"
						case "status":
							target.Status = 2
						}
						if domain == "record" || domain == "alert" {
							fixture := base.(*recordProjectionFixture)
							seed := fixture.db.Session(&gorm.Session{SkipHooks: true}).Unscoped()
							if change == "missing" {
								if seed.Delete(&model.MonitorConfig{}, target.ID).Error != nil {
									t.Fatal("fixture drift delete failed")
								}
							} else {
								copy := *target
								if seed.Save(&copy).Error != nil {
									t.Fatal("fixture drift write failed")
								}
							}
						}

					}
					return rows, total, err
				}, namespace: base.GetGeneratedConfigNamespacePage}
				switch c := h.publisher.(type) {
				case *recordRuleConfigCache:
					c.configDAO = d
				case *prometheusConfigCache:
					c.configDAO = d
				case *alertRuleConfigCache:
					c.configDAO = d
				}
				writes := h.writes()
				values := map[string]string{}
				for k, v := range h.values {
					values[k] = v
				}
				err := h.publisher.GenerateMainConfig(ctx)
				if change == "body" || change == "status" {
					current := h.rows[target.ID]
					if err != nil || current == nil || current.Status != 1 || current.ConfigContent != h.values[h.artifact("127.0.0.1")] {
						t.Fatal("same-owner fresh content/status did not repair")
					}
				} else if err == nil || h.writes() != writes || !reflect.DeepEqual(values, h.values) {
					t.Fatal("locator owner drift caused publication writes")
				}
				if d.fullCalls != 0 {
					t.Fatal("targeted plan used full-table fallback")
				}
			})
		}
	}
}
