package cache

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	configdao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/config"
)

func assertOldAMMainRetired(t *testing.T, r *alertContractRedis, db *alertContractConfigs, oldIPs ...string) {
	t.Helper()
	for _, ip := range oldIPs {
		if r.values[buildRedisKeyAlertManagerMain(ip)] != "" {
			t.Fatal("old main Redis artifact remains")
		}
		for _, row := range db.rows {
			if row.ConfigType == model.ConfigTypeAlertManager && row.InstanceIP == ip {
				t.Fatal("old main metadata remains")
			}
		}
	}
}
func TestAMMainDiscoversUnindexedCanonicalArtifactsAcrossRename(t *testing.T) {
	for _, rename := range []bool{false, true} {
		t.Run(fmt.Sprint(rename), func(t *testing.T) {
			c, _, r, db, pool, _ := alertContractCache(t)
			ctx := context.Background()
			if err := c.GenerateMainConfig(ctx); err != nil {
				t.Fatal(err)
			}
			clear(r.members[buildRedisSetKeyAlertManagerMainPoolIPs(pool.ID)])
			pool.AlertManagerInstances = model.StringList{"127.0.0.3"}
			if rename {
				pool.Name = "renamed"
			}
			if err := c.GenerateMainConfig(ctx); err != nil {
				t.Fatal(err)
			}
			assertOldAMMainRetired(t, r, db, "127.0.0.1", "127.0.0.2")
			if r.values[buildRedisKeyAlertManagerMain("127.0.0.3")] == "" {
				t.Fatal("current target was removed")
			}
		})
	}
}
func TestAMMainUnindexedRetirementSurvivesWriteFailuresAndRebuild(t *testing.T) {
	for _, stage := range []string{"members write", "main delete", "members delete", "hash write"} {
		t.Run(stage, func(t *testing.T) {
			c, _, r, db, pool, _ := alertContractCache(t)
			ctx := context.Background()
			if err := c.GenerateMainConfig(ctx); err != nil {
				t.Fatal(err)
			}
			clear(r.members[buildRedisSetKeyAlertManagerMainPoolIPs(pool.ID)])
			pool.AlertManagerInstances = model.StringList{"127.0.0.3"}
			pool.Name = "renamed"
			beforeSQL := db.writes
			r.failure = stage
			if err := c.GenerateMainConfig(ctx); err == nil {
				t.Fatal("injected failure reported completion")
			}
			if stage == "members write" && (db.writes <= beforeSQL || !amFixturePending(c, 2, pool.ID)) {
				t.Fatal("failed external membership write lost committed SQL retirement authority")
			}
			r.failure = ""
			c = rebuildAlertRetirementABA(c)
			if err := c.GenerateMainConfig(ctx); err != nil {
				t.Fatal(err)
			}
			assertOldAMMainRetired(t, r, db, "127.0.0.1", "127.0.0.2")
			if r.values[buildRedisKeyAlertManagerMain("127.0.0.3")] == "" {
				t.Fatal("current target lost during retry")
			}
		})
	}
}
func TestAMMainSameACKStillDiscoversDBOnlyRetirement(t *testing.T) {
	c, _, r, db, pool, _ := alertContractCache(t)
	ctx := context.Background()
	if err := c.GenerateMainConfig(ctx); err != nil {
		t.Fatal(err)
	}
	beforeACK := r.values[alertPublicationStateKey(pool.ID)]
	db.next++
	db.rows[db.next] = &model.MonitorConfig{Model: model.Model{ID: db.next}, PoolID: pool.ID, ConfigType: model.ConfigTypeAlertManager, InstanceIP: "127.0.0.9", Name: fmt.Sprintf(ConfigNameAlertManager, pool.ID, "127.0.0.9"), Status: 1}
	r.values[buildRedisKeyAlertManagerMain("127.0.0.9")] = "old"
	if err := c.GenerateMainConfig(ctx); err != nil {
		t.Fatal(err)
	}
	assertOldAMMainRetired(t, r, db, "127.0.0.9")
	if r.values[alertPublicationStateKey(pool.ID)] != beforeACK {
		t.Fatal("source version unexpectedly changed")
	}
}
func TestAMMainOwnershipConflictsFailBeforeAnyWrites(t *testing.T) {
	for _, position := range []string{"current", "indexed retired", "unindexed mixed retired"} {
		for _, mode := range []string{"foreign", "custom inactive", "foreign inactive", "duplicate owned"} {
			t.Run(position+"/"+mode, func(t *testing.T) {
				c, _, r, db, pool, _ := alertContractCache(t)
				ctx := context.Background()
				if err := c.GenerateMainConfig(ctx); err != nil {
					t.Fatal(err)
				}
				var target *model.MonitorConfig
				for _, row := range db.rows {
					if row.ConfigType == model.ConfigTypeAlertManager && row.InstanceIP == "127.0.0.2" {
						target = row
					}
				}
				if target == nil {
					t.Fatal("fixture missing main")
				}
				if position == "unindexed mixed retired" || mode == "duplicate owned" {
					copy := *target
					db.next++
					copy.ID = db.next
					db.rows[copy.ID] = &copy
					target = &copy
				}
				switch mode {
				case "foreign":
					target.PoolID = 99
				case "custom inactive":
					target.Name = "operator custom"
					target.Status = model.ConfigStatusInactive
				case "foreign inactive":
					target.PoolID = 99
					target.Status = model.ConfigStatusInactive
				}
				if position != "current" {
					pool.AlertManagerInstances = model.StringList{"127.0.0.1"}
				}
				if position == "unindexed mixed retired" {
					clear(r.members[buildRedisSetKeyAlertManagerMainPoolIPs(pool.ID)])
				}
				dw, rw := db.writes, r.writes
				if err := c.GenerateMainConfig(ctx); err == nil {
					t.Fatal("ownership conflict accepted")
				}
				if db.writes != dw || r.writes != rw {
					t.Fatal("ownership conflict wrote artifacts or pending")
				}
			})
		}
	}
}

type amMainInventoryPages struct {
	configdao.MonitorConfigDAO
	page func(int) ([]*model.MonitorConfig, int64, error)
}

func (s amMainInventoryPages) GetMonitorConfigList(ctx context.Context, req *model.GetMonitorConfigListReq) ([]*model.MonitorConfig, int64, error) {
	if req.ConfigType != nil && *req.ConfigType == model.ConfigTypeAlertManager {
		return s.page(req.Page)
	}
	return s.MonitorConfigDAO.GetMonitorConfigList(ctx, req)
}
func TestAMMainFullMetadataInventoryRejectsContradictionsBeforeWrites(t *testing.T) {
	for _, mode := range []string{"error", "early empty", "total drift", "negative total", "duplicate ID", "missing ID"} {
		t.Run(mode, func(t *testing.T) {
			c, _, r, db, pool, _ := alertContractCache(t)
			ctx := context.Background()
			if err := c.GenerateMainConfig(ctx); err != nil {
				t.Fatal(err)
			}
			clear(r.members[buildRedisSetKeyAlertManagerMainPoolIPs(pool.ID)])
			pool.AlertManagerInstances = model.StringList{"127.0.0.3"}
			first := make([]*model.MonitorConfig, 100)
			for i := range first {
				first[i] = &model.MonitorConfig{Model: model.Model{ID: i + 100}, ConfigType: model.ConfigTypeAlertManager, PoolID: 99, InstanceIP: fmt.Sprintf("192.0.2.%d", i+1), Name: "foreign"}
			}
			c.configDAO = amMainInventoryPages{MonitorConfigDAO: db, page: func(page int) ([]*model.MonitorConfig, int64, error) {
				if page == 1 {
					if mode == "negative total" {
						return nil, -1, nil
					}
					return first, 101, nil
				}
				switch mode {
				case "error":
					return nil, 0, errors.New("fixture page failure")
				case "early empty":
					return nil, 101, nil
				case "total drift":
					return []*model.MonitorConfig{{Model: model.Model{ID: 1000}, ConfigType: model.ConfigTypeAlertManager}}, 102, nil
				case "duplicate ID":
					return []*model.MonitorConfig{first[0]}, 101, nil
				default:
					return []*model.MonitorConfig{{ConfigType: model.ConfigTypeAlertManager}}, 101, nil
				}
			}}
			dw, rw := db.writes, r.writes
			if err := c.GenerateMainConfig(ctx); err == nil {
				t.Fatal("incomplete metadata inventory accepted")
			}
			if db.writes != dw || r.writes != rw {
				t.Fatal("incomplete scan authorized early writes")
			}
		})
	}
}
func TestAMMainInventoryClonesRowsBeforePageReuse(t *testing.T) {
	c, _, _, db, pool, _ := alertContractCache(t)
	first := make([]*model.MonitorConfig, 100)
	for i := range first {
		first[i] = &model.MonitorConfig{Model: model.Model{ID: i + 1}, ConfigType: model.ConfigTypeAlertManager, PoolID: 99, Name: "foreign"}
	}
	first[0] = &model.MonitorConfig{Model: model.Model{ID: 1}, ConfigType: model.ConfigTypeAlertManager, PoolID: pool.ID, InstanceIP: "127.0.0.9", Name: fmt.Sprintf(ConfigNameAlertManager, pool.ID, "127.0.0.9")}
	c.configDAO = amMainInventoryPages{MonitorConfigDAO: db, page: func(page int) ([]*model.MonitorConfig, int64, error) {
		if page == 1 {
			return first, 101, nil
		}
		first[0].Name = "reused custom"
		return []*model.MonitorConfig{{Model: model.Model{ID: 101}, ConfigType: model.ConfigTypeAlertManager, PoolID: 99, Name: "foreign"}}, 101, nil
	}}
	owners, ips, err := c.checkRetiredAlertMainOwnership(context.Background(), pool.ID, nil, map[string]bool{})
	if err != nil || len(ips) != 1 || owners["127.0.0.9"].Name != fmt.Sprintf(ConfigNameAlertManager, pool.ID, "127.0.0.9") {
		t.Fatal("later page reuse mutated the captured owner")
	}
}

func (s amMainInventoryPages) GetGeneratedConfigPoolCandidates(ctx context.Context, req configdao.GeneratedConfigPoolQuery) ([]*model.MonitorConfig, int64, error) {
	if req.ConfigType == model.ConfigTypeAlertManager {
		return s.page(req.Page)
	}
	return s.MonitorConfigDAO.GetGeneratedConfigPoolCandidates(ctx, req)
}
