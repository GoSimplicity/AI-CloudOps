package scrape_test

import (
	"context"
	"fmt"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	dao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/scrape"
	service "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/service/scrape"
	"go.uber.org/zap"
)

type poolSupportStore struct {
	dao.ScrapePoolDAO
	reads, writes int
	created       *model.MonitorScrapePool
	updated       *model.UpdateMonitorScrapePoolReq
}

func (s *poolSupportStore) CheckMonitorScrapePoolExists(context.Context, *model.MonitorScrapePool) (bool, error) {
	s.reads++
	return false, nil
}
func (s *poolSupportStore) GetMonitorScrapePoolList(context.Context, *model.GetMonitorScrapePoolListReq) ([]*model.MonitorScrapePool, int64, error) {
	s.reads++
	return []*model.MonitorScrapePool{{Model: model.Model{ID: 7}, Name: "existing", SupportAlert: 2, SupportRecord: 2}}, 1, nil
}
func (s *poolSupportStore) GetMonitorScrapePoolById(context.Context, int) (*model.MonitorScrapePool, error) {
	s.reads++
	return &model.MonitorScrapePool{Model: model.Model{ID: 7}, Name: "existing", SupportAlert: 2, SupportRecord: 2}, nil
}
func (s *poolSupportStore) CreateMonitorScrapePool(_ context.Context, row *model.MonitorScrapePool) error {
	s.writes++
	copy := *row
	s.created = &copy
	return nil
}
func (s *poolSupportStore) UpdateMonitorScrapePool(_ context.Context, req *model.UpdateMonitorScrapePoolReq) error {
	s.writes++
	copy := *req
	s.updated = &copy
	return nil
}

type poolSupportCache struct {
	calls atomic.Int32
	done  chan struct{}
}

func (c *poolSupportCache) MonitorCacheManager(context.Context) error {
	c.calls.Add(1)
	c.done <- struct{}{}
	return nil
}
func poolSupportService() (service.ScrapePoolService, *poolSupportStore, *poolSupportCache) {
	store := &poolSupportStore{}
	cache := &poolSupportCache{done: make(chan struct{}, 2)}
	return service.NewPrometheusPoolService(store, cache, zap.NewNop(), nil, nil), store, cache
}
func awaitPoolSupportRefresh(t *testing.T, cache *poolSupportCache) {
	t.Helper()
	select {
	case <-cache.done:
	case <-time.After(time.Second):
		t.Fatal("valid save did not refresh cache")
	}
	if cache.calls.Load() != 1 {
		t.Fatal("valid save refreshed more than once")
	}
}
func TestScrapePoolServiceRejectsInvalidSupportBeforeAnyDependency(t *testing.T) {
	for _, operation := range []string{"create", "update"} {
		for _, field := range []string{"alert", "record"} {
			values := []int8{-1, 3, 127}
			if operation == "update" {
				values = append(values, 0)
			}
			for _, value := range values {
				t.Run(fmt.Sprintf("%s/%s/%d", operation, field, value), func(t *testing.T) {
					svc, store, cache := poolSupportService()
					alert, record := int8(2), int8(2)
					if field == "alert" {
						alert = value
					} else {
						record = value
					}
					var err error
					if operation == "create" {
						err = svc.CreateMonitorScrapePool(context.Background(), &model.CreateMonitorScrapePoolReq{Name: "new", SupportAlert: alert, SupportRecord: record, ScrapeInterval: 30, ScrapeTimeout: 10})
					} else {
						err = svc.UpdateMonitorScrapePool(context.Background(), &model.UpdateMonitorScrapePoolReq{ID: 7, Name: "existing", SupportAlert: alert, SupportRecord: record, ScrapeInterval: 30, ScrapeTimeout: 10})
					}
					if err == nil || store.reads != 0 || store.writes != 0 || cache.calls.Load() != 0 {
						t.Fatal("invalid support reached a dependency, persistence or cache refresh")
					}
				})
			}
		}
	}
}
func TestScrapePoolServiceCreateDefaultAndExplicitSupport(t *testing.T) {
	for _, alert := range []int8{0, 1, 2} {
		for _, record := range []int8{0, 1, 2} {
			t.Run(fmt.Sprintf("%d/%d", alert, record), func(t *testing.T) {
				svc, store, cache := poolSupportService()
				req := &model.CreateMonitorScrapePoolReq{Name: "new", SupportAlert: alert, SupportRecord: record, ScrapeInterval: 30, ScrapeTimeout: 10}
				before := *req
				if err := svc.CreateMonitorScrapePool(context.Background(), req); err != nil {
					t.Fatal(err)
				}
				awaitPoolSupportRefresh(t, cache)
				expectedAlert, expectedRecord := alert, record
				if alert == 0 {
					expectedAlert = 2
				}
				if record == 0 {
					expectedRecord = 2
				}
				if store.writes != 1 || store.created == nil || store.created.SupportAlert != expectedAlert || store.created.SupportRecord != expectedRecord || !reflect.DeepEqual(before, *req) {
					t.Fatal("create did not persist the real disabled default or changed caller input")
				}
			})
		}
	}
}
func TestScrapePoolServiceUpdateRequiresBothExplicitFlags(t *testing.T) {
	for _, alert := range []int8{1, 2} {
		for _, record := range []int8{1, 2} {
			t.Run(fmt.Sprintf("%d/%d", alert, record), func(t *testing.T) {
				svc, store, cache := poolSupportService()
				req := &model.UpdateMonitorScrapePoolReq{ID: 7, Name: "existing", SupportAlert: alert, SupportRecord: record, ScrapeInterval: 30, ScrapeTimeout: 10}
				before := *req
				if err := svc.UpdateMonitorScrapePool(context.Background(), req); err != nil {
					t.Fatal(err)
				}
				awaitPoolSupportRefresh(t, cache)
				if store.writes != 1 || store.updated == nil || store.updated.SupportAlert != alert || store.updated.SupportRecord != record || !reflect.DeepEqual(before, *req) {
					t.Fatal("valid explicit update changed support flags")
				}
			})
		}
	}
}

func (*poolSupportCache) RetireDeletedScrapePool(context.Context, int) error { return nil }

func (*poolSupportCache) RetireDeletedAlertManagerPool(context.Context, int) error { return nil }
