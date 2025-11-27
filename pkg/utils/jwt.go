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

package utils

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	bearerPrefix                = "Bearer "
	authorizationHeaderKey      = "Authorization"
	defaultJWTExpirationMinutes = 30
	sessionKeyPattern           = "cloudops:user:ssid:%s"
	tokenBlacklistKeyPattern    = "cloudops:blacklist:token:%s"
)

type Handler interface {
	SetLoginToken(ctx *gin.Context, uid int, username string, accountType int8) (string, string, error)
	SetJWTToken(ctx *gin.Context, uid int, username string, ssid string, accountType int8) (string, error)
	ExtractToken(ctx *gin.Context) string
	CheckSession(ctx *gin.Context, ssid string) error
	ClearToken(ctx *gin.Context) error
	setRefreshToken(ctx *gin.Context, uid int, username string, ssid string, accountType int8) (string, error)
}

type UserClaims struct {
	jwt.RegisteredClaims
	Uid         int
	Username    string
	Ssid        string
	UserAgent   string
	ContentType string
	AccountType int8
}

type RefreshClaims struct {
	jwt.RegisteredClaims
	Uid         int
	Username    string
	Ssid        string
	AccountType int8
}

type handler struct {
	client        redis.Cmdable
	signingMethod jwt.SigningMethod
	jwtExpiration time.Duration
	rcExpiration  time.Duration
	key1          []byte
	key2          []byte
	issuer        string
}

func NewJWTHandler(c redis.Cmdable) Handler {
	key1 := viper.GetString("jwt.key1")
	key2 := viper.GetString("jwt.key2")
	issuer := viper.GetString("jwt.issuer")
	expirationMinutes := viper.GetInt64("jwt.expiration")
	if expirationMinutes <= 0 {
		expirationMinutes = defaultJWTExpirationMinutes
	}

	return &handler{
		client:        c,
		signingMethod: jwt.SigningMethodHS512,
		jwtExpiration: time.Minute * time.Duration(expirationMinutes),
		rcExpiration:  time.Hour * 24 * 7,
		key1:          []byte(key1),
		key2:          []byte(key2),
		issuer:        issuer,
	}
}

// SetLoginToken 设置长短Token
func (h *handler) SetLoginToken(ctx *gin.Context, uid int, username string, accountType int8) (string, string, error) {
	ssid := uuid.New().String()
	refreshToken, err := h.setRefreshToken(ctx, uid, username, ssid, accountType)
	if err != nil {
		return "", "", err
	}

	jwtToken, err := h.SetJWTToken(ctx, uid, username, ssid, accountType)

	if err != nil {
		return "", "", err
	}

	return jwtToken, refreshToken, nil
}

// SetJWTToken 设置短Token
func (h *handler) SetJWTToken(ctx *gin.Context, uid int, username string, ssid string, accountType int8) (string, error) {
	uc := UserClaims{
		Uid:         uid,
		Username:    username,
		Ssid:        ssid,
		UserAgent:   ctx.GetHeader("User-Agent"),
		ContentType: ctx.GetHeader("Content-Type"),
		AccountType: accountType,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(h.jwtExpiration)),
			Issuer:    h.issuer,
		},
	}

	return h.signClaims(uc, h.key1)
}

// setRefreshToken 设置长Token
func (h *handler) setRefreshToken(_ *gin.Context, uid int, username string, ssid string, accountType int8) (string, error) {
	rc := RefreshClaims{
		Uid:         uid,
		Username:    username,
		Ssid:        ssid,
		AccountType: accountType,
		RegisteredClaims: jwt.RegisteredClaims{
			// 设置刷新时间为一周
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(h.rcExpiration)),
		},
	}

	return h.signClaims(rc, h.key2)
}

// ExtractToken 提取 Authorization 头部中的 Token
func (h *handler) ExtractToken(ctx *gin.Context) string {
	token, err := h.extractBearerToken(ctx)
	if err != nil {
		return ""
	}
	return token
}

// CheckSession 检查会话状态
func (h *handler) CheckSession(ctx *gin.Context, ssid string) error {
	// 判断缓存中是否存在指定键
	c, err := h.client.Exists(ctx, fmt.Sprintf(sessionKeyPattern, ssid)).Result()
	if err != nil {
		return err
	}

	if c != 0 {
		return errors.New("token失效")
	}

	return nil
}

// ClearToken 清空 token，让 Authorization 中的用于验证的 token 失效
func (h *handler) ClearToken(ctx *gin.Context) error {
	// 获取 Authorization 头部中的 token
	authToken, err := h.extractBearerToken(ctx)
	if err != nil {
		return err
	}

	// 提取 token 的 claims 信息
	claims := &UserClaims{}
	token, err := jwt.ParseWithClaims(authToken, claims, func(token *jwt.Token) (interface{}, error) {
		return h.key1, nil
	})

	if err != nil || !token.Valid {
		return errors.New("invalid authorization token")
	}

	// 将 token 加入 Redis 黑名单
	if err := h.addToBlacklist(ctx, authToken, claims.ExpiresAt.Time); err != nil {
		return err
	}

	return nil
}

// 提取 Bearer Token
func (h *handler) extractBearerToken(ctx *gin.Context) (string, error) {
	authHeader := strings.TrimSpace(ctx.GetHeader(authorizationHeaderKey))
	if authHeader == "" {
		return "", errors.New("missing authorization token")
	}

	if !strings.HasPrefix(authHeader, bearerPrefix) {
		return "", errors.New("invalid authorization token format")
	}

	token := strings.TrimSpace(authHeader[len(bearerPrefix):])
	if token == "" {
		return "", errors.New("authorization token is empty")
	}
	return token, nil
}

// 将 token 加入 Redis 黑名单
func (h *handler) addToBlacklist(ctx *gin.Context, authToken string, expiresAt time.Time) error {
	remainingTime := time.Until(expiresAt)
	if remainingTime <= 0 {
		// 保证Redis中至少缓存一个极短的过期时间，避免永久存留
		remainingTime = time.Second
	}

	blacklistKey := fmt.Sprintf(tokenBlacklistKeyPattern, authToken)

	// 将 token 存入 Redis，并设置过期时间
	if err := h.client.Set(ctx, blacklistKey, "invalid", remainingTime).Err(); err != nil {
		return err
	}
	return nil
}

func (h *handler) signClaims(claims jwt.Claims, key []byte) (string, error) {
	t := jwt.NewWithClaims(h.signingMethod, claims)
	return t.SignedString(key)
}
