//go:build integration

package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	appconfig "github.com/GoSimplicity/AI-CloudOps/internal/config"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	alertdao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/alert"
	configdao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/config"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// The source DAOs, journal, sealed SQL lease and Redis runner are all real.
// Tracking compiles the same lease without executing a second publication.
type rootAMTrackingRunner struct {
	t      *testing.T
	client *redis.Client
	store  *ProjectionStore
	next   AMProjectionRunner
}

func (r *rootAMTrackingRunner) Project(ctx context.Context, lease configdao.ProjectionLease) error {
	snapshot, err := lease.Snapshot()
	if err != nil {
		return err
	}
	handle, err := compileProjection(r.store, lease, snapshot)
	if err != nil {
		return err
	}
	projectionTrack(r.t, r.client, handle)
	return r.next.Project(ctx, lease)
}

type rootAMNativeFixture struct {
	t        *testing.T
	ctx      context.Context
	client   *redis.Client
	db       *gorm.DB
	metadata configdao.MonitorConfigDAO
	journal  configdao.AMProjectionJournalDAO
	pools    alertdao.AlertManagerPoolDAO
	groups   alertdao.AlertManagerSendDAO
	pool     *model.MonitorAlertManagerPool
	group    *model.MonitorSendGroup
	store    *ProjectionStore
}

func rootAMNative(t *testing.T, id int) *rootAMNativeFixture {
	t.Helper()
	client := projectionRedisNative(t)
	_, metadata, db := projectionStoreSQL(t)
	if err := db.AutoMigrate(&model.MonitorAlertManagerPool{}, &model.MonitorSendGroup{}); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	logger := zap.NewNop()
	f := &rootAMNativeFixture{t: t, ctx: ctx, client: client, db: db, metadata: metadata,
		journal: configdao.NewAMProjectionJournalDAO(logger, db),
		pools:   alertdao.NewAlertManagerPoolDAO(db, logger), groups: alertdao.NewAlertManagerSendDAO(db, logger)}
	f.pool = &model.MonitorAlertManagerPool{Model: model.Model{ID: id}, Name: "native-am", AlertManagerInstances: model.StringList{"127.0.98.1"},
		ResolveTimeout: "5m", GroupWait: "30s", GroupInterval: "5m", RepeatInterval: "1h", GroupBy: model.StringList{"alertname"}, Receiver: "native-group"}
	if err := f.pools.CreateMonitorAlertManagerPool(ctx, f.pool); err != nil {
		t.Fatal(err)
	}
	f.group = &model.MonitorSendGroup{Model: model.Model{ID: id}, PoolID: id, Name: "native-group", NameZh: "native", Enable: 1, RepeatInterval: "1h", SendResolved: 1}
	if err := f.groups.CreateMonitorSendGroup(ctx, f.group); err != nil {
		t.Fatal(err)
	}
	store, err := NewProjectionStore(client)
	if err != nil {
		t.Fatal(err)
	}
	f.store = store
	t.Cleanup(func() { store.Close() })
	return f
}

func (f *rootAMNativeFixture) publisher(webhook string) AlertManagerConfigCache {
	return f.publisherWithSources(webhook, f.pools, f.groups)
}

func (f *rootAMNativeFixture) publisherWithSources(webhook string, pools alertdao.AlertManagerPoolDAO, groups alertdao.AlertManagerSendDAO) AlertManagerConfigCache {
	f.t.Helper()
	cfg := &appconfig.Config{}
	cfg.Server.Port = "8891"
	cfg.Prometheus.AlertWebhookAddr = webhook
	cfg.Prometheus.AlertWebhookFileDir = "/native/hooks"
	logger := zap.NewNop()
	runner := &rootAMTrackingRunner{t: f.t, client: f.client, store: f.store, next: NewAMProjectionRunner(f.store)}
	pub, err := NewAlertManagerConfigCache(cfg, logger, pools, groups, f.metadata, NewBatchConfigManager(f.metadata, logger), f.client,
		f.journal, configdao.NewAMProjectionHistoryReader(logger, f.db), runner)
	if err != nil {
		f.t.Fatal(err)
	}
	return pub
}

type rootAMPoolSnapshotHook struct {
	alertdao.AlertManagerPoolDAO
	afterSnapshot func()
	once          sync.Once
}

func (d *rootAMPoolSnapshotHook) GetMonitorAlertManagerPoolList(ctx context.Context, req *model.GetMonitorAlertManagerPoolListReq) ([]*model.MonitorAlertManagerPool, int64, error) {
	rows, total, err := d.AlertManagerPoolDAO.GetMonitorAlertManagerPoolList(ctx, req)
	if err == nil {
		d.once.Do(d.afterSnapshot)
	}
	return rows, total, err
}

func (d *rootAMPoolSnapshotHook) GetAlertPoolByID(ctx context.Context, id int) (*model.MonitorAlertManagerPool, error) {
	row, err := d.AlertManagerPoolDAO.GetAlertPoolByID(ctx, id)
	if err == nil {
		d.once.Do(d.afterSnapshot)
	}
	return row, err
}

type rootAMGroupSnapshotHook struct {
	alertdao.AlertManagerSendDAO
	afterSnapshot func()
	once          sync.Once
}

func (d *rootAMGroupSnapshotHook) GetMonitorSendGroupByPoolID(ctx context.Context, id int) ([]*model.MonitorSendGroup, int64, error) {
	rows, total, err := d.AlertManagerSendDAO.GetMonitorSendGroupByPoolID(ctx, id)
	if err == nil {
		d.once.Do(d.afterSnapshot)
	}
	return rows, total, err
}

func TestAMProjectionNativeOlderSourceCannotReplacePeerPublication(t *testing.T) {
	for index, scenario := range []struct {
		name     string
		subset   bool
		poolRead bool
	}{
		{"full pool snapshot", false, true},
		{"full group snapshot", false, false},
		{"subset pool snapshot", true, true},
		{"subset group snapshot", true, false},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			f := rootAMNative(t, 9820+index)
			newer := f.publisher("http://native-newer.invalid")
			if err := newer.GenerateMainConfig(f.ctx); err != nil {
				t.Fatal(err)
			}
			var full *configdao.ProjectionReceipt
			var subset *configdao.ProjectionReceipt
			keys := []string{buildRedisKeyAlertManagerMain(f.pool.AlertManagerInstances[0]), buildRedisKeyWebhookFile(f.pool.AlertManagerInstances[0], f.group.ID), buildRedisSetKeyAlertManagerMainPoolIPs(f.pool.ID), alertPublicationStateKey(f.pool.ID), alertRetirementIntentKey(f.pool.ID), projectionPoolKey(2, f.pool.ID, "ack"), projectionPoolKey(2, f.pool.ID, "fence"), projectionPoolKey(5, f.pool.ID, "ack"), projectionPoolKey(5, f.pool.ID, "fence")}
			var baseline map[string]rootProjectionValue
			called := false
			publishNewer := func() {
				called = true
				if scenario.poolRead {
					if err := f.db.WithContext(f.ctx).Model(&model.MonitorAlertManagerPool{}).Where("id = ?", f.pool.ID).Update("group_wait", "45s").Error; err != nil {
						t.Fatal(err)
					}
				} else {
					if err := f.db.WithContext(f.ctx).Model(&model.MonitorSendGroup{}).Where("id = ?", f.group.ID).Update("repeat_interval", "2h").Error; err != nil {
						t.Fatal(err)
					}
				}
				if err := newer.GenerateMainConfig(f.ctx); err != nil {
					t.Fatal(err)
				}
				current, err := f.pools.GetAlertPoolByID(f.ctx, f.pool.ID)
				if err != nil {
					t.Fatal(err)
				}
				if r, s := newer.GenerateRouteConfigForPool(f.ctx, current); len(r) == 0 || len(s) == 0 {
					t.Fatal("new peer subset did not publish")
				}
				full = f.projected(2)
				subset, err = f.journal.LoadLastProjectedProjection(f.ctx, 5, f.pool.ID)
				if err != nil {
					t.Fatal(err)
				}
				baseline = rootProjectionValues(t, f.client, keys)
			}
			pools := f.pools
			groups := f.groups
			if scenario.poolRead {
				pools = &rootAMPoolSnapshotHook{AlertManagerPoolDAO: pools, afterSnapshot: publishNewer}
			} else {
				groups = &rootAMGroupSnapshotHook{AlertManagerSendDAO: groups, afterSnapshot: publishNewer}
			}
			older := f.publisherWithSources("http://native-older.invalid", pools, groups)
			if scenario.subset {
				older.GenerateRouteConfigForPool(f.ctx, f.pool)
			} else {
				_ = older.GenerateMainConfig(f.ctx)
			}
			if !called {
				t.Fatal("actual AM source-read barrier was not reached")
			}
			currentSubset, err := f.journal.LoadLastProjectedProjection(f.ctx, 5, f.pool.ID)
			if err != nil {
				t.Fatal(err)
			}
			if f.projected(2).Reservation != full.Reservation || !reflect.DeepEqual(currentSubset, subset) {
				t.Fatal("older AM source replaced a completed peer publication")
			}
			if !reflect.DeepEqual(baseline, rootProjectionValues(t, f.client, keys)) {
				t.Fatal("older AM source changed peer raw, sets, ACK, SHA or fences")
			}
		})
	}
}

func (f *rootAMNativeFixture) projected(family int8) *configdao.ProjectionReceipt {
	f.t.Helper()
	got, err := f.journal.LoadLastProjectedProjection(f.ctx, family, f.pool.ID)
	if err != nil || got == nil || got.AMControls == nil {
		f.t.Fatal("AM publication has no durable paired-control receipt", err)
	}
	committed, err := f.journal.LoadLastCommittedProjection(f.ctx, family, f.pool.ID)
	if err != nil || committed == nil || committed.Reservation != got.Reservation {
		f.t.Fatal("AM committed and projected operations differ", err)
	}
	return got
}

func TestAMProjectionNativeSubsetIsolationAndStableNoop(t *testing.T) {
	f := rootAMNative(t, 9811)
	first := f.publisher("http://native-before.invalid")
	if err := first.GenerateMainConfig(f.ctx); err != nil {
		t.Fatal(err)
	}
	f.projected(2)
	protected := []string{buildRedisKeyAlertManagerMain(f.pool.AlertManagerInstances[0]), buildRedisSetKeyAlertManagerMainPoolIPs(f.pool.ID), alertPublicationStateKey(f.pool.ID), alertRetirementIntentKey(f.pool.ID), projectionPoolKey(2, f.pool.ID, "ack"), projectionPoolKey(2, f.pool.ID, "fence")}
	before := rootProjectionValues(t, f.client, protected)
	webhookKey := buildRedisKeyWebhookFile(f.pool.AlertManagerInstances[0], f.group.ID)
	oldBody, err := f.client.Get(f.ctx, webhookKey).Result()
	if err != nil {
		t.Fatal("initial webhook missing", err)
	}
	second := f.publisher("http://native-after.invalid")
	routes, receivers := second.GenerateRouteConfigForPool(f.ctx, f.pool)
	if len(routes) == 0 || len(receivers) == 0 {
		t.Fatal("subset wrapper failed to preserve rendered route contract")
	}
	f.projected(5)
	newBody, err := f.client.Get(f.ctx, webhookKey).Result()
	if err != nil || newBody == oldBody {
		t.Fatal("subset did not update its webhook artifact", err)
	}
	if !reflect.DeepEqual(before, rootProjectionValues(t, f.client, protected)) {
		t.Fatal("subset changed full-pool raw, membership, SHA, intents, ACK or fence")
	}
	if err := second.GenerateMainConfig(f.ctx); err != nil {
		t.Fatal(err)
	}
	full := f.projected(2)
	subset := f.projected(5)
	for i := 0; i < 3; i++ {
		if r, s := second.GenerateRouteConfigForPool(f.ctx, f.pool); len(r) == 0 || len(s) == 0 {
			t.Fatal("repeated subset failed")
		}
		if err := second.GenerateMainConfig(f.ctx); err != nil {
			t.Fatal(err)
		}
	}
	if f.projected(2).Reservation != full.Reservation || f.projected(5).Reservation != subset.Reservation {
		t.Fatal("unchanged full/subset alternation created new operations")
	}
}

func TestAMProjectionNativeEmptyGroupsRetireOnlyWebhook(t *testing.T) {
	f := rootAMNative(t, 9812)
	pub := f.publisher("http://native-empty.invalid")
	if err := pub.GenerateMainConfig(f.ctx); err != nil {
		t.Fatal(err)
	}
	f.projected(2)
	if err := f.db.WithContext(f.ctx).Model(&model.MonitorSendGroup{}).Where("id = ?", f.group.ID).Update("enable", 2).Error; err != nil {
		t.Fatal(err)
	}
	if err := pub.GenerateMainConfig(f.ctx); err != nil {
		t.Fatal(err)
	}
	f.projected(2)
	ip := f.pool.AlertManagerInstances[0]
	if _, err := f.client.Get(f.ctx, buildRedisKeyWebhookFile(ip, f.group.ID)).Result(); err != redis.Nil {
		t.Fatal("disabled group webhook was not retired", err)
	}
	if raw, err := f.client.Get(f.ctx, buildRedisKeyAlertManagerMain(ip)).Result(); err != nil || raw == "" {
		t.Fatal("empty group publication lost blackhole main", err)
	}
	if values, err := f.client.SMembers(f.ctx, buildRedisSetKeyAlertManagerMainPoolIPs(f.pool.ID)).Result(); err != nil || len(values) != 1 || values[0] != ip {
		t.Fatal("empty group publication lost pool membership", err)
	}
	if values, err := f.client.SMembers(f.ctx, alertRetirementIntentKey(f.pool.ID)).Result(); err != nil || len(values) != 0 {
		t.Fatal("completed retirement left pending webhook intents", err)
	}
}

func TestAMProjectionNativeRejectsUnsavedSubsetPool(t *testing.T) {
	f := rootAMNative(t, 9813)
	pub := f.publisher("http://native-stale.invalid")
	if err := pub.GenerateMainConfig(f.ctx); err != nil {
		t.Fatal(err)
	}
	full := f.projected(2)
	keys := []string{buildRedisKeyAlertManagerMain(f.pool.AlertManagerInstances[0]), buildRedisKeyWebhookFile(f.pool.AlertManagerInstances[0], f.group.ID), alertPublicationStateKey(f.pool.ID), projectionPoolKey(2, f.pool.ID, "ack"), projectionPoolKey(5, f.pool.ID, "ack")}
	before := rootProjectionValues(t, f.client, keys)
	unsaved := *f.pool
	unsaved.AlertManagerInstances = model.StringList{"127.0.98.99"}
	routes, receivers := pub.GenerateRouteConfigForPool(f.ctx, &unsaved)
	if routes != nil || receivers != nil {
		t.Fatal("unsaved external pool snapshot accepted for subset publication")
	}
	if !reflect.DeepEqual(before, rootProjectionValues(t, f.client, keys)) || f.projected(2).Reservation != full.Reservation {
		t.Fatal("rejected unsaved subset changed full publication")
	}
	if n, err := f.client.Exists(f.ctx, buildRedisKeyWebhookFile("127.0.98.99", f.group.ID)).Result(); err != nil || n != 0 {
		t.Fatal("unsaved subset emitted an external snapshot artifact", err)
	}
}

func TestAMProjectionNativeFailedReservationDoesNotInvalidateLastGood(t *testing.T) {
	for _, family := range []int8{2, 5} {
		t.Run(map[int8]string{2: "full attempt", 5: "subset attempt"}[family], func(t *testing.T) {
			f := rootAMNative(t, 9830+int(family))
			pub := f.publisher("http://native-reserved.invalid")
			if err := pub.GenerateMainConfig(f.ctx); err != nil {
				t.Fatal(err)
			}
			good := f.projected(2)
			versions, err := f.journal.ReadAMSourceVersions(f.ctx)
			if err != nil {
				t.Fatal(err)
			}
			var source configdao.AMSourceVersion
			for _, candidate := range versions {
				if candidate.PoolID == f.pool.ID {
					source = candidate
				}
			}
			if source.PoolID != f.pool.ID {
				t.Fatal("paired source version missing")
			}
			if _, err := f.journal.ReserveAMPoolOperation(f.ctx, configdao.PoolOperationRequest{Family: family, PoolID: f.pool.ID, Nonce: [16]byte{0xa5, byte(family)}}, source); err != nil {
				t.Fatal(err)
			}
			keys := []string{buildRedisKeyAlertManagerMain(f.pool.AlertManagerInstances[0]), buildRedisKeyWebhookFile(f.pool.AlertManagerInstances[0], f.group.ID), buildRedisSetKeyAlertManagerMainPoolIPs(f.pool.ID), alertPublicationStateKey(f.pool.ID), alertRetirementIntentKey(f.pool.ID), projectionPoolKey(2, f.pool.ID, "ack"), projectionPoolKey(2, f.pool.ID, "fence")}
			before := rootProjectionValues(t, f.client, keys)
			if err := f.journal.WithPoolProjectionLease(f.ctx, *good, func(lease configdao.ProjectionLease) error {
				return NewAMProjectionRunner(f.store).Project(f.ctx, lease)
			}); err != nil {
				t.Fatal("uncommitted attempt invalidated durable last-good projection", err)
			}
			if !reflect.DeepEqual(before, rootProjectionValues(t, f.client, keys)) || f.projected(2).Reservation != good.Reservation {
				t.Fatal("last-good replay after reservation changed publication")
			}
		})
	}
}

func TestAMProjectionNativeDelayedSubsetCannotRevertFullPublication(t *testing.T) {
	for index, retire := range []bool{false, true} {
		t.Run(map[bool]string{false: "full render", true: "full retirement"}[retire], func(t *testing.T) {
			f := rootAMNative(t, 9840+index)
			initial := f.publisher("http://native-initial.invalid")
			if err := initial.GenerateMainConfig(f.ctx); err != nil {
				t.Fatal(err)
			}
			f.projected(2)
			queue := rootNewRecordQueue(t, f.client.Options().Addr)
			options := *f.client.Options()
			options.Addr = queue.listener.Addr().String()
			options.Protocol = 2
			options.PoolSize = 1
			options.MinIdleConns = 0
			proxyClient := redis.NewClient(&options)
			defer proxyClient.Close()
			delayedStore, err := NewProjectionStore(proxyClient)
			if err != nil {
				t.Fatal(err)
			}
			defer delayedStore.Close()
			delayedFixture := *f
			delayedFixture.store = delayedStore
			older := delayedFixture.publisher("http://native-delayed.invalid")
			finished := make(chan bool, 1)
			go func() {
				routes, receivers := older.GenerateRouteConfigForPool(f.ctx, f.pool)
				finished <- routes != nil || receivers != nil
			}()
			select {
			case <-queue.seen:
			case <-f.ctx.Done():
				t.Fatal("subset never sent its actual artifact request")
			}
			select {
			case accepted := <-finished:
				if accepted {
					t.Fatal("unknown Redis result falsely completed subset")
				}
			case <-f.ctx.Done():
				t.Fatal("subset did not release SQL lease after proxy EOF")
			}
			pending, err := f.journal.LoadLastCommittedProjection(f.ctx, 5, f.pool.ID)
			if err != nil || pending == nil || pending.Phase != model.ConfigPublicationCommitted {
				t.Fatal("subset did not leave a durable pending operation", err)
			}
			if retire {
				if err := f.db.WithContext(f.ctx).Model(&model.MonitorSendGroup{}).Where("id = ?", f.group.ID).Update("enable", 2).Error; err != nil {
					t.Fatal(err)
				}
			}
			newer := f.publisher("http://native-final.invalid")
			if err := newer.GenerateMainConfig(f.ctx); err != nil {
				t.Fatal("full publication did not supersede pending subset", err)
			}
			good := f.projected(2)
			keys := []string{buildRedisKeyAlertManagerMain(f.pool.AlertManagerInstances[0]), buildRedisKeyWebhookFile(f.pool.AlertManagerInstances[0], f.group.ID), projectionNamespaceKey(5, f.pool.AlertManagerInstances[0]+"_"+fmt.Sprint(f.group.ID)), buildRedisSetKeyAlertManagerMainPoolIPs(f.pool.ID), alertPublicationStateKey(f.pool.ID), alertRetirementIntentKey(f.pool.ID), projectionPoolKey(2, f.pool.ID, "ack"), projectionPoolKey(2, f.pool.ID, "fence"), projectionPoolKey(5, f.pool.ID, "ack"), projectionPoolKey(5, f.pool.ID, "fence")}
			baseline := rootProjectionValues(t, f.client, keys)
			queue.unblock()
			select {
			case result := <-queue.result:
				if result.err != nil || result.response != "STALE" {
					t.Fatal("delayed subset was not rejected by actual Redis fencing", result.err)
				}
			case <-f.ctx.Done():
				t.Fatal("delayed subset packet was not evaluated")
			}
			if !reflect.DeepEqual(baseline, rootProjectionValues(t, f.client, keys)) || f.projected(2).Reservation != good.Reservation {
				t.Fatal("delayed subset changed newer full publication")
			}
			if retire {
				if n, err := f.client.Exists(f.ctx, buildRedisKeyWebhookFile(f.pool.AlertManagerInstances[0], f.group.ID)).Result(); err != nil || n != 0 {
					t.Fatal("delayed subset resurrected retired webhook", err)
				}
			}
		})
	}
}

func TestAMProjectionNativeSameHashFindsDatabaseOnlyRetirement(t *testing.T) {
	for _, kind := range []int8{2, 5} {
		t.Run(map[int8]string{2: "main", 5: "webhook"}[kind], func(t *testing.T) {
			f := rootAMNative(t, 9850+int(kind))
			pub := f.publisher("http://native-orphan.invalid")
			if err := pub.GenerateMainConfig(f.ctx); err != nil {
				t.Fatal(err)
			}
			first := f.projected(2)
			sha, err := f.client.Get(f.ctx, alertPublicationStateKey(f.pool.ID)).Result()
			if err != nil {
				t.Fatal(err)
			}
			const oldIP = "127.0.98.77"
			key := buildRedisKeyAlertManagerMain(oldIP)
			instance := oldIP
			name := fmt.Sprintf(ConfigNameAlertManager, f.pool.ID, oldIP)
			body := pub.GetConfigByIP(f.pool.AlertManagerInstances[0])
			if kind == 5 {
				key = buildRedisKeyWebhookFile(oldIP, f.group.ID)
				instance = fmt.Sprintf("%s_%d", oldIP, f.group.ID)
				name = fmt.Sprintf("webhook_%s_%d_%s", f.pool.Name, f.group.ID, oldIP)
				body = "http://native-orphan.invalid/api/v1/alerts/receive?send_group_id=" + fmt.Sprint(f.group.ID)
			}
			row := &model.MonitorConfig{PoolID: f.pool.ID, ConfigType: kind, InstanceIP: instance, Name: name, ConfigContent: body, ConfigHash: calculateConfigHash(body), Status: model.ConfigStatusActive}
			if err := f.metadata.CreateMonitorConfig(f.ctx, row); err != nil {
				t.Fatal(err)
			}
			if err := f.client.Set(f.ctx, key, body, 0).Err(); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if err := f.client.Del(ctx, key).Err(); err != nil {
					t.Error("could not clean database-only artifact")
				}
			})
			if err := pub.GenerateMainConfig(f.ctx); err != nil {
				t.Fatal("same-hash publication did not reconcile database-only owner", err)
			}
			current := f.projected(2)
			if current.Reservation.Generation <= first.Reservation.Generation || current.SourceHash != first.SourceHash {
				t.Fatal("same source did not issue a fresh retirement plan")
			}
			if got, err := f.client.Get(f.ctx, alertPublicationStateKey(f.pool.ID)).Result(); err != nil || got != sha {
				t.Fatal("retirement changed the unchanged-source legacy SHA", err)
			}
			if n, err := f.client.Exists(f.ctx, key).Result(); err != nil || n != 0 {
				t.Fatal("database-only obsolete raw artifact survived", err)
			}
			var retired model.MonitorConfig
			if err := f.db.WithContext(f.ctx).Unscoped().First(&retired, row.ID).Error; err != nil || retired.DeletedAt == 0 {
				t.Fatal("obsolete metadata owner was not durably retired", err)
			}
		})
	}
}

func TestAMProjectionNativeCoverageRejectsUnissuedACKBytes(t *testing.T) {
	f := rootAMNative(t, 9861)
	pub := f.publisher("http://native-ack.invalid")
	if err := pub.GenerateMainConfig(f.ctx); err != nil {
		t.Fatal(err)
	}
	f.projected(2)
	key := projectionPoolKey(2, f.pool.ID, "ack")
	raw, err := f.client.Get(f.ctx, key).Result()
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &fields); err != nil {
		t.Fatal("issued ACK is not JSON", err)
	}
	fields["unissued_extension"] = json.RawMessage(`true`)
	tampered, err := json.Marshal(fields)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.client.Set(f.ctx, key, tampered, 0).Err(); err != nil {
		t.Fatal(err)
	}
	err = pub.GenerateMainConfig(f.ctx)
	if err == nil {
		current, readErr := f.client.Get(f.ctx, key).Result()
		if readErr != nil || current != raw {
			t.Fatal("coverage certified ACK bytes that the completed operation never issued", readErr)
		}
	}
	// Replaying the original sealed plan can restore its issued ACK bytes.
	// A conflict can also fail closed; neither path may accept the extension.
}

func TestAMProjectionNativeDelayedFullCannotRevertSubset(t *testing.T) {
	f := rootAMNative(t, 9871)
	initial := f.publisher("http://native-full-initial.invalid")
	if err := initial.GenerateMainConfig(f.ctx); err != nil {
		t.Fatal(err)
	}
	first := f.projected(2)
	queue := rootNewRecordQueue(t, f.client.Options().Addr)
	options := *f.client.Options()
	options.Addr = queue.listener.Addr().String()
	options.Protocol, options.PoolSize, options.MinIdleConns = 2, 1, 0
	proxyClient := redis.NewClient(&options)
	defer proxyClient.Close()
	delayedStore, err := NewProjectionStore(proxyClient)
	if err != nil {
		t.Fatal(err)
	}
	defer delayedStore.Close()
	delayedFixture := *f
	delayedFixture.store = delayedStore
	older := delayedFixture.publisher("http://native-full-delayed.invalid")
	finished := make(chan error, 1)
	go func() { finished <- older.GenerateMainConfig(f.ctx) }()
	select {
	case <-queue.seen:
	case <-f.ctx.Done():
		t.Fatal("full publisher did not send its artifact request")
	}
	select {
	case err := <-finished:
		if err == nil {
			t.Fatal("unknown Redis result falsely completed full publication")
		}
	case <-f.ctx.Done():
		t.Fatal("full publisher did not release SQL lease after proxy EOF")
	}
	pending, err := f.journal.LoadLastCommittedProjection(f.ctx, 2, f.pool.ID)
	if err != nil || pending == nil || pending.Phase != model.ConfigPublicationCommitted || pending.Reservation == first.Reservation {
		t.Fatal("full publisher did not leave a new durable pending plan", err)
	}
	newer := f.publisher("http://native-subset-final.invalid")
	if routes, receivers := newer.GenerateRouteConfigForPool(f.ctx, f.pool); len(routes) == 0 || len(receivers) == 0 {
		t.Fatal("subset did not supersede pending full webhook")
	}
	good := f.projected(5)
	ip := f.pool.AlertManagerInstances[0]
	keys := []string{buildRedisKeyAlertManagerMain(ip), buildRedisKeyWebhookFile(ip, f.group.ID), projectionNamespaceKey(2, ip), projectionNamespaceKey(5, fmt.Sprintf("%s_%d", ip, f.group.ID)), buildRedisSetKeyAlertManagerMainPoolIPs(f.pool.ID), alertPublicationStateKey(f.pool.ID), alertRetirementIntentKey(f.pool.ID), projectionPoolKey(2, f.pool.ID, "ack"), projectionPoolKey(2, f.pool.ID, "fence"), projectionPoolKey(5, f.pool.ID, "ack"), projectionPoolKey(5, f.pool.ID, "fence")}
	baseline := rootProjectionValues(t, f.client, keys)
	queue.unblock()
	select {
	case result := <-queue.result:
		if result.err != nil || result.response != "STALE" {
			t.Fatal("full packet did not observe the newer subset namespace fence", result.err)
		}
	case <-f.ctx.Done():
		t.Fatal("delayed full packet was not evaluated")
	}
	if !reflect.DeepEqual(baseline, rootProjectionValues(t, f.client, keys)) || f.projected(5).Reservation != good.Reservation {
		t.Fatal("delayed full packet changed completed subset publication")
	}
	if err := newer.GenerateMainConfig(f.ctx); err != nil {
		t.Fatal("full publication did not recover after the newer subset", err)
	}
	if f.projected(2).Reservation.Generation <= pending.Reservation.Generation {
		t.Fatal("full recovery did not issue a fresh current generation")
	}
}

func rootAMHoldIntentRemoval(t *testing.T, f *rootAMNativeFixture) *rootRecordQueue {
	t.Helper()
	queue := rootNewRecordQueueMatching(t, f.client.Options().Addr, func(frame rootRecordRESP) bool {
		return rootIsProjectionModeRequest(frame, "intent_remove")
	})
	options := *f.client.Options()
	options.Addr = queue.listener.Addr().String()
	options.Protocol, options.PoolSize, options.MinIdleConns = 2, 1, 0
	proxyClient := redis.NewClient(&options)
	t.Cleanup(func() { proxyClient.Close() })
	store, err := NewProjectionStore(proxyClient)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	delayed := *f
	delayed.store = store
	pub := delayed.publisher("http://native-aba.invalid")
	finished := make(chan error, 1)
	go func() { finished <- pub.GenerateMainConfig(f.ctx) }()
	switch wait, err := rootObserveHeldCommand(f.ctx, queue.seen, finished, &queue.held); wait {
	case rootHeldWaitAccepted:
	case rootHeldWaitMissed:
		t.Fatal("publication returned before its intent removal request", err)
	case rootHeldWaitCompleted:
		t.Fatal("unknown intent removal result falsely completed publication")
	case rootHeldWaitNotReached:
		t.Fatal("publication did not reach actual intent removal request")
	case rootHeldWaitNotReleased:
		t.Fatal("intent removal EOF did not release SQL lease")
	default:
		t.Fatal("intent removal wait returned an unknown result", wait, err)
	}
	return queue
}

func TestAMProjectionNativeRepeatedRenameAndSecondABARecover(t *testing.T) {
	for index, restoreNewRow := range []bool{false, true} {
		t.Run(map[bool]string{false: "same row alias", true: "deleted row restoration"}[restoreNewRow], func(t *testing.T) {
			f := rootAMNative(t, 9880+index)
			pub := f.publisher("http://native-aba.invalid")
			if err := pub.GenerateMainConfig(f.ctx); err != nil {
				t.Fatal(err)
			}
			f.projected(2)
			ip := f.pool.AlertManagerInstances[0]
			instance := fmt.Sprintf("%s_%d", ip, f.group.ID)
			owners, err := f.metadata.GetMonitorConfigsByInstances(f.ctx, []string{instance}, 5)
			if err != nil || len(owners) != 1 {
				t.Fatal("initial webhook owner missing", err)
			}
			oldID := owners[0].ID
			if restoreNewRow {
				if err := f.db.WithContext(f.ctx).Model(&model.MonitorAlertManagerPool{}).Where("id = ?", f.pool.ID).Update("alert_manager_instances", model.StringList{}).Error; err != nil {
					t.Fatal(err)
				}
			} else {
				// A released retry locator is exactly the old owner JSON. It is
				// not permission to delete or rename the current SQL owner.
				member, err := json.Marshal(configdao.GeneratedConfigOwnerOf(owners[0]))
				if err != nil {
					t.Fatal(err)
				}
				if err := f.client.SAdd(f.ctx, alertRetirementIntentKey(f.pool.ID), string(member)).Err(); err != nil {
					t.Fatal(err)
				}
				if err := f.db.WithContext(f.ctx).Model(&model.MonitorAlertManagerPool{}).Where("id = ?", f.pool.ID).Update("name", "native-rename-one").Error; err != nil {
					t.Fatal(err)
				}
			}
			firstQueue := rootAMHoldIntentRemoval(t, f)
			if restoreNewRow {
				var retired model.MonitorConfig
				if err := f.db.WithContext(f.ctx).Unscoped().First(&retired, oldID).Error; err != nil || retired.DeletedAt == 0 {
					t.Fatal("first retirement did not durably tombstone the old owner", err)
				}
			}
			if n, err := f.client.SCard(f.ctx, alertRetirementIntentKey(f.pool.ID)).Result(); err != nil || n == 0 {
				t.Fatal("unknown removal lost its durable retry locator", err)
			}
			if err := f.db.WithContext(f.ctx).Model(&model.MonitorAlertManagerPool{}).Where("id = ?", f.pool.ID).Updates(map[string]any{"name": "native-rename-two", "alert_manager_instances": model.StringList{ip}}).Error; err != nil {
				t.Fatal(err)
			}
			secondQueue := rootAMHoldIntentRemoval(t, f)
			restored, err := f.metadata.GetMonitorConfigsByInstances(f.ctx, []string{instance}, 5)
			if err != nil || len(restored) != 1 {
				t.Fatal("restored or renamed owner missing", err)
			}
			if restoreNewRow == (restored[0].ID == oldID) {
				t.Fatal("test did not exercise its distinct-row or same-row identity contract")
			}
			if err := f.db.WithContext(f.ctx).Model(&model.MonitorSendGroup{}).Where("id = ?", f.group.ID).Update("enable", 2).Error; err != nil {
				t.Fatal(err)
			}
			if err := f.db.WithContext(f.ctx).Model(&model.MonitorAlertManagerPool{}).Where("id = ?", f.pool.ID).Update("name", "native-rename-three").Error; err != nil {
				t.Fatal(err)
			}
			if err := pub.GenerateMainConfig(f.ctx); err != nil {
				t.Fatal("repeated rename or second ABA prevented final retirement", err)
			}
			good := f.projected(2)
			if n, err := f.client.SCard(f.ctx, alertRetirementIntentKey(f.pool.ID)).Result(); err != nil || n != 0 {
				t.Fatal("completed retirement retained old retry locators", err)
			}
			if n, err := f.client.Exists(f.ctx, buildRedisKeyWebhookFile(ip, f.group.ID)).Result(); err != nil || n != 0 {
				t.Fatal("completed retirement retained the restored webhook", err)
			}
			keys := []string{buildRedisKeyAlertManagerMain(ip), buildRedisKeyWebhookFile(ip, f.group.ID), projectionNamespaceKey(5, instance), buildRedisSetKeyAlertManagerMainPoolIPs(f.pool.ID), alertRetirementIntentKey(f.pool.ID), alertPublicationStateKey(f.pool.ID), projectionPoolKey(2, f.pool.ID, "ack"), projectionPoolKey(2, f.pool.ID, "fence")}
			baseline := rootProjectionValues(t, f.client, keys)
			for _, queue := range []*rootRecordQueue{firstQueue, secondQueue} {
				queue.unblock()
				select {
				case result := <-queue.result:
					if result.err != nil || result.response != "STALE" {
						t.Fatal("old intent removal packet was not fenced by the completed recovery", result.err)
					}
				case <-f.ctx.Done():
					t.Fatal("old intent removal packet was not evaluated")
				}
			}
			if !reflect.DeepEqual(baseline, rootProjectionValues(t, f.client, keys)) {
				t.Fatal("old intent removal packet changed completed recovery")
			}
			if err := f.publisher("http://native-aba.invalid").GenerateMainConfig(f.ctx); err != nil {
				t.Fatal("rebuilt publisher forgot completed retirement", err)
			}
			if f.projected(2).Reservation != good.Reservation {
				t.Fatal("completed retirement did not converge to a stable no-op")
			}
		})
	}
}

func TestAMProjectionNativeMalformedIntentFailsBeforeReplayWrites(t *testing.T) {
	f := rootAMNative(t, 9891)
	pub := f.publisher("http://native-invalid-intent.invalid")
	if err := pub.GenerateMainConfig(f.ctx); err != nil {
		t.Fatal(err)
	}
	good := f.projected(2)
	if err := f.client.SAdd(f.ctx, alertRetirementIntentKey(f.pool.ID), "not-json").Err(); err != nil {
		t.Fatal(err)
	}
	versions, err := f.journal.ReadAMSourceVersions(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	ip := f.pool.AlertManagerInstances[0]
	keys := []string{buildRedisKeyAlertManagerMain(ip), buildRedisKeyWebhookFile(ip, f.group.ID), buildRedisSetKeyAlertManagerMainPoolIPs(f.pool.ID), alertRetirementIntentKey(f.pool.ID), alertPublicationStateKey(f.pool.ID), projectionPoolKey(2, f.pool.ID, "ack"), projectionPoolKey(2, f.pool.ID, "fence")}
	before := rootProjectionValues(t, f.client, keys)
	// Observe an actual write attempt before forwarding it, including an
	// idempotent write whose final DUMP would otherwise hide the side effect.
	queue := rootNewRecordQueueMatching(t, f.client.Options().Addr, func(frame rootRecordRESP) bool {
		if len(frame.args) == 0 {
			return false
		}
		switch strings.ToLower(string(frame.args[0])) {
		case "eval", "evalsha", "set", "del", "sadd", "srem", "hset":
			return true
		}
		return false
	})
	options := *f.client.Options()
	options.Addr = queue.listener.Addr().String()
	options.Protocol, options.PoolSize, options.MinIdleConns = 2, 1, 0
	observedClient := redis.NewClient(&options)
	t.Cleanup(func() { observedClient.Close() })
	store, err := NewProjectionStore(observedClient)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	observed := *f
	observed.client, observed.store = observedClient, store
	if err := observed.publisher("http://native-invalid-intent.invalid").GenerateMainConfig(f.ctx); err == nil {
		t.Fatal("malformed intent falsely completed same-hash publication")
	}
	if queue.held.Load() {
		t.Fatal("malformed intent reached a Redis write before readonly rejection")
	}
	currentVersions, err := f.journal.ReadAMSourceVersions(f.ctx)
	if err != nil || !reflect.DeepEqual(currentVersions, versions) {
		t.Fatal("malformed intent changed source counters or committed controls", err)
	}
	if f.projected(2).Reservation != good.Reservation || !reflect.DeepEqual(before, rootProjectionValues(t, f.client, keys)) {
		t.Fatal("malformed intent changed last-good projection")
	}
}
