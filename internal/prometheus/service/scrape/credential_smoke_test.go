//go:build smoke

package scrape

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	scrapeDAO "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/scrape"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/utils"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func credentialDatabase(t *testing.T) (*gorm.DB, ScrapeJobService, scrapeDAO.ScrapeJobDAO) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatal(err)
	}
	pool, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	pool.SetMaxOpenConns(1)
	t.Cleanup(func() { pool.Close() })
	if err := db.AutoMigrate(&model.MonitorScrapeJob{}); err != nil {
		t.Fatal(err)
	}
	dao := scrapeDAO.NewScrapeJobDAO(db, zap.NewNop())
	return db, NewPrometheusScrapeService(dao, quietCache{}, zap.NewNop(), nil), dao
}
func seedCredentialJob(t *testing.T, db *gorm.DB, token, file string) *model.MonitorScrapeJob {
	t.Helper()
	job := &model.MonitorScrapeJob{Name: "fixture", UserID: 2, PoolID: 1, Enable: 1, ServiceDiscoveryType: model.ServiceDiscoveryTypeStatic, IpAddress: "127.0.0.1:9100", BearerToken: token, BearerTokenFile: file, TlsCaContent: "public-fixture-ca", CreateUserName: "original creator"}
	if err := db.Create(job).Error; err != nil {
		t.Fatal(err)
	}
	return job
}
func TestCredentialReadsRedactTokenWithoutChangingLegacyStorage(t *testing.T) {
	db, svc, dao := credentialDatabase(t)
	const token = "private-legacy-fixture"
	job := seedCredentialJob(t, db, token, "")
	detail, err := svc.GetMonitorScrapeJobDetail(context.Background(), &model.GetMonitorScrapeJobDetailReq{ID: job.ID})
	if err != nil {
		t.Fatal(err)
	}
	list, err := svc.GetMonitorScrapeJobList(context.Background(), &model.GetMonitorScrapeJobListReq{ListReq: model.ListReq{Page: 1, Size: 10}})
	if err != nil {
		t.Fatal(err)
	}
	for _, response := range []any{detail, list} {
		body, err := json.Marshal(response)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(body), token) || strings.Contains(string(body), `"bearer_token":`) {
			t.Error("read response exposes token field")
		}
		if !strings.Contains(string(body), `"has_bearer_token":true`) || !strings.Contains(string(body), "public-fixture-ca") {
			t.Error("read response loses token presence or public CA")
		}
	}
	stored, err := dao.GetMonitorScrapeJobById(context.Background(), job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.BearerToken != token {
		t.Fatal("reading destructively redacted stored credential")
	}
	cfg, err := utils.ScrapeHTTPClientConfig(stored)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Authorization == nil || string(cfg.Authorization.Credentials) != token {
		t.Fatal("operational configuration lost legacy plaintext credential")
	}
}
func TestCredentialUpdateKeepClearReplaceAndSwitchSources(t *testing.T) {
	for _, tt := range []struct {
		name, oldToken, oldFile, extra, wantToken, wantFile string
		reject                                              bool
	}{
		{name: "omit retains", oldToken: "private-before", wantToken: "private-before"},
		{name: "clear", oldToken: "private-before", extra: `,"bearer_token":""`},
		{name: "replace", oldToken: "private-before", extra: `,"bearer_token":"private-after"`, wantToken: "private-after"},
		{name: "token to file", oldToken: "private-before", extra: `,"bearer_token":"","bearer_token_file":"/target/token"`, wantFile: "/target/token"},
		{name: "file to token", oldFile: "/target/token", extra: `,"bearer_token":"private-after","bearer_token_file":""`, wantToken: "private-after"},
		{name: "omitted token cannot create conflicting file", oldToken: "private-before", extra: `,"bearer_token_file":"/target/token"`, wantToken: "private-before", reject: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			db, svc, dao := credentialDatabase(t)
			job := seedCredentialJob(t, db, tt.oldToken, tt.oldFile)
			var req model.UpdateMonitorScrapeJobReq
			if err := json.Unmarshal([]byte(`{"name":"fixture","pool_id":1,"enable":1,"service_discovery_type":3,"ip_address":"127.0.0.1:9100"`+tt.extra+`}`), &req); err != nil {
				t.Fatal(err)
			}
			req.ID = job.ID
			err := svc.UpdateMonitorScrapeJob(context.Background(), &req)
			if (err != nil) != tt.reject {
				t.Fatal("unexpected merged credential validation result")
			}
			stored, err := dao.GetMonitorScrapeJobById(context.Background(), job.ID)
			if err != nil {
				t.Fatal(err)
			}
			if stored.BearerToken != tt.wantToken || stored.BearerTokenFile != tt.wantFile {
				t.Fatal("credential update did not preserve explicit keep/clear/replace semantics")
			}
			if stored.CreateUserName != "original creator" || stored.UserID != 2 {
				t.Fatal("configuration update changed creator identity")
			}
		})
	}
}

type pausedCredentialStore struct {
	scrapeDAO.ScrapeJobDAO
	snapshotRead chan struct{}
	rotated      chan struct{}
}

func (s *pausedCredentialStore) GetMonitorScrapeJobById(ctx context.Context, id int) (*model.MonitorScrapeJob, error) {
	snapshot, err := s.ScrapeJobDAO.GetMonitorScrapeJobById(ctx, id)
	close(s.snapshotRead)
	select {
	case <-s.rotated:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	return snapshot, err
}
func TestOmittedTokenPreservesRotationAfterServiceRead(t *testing.T) {
	db, _, dao := credentialDatabase(t)
	job := seedCredentialJob(t, db, "private-old", "")
	paused := &pausedCredentialStore{ScrapeJobDAO: dao, snapshotRead: make(chan struct{}), rotated: make(chan struct{})}
	svc := NewPrometheusScrapeService(paused, quietCache{}, zap.NewNop(), nil)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result := make(chan error, 1)
	go func() {
		result <- svc.UpdateMonitorScrapeJob(ctx, &model.UpdateMonitorScrapeJobReq{ID: job.ID, Name: "fixture", PoolID: 1, Enable: 1, ServiceDiscoveryType: model.ServiceDiscoveryTypeStatic})
	}()
	select {
	case <-paused.snapshotRead:
	case <-ctx.Done():
		t.Fatal("service did not reach snapshot boundary")
	}
	if err := db.Model(&model.MonitorScrapeJob{}).Where("id = ?", job.ID).Update("bearer_token", "private-rotated").Error; err != nil {
		close(paused.rotated)
		t.Fatal(err)
	}
	close(paused.rotated)
	select {
	case err := <-result:
		if err != nil {
			t.Fatal(err)
		}
	case <-ctx.Done():
		t.Fatal("configuration update did not complete")
	}
	stored, err := dao.GetMonitorScrapeJobById(ctx, job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.BearerToken != "private-rotated" {
		t.Fatal("omitted token overwrote a newer concurrent rotation")
	}
}
