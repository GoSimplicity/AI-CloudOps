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
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	alertdao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/alert"
	scrapedao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/scrape"
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
	return rootIsProjectionModeRequest(frame, "artifact") || rootIsProjectionModeRequest(frame, "artifact_batch")
}

func rootIsProjectionModeRequest(frame rootRecordRESP, mode string) bool {
	if len(frame.args) < 5 || !bytes.EqualFold(frame.args[0], []byte("eval")) {
		return false
	}
	keys, err := strconv.Atoi(string(frame.args[2]))
	if err != nil || keys < 0 || keys+5 > len(frame.args) {
		return false
	}
	wireKeys := make([]string, keys)
	for i := range wireKeys {
		wireKeys[i] = string(frame.args[i+3])
	}
	wire, err := decodeProjectionTestWire(string(frame.args[keys+3]), wireKeys)
	if err != nil || wire.Mode != mode {
		return false
	}
	if mode != "artifact_batch" {
		return keys+5 == len(frame.args)
	}
	var selected []string
	return json.Unmarshal(frame.args[keys+4], &selected) == nil && len(selected) > 1 && len(frame.args) == keys+5+len(selected)
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
	match       func(rootRecordRESP) bool
}

func rootNewRecordQueue(t *testing.T, backend string) *rootRecordQueue {
	t.Helper()
	return rootNewRecordQueueMatching(t, backend, rootIsRecordArtifactRequest)
}

func rootNewRecordQueueMatching(t *testing.T, backend string, match func(rootRecordRESP) bool) *rootRecordQueue {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal("cannot start local delayed-packet fixture")
	}
	q := &rootRecordQueue{listener: listener, backend: backend, seen: make(chan struct{}), release: make(chan struct{}), result: make(chan rootRecordQueueReply, 1), stop: make(chan struct{}), conns: map[net.Conn]bool{}}
	q.match = match
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
		if q.match(request) && q.held.CompareAndSwap(false, true) {
			// held is stored before this EOF. seen is closed only after it, so a
			// waiter can observe the publication error before seen. The original
			// Redis request stays queued until unblock.
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
	rootScrapeRuleNativeLatePackets(t, 4)
}

func TestRecordProjectionNativeLegacyOwnerCanFinishAfterReplacement(t *testing.T) {
	rootScrapeRuleNativeLegacyOwner(t, 4)
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
	rootScrapeRuleNativeOlderSource(t, 4)
}

func TestRecordProjectionNativeSameHashFindsNewDatabaseOnlyRetirement(t *testing.T) {
	rootScrapeRuleNativeSameHash(t, 4)
}
