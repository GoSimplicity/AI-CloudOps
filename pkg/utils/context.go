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
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// UserInfo 用户信息结构体
type UserInfo struct {
	UserID    int    `json:"userId"`
	Username  string `json:"username"`
	IP        string `json:"ip"`
	UserAgent string `json:"userAgent"`
}

// GetUserInfoFromGinContext 从gin.Context中获取用户信息
func GetUserInfoFromGinContext(c *gin.Context) *UserInfo {
	userInfo := &UserInfo{
		IP:        c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
	}

	// 尝试从gin.Context中获取用户信息
	if user, exists := c.Get("user"); exists {
		switch claims := user.(type) {
		case UserClaims:
			userInfo.UserID = claims.Uid
			userInfo.Username = claims.Username
		case map[string]interface{}:
			if uid, ok := claims["uid"].(float64); ok {
				userInfo.UserID = int(uid)
			}
			if username, ok := claims["username"].(string); ok {
				userInfo.Username = username
			}
		}
	}

	// fallback: 尝试从其他可能的key获取
	if userInfo.UserID == 0 {
		if userID, exists := c.Get("user_id"); exists {
			if uid, ok := userID.(int); ok {
				userInfo.UserID = uid
			}
		}
	}

	if userInfo.Username == "" {
		if username, exists := c.Get("username"); exists {
			if name, ok := username.(string); ok {
				userInfo.Username = name
			}
		}
	}

	return userInfo
}

// GetUserInfoFromHTTPRequest 从http.Request中获取用户信息
func GetUserInfoFromHTTPRequest(r *http.Request) *UserInfo {
	userInfo := &UserInfo{
		IP:        GetClientIP(r),
		UserAgent: r.UserAgent(),
	}

	// 从请求头中获取用户信息
	if userIDStr := r.Header.Get("X-User-ID"); userIDStr != "" {
		if userID, err := strconv.Atoi(userIDStr); err == nil {
			userInfo.UserID = userID
		}
	}

	if username := r.Header.Get("X-Username"); username != "" {
		userInfo.Username = username
	}

	// 尝试从URL参数获取
	if userInfo.UserID == 0 {
		if userIDStr := r.URL.Query().Get("user_id"); userIDStr != "" {
			if userID, err := strconv.Atoi(userIDStr); err == nil {
				userInfo.UserID = userID
			}
		}
	}

	return userInfo
}

// GetClientIP 获取客户端真实IP地址
func GetClientIP(r *http.Request) string {
	// 按优先级顺序检查各种头部
	ipHeaders := []string{
		"CF-Connecting-IP", // Cloudflare
		"X-Forwarded-For",  // 标准代理头
		"X-Real-IP",        // Nginx代理
		"X-Client-IP",      // Apache代理
		"X-Forwarded",      // 旧格式
		"Forwarded-For",    // 旧格式
		"Forwarded",        // RFC 7239
	}

	for _, header := range ipHeaders {
		if ip := extractValidIP(r.Header.Get(header)); ip != "" {
			return ip
		}
	}

	// 使用RemoteAddr作为后备方案
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		if ip := net.ParseIP(host); ip != nil {
			return host
		}
	}

	return "unknown"
}

// extractValidIP 从头部值中提取有效IP
func extractValidIP(headerValue string) string {
	if headerValue == "" {
		return ""
	}

	// 处理多个IP的情况（逗号分隔）
	ips := strings.Split(headerValue, ",")
	for _, ip := range ips {
		ip = strings.TrimSpace(ip)
		if isValidIP(ip) {
			return ip
		}
	}

	return ""
}

// isValidIP 检查IP是否有效
func isValidIP(ip string) bool {
	if ip == "" || ip == "unknown" || ip == "127.0.0.1" || ip == "::1" {
		return false
	}

	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}

	// 排除内网IP（可选，根据需求调整）
	if parsedIP.IsLoopback() || parsedIP.IsPrivate() {
		return false
	}

	return true
}

// GetUserInfoFromContext 从context.Context中获取用户信息
func GetUserInfoFromContext(ctx context.Context) *UserInfo {
	userInfo := &UserInfo{}

	// 支持多种context key格式
	contextKeys := map[string]interface{}{
		"user_id":    &userInfo.UserID,
		"userId":     &userInfo.UserID,
		"uid":        &userInfo.UserID,
		"username":   &userInfo.Username,
		"user_name":  &userInfo.Username,
		"client_ip":  &userInfo.IP,
		"ip":         &userInfo.IP,
		"user_agent": &userInfo.UserAgent,
		"userAgent":  &userInfo.UserAgent,
	}

	for key, target := range contextKeys {
		if value := ctx.Value(key); value != nil {
			switch ptr := target.(type) {
			case *int:
				if intVal, ok := value.(int); ok {
					*ptr = intVal
				} else if strVal, ok := value.(string); ok {
					if intVal, err := strconv.Atoi(strVal); err == nil {
						*ptr = intVal
					}
				}
			case *string:
				if strVal, ok := value.(string); ok {
					*ptr = strVal
				}
			}
		}
	}

	return userInfo
}

// SetUserInfoToContext 将用户信息设置到context中
func SetUserInfoToContext(ctx context.Context, userInfo *UserInfo) context.Context {
	if userInfo == nil {
		return ctx
	}

	ctx = context.WithValue(ctx, "user_id", userInfo.UserID)
	ctx = context.WithValue(ctx, "username", userInfo.Username)
	ctx = context.WithValue(ctx, "client_ip", userInfo.IP)
	ctx = context.WithValue(ctx, "user_agent", userInfo.UserAgent)

	return ctx
}

// SetUserInfoToGinContext 将用户信息设置到gin.Context中
func SetUserInfoToGinContext(c *gin.Context, userInfo *UserInfo) {
	if userInfo == nil {
		return
	}

	c.Set("user_id", userInfo.UserID)
	c.Set("username", userInfo.Username)
	c.Set("client_ip", userInfo.IP)
	c.Set("user_agent", userInfo.UserAgent)
}

// IsEmpty 检查用户信息是否为空
func (u *UserInfo) IsEmpty() bool {
	return u == nil || (u.UserID == 0 && u.Username == "")
}

// IsValid 检查用户信息是否有效
func (u *UserInfo) IsValid() bool {
	return u != nil && (u.UserID > 0 || u.Username != "")
}

// String 返回用户信息的字符串表示
func (u *UserInfo) String() string {
	if u == nil {
		return "UserInfo(nil)"
	}
	return fmt.Sprintf("UserInfo(ID:%d, Username:%s, IP:%s)", u.UserID, u.Username, u.IP)
}

// Clone 创建用户信息的副本
func (u *UserInfo) Clone() *UserInfo {
	if u == nil {
		return nil
	}

	return &UserInfo{
		UserID:    u.UserID,
		Username:  u.Username,
		IP:        u.IP,
		UserAgent: u.UserAgent,
	}
}
