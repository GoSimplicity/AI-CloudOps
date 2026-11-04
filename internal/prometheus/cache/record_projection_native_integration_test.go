//go:build integration

package cache

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"reflect"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	alertdao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/alert"
	configdao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/config"
	scrapedao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/scrape"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// Only the source inventory is fixed. The actual generator, metadata writes,
// journal, SQL lease, typed Store and Redis Lua are production implementations.
type rootRecordNativePool struct {
	scrapedao.ScrapePoolDAO
	pool model.MonitorScrapePool
}

func (s rootRecordNativePool) GetMonitorScrapePoolList(_ context.Context, req *model.GetMonitorScrapePoolListReq) ([]*model.MonitorScrapePool, int64, error) {
	if req.Page != 1 {
		return []*model.MonitorScrapePool{}, 1, nil
	}
	row := s.pool
	row.PrometheusInstances = append(model.StringList(nil), s.pool.PrometheusInstances...)
	return []*model.MonitorScrapePool{&row}, 1, nil
}

type rootRecordNativeRules struct {
	alertdao.AlertManagerRecordDAO
	pool      int
	expr      string
	forbidden bool
}

func (s rootRecordNativeRules) GetMonitorRecordRuleByPoolId(context.Context, int) ([]*model.MonitorRecordRule, int64, error) {
	if s.forbidden {
		return nil, 0, errors.New("off source must not read rules")
	}
	return []*model.MonitorRecordRule{{Model: model.Model{ID: 1}, PoolID: s.pool, Name: "native_total", Enable: 1, Expr: s.expr}}, 1, nil
}

type rootRecordRESP struct {
	raw     []byte
	payload []byte
	args    [][]byte
}

func rootReadRecordRESP(reader *bufio.Reader, depth int) (rootRecordRESP, error) {
	if depth > 16 {
		return rootRecordRESP{}, errors.New("fixture RESP depth exceeded")
	}
	line, err := reader.ReadBytes('\n')
	if err != nil {
		return rootRecordRESP{}, err
	}
	if len(line) < 3 || len(line) > 1024 || !bytes.HasSuffix(line, []byte("\r\n")) {
		return rootRecordRESP{}, errors.New("invalid fixture RESP header")
	}
	frame := rootRecordRESP{raw: append([]byte(nil), line...), payload: append([]byte(nil), line[1:len(line)-2]...)}
	switch line[0] {
	case '+', '-', ':':
		return frame, nil
	case '$', '*':
		count, err := strconv.Atoi(string(frame.payload))
		if err != nil || count < -1 || count > 16*1024*1024 {
			return rootRecordRESP{}, errors.New("invalid fixture RESP size")
		}
		if count == -1 {
			return frame, nil
		}
		if line[0] == '$' {
			data := make([]byte, count+2)
			if _, err := io.ReadFull(reader, data); err != nil {
				return rootRecordRESP{}, err
			}
			if !bytes.HasSuffix(data, []byte("\r\n")) {
				return rootRecordRESP{}, errors.New("invalid fixture RESP bulk")
			}
			frame.raw = append(frame.raw, data...)
			frame.payload = data[:count]
			return frame, nil
		}
		if count > 4096 {
			return rootRecordRESP{}, errors.New("fixture RESP array too large")
		}
		for i := 0; i < count; i++ {
			child, err := rootReadRecordRESP(reader, depth+1)
			if err != nil {
				return rootRecordRESP{}, err
			}
			frame.raw = append(frame.raw, child.raw...)
			frame.args = append(frame.args, child.payload)
		}
		return frame, nil
	default:
		return rootRecordRESP{}, errors.New("unsupported fixture RESP response")
	}
}
func rootIsRecordArtifactRequest(frame rootRecordRESP) bool {
	if len(frame.args) < 5 || !bytes.EqualFold(frame.args[0], []byte("eval")) {
		return false
	}
	keys, err := strconv.Atoi(string(frame.args[2]))
	if err != nil || keys < 0 || keys+5 != len(frame.args) {
		return false
	}
	var wire struct {
		Mode string `json:"mode"`
	}
	return json.Unmarshal(frame.args[keys+3], &wire) == nil && wire.Mode == "artifact"
}

type rootRecordQueueReply struct {
	response string
	err      error
}
type rootRecordQueue struct {
	listener    net.Listener
	backend     string
	seen        chan struct{}
	release     chan struct{}
	result      chan rootRecordQueueReply
	stop        chan struct{}
	held        atomic.Bool
	releaseOnce sync.Once
	stopOnce    sync.Once
	mu          sync.Mutex
	conns       map[net.Conn]bool
	wg          sync.WaitGroup
}

func rootNewRecordQueue(t *testing.T, backend string) *rootRecordQueue {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal("cannot start local delayed-packet fixture")
	}
	q := &rootRecordQueue{listener: listener, backend: backend, seen: make(chan struct{}), release: make(chan struct{}), result: make(chan rootRecordQueueReply, 1), stop: make(chan struct{}), conns: map[net.Conn]bool{}}
	q.wg.Add(1)
	go q.accept()
	t.Cleanup(q.close)
	return q
}
func (q *rootRecordQueue) add(conn net.Conn) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	select {
	case <-q.stop:
		return false
	default:
		q.conns[conn] = true
		return true
	}
}
func (q *rootRecordQueue) remove(conn net.Conn) {
	q.mu.Lock()
	delete(q.conns, conn)
	q.mu.Unlock()
	conn.Close()
}
func (q *rootRecordQueue) accept() {
	defer q.wg.Done()
	for {
		client, err := q.listener.Accept()
		if err != nil {
			return
		}
		if !q.add(client) {
			client.Close()
			return
		}
		q.wg.Add(1)
		go q.relay(client)
	}
}
func rootWriteRecordFrame(conn net.Conn, raw []byte) error {
	for len(raw) > 0 {
		n, err := conn.Write(raw)
		if err != nil {
			return err
		}
		if n == 0 {
			return io.ErrUnexpectedEOF
		}
		raw = raw[n:]
	}
	return nil
}
func (q *rootRecordQueue) relay(client net.Conn) {
	defer q.wg.Done()
	defer q.remove(client)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	backend, err := new(net.Dialer).DialContext(ctx, "tcp", q.backend)
	if err != nil {
		return
	}
	if !q.add(backend) {
		backend.Close()
		return
	}
	defer q.remove(backend)
	input, output := bufio.NewReader(client), bufio.NewReader(backend)
	for {
		request, err := rootReadRecordRESP(input, 0)
		if err != nil {
			return
		}
		if rootIsRecordArtifactRequest(request) && q.held.CompareAndSwap(false, true) {
			// The caller sees an unknown result and releases its SQL lease. The
			// original Redis request remains in this real TCP intermediary's queue.
			client.Close()
			close(q.seen)
			select {
			case <-q.release:
			case <-q.stop:
				return
			}
			backend.SetDeadline(time.Now().Add(10 * time.Second))
			err = rootWriteRecordFrame(backend, request.raw)
			var response rootRecordRESP
			if err == nil {
				response, err = rootReadRecordRESP(output, 0)
			}
			q.result <- rootRecordQueueReply{response: string(response.payload), err: err}
			return
		}
		backend.SetDeadline(time.Now().Add(10 * time.Second))
		if err := rootWriteRecordFrame(backend, request.raw); err != nil {
			return
		}
		response, err := rootReadRecordRESP(output, 0)
		if err != nil {
			return
		}
		if err := rootWriteRecordFrame(client, response.raw); err != nil {
			return
		}
	}
}
func (q *rootRecordQueue) unblock() { q.releaseOnce.Do(func() { close(q.release) }) }
func (q *rootRecordQueue) close() {
	q.stopOnce.Do(func() {
		close(q.stop)
		q.listener.Close()
		q.mu.Lock()
		for conn := range q.conns {
			conn.Close()
		}
		q.mu.Unlock()
	})
	q.wg.Wait()
}

func TestRecordProjectionNativeLatePacketsCannotDamageNewOwner(t *testing.T) {
	client := projectionRedisNative(t)
	for index, action := range []string{model.ConfigProjectionRender, model.ConfigProjectionRetire} {
		t.Run(action, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			_, metadata, db := projectionStoreSQL(t)
			journal := configdao.NewRecordProjectionJournalDAO(zap.NewNop(), db)
			history := configdao.NewGeneratedConfigHistoryReader(zap.NewNop(), db)
			normal, err := NewProjectionStore(client)
			if err != nil {
				t.Fatal(err)
			}
			defer normal.Close()
			const firstPool = 9091
			const nextPool = 9092
			ip := "127.0.90." + strconv.Itoa(index+1)
			makeRecord := func(pool int, enabled int8, expr string, store *ProjectionStore) RecordRuleConfigCache {
				source := model.MonitorScrapePool{Model: model.Model{ID: pool}, Name: "native-pool-" + strconv.Itoa(pool), SupportRecord: enabled, PrometheusInstances: model.StringList{ip}}
				return NewRecordRuleConfigCache(zap.NewNop(), rootRecordNativePool{pool: source}, rootRecordNativeRules{pool: pool, expr: expr, forbidden: enabled == 2}, metadata, NewBatchConfigManager(metadata, zap.NewNop()), client, journal, history, NewRecordProjectionRunner(store))
			}
			track := func(receipt configdao.ProjectionReceipt) {
				t.Helper()
				err := journal.WithPoolProjectionLease(ctx, receipt, func(lease configdao.ProjectionLease) error {
					snapshot, err := lease.Snapshot()
					if err != nil {
						return err
					}
					handle, err := compileProjection(normal, lease, snapshot)
					if err != nil {
						return err
					}
					projectionTrack(t, client, handle)
					return nil
				})
				if err != nil {
					t.Fatal("cannot track actual publisher fixture", err)
				}
			}
			initial := makeRecord(firstPool, 1, "sum(up)", normal)
			if err := initial.GenerateMainConfig(ctx); err != nil {
				t.Fatal("initial real publisher failed", err)
			}
			first, err := journal.LoadLastCommittedProjection(ctx, 4, firstPool)
			if err != nil || first == nil {
				t.Fatal("initial durable plan missing", err)
			}
			track(*first)
			firstRaw := initial.GetConfigByIP(ip)
			if firstRaw == "" {
				t.Fatal("initial real publisher did not serve raw bytes")
			}
			queue := rootNewRecordQueue(t, client.Options().Addr)
			options := *client.Options()
			options.Addr = queue.listener.Addr().String()
			options.Protocol = 2
			options.PoolSize = 1
			options.MinIdleConns = 0
			base := redis.NewClient(&options)
			defer base.Close()
			delayedStore, err := NewProjectionStore(base)
			if err != nil {
				t.Fatal(err)
			}
			defer delayedStore.Close()
			enabled := int8(1)
			if action == model.ConfigProjectionRetire {
				enabled = 2
			}
			old := makeRecord(firstPool, enabled, "sum(up) * 2", delayedStore)
			oldDone := make(chan error, 1)
			go func() { oldDone <- old.GenerateMainConfig(ctx) }()
			select {
			case <-queue.seen:
			case <-ctx.Done():
				t.Fatal("actual publisher never sent the held Redis artifact request")
			}
			select {
			case err := <-oldDone:
				if err == nil {
					t.Fatal("unknown Redis result falsely completed old publisher")
				}
			case <-ctx.Done():
				t.Fatal("old SQL lease did not finish after client EOF")
			}
			pending, err := journal.LoadLastCommittedProjection(ctx, 4, firstPool)
			if err != nil || pending == nil || pending.Phase != model.ConfigPublicationCommitted || len(pending.Artifacts) != 1 || pending.Artifacts[0].Action != action {
				t.Fatal("real old publisher did not leave its committed pending plan", err)
			}
			if action == model.ConfigProjectionRender {
				if pending.Artifacts[0].ConfigRowID == nil {
					t.Fatal("pending metadata identity missing")
				}
				if err := metadata.DeleteMonitorConfig(ctx, *pending.Artifacts[0].ConfigRowID); err != nil {
					t.Fatal(err)
				}
			}
			newer := makeRecord(nextPool, 1, "sum(up) * 3", normal)
			if err := newer.GenerateMainConfig(ctx); err != nil {
				t.Fatal("new owner's real publication failed", err)
			}
			good, err := journal.LoadLastProjectedProjection(ctx, 4, nextPool)
			if err != nil || good == nil {
				t.Fatal("new owner's projected SQL receipt missing", err)
			}
			track(*good)
			allKeys := []string{buildRedisKeyRecordRule(ip), projectionNamespaceKey(4, ip)}
			for _, pool := range []int{firstPool, nextPool} {
				allKeys = append(allKeys, projectionPoolKey(4, pool, "fence"), projectionPoolKey(4, pool, "ack"), recordPublicationKey(pool), buildRedisSetKeyRecordRulePoolIPs(pool))
			}
			before := rootProjectionValues(t, client, allKeys)
			newRaw := newer.GetConfigByIP(ip)
			if newRaw == "" || newRaw == firstRaw {
				t.Fatal("two real publications did not produce distinct bytes")
			}
			queue.unblock()
			select {
			case reply := <-queue.result:
				if reply.err != nil || reply.response != "STALE" {
					t.Fatalf("actual delayed %s packet did not get Redis STALE: %v", action, reply.err)
				}
			case <-ctx.Done():
				t.Fatal("delayed Redis packet was not actually evaluated")
			}
			if !reflect.DeepEqual(rootProjectionValues(t, client, allKeys), before) || newer.GetConfigByIP(ip) != newRaw {
				t.Fatal("delayed old publisher changed new body, members, ACK or fence")
			}
			current, err := journal.LoadLastProjectedProjection(ctx, 4, nextPool)
			if err != nil || current == nil || current.Reservation != good.Reservation {
				t.Fatal("new owner's durable completion changed", err)
			}
			{
				off := old
				if action == model.ConfigProjectionRender {
					if err := old.GenerateMainConfig(ctx); err == nil {
						t.Fatal("active old source reclaimed a foreign address")
					}
					off = makeRecord(firstPool, 2, "must not be read", normal)
				}
				if err := off.GenerateMainConfig(ctx); err != nil {
					t.Fatal("off publisher could not finish bookkeeping after a newer owner completed", err)
				}
				newOwnerKeys := []string{buildRedisKeyRecordRule(ip), projectionNamespaceKey(4, ip), projectionPoolKey(4, nextPool, "fence"), projectionPoolKey(4, nextPool, "ack"), recordPublicationKey(nextPool), buildRedisSetKeyRecordRulePoolIPs(nextPool)}
				for _, key := range newOwnerKeys {
					if rootProjectionValues(t, client, []string{key})[key] != before[key] {
						t.Fatal("off publisher changed the newer owner's raw bytes or completion proof")
					}
				}
				finished, err := journal.LoadLastProjectedProjection(ctx, 4, firstPool)
				if err != nil || finished == nil || finished.Phase != model.ConfigPublicationProjected || (action == model.ConfigProjectionRetire && finished.SourceHash != pending.SourceHash) || finished.Reservation.Generation <= pending.Reservation.Generation || len(finished.Artifacts) != 0 {
					t.Fatal("off publisher did not issue a fresh durable empty recovery", err)
				}
				if client.SCard(ctx, buildRedisSetKeyRecordRulePoolIPs(firstPool)).Val() != 0 {
					t.Fatal("off publisher left its old member locator behind")
				}
				// A completion is historical evidence. A later legitimate B body
				// revision must not make A forget its already resolved retirement.
				updatedOwner := makeRecord(nextPool, 1, "sum(up) * 4", normal)
				if err := updatedOwner.GenerateMainConfig(ctx); err != nil {
					t.Fatal("new owner could not update after superseding A", err)
				}
				stable := rootProjectionValues(t, client, newOwnerKeys)
				for retry := 0; retry < 2; retry++ {
					rebuilt := makeRecord(firstPool, 2, "must not be read", normal)
					if err := rebuilt.GenerateMainConfig(ctx); err != nil {
						t.Fatal("rebuilt off publisher forgot durable retirement resolution", err)
					}
					if !reflect.DeepEqual(rootProjectionValues(t, client, newOwnerKeys), stable) {
						t.Fatal("rebuilt off publisher changed the current owner's projection")
					}
				}
				newRaw = updatedOwner.GetConfigByIP(ip)
			}
			rows, err := metadata.GetMonitorConfigsByInstances(ctx, []string{ip}, 4)
			if err != nil || len(rows) != 1 || rows[0].PoolID != nextPool || rows[0].ConfigContent != newRaw {
				t.Fatal("delayed command changed the current metadata owner", err)
			}
		})
	}
}

func TestRecordProjectionNativeLegacyOwnerCanFinishAfterReplacement(t *testing.T) {
	client := projectionRedisNative(t)
	_, metadata, db := projectionStoreSQL(t)
	journal := configdao.NewRecordProjectionJournalDAO(zap.NewNop(), db)
	history := configdao.NewGeneratedConfigHistoryReader(zap.NewNop(), db)
	store, err := NewProjectionStore(client)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	const oldPool = 9341
	const newPool = 9342
	const ip = "127.0.93.4"
	legacy := &model.MonitorConfig{PoolID: oldPool, ConfigType: 4, InstanceIP: ip, Name: "prometheus_record_rule_9341_127.0.93.4.yaml", ConfigContent: "groups: [] # legacy before pool journal\n", Status: model.ConfigStatusActive}
	legacy.ConfigHash = calculateConfigHash(legacy.ConfigContent)
	if err := metadata.CreateMonitorConfig(ctx, legacy); err != nil {
		t.Fatal(err)
	}
	if err := metadata.DeleteMonitorConfig(ctx, legacy.ID); err != nil {
		t.Fatal(err)
	}
	if previous, err := journal.LoadLastCommittedProjection(ctx, 4, oldPool); err != nil || previous != nil {
		t.Fatal("legacy owner unexpectedly has an issued pool plan", err)
	}
	if client.Set(ctx, buildRedisKeyRecordRule(ip), legacy.ConfigContent, 0).Err() != nil || client.SAdd(ctx, buildRedisSetKeyRecordRulePoolIPs(oldPool), ip).Err() != nil {
		t.Fatal("could not seed owned legacy raw/member fixture")
	}
	t.Cleanup(func() {
		client.Del(context.Background(), buildRedisKeyRecordRule(ip), buildRedisSetKeyRecordRulePoolIPs(oldPool))
	})
	create := func(pool int, enabled int8) RecordRuleConfigCache {
		source := model.MonitorScrapePool{Model: model.Model{ID: pool}, Name: "native-legacy-" + strconv.Itoa(pool), SupportRecord: enabled, PrometheusInstances: model.StringList{ip}}
		return NewRecordRuleConfigCache(zap.NewNop(), rootRecordNativePool{pool: source}, rootRecordNativeRules{pool: pool, expr: "sum(up)", forbidden: enabled == 2}, metadata, NewBatchConfigManager(metadata, zap.NewNop()), client, journal, history, NewRecordProjectionRunner(store))
	}
	track := func(receipt configdao.ProjectionReceipt) {
		t.Helper()
		if err := journal.WithPoolProjectionLease(ctx, receipt, func(lease configdao.ProjectionLease) error {
			snapshot, err := lease.Snapshot()
			if err != nil {
				return err
			}
			handle, err := compileProjection(store, lease, snapshot)
			if err != nil {
				return err
			}
			projectionTrack(t, client, handle)
			return nil
		}); err != nil {
			t.Fatal(err)
		}
	}
	newer := create(newPool, 1)
	if err := newer.GenerateMainConfig(ctx); err != nil {
		t.Fatal("new owner failed to establish a complete replacement", err)
	}
	complete, err := journal.LoadLastProjectedProjection(ctx, 4, newPool)
	if err != nil || complete == nil {
		t.Fatal("replacement completion missing", err)
	}
	track(*complete)
	protected := []string{buildRedisKeyRecordRule(ip), projectionNamespaceKey(4, ip), projectionPoolKey(4, newPool, "fence"), projectionPoolKey(4, newPool, "ack"), recordPublicationKey(newPool), buildRedisSetKeyRecordRulePoolIPs(newPool)}
	before := rootProjectionValues(t, client, protected)
	for retry := 0; retry < 2; retry++ {
		if err := create(oldPool, 2).GenerateMainConfig(ctx); err != nil {
			t.Fatal("legacy off owner could not finish despite a verified complete replacement", err)
		}
		recovered, err := journal.LoadLastProjectedProjection(ctx, 4, oldPool)
		if err != nil || recovered == nil || len(recovered.Artifacts) != 0 || recovered.Phase != model.ConfigPublicationProjected {
			t.Fatal("legacy owner did not durably confirm its empty state", err)
		}
		track(*recovered)
		if client.SCard(ctx, buildRedisSetKeyRecordRulePoolIPs(oldPool)).Val() != 0 || !reflect.DeepEqual(rootProjectionValues(t, client, protected), before) {
			t.Fatal("legacy bookkeeping touched the replacement or retained obsolete members")
		}
	}
	// B is the current owner. A's already-deleted historical row must not
	// permanently prevent B from retiring its own configuration later.
	if err := create(newPool, 2).GenerateMainConfig(ctx); err != nil {
		t.Fatal("current owner could not retire because a foreign historical tombstone exists", err)
	}
	if client.Exists(ctx, buildRedisKeyRecordRule(ip)).Val() != 0 || client.SCard(ctx, buildRedisSetKeyRecordRulePoolIPs(newPool)).Val() != 0 {
		t.Fatal("current owner did not clear its own raw and member set")
	}
	if err := create(oldPool, 2).GenerateMainConfig(ctx); err != nil {
		t.Fatal("legacy resolved owner forgot its disposition after B retired", err)
	}
}

type rootRecordPoolSnapshotHook struct {
	scrapedao.ScrapePoolDAO
	afterSnapshot func()
	once          sync.Once
}

func (d *rootRecordPoolSnapshotHook) GetMonitorScrapePoolList(ctx context.Context, req *model.GetMonitorScrapePoolListReq) ([]*model.MonitorScrapePool, int64, error) {
	rows, total, err := d.ScrapePoolDAO.GetMonitorScrapePoolList(ctx, req)
	if err == nil {
		d.once.Do(d.afterSnapshot)
	}
	return rows, total, err
}

type rootRecordRuleSnapshotHook struct {
	alertdao.AlertManagerRecordDAO
	afterSnapshot func()
	once          sync.Once
}

func (d *rootRecordRuleSnapshotHook) GetMonitorRecordRuleByPoolId(ctx context.Context, pool int) ([]*model.MonitorRecordRule, int64, error) {
	rows, total, err := d.AlertManagerRecordDAO.GetMonitorRecordRuleByPoolId(ctx, pool)
	if err == nil {
		d.once.Do(d.afterSnapshot)
	}
	return rows, total, err
}

func TestRecordProjectionNativeOlderSourceCannotReserveAfterNewerPublication(t *testing.T) {
	client := projectionRedisNative(t)
	for index, scope := range []string{"pool snapshot", "rule snapshot"} {
		t.Run(scope, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			_, metadata, db := projectionStoreSQL(t)
			if err := db.AutoMigrate(&model.MonitorScrapePool{}, &model.MonitorRecordRule{}); err != nil {
				t.Fatal(err)
			}
			logger := zap.NewNop()
			pools := scrapedao.NewScrapePoolDAO(db, logger, nil)
			rules := alertdao.NewAlertManagerRecordDAO(db, logger, nil)
			pool := &model.MonitorScrapePool{Model: model.Model{ID: 9461 + index}, Name: "before-source-change", SupportRecord: 1, SupportAlert: 2, PrometheusInstances: model.StringList{"127.0.94." + strconv.Itoa(index+1)}}
			if err := pools.CreateMonitorScrapePool(ctx, pool); err != nil {
				t.Fatal(err)
			}
			rule := &model.MonitorRecordRule{PoolID: pool.ID, Name: "native_source_total", Expr: "sum(up)", Enable: 1}
			if err := rules.CreateMonitorRecordRule(ctx, rule); err != nil {
				t.Fatal(err)
			}
			journal := configdao.NewRecordProjectionJournalDAO(logger, db)
			history := configdao.NewGeneratedConfigHistoryReader(logger, db)
			store, err := NewProjectionStore(client)
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			create := func(sourcePools scrapedao.ScrapePoolDAO, sourceRules alertdao.AlertManagerRecordDAO) RecordRuleConfigCache {
				return NewRecordRuleConfigCache(logger, sourcePools, sourceRules, metadata, NewBatchConfigManager(metadata, logger), client, journal, history, NewRecordProjectionRunner(store))
			}
			newer := create(pools, rules)
			var completed *configdao.ProjectionReceipt
			var protected []string
			var baseline map[string]rootProjectionValue
			called := false
			publishNewSource := func() {
				called = true
				if scope == "pool snapshot" {
					req := &model.UpdateMonitorScrapePoolReq{ID: pool.ID, Name: "after-source-change", SupportRecord: 1, SupportAlert: 2, PrometheusInstances: append(model.StringList(nil), pool.PrometheusInstances...), ScrapeInterval: 30, ScrapeTimeout: 10, RemoteTimeoutSeconds: 5}
					if err := pools.UpdateMonitorScrapePool(ctx, req); err != nil {
						t.Fatal("real source pool update failed", err)
					}
				} else {
					changed := *rule
					changed.Expr = "sum(up) * 3"
					if err := rules.UpdateMonitorRecordRule(ctx, &changed); err != nil {
						t.Fatal("real source rule update failed", err)
					}
				}
				if err := newer.GenerateMainConfig(ctx); err != nil {
					t.Fatal("newer source publication failed", err)
				}
				var err error
				completed, err = journal.LoadLastProjectedProjection(ctx, 4, pool.ID)
				if err != nil || completed == nil {
					t.Fatal("new source was not fully acknowledged", err)
				}
				if err := journal.WithPoolProjectionLease(ctx, *completed, func(lease configdao.ProjectionLease) error {
					snapshot, err := lease.Snapshot()
					if err != nil {
						return err
					}
					handle, err := compileProjection(store, lease, snapshot)
					if err != nil {
						return err
					}
					projectionTrack(t, client, handle)
					protected = append([]string(nil), handle.keys...)
					baseline = rootProjectionValues(t, client, protected)
					return nil
				}); err != nil {
					t.Fatal(err)
				}
			}
			var older RecordRuleConfigCache
			if scope == "pool snapshot" {
				older = create(&rootRecordPoolSnapshotHook{ScrapePoolDAO: pools, afterSnapshot: publishNewSource}, rules)
			} else {
				older = create(pools, &rootRecordRuleSnapshotHook{AlertManagerRecordDAO: rules, afterSnapshot: publishNewSource})
			}
			err = older.GenerateMainConfig(ctx)
			if !called {
				t.Fatal("real source read barrier was not reached")
			}
			// A stale task may fail or coalesce into a safe no-op. It must never
			// replace the accepted source merely by reserving after a slow read.
			current, loadErr := journal.LoadLastProjectedProjection(ctx, 4, pool.ID)
			if loadErr != nil || current == nil {
				t.Fatal("cannot inspect publication after the delayed source read", loadErr)
			}
			if current.Reservation != completed.Reservation || current.SourceHash != completed.SourceHash {
				t.Fatalf("older source replaced generation %d with generation %d after newer completion (call failed=%v)", completed.Reservation.Generation, current.Reservation.Generation, err != nil)
			}
			accepted, loadErr := journal.LoadLastCommittedProjection(ctx, 4, pool.ID)
			if loadErr != nil || accepted == nil || accepted.Reservation != completed.Reservation {
				t.Fatal("older source changed the accepted metadata plan", loadErr)
			}
			if !reflect.DeepEqual(rootProjectionValues(t, client, protected), baseline) {
				t.Fatal("older source changed new raw, membership, SHA, ACK or fence")
			}
		})
	}
}

func TestRecordProjectionNativeSameHashFindsNewDatabaseOnlyRetirement(t *testing.T) {
	client := projectionRedisNative(t)
	_, metadata, db := projectionStoreSQL(t)
	journal := configdao.NewRecordProjectionJournalDAO(zap.NewNop(), db)
	history := configdao.NewGeneratedConfigHistoryReader(zap.NewNop(), db)
	store, err := NewProjectionStore(client)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	const poolID = 9551
	const desiredIP = "127.0.95.1"
	const obsoleteIP = "127.0.95.2"
	source := model.MonitorScrapePool{Model: model.Model{ID: poolID}, Name: "native-same-source", SupportRecord: 1, PrometheusInstances: model.StringList{desiredIP}}
	publisher := NewRecordRuleConfigCache(zap.NewNop(), rootRecordNativePool{pool: source}, rootRecordNativeRules{pool: poolID, expr: "sum(up)"}, metadata, NewBatchConfigManager(metadata, zap.NewNop()), client, journal, history, NewRecordProjectionRunner(store))
	if err := publisher.GenerateMainConfig(ctx); err != nil {
		t.Fatal(err)
	}
	first, err := journal.LoadLastProjectedProjection(ctx, 4, poolID)
	if err != nil || first == nil {
		t.Fatal(err)
	}
	if err := journal.WithPoolProjectionLease(ctx, *first, func(lease configdao.ProjectionLease) error {
		snapshot, err := lease.Snapshot()
		if err != nil {
			return err
		}
		handle, err := compileProjection(store, lease, snapshot)
		if err != nil {
			return err
		}
		projectionTrack(t, client, handle)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	original := publisher.GetConfigByIP(desiredIP)
	if original == "" {
		t.Fatal("initial desired raw content missing")
	}
	obsolete := &model.MonitorConfig{PoolID: poolID, ConfigType: 4, InstanceIP: obsoleteIP, Name: "prometheus_record_rule_9551_127.0.95.2.yaml", ConfigContent: "groups: [] # stale database-only owner\n", Status: model.ConfigStatusActive}
	obsolete.ConfigHash = calculateConfigHash(obsolete.ConfigContent)
	if err := metadata.CreateMonitorConfig(ctx, obsolete); err != nil {
		t.Fatal(err)
	}
	if client.Set(ctx, buildRedisKeyRecordRule(obsoleteIP), obsolete.ConfigContent, 0).Err() != nil {
		t.Fatal("could not seed unindexed stale raw")
	}
	t.Cleanup(func() {
		client.Del(context.Background(), buildRedisKeyRecordRule(obsoleteIP), projectionNamespaceKey(4, obsoleteIP))
	})
	members, memberErr := client.SMembers(ctx, buildRedisSetKeyRecordRulePoolIPs(poolID)).Result()
	if memberErr != nil {
		t.Fatal("cannot inspect the native member precondition", memberErr)
	}
	for _, ip := range members {
		if ip == obsoleteIP {
			t.Fatal("stale row must not be a pre-existing indexed target")
		}
	}
	if err := publisher.GenerateMainConfig(ctx); err != nil {
		t.Fatal("same-source reconciliation failed", err)
	}
	row, err := metadata.GetGeneratedMonitorConfigByID(ctx, obsolete.ID)
	if err != nil || row == nil || row.DeletedAt == 0 {
		t.Fatal("healthy old receipt hid a newly observed database-only stale owner", err)
	}
	if client.Exists(ctx, buildRedisKeyRecordRule(obsoleteIP)).Val() != 0 || publisher.GetConfigByIP(desiredIP) != original {
		t.Fatal("same-source retirement did not preserve desired and clear obsolete raw")
	}
	current, err := journal.LoadLastProjectedProjection(ctx, 4, poolID)
	if err != nil || current == nil || current.SourceHash != first.SourceHash || current.Reservation.Generation <= first.Reservation.Generation {
		t.Fatal("same-source metadata change did not get a fresh durable plan", err)
	}
}
