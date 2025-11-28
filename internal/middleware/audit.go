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
	"github.com/GoSimplicity/AI-CloudOps/pkg/jwt"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/datatypes"
)

const (
	OperationCreate = "CREATE"
	OperationUpdate = "UPDATE"
	OperationDelete = "DELETE"
	OperationQuery  = "VIEW"
	Unknown         = "UNKNOWN"

	// 请求体大小限制
	maxBodySize = 1024 * 1024 // 1MB
)

// 预定义跳过审计的路径
var skipAuditPaths = map[string]bool{
	"/api/user/login":                                   true,
	"/api/user/logout":                                  true,
	"/api/user/refresh_token":                           true,
	"/api/user/signup":                                  true,
	"/api/not_auth/getBindIps":                          true,
	"/api/not_auth/getTreeNodeBindIps":                  true,
	"/api/monitor/prometheus_configs/prometheus":        true,
	"/api/monitor/prometheus_configs/prometheus_alert":  true,
	"/api/monitor/prometheus_configs/prometheus_record": true,
	"/api/monitor/prometheus_configs/alertManager":      true,
}

var operationTypeMap = map[string]string{
	"POST":   OperationCreate,
	"PUT":    OperationUpdate,
	"PATCH":  OperationUpdate,
	"DELETE": OperationDelete,
	"GET":    OperationQuery,
}

// 使用对象池优化内存使用
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
		// 快速检查是否需要跳过审计
		if skipAuditPaths[c.Request.URL.Path] {
			c.Next()
			return
		}

		var requestBody datatypes.JSON
		startTime := time.Now()
		traceID := c.GetHeader("X-Trace-ID")

		// 只处理非GET请求的请求体
		if c.Request.Method != "GET" && c.Request.Body != nil {
			buf := bufferPool.Get().(*bytes.Buffer)
			buf.Reset()
			defer bufferPool.Put(buf)

			// 限制读取大小,避免内存溢出
			if _, err := io.CopyN(buf, c.Request.Body, maxBodySize); err != nil && err != io.EOF {
				m.l.Error("读取请求体失败", zap.Error(err))
			}

			// 尝试将请求体转换为JSON
			if buf.Len() > 0 {
				var jsonData interface{}
				if err := json.Unmarshal(buf.Bytes(), &jsonData); err == nil {
					if jsonBytes, err := json.Marshal(jsonData); err == nil {
						requestBody = jsonBytes
					}
				}
			}

			c.Request.Body = io.NopCloser(bytes.NewBuffer(buf.Bytes()))
		}

		// 包装ResponseWriter以捕获响应体
		blw := &bodyLogWriter{
			body:           bufferPool.Get().(*bytes.Buffer),
			ResponseWriter: c.Writer,
		}
		blw.body.Reset()
		defer bufferPool.Put(blw.body)

		c.Writer = blw
		c.Next()

		// 如果是GET请求，不记录审计日志，提前返回
		if c.Request.Method == "GET" {
			return
		}

		// 获取用户ID
		var userID int
		if user, exists := c.MustGet("user").(jwt.UserClaims); exists {
			userID = int(user.Uid)
		}

		// 获取错误信息
		var errorMsg string
		if len(c.Errors) > 0 {
			errorMsg = c.Errors.String()
		}

		// 处理响应体
		var responseBody datatypes.JSON
		if blw.body.Len() > 0 {
			var jsonData interface{}
			if err := json.Unmarshal(blw.body.Bytes(), &jsonData); err == nil {
				if jsonBytes, err := json.Marshal(jsonData); err == nil {
					responseBody = jsonBytes
				}
			}
		}

		// 构建审计日志请求
		auditLogReq := &model.CreateAuditLogRequest{
			UserID:        userID,
			TraceID:       traceID,
			IPAddress:     c.ClientIP(),
			UserAgent:     c.Request.UserAgent(),
			HttpMethod:    c.Request.Method,
			Endpoint:      c.Request.URL.Path,
			OperationType: parseOperationType(c.Request.Method),
			TargetType:    parseTargetType(c),
			TargetID:      parseTargetID(c, requestBody),
			StatusCode:    c.Writer.Status(),
			RequestBody:   requestBody,
			ResponseBody:  responseBody,
			Duration:      time.Since(startTime).Microseconds(), // 使用微秒
			ErrorMsg:      errorMsg,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		// 异步存储审计日志
		m.auditSvc.CreateAuditLogAsync(ctx, auditLogReq)
	}
}

// 解析目标类型
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

// 解析目标ID
func parseTargetID(c *gin.Context, reqBody datatypes.JSON) string {
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
