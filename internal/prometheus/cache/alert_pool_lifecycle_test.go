package cache

import (
	"context"
	"errors"
	"fmt"
	configdao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/config"
	"sync"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/config"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	alertdao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/alert"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/plugin/soft_delete"
)

type deletedAMInventory struct {
	alertdao.AlertManagerPoolDAO
	pool    *model.MonitorAlertManagerPool
	failure bool
	page    func(int) ([]*model.MonitorAlertManagerPool, int64, error)
}

func (s *deletedAMInventory) GetMonitorAlertManagerPoolList(context.Context, *model.GetMonitorAlertManagerPoolListReq) ([]*model.MonitorAlertManagerPool, int64, error) {
	if s.failure {
		return nil, 0, errors.New("fixture source failure")
	}
	if s.pool == nil || s.pool.DeletedAt > 0 {
		return nil, 0, nil
	}
	return []*model.MonitorAlertManagerPool{s.pool}, 1, nil
}
func (s *deletedAMInventory) GetMonitorAlertManagerPoolLifecycleList(_ context.Context, req *model.GetMonitorAlertManagerPoolListReq) ([]*model.MonitorAlertManagerPool, int64, error) {
	if s.failure {
		return nil, 0, errors.New("fixture lifecycle failure")
	}
	if s.page != nil {
		return s.page(req.Page)
	}
	if s.pool == nil {
		return nil, 0, nil
	}
	return []*model.MonitorAlertManagerPool{s.pool}, 1, nil
}
func (s *deletedAMInventory) GetMonitorAlertManagerPoolTombstone(_ context.Context, id int) (*model.MonitorAlertManagerPool, error) {
	if s.failure || s.pool == nil || s.pool.ID != id || s.pool.DeletedAt == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return cloneAlertPoolLifecycle(s.pool), nil
}
func markAMDeleted(pool *model.MonitorAlertManagerPool) {
	pool.DeletedAt = soft_delete.DeletedAt(1730000000)
	pool.Name = ""
	pool.AlertManagerInstances = model.StringList{"invalid old target"}
	pool.ResolveTimeout = "invalid"
}
func assertDeletedAMRetired(t *testing.T, r *alertContractRedis, db *alertContractConfigs) {
	t.Helper()
	for _, ip := range []string{"127.0.0.1", "127.0.0.2"} {
		if r.values[buildRedisKeyAlertManagerMain(ip)] != "" || r.values[buildRedisKeyWebhookFile(ip, 3)] != "" {
			t.Fatal("deleted AM artifact remains")
		}
	}
	for _, row := range db.rows {
		if row.PoolID == 7 && (row.ConfigType == model.ConfigTypeAlertManager || row.ConfigType == model.ConfigTypeWebhookFile) {
			t.Fatal("deleted owned AM metadata remains")
		}
	}
	if len(r.members[buildRedisSetKeyAlertManagerMainPoolIPs(7)]) != 0 || len(r.members[alertRetirementIntentKey(7)]) != 0 {
		t.Fatal("deleted AM retirement locator remains")
	}
}
func TestDeletedAMPoolRetiresBothKindsWithoutReadingBrokenGroups(t *testing.T) {
	for _, mode := range []string{"explicit", "periodic", "DB-only"} {
		t.Run(mode, func(t *testing.T) {
			c, g, r, db, pool, _ := alertContractCache(t)
			ctx := context.Background()
			inv := &deletedAMInventory{pool: pool}
			c.alertPoolDAO = inv
			if err := c.GenerateMainConfig(ctx); err != nil {
				t.Fatal(err)
			}
			calls := g.calls
			oldACK := r.values[alertPublicationStateKey(7)]
			for _, kind := range []int8{1, 3, 4} {
				db.next++
				db.rows[db.next] = &model.MonitorConfig{Model: model.Model{ID: db.next}, PoolID: 7, ConfigType: kind, Name: "same-ID Scrape output"}
			}
			if mode == "DB-only" {
				clear(r.members[buildRedisSetKeyAlertManagerMainPoolIPs(7)])
			}
			markAMDeleted(pool)
			g.fail = true
			var err error
			if mode == "explicit" {
				err = c.RetireDeletedAlertManagerPool(ctx, 7)
			} else {
				err = c.ReconcileDeletedAlertManagerPools(ctx)
			}
			if err != nil {
				t.Fatal(err)
			}
			assertDeletedAMRetired(t, r, db)
			if len(db.rows) != 3 || g.calls != calls || r.values[alertPublicationStateKey(7)] == oldACK {
				t.Fatal("deletion borrowed Scrape ownership, read groups or reused active ACK")
			}
			c = rebuildAlertRetirementABA(c)
			if err := c.ReconcileDeletedAlertManagerPools(ctx); err != nil {
				t.Fatal(err)
			}
			assertDeletedAMRetired(t, r, db)
		})
	}
}
func TestDeletedAMPoolFailuresRecoverFromTombstoneAfterRebuild(t *testing.T) {
	for _, stage := range []string{"hash read", "members read", "webhook list", "pending write", "members write", "intent add", "main delete", "members delete", "webhook DB delete", "webhook delete", "intent remove", "hash write"} {
		t.Run(stage, func(t *testing.T) {
			c, g, r, db, pool, _ := alertContractCache(t)
			ctx := context.Background()
			c.alertPoolDAO = &deletedAMInventory{pool: pool}
			if err := c.GenerateMainConfig(ctx); err != nil {
				t.Fatal(err)
			}
			clear(r.members[buildRedisSetKeyAlertManagerMainPoolIPs(7)])
			markAMDeleted(pool)
			g.fail = true
			wrapped := &alertRetirementRedis{alertContractRedis: r}
			c.redis = wrapped
			r.failure = stage
			db.failure = stage
			wrapped.failIntentAdd = stage == "intent add"
			wrapped.failIntentRemove = stage == "intent remove"
			if err := c.RetireDeletedAlertManagerPool(ctx, 7); err == nil {
				t.Fatal("injected retirement failure returned complete")
			}
			if stage == "pending write" || stage == "members write" || stage == "intent add" {
				if !amFixturePending(c, 2, pool.ID) {
					t.Fatal("failed external anchor lost durable SQL retirement plan")
				}
				for _, ip := range []string{"127.0.0.1", "127.0.0.2"} {
					if r.values[buildRedisKeyAlertManagerMain(ip)] == "" || r.values[buildRedisKeyWebhookFile(ip, 3)] == "" {
						t.Fatal("failed anchor changed visible output")
					}
				}
			}
			r.failure = ""
			db.failure = ""
			wrapped.failIntentAdd = false
			wrapped.failIntentRemove = false
			c = rebuildAlertRetirementABA(c)
			if err := c.ReconcileDeletedAlertManagerPools(ctx); err != nil {
				t.Fatal(err)
			}
			assertDeletedAMRetired(t, r, db)
		})
	}
}
func TestDeletedAMPoolRequiresTrueTombstoneAndPreservesConflicts(t *testing.T) {
	for _, mode := range []string{"active", "absent", "read error", "main foreign", "main custom inactive", "main mixed", "webhook mixed foreign", "webhook mixed custom inactive"} {
		t.Run(mode, func(t *testing.T) {
			c, _, r, db, pool, _ := alertContractCache(t)
			ctx := context.Background()
			inv := &deletedAMInventory{pool: pool}
			c.alertPoolDAO = inv
			if err := c.GenerateMainConfig(ctx); err != nil {
				t.Fatal(err)
			}
			if mode != "active" {
				markAMDeleted(pool)
			}
			switch mode {
			case "absent":
				inv.pool = nil
			case "read error":
				inv.failure = true
			default:
				for _, row := range db.rows {
					if row.ConfigType == model.ConfigTypeAlertManager && row.PoolID == 7 && row.InstanceIP == "127.0.0.2" {
						switch mode {
						case "main foreign":
							row.PoolID = 99
						case "main custom inactive":
							row.Name = "operator custom"
							row.Status = 2
						case "main mixed":
							copy := *row
							db.next++
							copy.ID = db.next
							copy.PoolID = 99
							db.rows[copy.ID] = &copy
						}
					}
					if row.ConfigType == model.ConfigTypeWebhookFile && row.InstanceIP == "127.0.0.2_3" && (mode == "webhook mixed foreign" || mode == "webhook mixed custom inactive") {
						copy := *row
						db.next++
						copy.ID = db.next
						if mode == "webhook mixed foreign" {
							copy.PoolID = 99
						} else {
							copy.Name = "custom"
							copy.Status = 2
						}
						db.rows[copy.ID] = &copy
						break
					}
				}
			}
			beforeDB, beforeRedis := copyAlertRows(db.rows), copyAlertRedis(r.values)
			rw := r.writes
			if err := c.RetireDeletedAlertManagerPool(ctx, 7); err == nil {
				t.Fatal("invalid deletion authorization/owner accepted")
			}
			if rw != r.writes || !sameAlertRetirementState(beforeRedis, r.values, beforeDB, db.rows) {
				t.Fatal("preflight conflict wrote metadata or Redis")
			}
		})
	}
}
func TestDeletedAMPoolInventoryFailureNeverAuthorizesEarlyRetirement(t *testing.T) {
	for _, mode := range []string{"error", "early empty", "total drift", "duplicate ID", "negative total"} {
		t.Run(mode, func(t *testing.T) {
			c, _, r, db, pool, _ := alertContractCache(t)
			ctx := context.Background()
			if err := c.GenerateMainConfig(ctx); err != nil {
				t.Fatal(err)
			}
			markAMDeleted(pool)
			first := []*model.MonitorAlertManagerPool{pool}
			for id := 10; len(first) < 100; id++ {
				first = append(first, &model.MonitorAlertManagerPool{Model: model.Model{ID: id}})
			}
			inv := &deletedAMInventory{pool: pool, page: func(page int) ([]*model.MonitorAlertManagerPool, int64, error) {
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
					return []*model.MonitorAlertManagerPool{{Model: model.Model{ID: 1000}}}, 102, nil
				default:
					return []*model.MonitorAlertManagerPool{pool}, 101, nil
				}
			}}
			c.alertPoolDAO = inv
			rw := r.writes
			rows := copyAlertRows(db.rows)
			values := copyAlertRedis(r.values)
			if err := c.ReconcileDeletedAlertManagerPools(ctx); err == nil {
				t.Fatal("incomplete lifecycle inventory authorized cleanup")
			}
			if rw != r.writes || !sameAlertRetirementState(values, r.values, rows, db.rows) {
				t.Fatal("early lifecycle page wrote outputs")
			}
		})
	}
}
func TestDeletedAMPoolWaitsForOldInflightAndRespectsCancellation(t *testing.T) {
	for _, cancelled := range []bool{false, true} {
		t.Run(fmt.Sprint(cancelled), func(t *testing.T) {
			c, g, r, db, pool, _ := alertContractCache(t)
			c.alertPoolDAO = &deletedAMInventory{pool: pool}
			held, release := make(chan struct{}), make(chan struct{})
			var once sync.Once
			r.onHashRead = func() { once.Do(func() { close(held); <-release }) }
			published := make(chan error, 1)
			go func() { published <- c.GenerateMainConfig(context.Background()) }()
			<-held
			markAMDeleted(pool)
			g.fail = true
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if cancelled {
				cancel()
			}
			retired := make(chan error, 1)
			go func() { retired <- c.RetireDeletedAlertManagerPool(ctx, 7) }()
			close(release)
			if err := <-published; err != nil {
				t.Fatal(err)
			}
			err := <-retired
			if cancelled {
				if !errors.Is(err, context.Canceled) {
					t.Fatal("cancelled retirement lost cancellation")
				}
				if err := c.ReconcileDeletedAlertManagerPools(context.Background()); err != nil {
					t.Fatal(err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
			assertDeletedAMRetired(t, r, db)
			if err := c.GenerateMainConfig(context.Background()); err != nil {
				t.Fatal(err)
			}
			assertDeletedAMRetired(t, r, db)
		})
	}
}
func TestDeletedAMPoolPreservesSecondABAOwnerIntentContracts(t *testing.T) {
	for _, replacement := range []string{"owned active", "owned inactive", "foreign", "custom inactive"} {
		t.Run(replacement, func(t *testing.T) {
			c, _, r, db, pool, _ := alertContractCache(t)
			ctx := context.Background()
			c.alertPoolDAO = &deletedAMInventory{pool: pool}
			if err := c.GenerateMainConfig(ctx); err != nil {
				t.Fatal(err)
			}
			wrapped := &alertRetirementRedis{alertContractRedis: r}
			c.redis = wrapped
			pool.AlertManagerInstances = model.StringList{"127.0.0.1"}
			r.failure = "webhook delete"
			if err := c.GenerateMainConfig(ctx); err == nil {
				t.Fatal("first retirement fault missing")
			}
			r.failure = ""
			wrapped.failIntentRemove = true
			pool.AlertManagerInstances = model.StringList{"127.0.0.1", "127.0.0.2"}
			c = rebuildAlertRetirementABA(c)
			if err := c.GenerateMainConfig(ctx); err == nil {
				t.Fatal("ABA cancellation fault missing")
			}
			var row *model.MonitorConfig
			for _, item := range db.rows {
				if item.ConfigType == model.ConfigTypeWebhookFile && item.InstanceIP == "127.0.0.2_3" {
					row = item
				}
			}
			if row == nil {
				t.Fatal("replacement absent")
			}
			switch replacement {
			case "owned inactive":
				row.Status = 2
			case "foreign":
				row.PoolID = 99
			case "custom inactive":
				row.Name = "custom"
				row.Status = 2
			}
			wrapped.failIntentRemove = false
			markAMDeleted(pool)
			c = rebuildAlertRetirementABA(c)
			rw := r.writes
			beforeRows, beforeRedis := copyAlertRows(db.rows), copyAlertRedis(r.values)
			err := c.RetireDeletedAlertManagerPool(ctx, 7)
			if replacement == "owned active" || replacement == "owned inactive" {
				if err != nil {
					t.Fatal(err)
				}
				assertDeletedAMRetired(t, r, db)
			} else if err == nil || rw != r.writes || !sameAlertRetirementState(beforeRedis, r.values, beforeRows, db.rows) {
				t.Fatal("ABA foreign/custom owner was not protected before writes")
			}
		})
	}
}

type deletedAMMetadataPages struct {
	*alertContractConfigs
	kind int8
	page func(int) ([]*model.MonitorConfig, int64, error)
}

func (s deletedAMMetadataPages) GetMonitorConfigList(ctx context.Context, req *model.GetMonitorConfigListReq) ([]*model.MonitorConfig, int64, error) {
	if req.ConfigType != nil && *req.ConfigType == s.kind {
		return s.page(req.Page)
	}
	return s.alertContractConfigs.GetMonitorConfigList(ctx, req)
}
func TestDeletedAMPoolBothMetadataKindsRequireCompleteInventory(t *testing.T) {
	for _, kind := range []int8{model.ConfigTypeAlertManager, model.ConfigTypeWebhookFile} {
		for _, mode := range []string{"error", "early empty", "total drift", "duplicate ID"} {
			t.Run(fmt.Sprintf("%d/%s", kind, mode), func(t *testing.T) {
				c, _, r, db, pool, _ := alertContractCache(t)
				ctx := context.Background()
				c.alertPoolDAO = &deletedAMInventory{pool: pool}
				if err := c.GenerateMainConfig(ctx); err != nil {
					t.Fatal(err)
				}
				markAMDeleted(pool)
				first := make([]*model.MonitorConfig, 100)
				for i := range first {
					first[i] = &model.MonitorConfig{Model: model.Model{ID: i + 100}, PoolID: 99, ConfigType: kind, Name: "foreign"}
				}
				c.configDAO = deletedAMMetadataPages{alertContractConfigs: db, kind: kind, page: func(page int) ([]*model.MonitorConfig, int64, error) {
					if page == 1 {
						return first, 101, nil
					}
					switch mode {
					case "error":
						return nil, 0, errors.New("fixture page failure")
					case "early empty":
						return nil, 101, nil
					case "total drift":
						return []*model.MonitorConfig{{Model: model.Model{ID: 1000}, ConfigType: kind}}, 102, nil
					default:
						return []*model.MonitorConfig{first[0]}, 101, nil
					}
				}}
				rw := r.writes
				rows, values := copyAlertRows(db.rows), copyAlertRedis(r.values)
				if err := c.RetireDeletedAlertManagerPool(ctx, 7); err == nil {
					t.Fatal("incomplete metadata accepted as authoritative retirement")
				}
				if rw != r.writes || !sameAlertRetirementState(values, r.values, rows, db.rows) {
					t.Fatal("incomplete metadata wrote output/pending")
				}
			})
		}
	}
}
func TestDeletedAMPoolLifecycleCopiesRowsAndSlicesBeforePageReuse(t *testing.T) {
	first := make([]*model.MonitorAlertManagerPool, 100)
	for i := range first {
		first[i] = &model.MonitorAlertManagerPool{Model: model.Model{ID: i + 1}}
	}
	first[0].DeletedAt = 10
	first[0].GroupBy = model.StringList{"alertname"}
	first[0].AlertManagerInstances = model.StringList{"127.0.0.1"}
	inv := &deletedAMInventory{page: func(page int) ([]*model.MonitorAlertManagerPool, int64, error) {
		if page == 1 {
			return first, 101, nil
		}
		first[0].ID = 999
		first[0].DeletedAt = 0
		first[0].GroupBy[0] = "reused"
		first[0].AlertManagerInstances[0] = "invalid"
		return []*model.MonitorAlertManagerPool{{Model: model.Model{ID: 101}}}, 101, nil
	}}
	rows, err := readAlertPoolLifecycleSnapshot(context.Background(), inv)
	if err != nil || rows[0].ID != 1 || rows[0].DeletedAt != 10 || rows[0].GroupBy[0] != "alertname" || rows[0].AlertManagerInstances[0] != "127.0.0.1" {
		t.Fatal("later page mutation changed captured deletion authorization")
	}
}

func TestDeletedAMPoolMalformedIntentRejectsBeforeAnyWrites(t *testing.T) {
	for _, entry := range []string{"not-json", "{}", `{"ID":1,"unexpected":true}`} {
		t.Run(entry, func(t *testing.T) {
			c, _, r, db, pool, _ := alertContractCache(t)
			ctx := context.Background()
			c.alertPoolDAO = &deletedAMInventory{pool: pool}
			if err := c.GenerateMainConfig(ctx); err != nil {
				t.Fatal(err)
			}
			markAMDeleted(pool)
			r.members[alertRetirementIntentKey(7)] = map[string]bool{entry: true}
			rw := r.writes
			rows, values := copyAlertRows(db.rows), copyAlertRedis(r.values)
			if err := c.RetireDeletedAlertManagerPool(ctx, 7); err == nil {
				t.Fatal("invalid retirement intent accepted")
			}
			if rw != r.writes || !sameAlertRetirementState(values, r.values, rows, db.rows) {
				t.Fatal("invalid intent wrote partial cleanup")
			}
		})
	}
}

type deletedAMManagerTask struct {
	AlertManagerConfigCache
	normalFailure error
}

func (t deletedAMManagerTask) GenerateMainConfig(context.Context) error { return t.normalFailure }
func TestDeletedAMPoolRecoveryPrecedesFailingNormalGeneration(t *testing.T) {
	c, g, r, db, pool, _ := alertContractCache(t)
	ctx := context.Background()
	c.alertPoolDAO = &deletedAMInventory{pool: pool}
	if err := c.GenerateMainConfig(ctx); err != nil {
		t.Fatal(err)
	}
	markAMDeleted(pool)
	g.fail = true
	normalFailure := errors.New("normal generation is paused/unavailable")
	manager, err := NewMonitorCache(&config.Config{}, &monitorPromTask{}, deletedAMManagerTask{AlertManagerConfigCache: c, normalFailure: normalFailure}, &monitorRuleTask{}, &monitorRecordTask{}, zap.NewNop())
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.MonitorCacheManager(ctx); !errors.Is(err, normalFailure) {
		t.Fatal("normal generation failure contract changed")
	}
	assertDeletedAMRetired(t, r, db)
}

func (s deletedAMMetadataPages) GetGeneratedConfigPoolCandidates(ctx context.Context, req configdao.GeneratedConfigPoolQuery) ([]*model.MonitorConfig, int64, error) {
	if req.ConfigType == s.kind {
		return s.page(req.Page)
	}
	return s.alertContractConfigs.MonitorConfigDAO.GetGeneratedConfigPoolCandidates(ctx, req)
}
