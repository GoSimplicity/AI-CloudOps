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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetUserInfoFromContextGeneric(t *testing.T) {
	tests := []struct {
		name     string
		ctx      context.Context
		expected *UserInfo
	}{
		{
			name: "完整用户信息",
			ctx: context.WithValue(context.WithValue(context.WithValue(context.WithValue(
				context.Background(),
				"user_id", 123,
			), "username", "testuser"), "client_ip", "192.168.1.1"), "user_agent", "Mozilla/5.0"),
			expected: &UserInfo{
				UserID:    123,
				Username:  "testuser",
				IP:        "192.168.1.1",
				UserAgent: "Mozilla/5.0",
			},
		},
		{
			name:     "空上下文",
			ctx:      context.Background(),
			expected: &UserInfo{},
		},
		{
			name: "部分用户信息",
			ctx: context.WithValue(context.WithValue(
				context.Background(),
				"user_id", 456,
			), "username", "partialuser"),
			expected: &UserInfo{
				UserID:   456,
				Username: "partialuser",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetUserInfoFromContextGeneric(tt.ctx)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSetUserInfoToContext(t *testing.T) {
	userInfo := &UserInfo{
		UserID:    789,
		Username:  "testuser",
		IP:        "10.0.0.1",
		UserAgent: "TestAgent/1.0",
	}

	ctx := SetUserInfoToContext(context.Background(), userInfo)

	// 验证设置的值
	assert.Equal(t, 789, ctx.Value("user_id"))
	assert.Equal(t, "testuser", ctx.Value("username"))
	assert.Equal(t, "10.0.0.1", ctx.Value("client_ip"))
	assert.Equal(t, "TestAgent/1.0", ctx.Value("user_agent"))
}

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name       string
		headers    map[string]string
		remoteAddr string
		expected   string
	}{
		{
			name: "X-Forwarded-For",
			headers: map[string]string{
				"X-Forwarded-For": "203.0.113.1",
			},
			expected: "203.0.113.1",
		},
		{
			name: "X-Real-IP",
			headers: map[string]string{
				"X-Real-IP": "198.51.100.1",
			},
			expected: "198.51.100.1",
		},
		{
			name: "CF-Connecting-IP",
			headers: map[string]string{
				"CF-Connecting-IP": "192.0.2.1",
			},
			expected: "192.0.2.1",
		},
		{
			name: "多个IP取第一个",
			headers: map[string]string{
				"X-Forwarded-For": "203.0.113.1, 10.0.0.1",
			},
			expected: "203.0.113.1",
		},
		{
			name: "带空格的IP",
			headers: map[string]string{
				"X-Forwarded-For": "  203.0.113.1  ",
			},
			expected: "203.0.113.1",
		},
		{
			name:       "RemoteAddr",
			headers:    map[string]string{},
			remoteAddr: "172.16.0.1:12345",
			expected:   "172.16.0.1",
		},
		{
			name:     "unknown",
			headers:  map[string]string{},
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/", nil)

			// 设置请求头
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			// 设置RemoteAddr
			if tt.remoteAddr != "" {
				req.RemoteAddr = tt.remoteAddr
			}

			result := GetClientIP(req)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetUserInfoFromHTTPContext(t *testing.T) {
	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Set("X-User-ID", "123")
	req.Header.Set("X-Username", "testuser")
	req.Header.Set("User-Agent", "TestAgent/1.0")
	req.RemoteAddr = "192.168.1.1:8080"

	userInfo := GetUserInfoFromHTTPContext(req)

	assert.Equal(t, "testuser", userInfo.Username)
	assert.Equal(t, "TestAgent/1.0", userInfo.UserAgent)
	assert.Equal(t, "192.168.1.1", userInfo.IP)
}

func TestIndexOf(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		sep      byte
		expected int
	}{
		{"找到字符", "hello,world", ',', 5},
		{"找不到字符", "hello", ',', -1},
		{"空字符串", "", ',', -1},
		{"第一个字符", "hello", 'h', 0},
		{"最后一个字符", "hello", 'o', 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := indexOf(tt.s, tt.sep)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTrimSpace(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"前导空格", "  hello", "hello"},
		{"尾随空格", "hello  ", "hello"},
		{"前后空格", "  hello  ", "hello"},
		{"无空格", "hello", "hello"},
		{"只有空格", "   ", ""},
		{"空字符串", "", ""},
		{"中间空格", "hello world", "hello world"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := trimSpace(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
