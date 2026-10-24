package app

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/mock"
	databaseLogger "github.com/GoSimplicity/AI-CloudOps/pkg/logger"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// historicalCandidateBcrypt 是用 188e907:mock/user_mock.go 中的旧内置口令离线生成的 bcrypt 哈希（cost=4），
// 用于确认 legacyMockAdminCandidate 的还原结果与历史值一致；测试中不出现口令明文。
const historicalCandidateBcrypt = "$2a$04$o/sQMuK0K6ikk.gPF/52GeMMQg1k7jHR2yYmsrG777oVtYVMjEA3e"

func newLegacyCheckApp(t *testing.T, migrate bool) (*App, *gorm.DB, *observer.ObservedLogs) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: databaseLogger.NewGORMLogger(zap.NewNop())})
	if err != nil {
		t.Fatal(err)
	}
	pool, _ := db.DB()
	pool.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = pool.Close() })
	if migrate {
		if err := db.AutoMigrate(&model.User{}); err != nil {
			t.Fatal(err)
		}
	}
	core, logs := observer.New(zapcore.DebugLevel)
	return &App{db: db, logger: zap.New(core)}, db, logs
}

func legacyCandidateHash(t *testing.T) string {
	t.Helper()
	candidate := legacyMockAdminCandidate()
	defer clear(candidate)
	hash, err := bcrypt.GenerateFromPassword(candidate, bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	return string(hash)
}

func strongHash(t *testing.T) string {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte("fixture-"+strings.Repeat("z", 12)), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	return string(hash)
}

func createUser(t *testing.T, db *gorm.DB, username, hash string) {
	t.Helper()
	if err := db.Create(&model.User{Username: username, Password: hash, RealName: "fixture", Enable: 1}).Error; err != nil {
		t.Fatal(err)
	}
}

func TestLegacyCandidateMatchesHistoricalValue(t *testing.T) {
	candidate := legacyMockAdminCandidate()
	defer clear(candidate)
	if len(candidate) == 0 {
		t.Fatal("候选值不能为空")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(historicalCandidateBcrypt), candidate); err != nil {
		t.Fatalf("候选值还原结果与 188e907 的历史值不一致: %v", err)
	}
	// 每次调用返回独立副本，调用方清零不影响后续比对。
	clear(candidate)
	again := legacyMockAdminCandidate()
	defer clear(again)
	if bcrypt.CompareHashAndPassword([]byte(historicalCandidateBcrypt), again) != nil {
		t.Fatal("清零调用方副本后不应影响下一次还原")
	}
}

func TestLegacyAdminPasswordRefusedInProduction(t *testing.T) {
	for _, env := range []string{"production", "PROD", " prd ", "", "test", "testing"} {
		t.Run(env, func(t *testing.T) {
			t.Setenv("ENV", env)
			a, db, logs := newLegacyCheckApp(t, true)
			createUser(t, db, mock.AdminUsername, legacyCandidateHash(t))
			err := a.checkLegacyAdminPassword(context.Background())
			if !errors.Is(err, ErrLegacyAdminPassword) {
				t.Fatalf("生产环境应拒绝启动, err=%v", err)
			}
			if !strings.Contains(err.Error(), "[SECURITY]") || !strings.Contains(err.Error(), "强口令") {
				t.Fatalf("提示应显眼并说明如何修复: %v", err)
			}
			if logs.Len() != 0 {
				t.Fatalf("拒绝启动时不应额外打印日志: %d", logs.Len())
			}
		})
	}
}

func TestLegacyAdminPasswordWarnsOutsideProduction(t *testing.T) {
	for _, env := range []string{"development", "dev", " DEV "} {
		t.Run(env, func(t *testing.T) {
			t.Setenv("ENV", env)
			a, db, logs := newLegacyCheckApp(t, true)
			createUser(t, db, mock.AdminUsername, legacyCandidateHash(t))
			if err := a.checkLegacyAdminPassword(context.Background()); err != nil {
				t.Fatalf("非生产环境只告警: %v", err)
			}
			entries := logs.FilterLevelExact(zapcore.WarnLevel).All()
			if len(entries) != 1 || !strings.HasPrefix(entries[0].Message, "[SECURITY]") || !strings.Contains(entries[0].Message, "修改 admin 口令") {
				t.Fatalf("应打印一条显眼的 [SECURITY] Warn: %+v", entries)
			}
			// 告警只包含固定文案，不携带任何口令或哈希字段。
			if len(entries[0].Context) != 0 {
				t.Fatalf("告警不应携带字段: %+v", entries[0].Context)
			}
		})
	}
}

func TestLegacyAdminPasswordSkipsWhenAdminMissing(t *testing.T) {
	for _, env := range []string{"production", "development"} {
		t.Run(env, func(t *testing.T) {
			t.Setenv("ENV", env)
			a, db, logs := newLegacyCheckApp(t, true)
			// 其他用户使用旧口令不在本检查范围内。
			createUser(t, db, "someone", legacyCandidateHash(t))
			if err := a.checkLegacyAdminPassword(context.Background()); err != nil {
				t.Fatal(err)
			}
			if logs.Len() != 0 {
				t.Fatalf("admin 不存在时应静默跳过: %d", logs.Len())
			}
		})
	}
}

func TestLegacyAdminPasswordMismatchPasses(t *testing.T) {
	t.Setenv("ENV", "production")
	a, db, logs := newLegacyCheckApp(t, true)
	createUser(t, db, mock.AdminUsername, strongHash(t))
	if err := a.checkLegacyAdminPassword(context.Background()); err != nil {
		t.Fatal(err)
	}
	if logs.Len() != 0 {
		t.Fatalf("口令不匹配时不应告警: %d", logs.Len())
	}
}

func TestLegacyAdminPasswordChecksEveryAdminRow(t *testing.T) {
	t.Setenv("ENV", "production")
	a, db, _ := newLegacyCheckApp(t, true)
	createUser(t, db, mock.AdminUsername, strongHash(t))
	createUser(t, db, mock.AdminUsername, legacyCandidateHash(t))
	if err := a.checkLegacyAdminPassword(context.Background()); !errors.Is(err, ErrLegacyAdminPassword) {
		t.Fatalf("任一同名 admin 使用旧口令都应拒绝, err=%v", err)
	}
}

func TestLegacyAdminPasswordQueryErrorPropagates(t *testing.T) {
	t.Run("production refuses", func(t *testing.T) {
		t.Setenv("ENV", "production")
		a, _, _ := newLegacyCheckApp(t, false) // 无用户表，查询必然失败
		err := a.checkLegacyAdminPassword(context.Background())
		if err == nil || errors.Is(err, ErrLegacyAdminPassword) || !strings.Contains(err.Error(), "查询用户失败") {
			t.Fatalf("生产环境读库失败应透传错误并拒绝启动: %v", err)
		}
	})
	t.Run("production keeps cancellation", func(t *testing.T) {
		t.Setenv("ENV", "production")
		a, _, _ := newLegacyCheckApp(t, true)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if err := a.checkLegacyAdminPassword(ctx); !errors.Is(err, context.Canceled) {
			t.Fatalf("应保留原始错误链: %v", err)
		}
	})
	t.Run("development warns", func(t *testing.T) {
		t.Setenv("ENV", "development")
		a, _, logs := newLegacyCheckApp(t, false)
		if err := a.checkLegacyAdminPassword(context.Background()); err != nil {
			t.Fatalf("非生产环境读库失败只告警: %v", err)
		}
		entries := logs.FilterLevelExact(zapcore.WarnLevel).All()
		if len(entries) != 1 || !strings.HasPrefix(entries[0].Message, "[SECURITY]") {
			t.Fatalf("应打印 [SECURITY] Warn: %+v", entries)
		}
	})
	t.Run("in-use helper returns error", func(t *testing.T) {
		a, _, _ := newLegacyCheckApp(t, false)
		inUse, err := a.legacyAdminPasswordInUse(context.Background())
		if err == nil || inUse {
			t.Fatalf("inUse=%v err=%v", inUse, err)
		}
	})
}
