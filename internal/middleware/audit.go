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

package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/system/service"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const (
	OperationCreate = "create"
	OperationUpdate = "update"
	OperationDelete = "delete"
	OperationQuery  = "query"
	Unknown         = "unknown"
)

var operationTypeMap = map[string]string{
	"POST":   OperationCreate,
	"PUT":    OperationUpdate,
	"PATCH":  OperationUpdate,
	"DELETE": OperationDelete,
	"GET":    OperationQuery,
}

var bufferPool = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w bodyLogWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

type AuditLogMiddleware struct {
	auditSvc service.AuditService
	l        *zap.Logger
}

func NewAuditLogMiddleware(auditSvc service.AuditService, l *zap.Logger) *AuditLogMiddleware {
	return &AuditLogMiddleware{
		auditSvc: auditSvc,
		l:        l,
	}
}

func (m *AuditLogMiddleware) AuditLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 跳过登录接口的审计
		if c.Request.URL.Path == "/api/user/login" ||
			c.Request.URL.Path == "/api/user/logout" ||
			c.Request.URL.Path == "/api/user/refresh_token" ||
			c.Request.URL.Path == "/api/user/signup" ||
			c.Request.URL.Path == "/api/not_auth/getTreeNodeBindIps" ||
			c.Request.URL.Path == "/api/monitor/prometheus_configs/prometheus" ||
			c.Request.URL.Path == "/api/monitor/prometheus_configs/prometheus_alert" ||
			c.Request.URL.Path == "/api/monitor/prometheus_configs/prometheus_record" ||
			c.Request.URL.Path == "/api/monitor/prometheus_configs/alertManager" {
			c.Next()
			return
		}

		var requestBody []byte

		if c.Request.Body != nil {
			// 使用bufferPool复用buffer
			buf := bufferPool.Get().(*bytes.Buffer)
			buf.Reset()
			defer bufferPool.Put(buf)

			// 限制读取大小,避免内存溢出
			if _, err := io.CopyN(buf, c.Request.Body, 1024*1024); err != nil && err != io.EOF {
				m.l.Error("读取请求体失败", zap.Error(err))
			}
			requestBody = buf.Bytes()
			c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
		}

		// 包装ResponseWriter以捕获响应体
		blw := &bodyLogWriter{
			body:           bufferPool.Get().(*bytes.Buffer),
			ResponseWriter: c.Writer,
		}
		blw.body.Reset()
		defer bufferPool.Put(blw.body)

		c.Writer = blw

		startTime := time.Now()
		c.Next()

		// 获取用户id,如果不存在则使用0表示未登录用户
		var userID uint
		if user, exists := c.Get("user"); exists {
			if uc, ok := user.(*utils.UserClaims); ok {
				userID = uint(uc.Uid)
			}
		}

		// 构建基础日志
		auditLog := &model.AuditLog{
			UserID:        userID,
			IPAddress:     c.ClientIP(),
			UserAgent:     c.Request.UserAgent(),
			HttpMethod:    c.Request.Method,
			Endpoint:      c.Request.URL.Path,
			OperationType: parseOperationType(c.Request.Method),
			TargetType:    parseTargetType(c),
			TargetID:      parseTargetID(c, requestBody),
			StatusCode:    c.Writer.Status(),
			RequestBody:   requestBody,
			ResponseBody:  blw.body.Bytes(),
			CreatedAt:     startTime.Unix(),
			Duration:      time.Since(startTime).Milliseconds(),
		}

		// 异步存储
		go func(log *model.AuditLog) {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()

			if err := m.auditSvc.RecordOperationLog(ctx, log); err != nil {
				m.l.Error("保存审计日志失败", zap.Error(err))
			}
		}(auditLog)
	}
}

func parseTargetType(c *gin.Context) string {
	path := c.Request.URL.Path
	if path == "" {
		return Unknown
	}

	parts := strings.Split(strings.TrimPrefix(path, "/api/"), "/")
	if len(parts) > 0 && parts[0] != "" {
		return parts[0]
	}

	return Unknown
}

// 常见ID字段名
var idFields = []string{"id", "ID", "Id", "targetId", "target_id"}

func parseTargetID(c *gin.Context, reqBody []byte) string {
	// 优先从URL参数获取
	if id := c.Param("id"); id != "" {
		return id
	}

	// 从查询参数获取
	if id := c.Query("id"); id != "" {
		return id
	}

	// 从请求体获取
	if len(reqBody) > 0 {
		// 尝试解析为通用结构
		var body map[string]interface{}
		if err := json.Unmarshal(reqBody, &body); err == nil {
			for _, key := range idFields {
				if val, ok := body[key]; ok {
					switch v := val.(type) {
					case string:
						return v
					case float64:
						return strconv.FormatFloat(v, 'f', 0, 64)
					case int:
						return strconv.Itoa(v)
					}
				}
			}
		}

		// 尝试解析为数组
		var ids []interface{}
		if json.Unmarshal(reqBody, &ids) == nil && len(ids) > 0 {
			if id, ok := ids[0].(string); ok {
				return id
			}
		}
	}

	return "0"
}

// 解析操作类型
func parseOperationType(method string) string {
	if opType, ok := operationTypeMap[method]; ok {
		return opType
	}
	return Unknown
}
