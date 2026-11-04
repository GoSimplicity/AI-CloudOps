package config

import (
	"context"
	"errors"
	"fmt"
	"math"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func generatedQueryFixture(t *testing.T) (*monitorConfigDAO, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "query.sqlite")), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatal(err)
	}
	conn, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	conn.SetMaxOpenConns(1)
	t.Cleanup(func() { conn.Close() })
	// Model-compatible test table with a deliberately weaker legacy collation.
	if err := db.Exec(`CREATE TABLE cl_monitor_configs (id INTEGER PRIMARY KEY AUTOINCREMENT,created_at DATETIME,updated_at DATETIME,deleted_at INTEGER DEFAULT 0,name TEXT NOT NULL,pool_id INTEGER NOT NULL,instance_ip TEXT COLLATE NOCASE NOT NULL,config_type INTEGER NOT NULL,config_content TEXT NOT NULL,config_hash TEXT NOT NULL,status INTEGER DEFAULT 1,last_generated_time INTEGER NOT NULL)`).Error; err != nil {
		t.Fatal(err)
	}
	return NewMonitorConfigDAO(zap.NewNop(), db).(*monitorConfigDAO), db
}
func seedGeneratedQueryRows(t *testing.T, db *gorm.DB) {
	t.Helper()
	rows := []*model.MonitorConfig{
		{Model: model.Model{ID: 1}, PoolID: 7, ConfigType: 4, InstanceIP: "192.0.2.1", Name: "canonical", Status: 1},
		{Model: model.Model{ID: 2}, PoolID: 7, ConfigType: 4, InstanceIP: "192.0.2.1", Name: "自定义", Status: 2},
		{Model: model.Model{ID: 3}, PoolID: 99, ConfigType: 4, InstanceIP: "192.0.2.1", Name: "foreign", Status: 2},
		{Model: model.Model{ID: 4}, PoolID: 7, ConfigType: 4, InstanceIP: "192.0.2.2", Name: "other", Status: 2},
		{Model: model.Model{ID: 5}, PoolID: 7, ConfigType: 1, InstanceIP: "192.0.2.1", Name: "other kind", Status: 1},
		{Model: model.Model{ID: 6, DeletedAt: 10}, PoolID: 7, ConfigType: 4, InstanceIP: "192.0.2.1", Name: "deleted", Status: 1},
		{Model: model.Model{ID: 7}, PoolID: 7, ConfigType: 4, InstanceIP: "2001:db8::1", Name: "lower", Status: 1},
		{Model: model.Model{ID: 8}, PoolID: 7, ConfigType: 4, InstanceIP: "2001:DB8::1", Name: "upper", Status: 2},
		{Model: model.Model{ID: 9}, PoolID: 7, ConfigType: 4, InstanceIP: "192.0.2.3", Name: "no suffix", Status: 1},
		{Model: model.Model{ID: 10}, PoolID: 7, ConfigType: 4, InstanceIP: "192.0.2.3 ", Name: "trailing space", Status: 2},
		{Model: model.Model{ID: 11}, PoolID: 99, ConfigType: 4, InstanceIP: " 192.0.2.3", Name: "leading space", Status: 1},
	}
	for _, row := range rows {
		row.ConfigContent = "fixture bytes"
		row.ConfigHash = "fixture-hash"
		row.LastGeneratedTime = 1
	}
	if err := db.Create(rows).Error; err != nil {
		t.Fatal(err)
	}
}
func assertConfigQueryIDs(t *testing.T, rows []*model.MonitorConfig, total int64, err error, wantTotal int64, want ...int) {
	t.Helper()
	got := make([]int, 0, len(rows))
	for _, row := range rows {
		got = append(got, row.ID)
	}
	if err != nil || total != wantTotal || !reflect.DeepEqual(got, append([]int{}, want...)) {
		t.Fatalf("unexpected page IDs=%v total=%d error=%v", got, total, err)
	}
}

// Shared by real SQLite and opt-in MySQL; assertions exercise executed queries,
// including duplicate rows and weak-column-collation byte boundaries.
func exerciseGeneratedConfigQueries(t *testing.T, dao *monitorConfigDAO, db *gorm.DB) {
	t.Helper()
	seedGeneratedQueryRows(t, db)
	ctx := context.Background()
	t.Run("pool candidates preserve custom and inactive", func(t *testing.T) {
		for page, want := range [][]int{{1, 2}, {4, 7}, {8, 9}, {10}, {}} {
			rows, total, err := dao.GetGeneratedConfigPoolCandidates(ctx, GeneratedConfigPoolQuery{PoolID: 7, ConfigType: 4, Page: page + 1, Size: 2})
			assertConfigQueryIDs(t, rows, total, err, 7, want...)
		}
		rows, total, err := dao.GetGeneratedConfigPoolCandidates(ctx, GeneratedConfigPoolQuery{PoolID: 99, ConfigType: 4, Page: 1, Size: 100})
		assertConfigQueryIDs(t, rows, total, err, 2, 3, 11)
	})
	t.Run("namespace preserves all owners and duplicate rows", func(t *testing.T) {
		for page, want := range [][]int{{1, 2}, {3, 4}, {}} {
			rows, total, err := dao.GetGeneratedConfigNamespacePage(ctx, GeneratedConfigKeysQuery{ConfigType: 4, InstanceKeys: []string{"192.0.2.2", "192.0.2.1"}, Page: page + 1, Size: 2})
			assertConfigQueryIDs(t, rows, total, err, 4, want...)
		}
	})
	t.Run("byte identity", func(t *testing.T) {
		for _, tc := range []struct {
			key string
			id  int
		}{{"2001:db8::1", 7}, {"2001:DB8::1", 8}, {"192.0.2.3", 9}, {"192.0.2.3 ", 10}, {" 192.0.2.3", 11}} {
			t.Run(fmt.Sprint(tc.id), func(t *testing.T) {
				rows, total, err := dao.GetGeneratedConfigNamespacePage(ctx, GeneratedConfigKeysQuery{ConfigType: 4, InstanceKeys: []string{tc.key}, Page: 1, Size: 100})
				assertConfigQueryIDs(t, rows, total, err, 1, tc.id)
			})
		}
	})
	t.Run("boundary empty and duplicate inputs", func(t *testing.T) {
		keys := make([]string, GeneratedConfigQueryLimit)
		for i := range keys {
			keys[i] = fmt.Sprint("unmatched-", i)
		}
		keys[0] = "192.0.2.1"
		rows, total, err := dao.GetGeneratedConfigNamespacePage(ctx, GeneratedConfigKeysQuery{ConfigType: 4, InstanceKeys: keys, Page: 1, Size: 100})
		assertConfigQueryIDs(t, rows, total, err, 3, 1, 2, 3)
		if rows, total, err := dao.GetGeneratedConfigNamespacePage(ctx, GeneratedConfigKeysQuery{ConfigType: 4, InstanceKeys: append(keys, "over-cap-nonempty"), Page: 1, Size: 100}); !errors.Is(err, ErrInvalidGeneratedConfigQuery) || rows != nil || total != 0 {
			t.Fatal("over-cap nonempty keys were truncated or accepted")
		}
		rows, total, err = dao.GetGeneratedConfigNamespacePage(ctx, GeneratedConfigKeysQuery{ConfigType: 4, InstanceKeys: []string{"2001:db8::1", "2001:db8::1"}, Page: 1, Size: 100})
		assertConfigQueryIDs(t, rows, total, err, 1, 7)
		rows, total, err = dao.GetGeneratedConfigNamespacePage(ctx, GeneratedConfigKeysQuery{ConfigType: 4, Page: 1, Size: 100})
		assertConfigQueryIDs(t, rows, total, err, 0)
	})
	t.Run("ordinary API contract unchanged", func(t *testing.T) {
		// The old API deliberately retains the database's normal IN collation.
		rows, err := dao.GetMonitorConfigsByInstances(ctx, []string{"2001:db8::1"}, 4)
		if err != nil || len(rows) != 2 {
			t.Fatal("ordinary instance lookup contract was changed")
		}
		rows, total, err := dao.GetGeneratedConfigPoolCandidates(ctx, GeneratedConfigPoolQuery{PoolID: 7, ConfigType: 1, Page: 1, Size: 100})
		assertConfigQueryIDs(t, rows, total, err, 1, 5)
		for _, kind := range []int8{2, 3, 5} {
			rows, total, err := dao.GetGeneratedConfigPoolCandidates(ctx, GeneratedConfigPoolQuery{PoolID: 7, ConfigType: kind, Page: 1, Size: 100})
			assertConfigQueryIDs(t, rows, total, err, 0)
		}
	})
}
func TestGeneratedConfigQueriesSQLite(t *testing.T) {
	dao, db := generatedQueryFixture(t)
	exerciseGeneratedConfigQueries(t, dao, db)
}

func TestGeneratedConfigQueriesRejectInvalidRequestsWithoutSQL(t *testing.T) {
	dao, db := generatedQueryFixture(t)
	reads := 0
	if err := db.Callback().Query().Before("gorm:query").Register("fixture:query-count", func(*gorm.DB) { reads++ }); err != nil {
		t.Fatal(err)
	}
	for _, req := range []GeneratedConfigPoolQuery{{PoolID: 0, ConfigType: 4, Page: 1, Size: 100}, {PoolID: -1, ConfigType: 4, Page: 1, Size: 100}, {PoolID: 7, ConfigType: 0, Page: 1, Size: 100}, {PoolID: 7, ConfigType: 6, Page: 1, Size: 100}, {PoolID: 7, ConfigType: 4, Page: 0, Size: 100}, {PoolID: 7, ConfigType: 4, Page: -1, Size: 100}, {PoolID: 7, ConfigType: 4, Page: 1, Size: 0}, {PoolID: 7, ConfigType: 4, Page: 1, Size: 101}, {PoolID: 7, ConfigType: 4, Page: math.MaxInt, Size: 100}} {
		rows, total, err := dao.GetGeneratedConfigPoolCandidates(context.Background(), req)
		if !errors.Is(err, ErrInvalidGeneratedConfigQuery) || rows != nil || total != 0 {
			t.Fatal("invalid pool query accepted")
		}
	}
	for _, req := range []GeneratedConfigKeysQuery{{ConfigType: 0, Page: 1, Size: 100}, {ConfigType: 6, Page: 1, Size: 100}, {ConfigType: 4, Page: 0, Size: 100}, {ConfigType: 4, Page: 1, Size: 0}, {ConfigType: 4, Page: 1, Size: 101}, {ConfigType: 4, Page: math.MaxInt, Size: 100}, {ConfigType: 4, InstanceKeys: make([]string, 101), Page: 1, Size: 100}, {ConfigType: 4, InstanceKeys: []string{""}, Page: 1, Size: 100}} {
		rows, total, err := dao.GetGeneratedConfigNamespacePage(context.Background(), req)
		if !errors.Is(err, ErrInvalidGeneratedConfigQuery) || rows != nil || total != 0 {
			t.Fatal("invalid namespace query accepted or truncated")
		}
	}
	for _, keys := range [][]string{nil, {}} {
		rows, total, err := dao.GetGeneratedConfigNamespacePage(context.Background(), GeneratedConfigKeysQuery{ConfigType: 4, InstanceKeys: keys, Page: 1, Size: 100})
		assertConfigQueryIDs(t, rows, total, err, 0)
	}
	if reads != 0 {
		t.Fatal("invalid or empty-key query reached SQL")
	}
}
func TestGeneratedConfigQueriesCancellationAndReadFailure(t *testing.T) {
	for _, operation := range []string{"pool", "namespace"} {
		for _, mode := range []string{"pre-cancel", "count-cancel", "page-cancel", "SQL failure"} {
			t.Run(operation+"/"+mode, func(t *testing.T) {
				dao, db := generatedQueryFixture(t)
				seedGeneratedQueryRows(t, db)
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				calls := 0
				if mode == "pre-cancel" {
					cancel()
				}
				if mode == "SQL failure" {
					if err := db.Migrator().DropTable(&model.MonitorConfig{}); err != nil {
						t.Fatal(err)
					}
				}
				if err := db.Callback().Query().Before("gorm:query").Register("fixture:cancel-query", func(*gorm.DB) {
					calls++
					if mode == "count-cancel" || mode == "page-cancel" && calls == 2 {
						cancel()
					}
				}); err != nil {
					t.Fatal(err)
				}
				var rows []*model.MonitorConfig
				var total int64
				var err error
				if operation == "pool" {
					rows, total, err = dao.GetGeneratedConfigPoolCandidates(ctx, GeneratedConfigPoolQuery{PoolID: 7, ConfigType: 4, Page: 1, Size: 100})
				} else {
					rows, total, err = dao.GetGeneratedConfigNamespacePage(ctx, GeneratedConfigKeysQuery{ConfigType: 4, InstanceKeys: []string{"192.0.2.1"}, Page: 1, Size: 100})
				}
				if err == nil || rows != nil || total != 0 {
					t.Fatal("failed read returned an authoritative page")
				}
				if mode != "SQL failure" && !errors.Is(err, context.Canceled) {
					t.Fatal("cancelled query lost cancellation identity")
				}
				if mode == "SQL failure" && !errors.Is(err, ErrGeneratedConfigQuery) {
					t.Fatal("query failure classification lost")
				}
				if mode == "pre-cancel" && calls != 0 {
					t.Fatal("pre-cancelled query touched SQL")
				}
			})
		}
	}
}
