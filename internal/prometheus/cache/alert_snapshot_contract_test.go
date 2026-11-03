package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	altconfig "github.com/prometheus/alertmanager/config"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/config"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
)

func TestAlertSnapshotIsDetachedAndStable(t *testing.T) {
	ctx := context.Background()
	c, g, r, db, pool, _ := alertContractCache(t)
	r.onHashRead = func() {
		r.onHashRead = nil
		g.rows[0].RepeatInterval = "2h"
		pool.GroupWait = "60s"
		c.alertWebhookAddr = "http://changed-constructor.invalid"
	}
	if err := c.GenerateMainConfig(ctx); err != nil {
		t.Fatal(err)
	}
	key := buildRedisKeyAlertManagerMain("127.0.0.1")
	hashKey := alertPublicationStateKey(pool.ID)
	firstHash := r.values[hashKey]
	cfg := parseAlertContract(t, r.values[key])
	if cfg.Route.GroupWait.String() != "30s" || cfg.Route.Routes[0].RepeatInterval.String() != "1h" || g.calls != 1 {
		t.Fatal("renderer reread a mutable source after taking its snapshot")
	}
	hookKey := buildRedisKeyWebhookFile("127.0.0.1", 3)
	if !strings.Contains(r.values[hookKey], "private-webhook-url-canary") || strings.Contains(r.values[hookKey], "changed-constructor") {
		t.Fatal("Redis webhook did not use the hashed webhook snapshot")
	}
	for _, row := range db.rows {
		if row.ConfigType == model.ConfigTypeWebhookFile && row.InstanceIP == "127.0.0.1_3" && row.ConfigContent != r.values[hookKey] {
			t.Fatal("DB webhook and Redis webhook used different snapshots")
		}
	}
	if err := c.GenerateMainConfig(ctx); err != nil {
		t.Fatal(err)
	}
	cfg = parseAlertContract(t, r.values[key])
	if cfg.Route.GroupWait.String() != "1m" || cfg.Route.Routes[0].RepeatInterval.String() != "2h" || g.calls != 2 || r.values[hashKey] == firstHash || !strings.Contains(r.values[hookKey], "changed-constructor") {
		t.Fatal("next generation did not publish the changed input")
	}
}
func TestAlertSourceOrderDoesNotRepublish(t *testing.T) {
	c, g, r, db, pool, _ := alertContractCache(t)
	second := *g.rows[0]
	second.ID = 4
	second.Name = "another-group"
	g.rows = append(g.rows, &second)
	if err := c.GenerateMainConfig(context.Background()); err != nil {
		t.Fatal(err)
	}
	hashKey := alertPublicationStateKey(pool.ID)
	before := r.values[hashKey]
	writes := db.writes
	g.rows[0], g.rows[1] = g.rows[1], g.rows[0]
	if err := c.GenerateMainConfig(context.Background()); err != nil {
		t.Fatal(err)
	}
	if r.values[hashKey] != before || db.writes != writes {
		t.Fatal("unchanged source reordered routes, receivers or snapshot hash")
	}
}
func TestAlertCleanupPreservesForeignAndCustomRows(t *testing.T) {
	c, g, r, db, pool, _ := alertContractCache(t)
	ctx := context.Background()
	if err := c.GenerateMainConfig(ctx); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name string
		row  model.MonitorConfig
	}{
		{"other pool", model.MonitorConfig{PoolID: 99, ConfigType: model.ConfigTypeWebhookFile, Name: "webhook_other_901_127.0.0.2", InstanceIP: "127.0.0.2_901", ConfigContent: "keep"}},
		{"custom name", model.MonitorConfig{PoolID: pool.ID, ConfigType: model.ConfigTypeWebhookFile, Name: "custom-user-config", InstanceIP: "127.0.0.2_902", ConfigContent: "keep"}},
		{"other kind", model.MonitorConfig{PoolID: pool.ID, ConfigType: model.ConfigTypePrometheus, Name: "webhook_fixture_903_127.0.0.2", InstanceIP: "127.0.0.2_903", ConfigContent: "keep"}},
		{"mismatched generated key", model.MonitorConfig{PoolID: pool.ID, ConfigType: model.ConfigTypeWebhookFile, Name: "webhook_fixture_904_127.0.0.2", InstanceIP: "127.0.0.2_905", ConfigContent: "keep"}},
		{"no generated key", model.MonitorConfig{PoolID: pool.ID, ConfigType: model.ConfigTypeWebhookFile, Name: "webhook_fixture_906_127.0.0.2", InstanceIP: "custom", ConfigContent: "keep"}},
	}
	for i := range cases {
		db.next++
		cases[i].row.ID = db.next
		copy := cases[i].row
		db.rows[copy.ID] = &copy
	}
	for id := 901; id <= 906; id++ {
		r.values[buildRedisKeyWebhookFile("127.0.0.2", id)] = "keep"
	}
	g.rows = nil
	if err := c.GenerateMainConfig(ctx); err != nil {
		t.Fatal(err)
	}
	for _, item := range cases {
		if db.rows[item.row.ID] == nil || db.rows[item.row.ID].ConfigContent != "keep" {
			t.Fatalf("cleanup deleted %s", item.name)
		}
	}
	for id := 901; id <= 906; id++ {
		if r.values[buildRedisKeyWebhookFile("127.0.0.2", id)] != "keep" {
			t.Fatal("cleanup guessed an unowned Redis key")
		}
	}
	for _, row := range db.rows {
		if row.PoolID == pool.ID && row.ConfigType == model.ConfigTypeWebhookFile && strings.HasPrefix(row.Name, "webhook_private-pool-name-canary_") {
			t.Fatal("old owned webhook survived")
		}
	}
}
func TestAlertCleanupReadsAllMetadataPages(t *testing.T) {
	c, g, _, db, _, _ := alertContractCache(t)
	for i := 1; i <= 110; i++ {
		db.next++
		db.rows[db.next] = &model.MonitorConfig{Model: model.Model{ID: db.next}, PoolID: 100 + i, ConfigType: model.ConfigTypeWebhookFile, Name: fmt.Sprintf("custom-%d", i), InstanceIP: fmt.Sprintf("custom-%d", i), ConfigContent: "keep"}
	}
	if err := c.GenerateMainConfig(context.Background()); err != nil {
		t.Fatal(err)
	}
	g.rows = nil
	if err := c.GenerateMainConfig(context.Background()); err != nil {
		t.Fatal(err)
	}
	kept := 0
	for _, row := range db.rows {
		if row.ConfigType == model.ConfigTypeWebhookFile {
			kept++
		}
	}
	if kept != 110 {
		t.Fatal("paged cleanup skipped owned rows or deleted foreign rows")
	}
}
func TestAlertPublicRouteWrapperKeepsSuccessfulSideEffects(t *testing.T) {
	c, _, r, db, pool, _ := alertContractCache(t)
	routes, receivers := c.GenerateRouteConfigForPool(context.Background(), pool)
	if len(routes) != 1 || len(receivers) != 1 || len(db.rows) != 2 {
		t.Fatal("public wrapper lost route or webhook persistence")
	}
	for _, ip := range pool.AlertManagerInstances {
		if r.values[buildRedisKeyWebhookFile(ip, 3)] == "" {
			t.Fatal("public wrapper did not publish webhook file")
		}
		if r.values[buildRedisKeyAlertManagerMain(ip)] != "" {
			t.Fatal("route wrapper published a main configuration")
		}
	}
	db.failure = "webhook DB"
	c.alertWebhookAddr = "http://rotated.invalid"
	routes, receivers = c.GenerateRouteConfigForPool(context.Background(), pool)
	if routes != nil || receivers != nil {
		t.Fatal("public wrapper hid failed side-effect publication")
	}
}
func TestAlertInvalidSnapshotPreservesLastGood(t *testing.T) {
	for _, mode := range []string{"nil group", "invalid group ID", "duplicate receiver", "mixed wildcard group labels", "invalid route interval"} {
		t.Run(mode, func(t *testing.T) {
			c, g, r, _, pool, logs := alertContractCache(t)
			ctx := context.Background()
			if err := c.GenerateMainConfig(ctx); err != nil {
				t.Fatal(err)
			}
			key := buildRedisKeyAlertManagerMain("127.0.0.1")
			hashKey := alertPublicationStateKey(pool.ID)
			before, hash := r.values[key], r.values[hashKey]
			switch mode {
			case "nil group":
				g.rows = []*model.MonitorSendGroup{nil}
			case "invalid group ID":
				g.rows[0].ID = 0
			case "duplicate receiver":
				copy := *g.rows[0]
				copy.ID = 4
				g.rows = append(g.rows, &copy)
			case "mixed wildcard group labels":
				pool.GroupBy = model.StringList{"...", "alertname"}
			case "invalid route interval":
				pool.GroupInterval = "0s"
			}
			if c.GenerateMainConfig(ctx) == nil {
				t.Fatal("invalid snapshot reported success")
			}
			if r.values[key] != before || r.values[hashKey] != hash {
				t.Fatal("invalid snapshot changed or acknowledged last-good")
			}
			assertAlertLogsSafe(t, logs)
		})
	}
}
func TestAlertZeroEnabledStartsWithSafeConfiguration(t *testing.T) {
	for _, enable := range []int8{0, 2} {
		t.Run(fmt.Sprint(enable), func(t *testing.T) {
			c, g, r, db, _, _ := alertContractCache(t)
			g.rows[0].Enable = enable
			if err := c.GenerateMainConfig(context.Background()); err != nil {
				t.Fatal(err)
			}
			cfg := parseAlertContract(t, r.values[buildRedisKeyAlertManagerMain("127.0.0.1")])
			if len(cfg.Route.Routes) != 0 || len(cfg.Receivers) != 1 || len(cfg.Receivers[0].WebhookConfigs) != 0 {
				t.Fatal("zero enabled source fabricated an active receiver")
			}
			for _, row := range db.rows {
				if row.ConfigType == model.ConfigTypeWebhookFile {
					t.Fatal("zero enabled source fabricated a webhook file")
				}
			}
		})
	}
}
func TestAlertCancellationDoesNotPublishOrAcknowledge(t *testing.T) {
	c, _, r, db, pool, _ := alertContractCache(t)
	if err := c.GenerateMainConfig(context.Background()); err != nil {
		t.Fatal(err)
	}
	key := buildRedisKeyAlertManagerMain("127.0.0.1")
	hashKey := alertPublicationStateKey(pool.ID)
	before, hash, writes := r.values[key], r.values[hashKey], db.writes
	ctx, cancel := context.WithCancel(context.Background())
	pool.GroupWait = "45s"
	r.onHashRead = cancel
	if err := c.GenerateMainConfig(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal("cancellation lost")
	}
	if r.values[key] != before || r.values[hashKey] != hash || db.writes != writes {
		t.Fatal("canceled snapshot published or acknowledged")
	}
}
func TestAlertConcurrentGenerationsAreSerialized(t *testing.T) {
	c, g, r, _, _, _ := alertContractCache(t)
	const count = 8
	var wg sync.WaitGroup
	failures := make(chan error, count)
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := c.GenerateMainConfig(context.Background()); err != nil {
				failures <- err
			}
		}()
	}
	wg.Wait()
	close(failures)
	for err := range failures {
		t.Fatal(err)
	}
	if g.calls != count {
		t.Fatal("concurrent generation dropped a source snapshot")
	}
	parseAlertContract(t, r.values[buildRedisKeyAlertManagerMain("127.0.0.1")])
}

func TestAlertRenameUpdatesWebhookMetadataWithUnchangedBody(t *testing.T) {
	c, g, r, db, pool, _ := alertContractCache(t)
	ctx := context.Background()
	if err := c.GenerateMainConfig(ctx); err != nil {
		t.Fatal(err)
	}
	var previous model.MonitorConfig
	for _, row := range db.rows {
		if row.ConfigType == model.ConfigTypeWebhookFile && row.InstanceIP == "127.0.0.1_3" {
			previous = *row
		}
	}
	if previous.ID == 0 {
		t.Fatal("baseline webhook metadata missing")
	}
	pool.Name = "renamed-pool"
	if err := c.GenerateMainConfig(ctx); err != nil {
		t.Fatal(err)
	}
	current := db.rows[previous.ID]
	if current == nil || current.Name != "webhook_renamed-pool_3_127.0.0.1" || current.ConfigHash != previous.ConfigHash || current.ConfigContent != previous.ConfigContent {
		t.Fatal("same webhook body skipped file-name metadata rotation")
	}
	parsed := parseAlertContract(t, r.values[buildRedisKeyAlertManagerMain("127.0.0.1")])
	if parsed.Receivers[0].WebhookConfigs[0].URLFile != "/typed/hooks/webhook_renamed-pool_3.yml" {
		t.Fatal("main file and webhook metadata names disagree")
	}
	g.rows = nil
	if err := c.GenerateMainConfig(ctx); err != nil {
		t.Fatal(err)
	}
	for _, row := range db.rows {
		if row.ConfigType == model.ConfigTypeWebhookFile {
			t.Fatal("renamed owned webhook was not retired")
		}
	}
}
func TestAlertSaveDoesNotAdoptForeignOrCustomArtifact(t *testing.T) {
	for _, kind := range []int8{model.ConfigTypeAlertManager, model.ConfigTypeWebhookFile} {
		for _, ownership := range []string{"foreign", "custom"} {
			t.Run(fmt.Sprintf("%d-%s", kind, ownership), func(t *testing.T) {
				c, _, r, db, pool, _ := alertContractCache(t)
				ip := "127.0.0.1"
				if kind == model.ConfigTypeWebhookFile {
					ip += "_3"
				}
				row := &model.MonitorConfig{Model: model.Model{ID: 1}, PoolID: pool.ID, ConfigType: kind, Name: "custom-preserved", InstanceIP: ip, ConfigContent: "custom preserved body", ConfigHash: "custom-preserved-hash"}
				if ownership == "foreign" {
					row.PoolID = 99
				}
				db.rows[1] = row
				db.next = 1
				if c.GenerateMainConfig(context.Background()) == nil {
					t.Fatal("generator adopted an unowned artifact key")
				}
				after := db.rows[1]
				if after == nil || after.Name != row.Name || after.PoolID != row.PoolID || after.ConfigContent != "custom preserved body" || after.ConfigHash != "custom-preserved-hash" {
					t.Fatal("unowned artifact was overwritten or deleted")
				}
				if r.values[alertPublicationStateKey(pool.ID)] != "" {
					t.Fatal("conflicting artifact was acknowledged")
				}
			})
		}
	}
}

func TestAlertGroupBySerializationUsesSDKContract(t *testing.T) {
	for _, labels := range [][]string{{"service", "namespace"}, {"..."}, {"环境"}, {"...", "alertname"}, {"alertname", "alertname"}} {
		t.Run(fmt.Sprint(labels), func(t *testing.T) {
			rawLabels, _ := json.Marshal(labels)
			expected, parseErr := altconfig.Load("route:\n  receiver: noop\n  group_by: " + string(rawLabels) + "\nreceivers:\n  - name: noop\n")
			c, _, r, _, pool, _ := alertContractCache(t)
			pool.GroupBy = model.StringList(labels)
			err := c.GenerateMainConfig(context.Background())
			if (err == nil) != (parseErr == nil) {
				t.Fatal("generator and fixed SDK disagree about group_by validity")
			}
			if parseErr != nil {
				return
			}
			actual := parseAlertContract(t, r.values[buildRedisKeyAlertManagerMain("127.0.0.1")])
			if fmt.Sprint(actual.Route.GroupBy) != fmt.Sprint(expected.Route.GroupBy) || actual.Route.GroupByAll != expected.Route.GroupByAll {
				t.Fatal("configured grouping did not reach the serialized route")
			}
		})
	}
}

func TestAlertCleanupRejectsNoncanonicalWebhookKeys(t *testing.T) {
	for _, suffix := range []string{"03", "+3", "0", "-1", "invalid"} {
		t.Run(suffix, func(t *testing.T) {
			c, _, r, db, pool, _ := alertContractCache(t)
			ctx := context.Background()
			if err := c.GenerateMainConfig(ctx); err != nil {
				t.Fatal(err)
			}
			db.next++
			id := db.next
			db.rows[id] = &model.MonitorConfig{Model: model.Model{ID: id}, PoolID: pool.ID, ConfigType: model.ConfigTypeWebhookFile, Name: fmt.Sprintf("webhook_%s_3_127.0.0.1", pool.Name), InstanceIP: "127.0.0.1_" + suffix, ConfigContent: "unowned"}
			pool.GroupWait = "45s"
			if err := c.GenerateMainConfig(ctx); err != nil {
				t.Fatal(err)
			}
			if db.rows[id] == nil {
				t.Fatal("cleanup adopted a noncanonical metadata key")
			}
			if r.values[buildRedisKeyWebhookFile("127.0.0.1", 3)] == "" {
				t.Fatal("unowned metadata alias deleted an active webhook")
			}
		})
	}
}

type alertContractPagedPools struct {
	alertContractPools
	invalid *model.MonitorAlertManagerPool
	fail    bool
}

func (s alertContractPagedPools) GetMonitorAlertManagerPoolList(_ context.Context, req *model.GetMonitorAlertManagerPoolListReq) ([]*model.MonitorAlertManagerPool, int64, error) {
	if s.fail {
		return nil, 0, errors.New("private-backend-error-canary")
	}
	if req.Page == 1 {
		rows := []*model.MonitorAlertManagerPool{s.invalid}
		for i := 1; i < 100; i++ {
			rows = append(rows, &model.MonitorAlertManagerPool{Model: model.Model{ID: 100 + i}, Name: "empty-instance-pool"})
		}
		return rows, 101, nil
	}
	if req.Page == 2 {
		return []*model.MonitorAlertManagerPool{s.pool}, 101, nil
	}
	return nil, 101, nil
}
func TestAlertFailedPoolDoesNotStarveLaterPages(t *testing.T) {
	c, _, r, _, pool, _ := alertContractCache(t)
	bad := *pool
	bad.ID = 8
	bad.Name = "invalid-pool"
	bad.GroupBy = model.StringList{"...", "alertname"}
	c.alertPoolDAO = alertContractPagedPools{alertContractPools: alertContractPools{pool: pool}, invalid: &bad}
	if c.GenerateMainConfig(context.Background()) == nil {
		t.Fatal("invalid first page not reported")
	}
	parseAlertContract(t, r.values[buildRedisKeyAlertManagerMain("127.0.0.1")])
	if r.values[alertPublicationStateKey(pool.ID)] == "" {
		t.Fatal("failed first page starved valid later page")
	}
}
func TestAlertPoolReadFailureIsSanitized(t *testing.T) {
	c, _, r, _, _, logs := alertContractCache(t)
	ctx := context.Background()
	if err := c.GenerateMainConfig(ctx); err != nil {
		t.Fatal(err)
	}
	key := buildRedisKeyAlertManagerMain("127.0.0.1")
	before := r.values[key]
	c.alertPoolDAO = alertContractPagedPools{fail: true}
	err := c.GenerateMainConfig(ctx)
	if err == nil || strings.Contains(err.Error(), "private-backend-error-canary") || r.values[key] != before {
		t.Fatal("pool read failure lost last-good or exposed backend contents")
	}
	assertAlertLogsSafe(t, logs)
}

type alertCanceledWrite struct {
	*alertContractConfigs
	cancel context.CancelFunc
}

func (s alertCanceledWrite) BatchUpdateMonitorConfigs(ctx context.Context, rows []*model.MonitorConfig) error {
	for _, row := range rows {
		if row.ConfigType == model.ConfigTypeAlertManager {
			s.cancel()
			return context.Canceled
		}
	}
	return s.alertContractConfigs.BatchUpdateMonitorConfigs(ctx, rows)
}
func TestAlertCancellationDuringPersistenceRemainsDiscoverable(t *testing.T) {
	c, _, r, db, pool, _ := alertContractCache(t)
	if err := c.GenerateMainConfig(context.Background()); err != nil {
		t.Fatal(err)
	}
	key := buildRedisKeyAlertManagerMain("127.0.0.1")
	hashKey := alertPublicationStateKey(pool.ID)
	before, hash := r.values[key], r.values[hashKey]
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c.configDAO = alertCanceledWrite{alertContractConfigs: db, cancel: cancel}
	pool.GroupWait = "45s"
	if err := c.GenerateMainConfig(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal("persistence cancellation was hidden by a publication phase error")
	}
	if r.values[key] != before || (r.values[hashKey] != hash && !strings.HasPrefix(r.values[hashKey], "pending:")) {
		t.Fatal("canceled persistence published or acknowledged configuration")
	}
}

func copyAlertRows(rows map[int]*model.MonitorConfig) map[int]*model.MonitorConfig {
	out := map[int]*model.MonitorConfig{}
	for id, row := range rows {
		copy := *row
		out[id] = &copy
	}
	return out
}
func copyAlertRedis(values map[string]string) map[string]string {
	out := map[string]string{}
	for key, value := range values {
		out[key] = value
	}
	return out
}
func TestAlertMixedOwnershipRejectsWholeSnapshotBeforeWrites(t *testing.T) {
	for _, kind := range []int8{model.ConfigTypeAlertManager, model.ConfigTypeWebhookFile} {
		for _, owner := range []string{"custom", "foreign"} {
			for _, changed := range []bool{false, true} {
				t.Run(fmt.Sprintf("%d-%s-changed%v", kind, owner, changed), func(t *testing.T) {
					c, _, r, db, pool, _ := alertContractCache(t)
					ctx := context.Background()
					if err := c.GenerateMainConfig(ctx); err != nil {
						t.Fatal(err)
					}
					key := "127.0.0.1"
					if kind == model.ConfigTypeWebhookFile {
						key += "_3"
					}
					db.next++
					id := db.next
					row := &model.MonitorConfig{Model: model.Model{ID: id}, PoolID: pool.ID, ConfigType: kind, Name: "custom-mixed-preserved", InstanceIP: key, ConfigContent: "custom mixed body", ConfigHash: "custom mixed hash"}
					if owner == "foreign" {
						row.PoolID = 99
					}
					db.rows[id] = row
					beforeRows := copyAlertRows(db.rows)
					beforeRedis := copyAlertRedis(r.values)
					dbWrites, redisWrites := db.writes, r.writes
					if changed {
						pool.GroupWait = "45s"
						c.alertWebhookAddr = "http://different-snapshot.invalid"
					}
					if c.GenerateMainConfig(ctx) == nil {
						t.Fatal("mixed ownership was acknowledged")
					}
					if db.writes != dbWrites || r.writes != redisWrites || !reflect.DeepEqual(beforeRows, db.rows) || !reflect.DeepEqual(beforeRedis, r.values) {
						t.Fatal("mixed ownership changed metadata, Redis artifacts or ACK")
					}
				})
			}
		}
	}
}
func TestAlertMixedRetiredOwnershipCannotDeleteAmbiguousKeys(t *testing.T) {
	for _, kind := range []int8{model.ConfigTypeAlertManager, model.ConfigTypeWebhookFile} {
		for _, owner := range []string{"custom", "foreign"} {
			t.Run(fmt.Sprintf("%d-%s", kind, owner), func(t *testing.T) {
				c, g, r, db, pool, _ := alertContractCache(t)
				ctx := context.Background()
				if err := c.GenerateMainConfig(ctx); err != nil {
					t.Fatal(err)
				}
				key := "127.0.0.2"
				if kind == model.ConfigTypeWebhookFile {
					key += "_3"
				}
				db.next++
				id := db.next
				row := &model.MonitorConfig{Model: model.Model{ID: id}, PoolID: pool.ID, ConfigType: kind, Name: "custom-retired-preserved", InstanceIP: key, ConfigContent: "custom retired body"}
				if owner == "foreign" {
					row.PoolID = 99
				}
				db.rows[id] = row
				beforeRows := copyAlertRows(db.rows)
				beforeRedis := copyAlertRedis(r.values)
				dbWrites, redisWrites := db.writes, r.writes
				pool.AlertManagerInstances = model.StringList{"127.0.0.1"}
				g.rows = nil
				if c.GenerateMainConfig(ctx) == nil {
					t.Fatal("ambiguous retired ownership was acknowledged")
				}
				if db.writes != dbWrites || r.writes != redisWrites || !reflect.DeepEqual(beforeRows, db.rows) || !reflect.DeepEqual(beforeRedis, r.values) {
					t.Fatal("cleanup changed an ambiguously owned key or metadata")
				}
			})
		}
	}
}
func TestAlertFailedSnapshotThenSourceRevertsRestoresAcknowledgedContents(t *testing.T) {
	for _, stage := range []string{"hash read", "members read", "main cache", "webhook cache", "members write", "main delete", "webhook delete", "members delete", "hash write", "DB read", "main DB", "webhook DB", "webhook list", "webhook DB delete", "pending write"} {
		for _, target := range []string{"A", "B"} {
			for _, rebuild := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s-return%s-rebuild%v", stage, target, rebuild), func(t *testing.T) {
					c, g, r, db, pool, _ := alertContractCache(t)
					ctx := context.Background()
					if err := c.GenerateMainConfig(ctx); err != nil {
						t.Fatal(err)
					}
					mainKey := buildRedisKeyAlertManagerMain("127.0.0.1")
					hookKey := buildRedisKeyWebhookFile("127.0.0.1", 3)
					hashKey := alertPublicationStateKey(pool.ID)
					beforeMain, beforeHook, beforeHash, beforeAddress := r.values[mainKey], r.values[hookKey], r.values[hashKey], c.alertWebhookAddr
					pool.GroupWait = "45s"
					pool.AlertManagerInstances = model.StringList{"127.0.0.1"}
					c.alertWebhookAddr = "http://failed-b-snapshot.invalid"
					r.failure, db.failure = stage, stage
					if c.GenerateMainConfig(ctx) == nil {
						t.Fatal("B publication failure reported success")
					}
					r.failure, db.failure = "", ""
					if target == "A" {
						pool.GroupWait = "30s"
						pool.AlertManagerInstances = model.StringList{"127.0.0.1", "127.0.0.2"}
						c.alertWebhookAddr = beforeAddress
					}
					if rebuild {
						next, err := NewAlertManagerConfigCache(&config.Config{Server: config.ServerConfig{Port: c.serverPort}, Prometheus: config.PrometheusConfig{AlertWebhookAddr: c.alertWebhookAddr, AlertWebhookFileDir: c.webhookFileDir}}, c.logger, alertContractPools{pool: pool}, g, db, NewBatchConfigManager(db, c.logger), r)
						if err != nil {
							t.Fatal(err)
						}
						c = next.(*alertManagerConfigCache)
					}
					if err := c.GenerateMainConfig(ctx); err != nil {
						t.Fatal(err)
					}
					if strings.HasPrefix(r.values[hashKey], "pending:") || len(r.values[hashKey]) != 64 {
						t.Fatal("successful recovery did not acknowledge the full snapshot")
					}
					if target == "A" {
						if r.values[mainKey] != beforeMain || r.values[hookKey] != beforeHook || r.values[hashKey] != beforeHash {
							t.Fatal("obsolete A ACK skipped restoring partial B artifacts")
						}
						if r.values[buildRedisKeyAlertManagerMain("127.0.0.2")] == "" || r.values[buildRedisKeyWebhookFile("127.0.0.2", 3)] != beforeHook {
							t.Fatal("A instance removed during B was not restored")
						}
					} else {
						parsed := parseAlertContract(t, r.values[mainKey])
						if parsed.Route.GroupWait.String() != "45s" || !strings.Contains(r.values[hookKey], "failed-b-snapshot") || r.values[hashKey] == beforeHash {
							t.Fatal("B recovery reused the previous A snapshot")
						}
						if r.values[buildRedisKeyAlertManagerMain("127.0.0.2")] != "" || r.values[buildRedisKeyWebhookFile("127.0.0.2", 3)] != "" {
							t.Fatal("B recovery retained retired A artifacts")
						}
					}
					for _, row := range db.rows {
						if row.ConfigType == model.ConfigTypeAlertManager && row.InstanceIP == "127.0.0.1" && row.ConfigContent != r.values[mainKey] {
							t.Fatal("DB main and Redis disagree after recovery")
						}
						if row.ConfigType == model.ConfigTypeWebhookFile && row.InstanceIP == "127.0.0.1_3" && row.ConfigContent != r.values[hookKey] {
							t.Fatal("DB webhook and Redis disagree after recovery")
						}
					}
				})
			}
		}
	}
}
func TestAlertPendingMarkerFailureAllowsNoArtifactWrite(t *testing.T) {
	c, _, r, db, pool, _ := alertContractCache(t)
	ctx := context.Background()
	if err := c.GenerateMainConfig(ctx); err != nil {
		t.Fatal(err)
	}
	beforeRows := copyAlertRows(db.rows)
	beforeRedis := copyAlertRedis(r.values)
	dbWrites, redisWrites := db.writes, r.writes
	pool.GroupWait = "45s"
	c.alertWebhookAddr = "http://pending-write-failure.invalid"
	r.failure = "pending write"
	if c.GenerateMainConfig(ctx) == nil {
		t.Fatal("failed pending marker reported success")
	}
	if db.writes != dbWrites || r.writes != redisWrites+1 || !reflect.DeepEqual(beforeRows, db.rows) || !reflect.DeepEqual(beforeRedis, r.values) {
		t.Fatal("artifact writes preceded a reliable pending marker")
	}
}
func TestAlertPublicWrapperAndMainPublishSerialize(t *testing.T) {
	c, _, r, _, pool, _ := alertContractCache(t)
	var wg sync.WaitGroup
	errs := make(chan error, 8)
	for i := 0; i < 4; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			if err := c.GenerateMainConfig(context.Background()); err != nil {
				errs <- err
			}
		}()
		go func() {
			defer wg.Done()
			routes, _ := c.GenerateRouteConfigForPool(context.Background(), pool)
			if len(routes) != 1 {
				errs <- errors.New("wrapper route missing")
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
	parseAlertContract(t, r.values[buildRedisKeyAlertManagerMain("127.0.0.1")])
}

func TestAlertStablePublicationKeyDoesNotTrustOrTouchLegacyNames(t *testing.T) {
	for _, name := range []string{"legacy-pool", "pool:7"} {
		t.Run(name, func(t *testing.T) {
			c, _, r, _, pool, _ := alertContractCache(t)
			pool.Name = name
			ctx := context.Background()
			if err := c.GenerateMainConfig(ctx); err != nil {
				t.Fatal(err)
			}
			stable := alertPublicationStateKey(pool.ID)
			legacy := buildRedisHashKeyAlertManager(pool.Name)
			if stable == legacy {
				t.Fatal("stable ID namespace overlaps a possible legacy name")
			}
			acknowledged := r.values[stable]
			if len(acknowledged) != 64 {
				t.Fatal("missing stable baseline")
			}
			r.values[legacy] = acknowledged
			delete(r.values, stable)
			writes := r.writes
			if err := c.GenerateMainConfig(ctx); err != nil {
				t.Fatal(err)
			}
			if r.values[stable] != acknowledged || r.writes == writes {
				t.Fatal("missing stable ACK trusted a legacy name key instead of refreshing")
			}
			if r.values[legacy] != acknowledged {
				t.Fatal("migration deleted or mirrored the legacy key")
			}
		})
	}
}
func TestAlertRenamePendingStateSurvivesReversionAndRebuild(t *testing.T) {
	for _, stage := range []string{"main cache", "webhook cache", "members write", "main delete", "webhook delete", "members delete", "hash write", "main DB", "webhook DB", "webhook DB delete", "pending write"} {
		for _, target := range []string{"A", "B"} {
			for _, rebuild := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s-return%s-rebuild%v", stage, target, rebuild), func(t *testing.T) {
					c, g, r, db, pool, _ := alertContractCache(t)
					ctx := context.Background()
					pool.Name = "name-a"
					if err := c.GenerateMainConfig(ctx); err != nil {
						t.Fatal(err)
					}
					stable := alertPublicationStateKey(pool.ID)
					mainKey := buildRedisKeyAlertManagerMain("127.0.0.1")
					beforeMain, beforeHash := r.values[mainKey], r.values[stable]
					oldA, oldB := buildRedisHashKeyAlertManager("name-a"), buildRedisHashKeyAlertManager("name-b")
					r.values[oldA] = "legacy-a-untrusted"
					r.values[oldB] = "legacy-b-untrusted"
					pool.Name = "name-b"
					pool.AlertManagerInstances = model.StringList{"127.0.0.1"}
					r.failure, db.failure = stage, stage
					if c.GenerateMainConfig(ctx) == nil {
						t.Fatal("renamed B publication failure reported success")
					}
					r.failure, db.failure = "", ""
					if target == "A" {
						pool.Name = "name-a"
						pool.AlertManagerInstances = model.StringList{"127.0.0.1", "127.0.0.2"}
					}
					if rebuild {
						next, err := NewAlertManagerConfigCache(&config.Config{Server: config.ServerConfig{Port: c.serverPort}, Prometheus: config.PrometheusConfig{AlertWebhookAddr: c.alertWebhookAddr, AlertWebhookFileDir: c.webhookFileDir}}, c.logger, alertContractPools{pool: pool}, g, db, NewBatchConfigManager(db, c.logger), r)
						if err != nil {
							t.Fatal(err)
						}
						c = next.(*alertManagerConfigCache)
					}
					if err := c.GenerateMainConfig(ctx); err != nil {
						t.Fatal(err)
					}
					if len(r.values[stable]) != 64 || strings.HasPrefix(r.values[stable], "pending:") {
						t.Fatal("renamed recovery did not commit stable ID state")
					}
					parsed := parseAlertContract(t, r.values[mainKey])
					wantedFile := fmt.Sprintf("/typed/hooks/webhook_%s_3.yml", pool.Name)
					if parsed.Receivers[0].WebhookConfigs[0].URLFile != wantedFile {
						t.Fatal("renamed recovery served another name's configuration")
					}
					if target == "A" && (r.values[mainKey] != beforeMain || r.values[stable] != beforeHash) {
						t.Fatal("old name ACK bypassed restoration after failed rename")
					}
					if target == "B" && (r.values[mainKey] == beforeMain || r.values[stable] == beforeHash) {
						t.Fatal("renamed B reused A configuration")
					}
					for _, row := range db.rows {
						if row.ConfigType == model.ConfigTypeWebhookFile && row.InstanceIP == "127.0.0.1_3" && row.Name != fmt.Sprintf("webhook_%s_3_127.0.0.1", pool.Name) {
							t.Fatal("recovered main and webhook metadata names differ")
						}
					}
					if r.values[oldA] != "legacy-a-untrusted" || r.values[oldB] != "legacy-b-untrusted" {
						t.Fatal("publisher touched legacy name ACKs")
					}
				})
			}
		}
	}
}
