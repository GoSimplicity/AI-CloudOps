package alert_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	api "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/api"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/cache"
	alert "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/alert"
	service "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/service/alert"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func amLifecycleDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "am.sqlite")), &gorm.Config{Logger: logger.Discard, DisableForeignKeyConstraintWhenMigrating: true})
	if err != nil {
		t.Fatal(err)
	}
	conn, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	conn.SetMaxOpenConns(1)
	t.Cleanup(func() { conn.Close() })
	if err := db.AutoMigrate(&model.MonitorAlertManagerPool{}, &model.MonitorSendGroup{}, &model.MonitorScrapePool{}); err != nil {
		t.Fatal(err)
	}
	for _, id := range []int{7, 8} {
		if err := db.Create(&model.MonitorAlertManagerPool{Model: model.Model{ID: id}, Name: fmt.Sprint("AM", id)}).Error; err != nil {
			t.Fatal(err)
		}
	}
	return db
}
func amDAOs(db *gorm.DB) (alert.AlertManagerPoolDAO, alert.AlertManagerSendDAO) {
	return alert.NewAlertManagerPoolDAO(db, zap.NewNop()), alert.NewAlertManagerSendDAO(db, zap.NewNop())
}

type amLifecycleCache struct{ cache.MonitorCache }

func (amLifecycleCache) MonitorCacheManager(context.Context) error          { return nil }
func (amLifecycleCache) RetireDeletedScrapePool(context.Context, int) error { return nil }

func TestAMPoolDeletionProtectsAllUndeletedSendGroups(t *testing.T) {
	for _, enabled := range []int8{1, 2} {
		t.Run(fmt.Sprint(enabled), func(t *testing.T) {
			db := amLifecycleDB(t)
			p, g := amDAOs(db)
			ctx := context.Background()
			row := &model.MonitorSendGroup{Name: "group", PoolID: 7, Enable: enabled}
			if err := g.CreateMonitorSendGroup(ctx, row); err != nil {
				t.Fatal(err)
			}
			if err := p.DeleteMonitorAlertManagerPool(ctx, 7); err == nil {
				t.Fatal("DAO deleted referenced AM pool")
			}
			svc := service.NewAlertManagerPoolService(p, g, zap.NewNop(), amLifecycleCache{})
			if err := svc.DeleteMonitorAlertManagerPool(ctx, &model.DeleteMonitorAlertManagerPoolReq{ID: 7}); err == nil {
				t.Fatal("service deleted referenced AM pool")
			}
			if err := g.DeleteMonitorSendGroup(ctx, row.ID); err != nil {
				t.Fatal(err)
			}
			if err := p.DeleteMonitorAlertManagerPool(ctx, 7); err != nil {
				t.Fatal(err)
			}
		})
	}
}
func TestAMSendGroupRejectsUnavailableTargetWithoutBorrowingScrapeIdentity(t *testing.T) {
	for _, target := range []int{0, -1, 99, 7} {
		for _, operation := range []string{"create", "update"} {
			t.Run(fmt.Sprintf("%s/%d", operation, target), func(t *testing.T) {
				db := amLifecycleDB(t)
				p, g := amDAOs(db)
				ctx := context.Background()
				if err := p.DeleteMonitorAlertManagerPool(ctx, 7); err != nil {
					t.Fatal(err)
				}
				if err := db.Create(&model.MonitorScrapePool{Model: model.Model{ID: 7}, Name: "unrelated-scrape"}).Error; err != nil {
					t.Fatal(err)
				}
				var err error
				if operation == "create" {
					err = g.CreateMonitorSendGroup(ctx, &model.MonitorSendGroup{Name: "new", PoolID: target, Enable: 2})
				} else {
					row := &model.MonitorSendGroup{Name: "existing", PoolID: 8, Enable: 2}
					if err := g.CreateMonitorSendGroup(ctx, row); err != nil {
						t.Fatal(err)
					}
					row.PoolID = target
					row.Name = "changed"
					err = g.UpdateMonitorSendGroup(ctx, row)
					if target == 0 {
						if err != nil {
							t.Fatal(err)
						}
						got, e := g.GetMonitorSendGroupByID(ctx, row.ID)
						if e != nil || got.PoolID != 8 {
							t.Fatal("zero PoolID stopped preserving stored target")
						}
						return
					}
					got, e := g.GetMonitorSendGroupByID(ctx, row.ID)
					if e != nil || got.PoolID != 8 || got.Name != "existing" {
						t.Fatal("failed migration changed stored definition")
					}
				}
				if err == nil {
					t.Fatal("unavailable AM target accepted")
				}
			})
		}
	}
}
func TestAMSendGroupMigrationAndAssociationContract(t *testing.T) {
	db := amLifecycleDB(t)
	p, g := amDAOs(db)
	ctx := context.Background()
	users := []*model.User{{Username: "first"}, {Username: "second"}}
	if err := db.Create(users).Error; err != nil {
		t.Fatal(err)
	}
	row := &model.MonitorSendGroup{Name: "before", NameZh: "保持名称", PoolID: 7, Enable: 2, StaticReceiveUsers: users[:1], FirstUpgradeUsers: users[:1], SecondUpgradeUsers: users[:1]}
	if err := g.CreateMonitorSendGroup(ctx, row); err != nil {
		t.Fatal(err)
	}
	if err := g.UpdateMonitorSendGroup(ctx, &model.MonitorSendGroup{Model: model.Model{ID: row.ID}, Name: "after", PoolID: 8, StaticReceiveUsers: users[1:], FirstUpgradeUsers: users[1:], SecondUpgradeUsers: users[1:]}); err != nil {
		t.Fatal(err)
	}
	got, err := g.GetMonitorSendGroupByID(ctx, row.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.PoolID != 8 || got.Enable != 2 || got.NameZh != "保持名称" {
		t.Fatal("struct update changed omitted fields")
	}
	for _, list := range [][]*model.User{got.StaticReceiveUsers, got.FirstUpgradeUsers, got.SecondUpgradeUsers} {
		if len(list) != 1 || list[0].ID != users[1].ID {
			t.Fatal("user association replacement changed")
		}
	}
	if err := p.DeleteMonitorAlertManagerPool(ctx, 7); err != nil {
		t.Fatal(err)
	}
	if err := p.DeleteMonitorAlertManagerPool(ctx, 8); err == nil {
		t.Fatal("migrated definition did not protect target")
	}
	if err := g.UpdateMonitorSendGroup(ctx, &model.MonitorSendGroup{Model: model.Model{ID: row.ID}, Name: "without-associations"}); err != nil {
		t.Fatal(err)
	}
	got, err = g.GetMonitorSendGroupByID(ctx, row.ID)
	if err != nil || got.PoolID != 8 || got.Enable != 2 || len(got.StaticReceiveUsers)+len(got.FirstUpgradeUsers)+len(got.SecondUpgradeUsers) != 0 {
		t.Fatal("zero target/empty association update contract changed")
	}
}
func TestAMSendGroupAssociationFailureRollsBackDefinitionAndAllLinks(t *testing.T) {
	db := amLifecycleDB(t)
	_, g := amDAOs(db)
	ctx := context.Background()
	user := &model.User{Username: "linked"}
	if err := db.Create(user).Error; err != nil {
		t.Fatal(err)
	}
	row := &model.MonitorSendGroup{Name: "before", PoolID: 7, Enable: 2, StaticReceiveUsers: []*model.User{user}, FirstUpgradeUsers: []*model.User{user}, SecondUpgradeUsers: []*model.User{user}}
	if err := g.CreateMonitorSendGroup(ctx, row); err != nil {
		t.Fatal(err)
	}
	fired := false
	if err := db.Callback().Delete().Before("gorm:delete").Register("fixture:association-failure", func(tx *gorm.DB) {
		if tx.Statement.Table == "cl_monitor_send_group_second_upgrade_users" {
			fired = true
			tx.AddError(errors.New("fixture failure"))
		}
	}); err != nil {
		t.Fatal(err)
	}
	if err := g.UpdateMonitorSendGroup(ctx, &model.MonitorSendGroup{Model: model.Model{ID: row.ID}, Name: "changed", PoolID: 8}); err == nil || !fired {
		t.Fatal("association failure not exercised")
	}
	got, err := g.GetMonitorSendGroupByID(ctx, row.ID)
	if err != nil || got.PoolID != 7 || got.Name != "before" {
		t.Fatal("failed association transaction changed definition")
	}
	if len(got.StaticReceiveUsers) != 1 || len(got.FirstUpgradeUsers) != 1 || len(got.SecondUpgradeUsers) != 1 {
		t.Fatal("failed association transaction lost earlier user links")
	}
}

type amInterleavedGroups struct {
	alert.AlertManagerSendDAO
	insert func() error
}

func (d amInterleavedGroups) GetMonitorSendGroupByPoolID(ctx context.Context, id int) ([]*model.MonitorSendGroup, int64, error) {
	rows, n, err := d.AlertManagerSendDAO.GetMonitorSendGroupByPoolID(ctx, id)
	if err != nil {
		return nil, 0, err
	}
	if err := d.insert(); err != nil {
		return nil, 0, err
	}
	return rows, n, nil
}
func TestAMDeleteHTTPRejectsGroupCommittedAfterServicePrecheck(t *testing.T) {
	db := amLifecycleDB(t)
	p, g := amDAOs(db)
	wrapped := amInterleavedGroups{AlertManagerSendDAO: g, insert: func() error {
		return g.CreateMonitorSendGroup(context.Background(), &model.MonitorSendGroup{Name: "interleaved", PoolID: 7, Enable: 2})
	}}
	svc := service.NewAlertManagerPoolService(p, wrapped, zap.NewNop(), amLifecycleCache{})
	router := gin.New()
	router.DELETE("/pool/:id", api.NewAlertPoolHandler(svc).DeleteMonitorAlertManagerPool)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest("DELETE", "/pool/7", nil))
	var body struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Code == 0 {
		t.Fatal("HTTP service precheck/delete race was accepted")
	}
	if _, err := p.GetAlertPoolByID(context.Background(), 7); err != nil {
		t.Fatal("referenced parent was deleted")
	}
}

// SQLite serializes transactions in this fixture; native opt-in tests verify
// actual parent SELECT FOR UPDATE contention separately.
func TestAMConcurrentCreateAndDeleteHasOneSafeWinner(t *testing.T) {
	db := amLifecycleDB(t)
	p, g := amDAOs(db)
	start := make(chan struct{})
	var wg sync.WaitGroup
	var createErr, deleteErr error
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-start
		createErr = g.CreateMonitorSendGroup(context.Background(), &model.MonitorSendGroup{Name: "racing", PoolID: 7, Enable: 2})
	}()
	go func() { defer wg.Done(); <-start; deleteErr = p.DeleteMonitorAlertManagerPool(context.Background(), 7) }()
	close(start)
	wg.Wait()
	if (createErr == nil) == (deleteErr == nil) {
		t.Fatalf("unsafe outcomes create=%v delete=%v", createErr, deleteErr)
	}
}

func (amLifecycleCache) RetireDeletedAlertManagerPool(context.Context, int) error { return nil }
