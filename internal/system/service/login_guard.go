/*
 * MIT License
 *
 * Copyright (c) 2024 Bamboo
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 * THE SOFTWARE.
 *
 */

package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/config"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

const loginKeyPrefix = "cloudops:auth:login:"

// ErrLoginUnavailable 表示无法确认或写入锁定状态。
// 登录会失败关闭，避免 Redis 故障时绕过锁定；错误文本不含内部原因。
var ErrLoginUnavailable = errors.New("登录服务暂不可用，请稍后重试")

// ErrLoginLocked 表示用户名或来源 IP 处于临时锁定。
var ErrLoginLocked = errors.New("login temporarily locked")

// LoginLockedError 是返回给 API 层的锁定结果，Retry-After 用剩余时间。
type LoginLockedError struct {
	RetryAfter time.Duration
}

func (e *LoginLockedError) Error() string {
	if e == nil {
		return "登录失败次数过多，请 1 分钟后再试"
	}
	return fmt.Sprintf("登录失败次数过多，请 %d 分钟后再试", e.Minutes())
}

func (e *LoginLockedError) Unwrap() error { return ErrLoginLocked }

func (e *LoginLockedError) Minutes() int {
	minutes := (e.RetryAfterSeconds() + 59) / 60
	if minutes < 1 {
		return 1
	}
	return minutes
}

func (e *LoginLockedError) RetryAfterSeconds() int {
	if e == nil {
		return 1
	}
	return retrySeconds(e.RetryAfter)
}

// LoginGuard 按「用户名+IP」、用户名和 IP 记录登录失败并执行临时锁定。
type LoginGuard interface {
	Check(ctx context.Context, username, clientIP string) error
	RecordFailure(ctx context.Context, username, clientIP string) error
	RecordSuccess(ctx context.Context, username, clientIP string) error
}

type loginGuard struct {
	client redis.Cmdable
	policy config.LoginLockoutConfig
	audit  AuditService
	logger *zap.Logger
}

// 一次失败同时推进三组计数。每组首次写入时设置窗口；达到阈值后写入锁定键且不延长已有锁定。
// 返回 {pair, username, ip}：0 未锁定，1 本次新建锁定，2 锁定已存在。
var loginFailScript = redis.NewScript(`
local function bump(failKey, lockKey, windowMs, maxn, lockMs)
  local fails = redis.call("INCR", failKey)
  if fails == 1 then
    redis.call("PEXPIRE", failKey, windowMs)
  end
  if tonumber(fails) >= tonumber(maxn) then
    local created = redis.call("SET", lockKey, "1", "NX", "PX", lockMs)
    redis.call("DEL", failKey)
    if created then
      return 1
    end
    return 2
  end
  return 0
end
return {
  bump(KEYS[1], KEYS[2], ARGV[1], ARGV[2], ARGV[3]),
  bump(KEYS[3], KEYS[4], ARGV[4], ARGV[5], ARGV[6]),
  bump(KEYS[5], KEYS[6], ARGV[7], ARGV[8], ARGV[9])
}
`)

func NewLoginGuard(client redis.Cmdable, policy config.LoginLockoutConfig, audit AuditService, logger *zap.Logger) LoginGuard {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &loginGuard{
		client: client,
		policy: policy.Normalized(),
		audit:  audit,
		logger: logger,
	}
}

func (g *loginGuard) Check(ctx context.Context, username, clientIP string) error {
	if g == nil || g.client == nil {
		return ErrLoginUnavailable
	}
	user := canonicalUser(username)
	ip := canonicalIP(clientIP)
	retry, err := g.longestLock(ctx, user, ip)
	if err != nil {
		g.logger.Error("查询登录锁定失败", zap.String("username", clip(username, 128)), zap.String("ip", ip), zap.Error(err))
		return ErrLoginUnavailable
	}
	if retry <= 0 {
		return nil
	}
	g.logger.Warn("拒绝已锁定的登录",
		zap.String("username", clip(username, 128)),
		zap.String("ip", ip),
		zap.Int("retry_after_seconds", retrySeconds(retry)),
	)
	return &LoginLockedError{RetryAfter: retry}
}

func (g *loginGuard) RecordFailure(ctx context.Context, username, clientIP string) error {
	if g == nil || g.client == nil {
		g.logUnavailable("记录登录失败时存储不可用")
		return ErrLoginUnavailable
	}
	user := canonicalUser(username)
	ip := canonicalIP(clientIP)
	codes, err := g.bumpAll(ctx, user, ip)
	if err != nil {
		g.logger.Error("记录登录失败次数失败", zap.String("username", clip(username, 128)), zap.String("ip", ip), zap.Error(err))
		return ErrLoginUnavailable
	}
	if codes[0] == 0 && codes[1] == 0 && codes[2] == 0 {
		return nil
	}
	retry, err := g.longestLock(ctx, user, ip)
	if err != nil || retry <= 0 {
		retry = g.policy.PairLock
		if g.policy.UsernameLock > retry {
			retry = g.policy.UsernameLock
		}
		if g.policy.IPLock > retry {
			retry = g.policy.IPLock
		}
	}
	names := []string{"pair", "username", "ip"}
	var dims []string
	for i, code := range codes {
		if code == 1 {
			dims = append(dims, names[i])
		}
	}
	if len(dims) > 0 {
		g.writeLockAudit(ctx, username, ip, strings.Join(dims, ","), retry)
	}
	return &LoginLockedError{RetryAfter: retry}
}

func (g *loginGuard) RecordSuccess(ctx context.Context, username, clientIP string) error {
	if g == nil || g.client == nil {
		return ErrLoginUnavailable
	}
	user := canonicalUser(username)
	ip := canonicalIP(clientIP)
	if err := g.client.Del(ctx, pairFailKey(user, ip), userFailKey(user)).Err(); err != nil {
		return err
	}
	return nil
}

func (g *loginGuard) bumpAll(ctx context.Context, user, ip string) ([3]int64, error) {
	values, err := loginFailScript.Run(ctx, g.client, []string{
		pairFailKey(user, ip), pairLockKey(user, ip),
		userFailKey(user), userLockKey(user),
		ipFailKey(ip), ipLockKey(ip),
	},
		positiveMillis(g.policy.PairWindow), g.policy.PairMaxFailures, positiveMillis(g.policy.PairLock),
		positiveMillis(g.policy.UsernameWindow), g.policy.UsernameMaxFailures, positiveMillis(g.policy.UsernameLock),
		positiveMillis(g.policy.IPWindow), g.policy.IPMaxFailures, positiveMillis(g.policy.IPLock),
	).Slice()
	if err != nil {
		return [3]int64{}, err
	}
	if len(values) != 3 {
		return [3]int64{}, fmt.Errorf("登录计数脚本返回 %d 项", len(values))
	}
	var codes [3]int64
	for i, value := range values {
		switch n := value.(type) {
		case int64:
			codes[i] = n
		case int:
			codes[i] = int64(n)
		default:
			return [3]int64{}, fmt.Errorf("登录计数脚本返回了非整数")
		}
	}
	return codes, nil
}

func (g *loginGuard) longestLock(ctx context.Context, user, ip string) (time.Duration, error) {
	var retry time.Duration
	for _, key := range []string{pairLockKey(user, ip), userLockKey(user), ipLockKey(ip)} {
		ttl, err := g.remaining(ctx, key)
		if err != nil {
			return 0, err
		}
		if ttl > retry {
			retry = ttl
		}
	}
	return retry, nil
}

func (g *loginGuard) remaining(ctx context.Context, key string) (time.Duration, error) {
	ttl, err := g.client.TTL(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	if ttl > 0 {
		return ttl, nil
	}
	return 0, nil
}

func (g *loginGuard) writeLockAudit(ctx context.Context, username, ip, dimension string, retry time.Duration) {
	seconds := retrySeconds(retry)
	g.logger.Warn("登录已临时锁定",
		zap.String("username", clip(username, 128)),
		zap.String("ip", ip),
		zap.String("dimension", dimension),
		zap.Int("retry_after_seconds", seconds),
	)
	if g.audit == nil {
		return
	}
	g.audit.CreateAuditLogAsync(ctx, &model.CreateAuditLogRequest{
		IPAddress:     clip(ip, 45),
		HttpMethod:    http.MethodPost,
		Endpoint:      "/api/user/login",
		OperationType: "LOCK",
		TargetType:    "user",
		TargetID:      clip(username, 255),
		StatusCode:    http.StatusTooManyRequests,
		ErrorMsg:      fmt.Sprintf("登录临时锁定 dimension=%s retry_after_seconds=%d", dimension, seconds),
	})
}

func (g *loginGuard) logUnavailable(message string) {
	if g != nil && g.logger != nil {
		g.logger.Error(message)
	}
}

func canonicalUser(username string) string {
	return strings.ToLower(strings.TrimSpace(username))
}

func canonicalIP(ip string) string {
	ip = strings.TrimSpace(ip)
	if host, _, err := net.SplitHostPort(ip); err == nil {
		ip = host
	}
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return "unknown"
	}
	return parsed.String()
}

func pairMaterial(user, ip string) string { return user + "\x00" + ip }

func pairFailKey(user, ip string) string {
	return loginKeyPrefix + "fail:pair:" + digest(pairMaterial(user, ip))
}
func pairLockKey(user, ip string) string {
	return loginKeyPrefix + "lock:pair:" + digest(pairMaterial(user, ip))
}
func userFailKey(user string) string { return loginKeyPrefix + "fail:user:" + digest(user) }
func userLockKey(user string) string { return loginKeyPrefix + "lock:user:" + digest(user) }
func ipFailKey(ip string) string     { return loginKeyPrefix + "fail:ip:" + digest(ip) }
func ipLockKey(ip string) string     { return loginKeyPrefix + "lock:ip:" + digest(ip) }

func digest(part string) string {
	sum := sha256.Sum256([]byte(part))
	return hex.EncodeToString(sum[:])
}

func positiveMillis(d time.Duration) int64 {
	if d < time.Second {
		d = time.Second
	}
	return d.Milliseconds()
}

func retrySeconds(d time.Duration) int {
	if d <= 0 {
		return 1
	}
	seconds := int((d + time.Second - 1) / time.Second)
	if seconds < 1 {
		return 1
	}
	return seconds
}

func clip(value string, max int) string {
	value = strings.TrimSpace(strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' {
			return -1
		}
		return r
	}, value))
	runes := []rune(value)
	if max > 0 && len(runes) > max {
		return string(runes[:max])
	}
	return value
}
