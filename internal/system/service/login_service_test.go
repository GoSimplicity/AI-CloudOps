package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/config"
	"github.com/GoSimplicity/AI-CloudOps/internal/constants"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/system/dao"
	userutils "github.com/GoSimplicity/AI-CloudOps/internal/system/utils"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type loginUserDAO struct {
	dao.UserDAO
	users   map[string]*model.User
	queried []string
	err     error
}

func (d *loginUserDAO) GetByUsername(_ context.Context, username string) (*model.User, error) {
	d.queried = append(d.queried, username)
	if d.err != nil {
		return nil, d.err
	}
	user, ok := d.users[username]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return user, nil
}

type memoryGuard struct {
	checks     []error
	failures   []error
	successErr error
	failCount  int
	okCount    int
	lastUser   string
	lastIP     string
}

func (g *memoryGuard) Check(context.Context, string, string) error {
	if len(g.checks) == 0 {
		return nil
	}
	err := g.checks[0]
	g.checks = g.checks[1:]
	return err
}

func (g *memoryGuard) RecordFailure(_ context.Context, username, ip string) error {
	g.failCount++
	g.lastUser = username
	g.lastIP = ip
	if len(g.failures) == 0 {
		return nil
	}
	err := g.failures[0]
	g.failures = g.failures[1:]
	return err
}

func (g *memoryGuard) RecordSuccess(context.Context, string, string) error {
	g.okCount++
	return g.successErr
}

func swapCompare(t *testing.T, fn func(string, string) error) {
	t.Helper()
	original := comparePassword
	comparePassword = fn
	t.Cleanup(func() { comparePassword = original })
}

func TestDummyPasswordHashUsesDefaultCost(t *testing.T) {
	cost, err := bcrypt.Cost([]byte(userutils.LoginTimingHash()))
	if err != nil || cost != bcrypt.DefaultCost {
		t.Fatalf("cost=%d err=%v", cost, err)
	}
	start := time.Now()
	if err = comparePassword(userutils.LoginTimingHash(), "not-a-real-password"); err == nil {
		t.Fatal("dummy hash accepted an arbitrary password")
	}
	if time.Since(start) < 10*time.Millisecond {
		t.Fatal("dummy compare was cheaper than a normal bcrypt compare")
	}
}

func TestLoginUnifiedCredentialFailureAndTimingCompare(t *testing.T) {
	hash, err := userutils.HashPassword("right-password")
	if err != nil {
		t.Fatal(err)
	}
	users := &loginUserDAO{users: map[string]*model.User{
		"alice":  {Model: model.Model{ID: 7}, Username: "alice", Password: hash, Enable: 1},
		"frozen": {Model: model.Model{ID: 8}, Username: "frozen", Password: hash, Enable: 2},
	}}
	guard := &memoryGuard{}
	core, logs := observer.New(zapcore.DebugLevel)
	svc := NewUserService(users, nil, zap.New(core), guard)
	var seen []string
	swapCompare(t, func(hashed, raw string) error {
		seen = append(seen, hashed+"|"+raw)
		return comparePasswordBackup(hashed, raw)
	})

	const secret = "s3cret-login-value"
	ctx := context.Background()
	_, err = svc.Login(ctx, &model.UserLoginReq{Username: " ghost ", Password: secret}, "203.0.113.8")
	if !errors.Is(err, constants.ErrorInvalidCredentials) || guard.failCount != 1 || guard.lastUser != "ghost" {
		t.Fatalf("unknown user err=%v fails=%d user=%s", err, guard.failCount, guard.lastUser)
	}
	_, err = svc.Login(ctx, &model.UserLoginReq{Username: "alice", Password: secret}, "203.0.113.8")
	if !errors.Is(err, constants.ErrorInvalidCredentials) || guard.failCount != 2 {
		t.Fatalf("wrong password err=%v fails=%d", err, guard.failCount)
	}
	_, err = svc.Login(ctx, &model.UserLoginReq{Username: "frozen", Password: "right-password"}, "203.0.113.8")
	if !errors.Is(err, constants.ErrorInvalidCredentials) || guard.okCount != 0 {
		t.Fatalf("disabled account err=%v success=%d", err, guard.okCount)
	}
	if len(seen) != 3 || !strings.HasPrefix(seen[0], userutils.LoginTimingHash()+"|"+secret) || !strings.HasPrefix(seen[1], hash+"|"+secret) || !strings.HasPrefix(seen[2], hash+"|right-password") {
		t.Fatalf("compares=%v", seen)
	}
	assertNoSecret(t, logs, secret)
	for _, queried := range users.queried {
		if queried != strings.TrimSpace(queried) {
			t.Fatalf("username not trimmed: %q", queried)
		}
	}
}

func comparePasswordBackup(hashed, raw string) error {
	return userutils.ComparePassword(hashed, raw)
}

func TestLoginSuccessClearsAndLockReplacesCredentialError(t *testing.T) {
	hash, err := userutils.HashPassword("right-password")
	if err != nil {
		t.Fatal(err)
	}
	users := &loginUserDAO{users: map[string]*model.User{
		"alice": {Model: model.Model{ID: 7}, Username: "alice", Password: hash, Enable: 1},
	}}
	guard := &memoryGuard{}
	svc := NewUserService(users, nil, zap.NewNop(), guard)
	user, err := svc.Login(context.Background(), &model.UserLoginReq{Username: " alice ", Password: "right-password"}, "198.51.100.4")
	if err != nil || user == nil || user.ID != 7 || guard.okCount != 1 || guard.lastIP != "" && guard.failCount != 0 {
		t.Fatalf("success user=%v err=%v guard=%+v", user, err, guard)
	}
	if users.queried[0] != "alice" {
		t.Fatalf("lookup=%v", users.queried)
	}

	guard.failures = []error{&LoginLockedError{RetryAfter: 15 * time.Minute}}
	_, err = svc.Login(context.Background(), &model.UserLoginReq{Username: "alice", Password: "wrong-password"}, "198.51.100.4")
	var locked *LoginLockedError
	if !errors.As(err, &locked) || locked.Error() != "登录失败次数过多，请 15 分钟后再试" {
		t.Fatalf("lock result=%v", err)
	}

	guard.checks = []error{ErrLoginUnavailable}
	_, err = svc.Login(context.Background(), &model.UserLoginReq{Username: "alice", Password: "right-password"}, "198.51.100.4")
	if !errors.Is(err, ErrLoginUnavailable) || guard.okCount != 1 {
		t.Fatalf("unavailable err=%v success=%d", err, guard.okCount)
	}
}

func TestLoginAlreadyLockedSkipsPasswordCompare(t *testing.T) {
	guard := &memoryGuard{checks: []error{&LoginLockedError{RetryAfter: time.Minute}}}
	svc := NewUserService(&loginUserDAO{}, nil, zap.NewNop(), guard)
	called := false
	swapCompare(t, func(string, string) error {
		called = true
		return nil
	})
	_, err := svc.Login(context.Background(), &model.UserLoginReq{Username: "alice", Password: "right-password"}, "203.0.113.8")
	if !errors.As(err, new(*LoginLockedError)) || called {
		t.Fatalf("err=%v compared=%v", err, called)
	}
}

func TestLoginNilGuardFailsClosed(t *testing.T) {
	svc := NewUserService(&loginUserDAO{}, nil, zap.NewNop(), nil)
	_, err := svc.Login(context.Background(), &model.UserLoginReq{Username: "alice", Password: "right-password"}, "203.0.113.8")
	if !errors.Is(err, ErrLoginUnavailable) {
		t.Fatal(err)
	}
}

func TestLoginSuccessSurvivesCounterResetFailure(t *testing.T) {
	hash, err := userutils.HashPassword("right-password")
	if err != nil {
		t.Fatal(err)
	}
	guard := &memoryGuard{successErr: errors.New("redis reset failed")}
	users := &loginUserDAO{users: map[string]*model.User{
		"alice": {Username: "alice", Password: hash, Enable: 1},
	}}
	core, logs := observer.New(zapcore.DebugLevel)
	svc := NewUserService(users, nil, zap.New(core), guard)
	user, err := svc.Login(context.Background(), &model.UserLoginReq{Username: "alice", Password: "right-password"}, "203.0.113.8")
	if err != nil || user == nil || guard.okCount != 1 {
		t.Fatalf("user=%v err=%v", user, err)
	}
	assertNoSecret(t, logs, "right-password")
}

func TestAnotherIPCanLoginAfterPairLock(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	hash, err := userutils.HashPassword("right-password")
	if err != nil {
		t.Fatal(err)
	}
	users := &loginUserDAO{users: map[string]*model.User{
		"alice": {Model: model.Model{ID: 7}, Username: "alice", Password: hash, Enable: 1},
	}}
	guard := NewLoginGuard(client, config.LoginLockoutConfig{}.Normalized(), nil, zap.NewNop())
	svc := NewUserService(users, nil, zap.NewNop(), guard)
	ctx := context.Background()
	for i := 0; i < 4; i++ {
		_, err = svc.Login(ctx, &model.UserLoginReq{Username: "alice", Password: "wrong-password"}, "203.0.113.8")
		if !errors.Is(err, constants.ErrorInvalidCredentials) {
			t.Fatalf("failure %d: %v", i+1, err)
		}
	}
	_, err = svc.Login(ctx, &model.UserLoginReq{Username: "alice", Password: "wrong-password"}, "203.0.113.8")
	if !errors.As(err, new(*LoginLockedError)) {
		t.Fatalf("pair lock: %v", err)
	}
	user, err := svc.Login(ctx, &model.UserLoginReq{Username: "alice", Password: "right-password"}, "203.0.113.9")
	if err != nil || user == nil || user.ID != 7 {
		t.Fatalf("other IP login: user=%v err=%v", user, err)
	}
	_, err = svc.Login(ctx, &model.UserLoginReq{Username: "alice", Password: "right-password"}, "203.0.113.8")
	if !errors.As(err, new(*LoginLockedError)) {
		t.Fatalf("original IP should stay locked: %v", err)
	}
}

func assertNoSecret(t *testing.T, logs *observer.ObservedLogs, secret string) {
	t.Helper()
	for _, entry := range logs.All() {
		if strings.Contains(entry.Message, secret) {
			t.Fatalf("log message contains secret: %s", entry.Message)
		}
		for _, field := range entry.Context {
			if strings.Contains(field.Key, "password") || strings.Contains(field.String, secret) || strings.Contains(strings.ToLower(field.Key), "passwd") {
				t.Fatalf("log field %s contains secret material", field.Key)
			}
		}
	}
}
