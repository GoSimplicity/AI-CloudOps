package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/config"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type recordingAudit struct {
	AuditService
	reqs []*model.CreateAuditLogRequest
}

func (a *recordingAudit) CreateAuditLogAsync(_ context.Context, req *model.CreateAuditLogRequest) {
	a.reqs = append(a.reqs, req)
}

func testGuard(t *testing.T, policy config.LoginLockoutConfig) (*miniredis.Miniredis, LoginGuard, *recordingAudit) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	audit := &recordingAudit{}
	return mr, NewLoginGuard(client, policy, audit, zap.NewNop()), audit
}

func testPolicy() config.LoginLockoutConfig {
	return config.LoginLockoutConfig{
		PairMaxFailures:     5,
		PairWindow:          15 * time.Minute,
		PairLock:            15 * time.Minute,
		UsernameMaxFailures: 50,
		UsernameWindow:      15 * time.Minute,
		UsernameLock:        15 * time.Minute,
		IPMaxFailures:       20,
		IPWindow:            15 * time.Minute,
		IPLock:              15 * time.Minute,
	}
}

func TestLoginGuardCountsLocksAndUnlocks(t *testing.T) {
	mr, guard, audit := testGuard(t, testPolicy())
	ctx := context.Background()
	const secret = "s3cret-login-value"

	for i := 0; i < 4; i++ {
		if err := guard.RecordFailure(ctx, "Alice", "203.0.113.8"); err != nil {
			t.Fatalf("failure %d: %v", i+1, err)
		}
	}
	if err := guard.Check(ctx, "alice", "203.0.113.8"); err != nil {
		t.Fatalf("pair locked too early: %v", err)
	}
	err := guard.RecordFailure(ctx, "alice", "203.0.113.8")
	var locked *LoginLockedError
	if !errors.As(err, &locked) || locked.Minutes() != 15 {
		t.Fatalf("fifth failure = %v", err)
	}
	if len(audit.reqs) != 1 {
		t.Fatalf("lock audit count=%d", len(audit.reqs))
	}
	body := audit.reqs[0]
	if body.StatusCode != 429 || body.OperationType != "LOCK" || body.Endpoint != "/api/user/login" {
		t.Fatalf("audit=%+v", body)
	}
	if len(body.RequestBody) != 0 || strings.Contains(body.ErrorMsg, secret) || strings.Contains(body.TargetID, secret) {
		t.Fatalf("audit kept a secret or request body: %+v", body)
	}
	if !strings.Contains(body.ErrorMsg, "dimension=pair") {
		t.Fatalf("audit reason=%s", body.ErrorMsg)
	}
	if err = guard.Check(ctx, "ALICE", "198.51.100.10"); err != nil {
		t.Fatalf("another IP should still be allowed: %v", err)
	}
	if _, err = mr.Get(pairFailKey(canonicalUser("alice"), canonicalIP("203.0.113.8"))); err == nil {
		t.Fatal("pair failure counter should be removed once locked")
	}

	mr.FastForward(15*time.Minute + time.Second)
	if err = guard.Check(ctx, "alice", "203.0.113.8"); err != nil {
		t.Fatalf("lock did not expire: %v", err)
	}
}

func TestLoginGuardUsernameLocksAfterWideThreshold(t *testing.T) {
	policy := testPolicy()
	policy.PairMaxFailures = 1000
	policy.IPMaxFailures = 1000
	policy.UsernameMaxFailures = 50
	_, guard, audit := testGuard(t, policy)
	ctx := context.Background()
	for i := 0; i < 49; i++ {
		if err := guard.RecordFailure(ctx, "alice", "203.0.113.8"); err != nil {
			t.Fatalf("failure %d: %v", i+1, err)
		}
	}
	if err := guard.Check(ctx, "alice", "198.51.100.20"); err != nil {
		t.Fatalf("username locked before 50: %v", err)
	}
	err := guard.RecordFailure(ctx, "alice", "203.0.113.8")
	var locked *LoginLockedError
	if !errors.As(err, &locked) {
		t.Fatalf("50th failure = %v", err)
	}
	if err = guard.Check(ctx, "alice", "198.51.100.21"); !errors.As(err, &locked) {
		t.Fatalf("username lock did not apply to another IP: %v", err)
	}
	if len(audit.reqs) != 1 || !strings.Contains(audit.reqs[0].ErrorMsg, "dimension=username") {
		t.Fatalf("audit=%v", audit.reqs)
	}
}

func TestLoginGuardSuccessClearsUsernameCounterOnly(t *testing.T) {
	_, guard, audit := testGuard(t, testPolicy())
	ctx := context.Background()
	for i := 0; i < 4; i++ {
		if err := guard.RecordFailure(ctx, "bob", "203.0.113.20"); err != nil {
			t.Fatal(err)
		}
	}
	if err := guard.RecordSuccess(ctx, "Bob", "203.0.113.20"); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 4; i++ {
		if err := guard.RecordFailure(ctx, "bob", "203.0.113.20"); err != nil {
			t.Fatalf("post-success failure %d: %v", i+1, err)
		}
	}
	if err := guard.Check(ctx, "bob", "203.0.113.20"); err != nil {
		t.Fatalf("username counter was not cleared: %v", err)
	}
	if err := guard.RecordFailure(ctx, "bob", "203.0.113.20"); !errors.As(err, new(*LoginLockedError)) {
		t.Fatalf("fifth failure after reset = %v", err)
	}
	if len(audit.reqs) != 1 {
		t.Fatalf("audit count=%d", len(audit.reqs))
	}
}

func TestLoginGuardIPLockSurvivesUsernameSuccess(t *testing.T) {
	policy := testPolicy()
	policy.IPMaxFailures = 3
	_, guard, audit := testGuard(t, policy)
	ctx := context.Background()
	for _, name := range []string{"u1", "u2"} {
		if err := guard.RecordFailure(ctx, name, "203.0.113.30"); err != nil {
			t.Fatal(err)
		}
	}
	if err := guard.RecordSuccess(ctx, "u1", "203.0.113.30"); err != nil {
		t.Fatal(err)
	}
	err := guard.RecordFailure(ctx, "u3", "203.0.113.30")
	var locked *LoginLockedError
	if !errors.As(err, &locked) {
		t.Fatalf("ip lock = %v", err)
	}
	if len(audit.reqs) != 1 || !strings.Contains(audit.reqs[0].ErrorMsg, "dimension=ip") {
		t.Fatalf("audit=%v", audit.reqs)
	}
	if err = guard.Check(ctx, "fresh-user", "203.0.113.30"); !errors.As(err, &locked) {
		t.Fatalf("ip still locked for a new username: %v", err)
	}
	if err = guard.Check(ctx, "u3", "198.51.100.30"); err != nil {
		t.Fatalf("another IP should not inherit the lock: %v", err)
	}
}

func TestLoginGuardRedisUnavailableFailsClosed(t *testing.T) {
	mr, guard, _ := testGuard(t, testPolicy())
	addr := mr.Addr()
	mr.Close()
	err := guard.Check(context.Background(), "alice", "203.0.113.8")
	if !errors.Is(err, ErrLoginUnavailable) || err.Error() != "登录服务暂不可用，请稍后重试" {
		t.Fatalf("check: %v", err)
	}
	if strings.Contains(err.Error(), "s3cret-login-value") || strings.Contains(err.Error(), addr) {
		t.Fatalf("unavailable error leaked internals: %v", err)
	}
	err = guard.RecordFailure(context.Background(), "alice", "203.0.113.8")
	if !errors.Is(err, ErrLoginUnavailable) {
		t.Fatalf("record: %v", err)
	}
}

func TestLoginLockedMessageRoundsUp(t *testing.T) {
	err := &LoginLockedError{RetryAfter: 90 * time.Second}
	if err.Error() != "登录失败次数过多，请 2 分钟后再试" || err.RetryAfterSeconds() != 90 {
		t.Fatal(err.Error(), err.RetryAfterSeconds())
	}
	if !errors.Is(err, ErrLoginLocked) {
		t.Fatal("locked sentinel")
	}
}
