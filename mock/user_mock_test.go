package mock

import (
	"bytes"
	"errors"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/constants"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// 测试用口令，运行时拼接，不是任何环境的真实口令
var fixturePassword = "mock-fixture-" + strings.Repeat("x", 6) + "-0001"

func newUserMockDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	pool, _ := db.DB()
	pool.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = pool.Close() })
	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	buf := &bytes.Buffer{}
	log.SetOutput(buf)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
	return buf
}

func newMockWithEnv(db *gorm.DB, value string) *UserMock {
	m := NewUserMock(db)
	m.getenv = func(key string) string {
		if key == MockAdminPasswordEnv {
			return value
		}
		return ""
	}
	return m
}

func countUsers(t *testing.T, db *gorm.DB) int64 {
	t.Helper()
	var n int64
	if err := db.Model(&model.User{}).Count(&n).Error; err != nil {
		t.Fatal(err)
	}
	return n
}

func TestCreateUserAdminSkipsWithoutEnvPassword(t *testing.T) {
	db := newUserMockDB(t)
	logs := captureLog(t)
	if err := newMockWithEnv(db, "").CreateUserAdmin(); err != nil {
		t.Fatal(err)
	}
	if n := countUsers(t, db); n != 0 {
		t.Fatalf("未设置 %s 时不应创建管理员, users=%d", MockAdminPasswordEnv, n)
	}
	if !strings.Contains(logs.String(), MockAdminPasswordEnv) {
		t.Fatal("跳过时应提示如何设置口令")
	}
}

func TestCreateUserAdminRejectsShortPassword(t *testing.T) {
	t.Setenv("ENV", "production")
	db := newUserMockDB(t)
	short := strings.Repeat("s", MinMockAdminPasswordLen-1)
	logs := captureLog(t)
	if err := newMockWithEnv(db, short).CreateUserAdmin(); !errors.Is(err, ErrMockAdminPasswordTooShort) {
		t.Fatalf("err=%v", err)
	}
	if n := countUsers(t, db); n != 0 {
		t.Fatalf("users=%d", n)
	}
	if strings.Contains(logs.String(), short) {
		t.Fatal("日志不得包含口令")
	}
}

func TestCreateUserAdminUsesEnvPasswordAndNeverLogsIt(t *testing.T) {
	db := newUserMockDB(t)
	logs := captureLog(t)
	if err := newMockWithEnv(db, fixturePassword).CreateUserAdmin(); err != nil {
		t.Fatal(err)
	}
	var admin model.User
	if err := db.Where("username = ?", AdminUsername).First(&admin).Error; err != nil {
		t.Fatal(err)
	}
	if admin.Password == fixturePassword {
		t.Fatal("口令不得明文入库")
	}
	if bcrypt.CompareHashAndPassword([]byte(admin.Password), []byte(fixturePassword)) != nil {
		t.Fatal("入库哈希应对应环境变量中的口令")
	}
	if strings.Contains(logs.String(), fixturePassword) {
		t.Fatal("日志不得包含口令")
	}
}

func TestCreateUserAdminDoesNotTouchExistingUsers(t *testing.T) {
	db := newUserMockDB(t)
	if err := db.Create(&model.User{Username: "someone", Password: "hash", RealName: "x", Enable: 1}).Error; err != nil {
		t.Fatal(err)
	}
	if err := newMockWithEnv(db, fixturePassword).CreateUserAdmin(); err != nil {
		t.Fatal(err)
	}
	if n := countUsers(t, db); n != 1 {
		t.Fatalf("已有用户时不应再创建管理员, users=%d", n)
	}
}

func TestNewUserMockReadsProcessEnv(t *testing.T) {
	t.Setenv(MockAdminPasswordEnv, fixturePassword)
	if got := NewUserMock(nil).getenv(MockAdminPasswordEnv); got != fixturePassword {
		t.Fatal("默认应从进程环境变量读取口令")
	}
	if os.Getenv(MockAdminPasswordEnv) != "" {
		t.Fatal("读取后应立即清除环境变量")
	}
	if _, ok := os.LookupEnv(MockAdminPasswordEnv); ok {
		t.Fatal("读取后应从进程环境中删除")
	}
}

func TestCreateUserAdminWarnsOnShortPasswordOutsideProduction(t *testing.T) {
	for _, env := range []string{"development", "Dev", " dev "} {
		t.Run(env, func(t *testing.T) {
			t.Setenv("ENV", env)
			db := newUserMockDB(t)
			short := strings.Repeat("s", MinMockAdminPasswordLen-1)
			logs := captureLog(t)
			if err := newMockWithEnv(db, short).CreateUserAdmin(); err != nil {
				t.Fatalf("非生产环境短口令只告警: %v", err)
			}
			if n := countUsers(t, db); n != 0 {
				t.Fatalf("短口令不应创建管理员, users=%d", n)
			}
			out := logs.String()
			if !strings.Contains(out, "[SECURITY]") || !strings.Contains(out, MockAdminPasswordEnv) {
				t.Fatal("应打印显眼的 [SECURITY] 告警并指明变量")
			}
			if strings.Contains(out, short) {
				t.Fatal("日志不得包含口令")
			}
		})
	}
}

func TestCreateUserAdminCountErrorPropagates(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	pool, _ := db.DB()
	t.Cleanup(func() { _ = pool.Close() })
	logs := captureLog(t)
	// 未建表：用户计数查询必然失败，错误必须透传而不是被当作空表继续创建
	err = newMockWithEnv(db, fixturePassword).CreateUserAdmin()
	if err == nil || !strings.Contains(err.Error(), "查询用户数量失败") {
		t.Fatalf("err=%v", err)
	}
	if strings.Contains(err.Error(), fixturePassword) || strings.Contains(logs.String(), fixturePassword) {
		t.Fatal("错误与日志不得包含口令")
	}
}

func TestCreateUserAdminClearsProcessEnvEvenWhenUsersExist(t *testing.T) {
	t.Setenv("ENV", "development")
	t.Setenv(MockAdminPasswordEnv, fixturePassword)
	db := newUserMockDB(t)
	if err := db.Create(&model.User{Username: "someone", Password: "hash", RealName: "x", Enable: 1}).Error; err != nil {
		t.Fatal(err)
	}
	if err := NewUserMock(db).CreateUserAdmin(); err != nil {
		t.Fatal(err)
	}
	if _, ok := os.LookupEnv(MockAdminPasswordEnv); ok {
		t.Fatal("即使跳过创建也应清除口令环境变量")
	}
}

func TestCreateUserAdminUsesUserAccountType(t *testing.T) {
	t.Setenv("ENV", "development")
	db := newUserMockDB(t)
	if err := newMockWithEnv(db, fixturePassword).CreateUserAdmin(); err != nil {
		t.Fatal(err)
	}
	var admin model.User
	if err := db.Where("username = ?", AdminUsername).First(&admin).Error; err != nil {
		t.Fatal(err)
	}
	if admin.AccountType != constants.AccountTypeUser || AdminAccountType != constants.AccountTypeUser {
		t.Fatalf("mock 管理员应与 bootstrap 一致使用 AccountTypeUser, got=%d", admin.AccountType)
	}
}
