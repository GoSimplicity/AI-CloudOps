package app

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/constants"
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

// 测试用口令，运行时拼接，不是任何环境的真实口令。
var (
	bootstrapFixturePassword = "bootstrap-fixture-" + strings.Repeat("b", 6)
	mockFixturePassword      = "mock-fixture-" + strings.Repeat("m", 8)
)

func newAdminTestApp(t *testing.T, models ...any) (*App, *gorm.DB, *observer.ObservedLogs) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: databaseLogger.NewGORMLogger(zap.NewNop())})
	if err != nil {
		t.Fatal(err)
	}
	pool, _ := db.DB()
	pool.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = pool.Close() })
	if len(models) > 0 {
		if err := db.AutoMigrate(models...); err != nil {
			t.Fatal(err)
		}
	}
	core, logs := observer.New(zapcore.DebugLevel)
	return &App{db: db, logger: zap.New(core)}, db, logs
}

func assertEnvCleared(t *testing.T, keys ...string) {
	t.Helper()
	for _, key := range keys {
		if v := os.Getenv(key); v != "" {
			t.Fatalf("%s 读取后应已清除", key)
		}
		if _, ok := os.LookupEnv(key); ok {
			t.Fatalf("%s 应从进程环境中删除", key)
		}
	}
}

func assertLogsExclude(t *testing.T, logs *observer.ObservedLogs, secrets ...string) {
	t.Helper()
	for _, entry := range logs.All() {
		text := entry.Message
		for _, f := range entry.Context {
			text += " " + f.Key + "=" + f.String
			if f.Interface != nil {
				if e, ok := f.Interface.(error); ok {
					text += " " + e.Error()
				}
			}
		}
		for _, s := range secrets {
			if s != "" && strings.Contains(text, s) {
				t.Fatal("日志不得包含口令")
			}
		}
	}
}

func TestBootstrapAdminQueryErrorPropagates(t *testing.T) {
	t.Setenv("ENV", "development")
	t.Setenv(BootstrapAdminPasswordEnv, bootstrapFixturePassword)
	a, _, logs := newAdminTestApp(t) // 无用户表
	err := a.bootstrapAdmin(context.Background())
	if err == nil || !strings.Contains(err.Error(), "查询用户失败") {
		t.Fatalf("用户计数查询失败应透传错误: %v", err)
	}
	if strings.Contains(err.Error(), bootstrapFixturePassword) {
		t.Fatal("错误不得包含口令")
	}
	assertLogsExclude(t, logs, bootstrapFixturePassword)
	assertEnvCleared(t, BootstrapAdminPasswordEnv)
}

func TestBootstrapAdminShortPassword(t *testing.T) {
	short := strings.Repeat("s", MinBootstrapAdminPasswordLen-1)
	t.Run("production refuses", func(t *testing.T) {
		t.Setenv("ENV", "production")
		t.Setenv(BootstrapAdminPasswordEnv, short)
		a, db, logs := newAdminTestApp(t, &model.User{})
		err := a.bootstrapAdmin(context.Background())
		if err == nil || !strings.Contains(err.Error(), BootstrapAdminPasswordEnv) || strings.Contains(err.Error(), short) {
			t.Fatalf("生产环境短口令应报错且不含口令: %v", err)
		}
		if n := countAppUsers(t, db); n != 0 {
			t.Fatalf("users=%d", n)
		}
		assertLogsExclude(t, logs, short)
		assertEnvCleared(t, BootstrapAdminPasswordEnv)
	})
	t.Run("development warns and skips", func(t *testing.T) {
		t.Setenv("ENV", "development")
		t.Setenv(BootstrapAdminPasswordEnv, short)
		a, db, logs := newAdminTestApp(t, &model.User{})
		if err := a.bootstrapAdmin(context.Background()); err != nil {
			t.Fatalf("非生产环境短口令只告警: %v", err)
		}
		if n := countAppUsers(t, db); n != 0 {
			t.Fatalf("短口令不应创建管理员, users=%d", n)
		}
		warns := logs.FilterLevelExact(zapcore.WarnLevel).All()
		if len(warns) != 1 || !strings.HasPrefix(warns[0].Message, "[SECURITY]") || !strings.Contains(warns[0].Message, BootstrapAdminPasswordEnv) {
			t.Fatalf("应打印显眼的 [SECURITY] Warn 并指明变量: %+v", warns)
		}
		assertLogsExclude(t, logs, short)
		assertEnvCleared(t, BootstrapAdminPasswordEnv)
	})
}

func TestBootstrapAdminUsesUserAccountTypeAndClearsEnv(t *testing.T) {
	t.Setenv("ENV", "production")
	t.Setenv(BootstrapAdminPasswordEnv, bootstrapFixturePassword)
	a, db, logs := newAdminTestApp(t, &model.User{})
	if err := a.bootstrapAdmin(context.Background()); err != nil {
		t.Fatal(err)
	}
	admin := loadAdmin(t, db)
	if admin.AccountType != constants.AccountTypeUser {
		t.Fatalf("AccountType=%d want %d", admin.AccountType, constants.AccountTypeUser)
	}
	assertLogsExclude(t, logs, bootstrapFixturePassword)
	assertEnvCleared(t, BootstrapAdminPasswordEnv)
}

func TestScrubAdminPasswordEnvClearsBoth(t *testing.T) {
	t.Setenv(BootstrapAdminPasswordEnv, bootstrapFixturePassword)
	t.Setenv(mock.MockAdminPasswordEnv, mockFixturePassword)
	(&App{logger: zap.NewNop()}).scrubAdminPasswordEnv()
	assertEnvCleared(t, BootstrapAdminPasswordEnv, mock.MockAdminPasswordEnv)
}

// 与 Run 的顺序一致：bootstrapAdmin -> initMockData（mock 开启时）-> scrubAdminPasswordEnv。
func TestBootstrapAndMockAdminCombinations(t *testing.T) {
	for _, tt := range []struct {
		name               string
		bootstrap, mockPwd string
		existingUser       bool
		wantUsers          int64
		wantPassword       string // 期望 admin 对应的口令；空表示不应存在 admin
	}{
		{"neither set", "", "", false, 0, ""},
		{"only bootstrap", bootstrapFixturePassword, "", false, 1, bootstrapFixturePassword},
		{"only mock", "", mockFixturePassword, false, 1, mockFixturePassword},
		{"both set bootstrap wins", bootstrapFixturePassword, mockFixturePassword, false, 1, bootstrapFixturePassword},
		{"existing users untouched", bootstrapFixturePassword, mockFixturePassword, true, 1, ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("ENV", "development")
			t.Setenv(BootstrapAdminPasswordEnv, tt.bootstrap)
			t.Setenv(mock.MockAdminPasswordEnv, tt.mockPwd)
			a, db, logs := newAdminTestApp(t, &model.User{}, &model.Api{})
			if tt.existingUser {
				if err := db.Create(&model.User{Username: "someone", Password: "x", RealName: "x", Enable: 1}).Error; err != nil {
					t.Fatal(err)
				}
			}
			ctx := context.Background()
			if err := a.bootstrapAdmin(ctx); err != nil {
				t.Fatal(err)
			}
			if err := a.initMockData(ctx); err != nil {
				t.Fatal(err)
			}
			a.scrubAdminPasswordEnv()

			if n := countAppUsers(t, db); n != tt.wantUsers {
				t.Fatalf("users=%d want %d", n, tt.wantUsers)
			}
			var admins []model.User
			if err := db.Where("username = ?", mock.AdminUsername).Find(&admins).Error; err != nil {
				t.Fatal(err)
			}
			if tt.wantPassword == "" {
				if len(admins) != 0 {
					t.Fatalf("不应创建 admin: %d", len(admins))
				}
			} else {
				if len(admins) != 1 {
					t.Fatalf("应恰好有一个 admin: %d", len(admins))
				}
				if bcrypt.CompareHashAndPassword([]byte(admins[0].Password), []byte(tt.wantPassword)) != nil {
					t.Fatal("admin 口令来源不符合预期")
				}
				if admins[0].AccountType != constants.AccountTypeUser {
					t.Fatalf("AccountType=%d", admins[0].AccountType)
				}
			}
			assertLogsExclude(t, logs, bootstrapFixturePassword, mockFixturePassword)
			assertEnvCleared(t, BootstrapAdminPasswordEnv, mock.MockAdminPasswordEnv)
		})
	}
}

func countAppUsers(t *testing.T, db *gorm.DB) int64 {
	t.Helper()
	var n int64
	if err := db.Model(&model.User{}).Count(&n).Error; err != nil {
		t.Fatal(err)
	}
	return n
}

func loadAdmin(t *testing.T, db *gorm.DB) model.User {
	t.Helper()
	var admin model.User
	if err := db.Where("username = ?", mock.AdminUsername).First(&admin).Error; err != nil {
		t.Fatal(err)
	}
	return admin
}
