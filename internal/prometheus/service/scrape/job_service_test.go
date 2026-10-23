package scrape

import (
	"context"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/cache"
	scrapeDAO "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/scrape"
	"go.uber.org/zap"
	"reflect"
	"testing"
)

type jobStore struct {
	scrapeDAO.ScrapeJobDAO
	saved *model.MonitorScrapeJob
}

func (*jobStore) CheckMonitorScrapeJobExists(context.Context, string) (bool, error) {
	return false, nil
}
func (*jobStore) CheckMonitorInstanceExists(context.Context, int) (bool, error) { return true, nil }
func (s *jobStore) CreateMonitorScrapeJob(_ context.Context, job *model.MonitorScrapeJob) error {
	s.saved = job
	return nil
}
func (s *jobStore) UpdateMonitorScrapeJob(_ context.Context, job *model.MonitorScrapeJob, _ *string) error {
	s.saved = job
	return nil
}
func (*jobStore) GetMonitorScrapeJobById(context.Context, int) (*model.MonitorScrapeJob, error) {
	return &model.MonitorScrapeJob{Name: "fixture"}, nil
}

type quietCache struct{ cache.MonitorCache }

func (quietCache) MonitorCacheManager(context.Context) error { return nil }

func TestCreateAndUpdateSaveHTTPDiscoveryNodes(t *testing.T) {
	store := &jobStore{}
	svc := NewPrometheusScrapeService(store, quietCache{}, zap.NewNop(), nil)
	ids := model.StringList{"7", "8"}
	if err := svc.CreateMonitorScrapeJob(context.Background(), &model.CreateMonitorScrapeJobReq{Name: "fixture", TreeNodeIDs: ids}); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(store.saved.TreeNodeIDs, ids) {
		t.Fatal("create lost discovery nodes")
	}
	if err := svc.UpdateMonitorScrapeJob(context.Background(), &model.UpdateMonitorScrapeJobReq{ID: 1, Name: "fixture", TreeNodeIDs: ids}); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(store.saved.TreeNodeIDs, ids) {
		t.Fatal("update lost discovery nodes")
	}
}
func TestConflictingCredentialsRejectedBeforeStorage(t *testing.T) {
	store := &jobStore{}
	svc := NewPrometheusScrapeService(store, quietCache{}, zap.NewNop(), nil)
	if err := svc.CreateMonitorScrapeJob(context.Background(), &model.CreateMonitorScrapeJobReq{BearerToken: "fixture", BearerTokenFile: "/fixture"}); err == nil {
		t.Fatal("conflicting token sources accepted")
	}
	if err := svc.UpdateMonitorScrapeJob(context.Background(), &model.UpdateMonitorScrapeJobReq{ID: 1, TlsCaContent: "fixture", TlsCaFilePath: "/fixture"}); err == nil {
		t.Fatal("conflicting CA sources accepted")
	}
	if store.saved != nil {
		t.Fatal("invalid authentication reached storage")
	}
}
