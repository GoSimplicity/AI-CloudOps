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
	"net/http"

	"github.com/gin-gonic/gin"
)

// UserInfo 用户信息结构体
type UserInfo struct {
	UserID    int    `json:"userId"`
	Username  string `json:"username"`
	IP        string `json:"ip"`
	UserAgent string `json:"userAgent"`
}

// GetUserInfoFromContext 从gin.Context中获取用户信息
func GetUserInfoFromContext(c *gin.Context) *UserInfo {
	userInfo := &UserInfo{
		IP:        c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
	}

	// 尝试从gin.Context中获取用户信息
	if user, exists := c.Get("user"); exists {
		if claims, ok := user.(UserClaims); ok {
			userInfo.UserID = claims.Uid
			userInfo.Username = claims.Username
		}
	}

	return userInfo
}

// GetUserInfoFromHTTPContext 从http.Request中获取用户信息
func GetUserInfoFromHTTPContext(r *http.Request) *UserInfo {
	userInfo := &UserInfo{
		IP:        GetClientIP(r),
		UserAgent: r.UserAgent(),
	}

	// 从请求头中获取用户信息（如果有的话）
	if userID := r.Header.Get("X-User-ID"); userID != "" {
		// 这里可以解析用户ID，暂时保持为空
	}
	if username := r.Header.Get("X-Username"); username != "" {
		userInfo.Username = username
	}

	return userInfo
}

// GetClientIP 获取客户端IP地址
func GetClientIP(r *http.Request) string {
	// 尝试从各种头部获取真实IP
	headers := []string{
		"X-Forwarded-For",
		"X-Real-IP",
		"X-Client-IP",
		"CF-Connecting-IP", // Cloudflare
		"X-Forwarded",
		"Forwarded-For",
		"Forwarded",
	}

	for _, header := range headers {
		if ip := r.Header.Get(header); ip != "" {
			// 如果是逗号分隔的多个IP，取第一个
			if idx := indexOf(ip, ','); idx != -1 {
				ip = ip[:idx]
			}
			// 去除空格
			ip = trimSpace(ip)
			if ip != "" && ip != "unknown" {
				return ip
			}
		}
	}

	// 如果头部中没有，使用RemoteAddr
	if r.RemoteAddr != "" {
		// 去除端口号
		if idx := indexOf(r.RemoteAddr, ':'); idx != -1 {
			return r.RemoteAddr[:idx]
		}
		return r.RemoteAddr
	}

	return "unknown"
}

// GetUserInfoFromContext 从context.Context中获取用户信息（通用版本）
func GetUserInfoFromContextGeneric(ctx context.Context) *UserInfo {
	userInfo := &UserInfo{}

	// 尝试从context中获取用户信息
	if userID, ok := ctx.Value("user_id").(int); ok {
		userInfo.UserID = userID
	}
	if username, ok := ctx.Value("username").(string); ok {
		userInfo.Username = username
	}
	if ip, ok := ctx.Value("client_ip").(string); ok {
		userInfo.IP = ip
	}
	if userAgent, ok := ctx.Value("user_agent").(string); ok {
		userInfo.UserAgent = userAgent
	}

	return userInfo
}

// SetUserInfoToContext 将用户信息设置到context中
func SetUserInfoToContext(ctx context.Context, userInfo *UserInfo) context.Context {
	ctx = context.WithValue(ctx, "user_id", userInfo.UserID)
	ctx = context.WithValue(ctx, "username", userInfo.Username)
	ctx = context.WithValue(ctx, "client_ip", userInfo.IP)
	ctx = context.WithValue(ctx, "user_agent", userInfo.UserAgent)
	return ctx
}

// 辅助函数
func indexOf(s string, sep byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == sep {
			return i
		}
	}
	return -1
}

func trimSpace(s string) string {
	start := 0
	end := len(s)

	// 去除前导空格
	for start < end && s[start] == ' ' {
		start++
	}

	// 去除尾随空格
	for end > start && s[end-1] == ' ' {
		end--
	}

	return s[start:end]
}
