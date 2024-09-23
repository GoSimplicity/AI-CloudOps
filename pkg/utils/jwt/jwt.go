package jwt

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

type Handler interface {
	SetLoginToken(ctx *gin.Context, uid int) (string, string, error)
	SetJWTToken(ctx *gin.Context, uid int, ssid string) (string, error)
	ExtractToken(ctx *gin.Context) string
	CheckSession(ctx *gin.Context, ssid string) error
	ClearToken(ctx *gin.Context) error
	setRefreshToken(ctx *gin.Context, uid int, ssid string) (string, error)
}

type UserClaims struct {
	jwt.RegisteredClaims
	Uid         int
	Ssid        string
	UserAgent   string
	ContentType string
}

type RefreshClaims struct {
	jwt.RegisteredClaims
	Uid  int
	Ssid string
}

type handler struct {
	client        redis.Cmdable
	signingMethod jwt.SigningMethod
	rcExpiration  time.Duration
	key1          []byte
	key2          []byte
	issuer        string
}

func NewJWTHandler(c redis.Cmdable) Handler {
	key1 := viper.GetString("jwt.key1")
	key2 := viper.GetString("jwt.key2")
	issuer := viper.GetString("jwt.issuer")

	return &handler{
		client:        c,
		signingMethod: jwt.SigningMethodHS512,
		rcExpiration:  time.Hour * 24 * 7,
		key1:          []byte(key1),
		key2:          []byte(key2),
		issuer:        issuer,
	}
}

// SetLoginToken 设置长短Token
func (h *handler) SetLoginToken(ctx *gin.Context, uid int) (string, string, error) {
	ssid := uuid.New().String()
	refreshToken, err := h.setRefreshToken(ctx, uid, ssid)
	if err != nil {
		return "", "", err
	}

	jwtToken, err := h.SetJWTToken(ctx, uid, ssid)

	if err != nil {
		return "", "", err
	}

	return jwtToken, refreshToken, nil
}

// SetJWTToken 设置短Token
func (h *handler) SetJWTToken(ctx *gin.Context, uid int, ssid string) (string, error) {
	// 从配置文件中获取JWT的过期时间
	expirationMinutes := viper.GetInt64("jwt.expiration")

	// 如果未设置或值无效，设置一个默认值，例如30分钟
	if expirationMinutes <= 0 {
		expirationMinutes = 30
	}

	// 构建用户声明信息
	uc := UserClaims{
		Uid:         uid,
		Ssid:        ssid,
		UserAgent:   ctx.GetHeader("User-Agent"),
		ContentType: ctx.GetHeader("Content-Type"),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute * time.Duration(expirationMinutes))),
			Issuer:    h.issuer,
		},
	}

	token := jwt.NewWithClaims(h.signingMethod, uc)
	// 进行签名
	signedString, err := token.SignedString(h.key1)
	if err != nil {
		return "", err
	}

	return signedString, nil
}

// setRefreshToken 设置长Token
func (h *handler) setRefreshToken(_ *gin.Context, uid int, ssid string) (string, error) {
	rc := RefreshClaims{
		Uid:  uid,
		Ssid: ssid,
		RegisteredClaims: jwt.RegisteredClaims{
			// 设置刷新时间为一周
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(h.rcExpiration)),
		},
	}

	t := jwt.NewWithClaims(h.signingMethod, rc)
	signedString, err := t.SignedString(h.key2)
	if err != nil {
		return "", err
	}

	return signedString, nil
}

// ExtractToken 提取 Authorization 头部中的 Token
func (h *handler) ExtractToken(ctx *gin.Context) string {
	authCode := ctx.GetHeader("Authorization")
	if authCode == "" {
		return ""
	}

	// Authorization 头部格式需为 Bearer string
	s := strings.Split(authCode, " ")

	if len(s) != 2 {
		return ""
	}

	return s[1]
}

// CheckSession 检查会话状态
func (h *handler) CheckSession(ctx *gin.Context, ssid string) error {
	// 判断缓存中是否存在指定键
	c, err := h.client.Exists(ctx, fmt.Sprintf("linkme:user:ssid:%s", ssid)).Result()
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
	authHeader := ctx.GetHeader("Authorization")
	if authHeader == "" {
		return "", errors.New("missing authorization token")
	}

	const bearerPrefix = "Bearer "
	if len(authHeader) <= len(bearerPrefix) || authHeader[:len(bearerPrefix)] != bearerPrefix {
		return "", errors.New("invalid authorization token format")
	}

	return authHeader[len(bearerPrefix):], nil
}

// 将 token 加入 Redis 黑名单
func (h *handler) addToBlacklist(ctx *gin.Context, authToken string, expiresAt time.Time) error {
	remainingTime := time.Until(expiresAt)
	blacklistKey := fmt.Sprintf("blacklist:token:%s", authToken)

	// 将 token 存入 Redis，并设置过期时间
	if err := h.client.Set(ctx, blacklistKey, "invalid", remainingTime).Err(); err != nil {
		return err
	}
	return nil
}
