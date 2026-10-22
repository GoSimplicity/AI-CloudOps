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
	"context"
	"encoding/json"
	"strconv"
	"strings"
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
		if skipAuditPaths[c.Request.URL.Path] {
			c.Next()
			return
		}

		var requestBody datatypes.JSON
		startTime := time.Now()
		traceID := c.GetHeader("X-Trace-ID")

		// Audit metadata only: bodies can contain passwords, kubeconfigs and large streams.
		c.Next()

		// 如果是GET请求，不记录审计日志，提前返回
		if c.Request.Method == "GET" {
			return
		}

		// 获取用户ID
		var userID int
		if user, exists := c.Value("user").(jwt.UserClaims); exists {
			userID = int(user.Uid)
		}

		var errorMsg string
		if len(c.Errors) > 0 {
			errorMsg = c.Errors.String()
		}

		var responseBody datatypes.JSON

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

	if id := c.Query("id"); id != "" {
		return id
	}

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
