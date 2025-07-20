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
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ContainerExecPermission 容器执行权限结构
type ContainerExecPermission struct {
	logger *zap.Logger
}

// NewContainerExecPermission 创建权限中间件
func NewContainerExecPermission(logger *zap.Logger) *ContainerExecPermission {
	return &ContainerExecPermission{
		logger: logger,
	}
}

// CheckExecPermission 检查容器执行权限
func (p *ContainerExecPermission) CheckExecPermission() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取用户信息
		userInfo, exists := c.Get("user")
		if !exists {
			p.logger.Warn("用户未认证")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "用户未认证"})
			c.Abort()
			return
		}

		user, ok := userInfo.(map[string]interface{})
		if !ok {
			p.logger.Error("用户信息格式错误")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "用户信息格式错误"})
			c.Abort()
			return
		}

		// 获取用户角色
		userRole, exists := user["role"]
		if !exists {
			p.logger.Warn("用户角色信息缺失")
			c.JSON(http.StatusForbidden, gin.H{"error": "用户角色信息缺失"})
			c.Abort()
			return
		}

		// 获取请求路径和方法
		path := c.Request.URL.Path
		method := c.Request.Method

		// 检查权限
		if !p.hasPermission(userRole.(string), method, path) {
			p.logger.Warn("用户权限不足",
				zap.String("user_role", userRole.(string)),
				zap.String("method", method),
				zap.String("path", path))
			c.JSON(http.StatusForbidden, gin.H{"error": "权限不足"})
			c.Abort()
			return
		}

		// 记录操作日志
		p.logger.Info("容器执行权限检查通过",
			zap.String("user_role", userRole.(string)),
			zap.String("method", method),
			zap.String("path", path))

		c.Next()
	}
}

// hasPermission 检查用户是否有相应权限
func (p *ContainerExecPermission) hasPermission(role, method, path string) bool {
	// 定义权限规则
	permissions := map[string]map[string][]string{
		"admin": {
			"GET": {
				"/api/k8s/containers/*/exec/history",
				"/api/k8s/containers/*/sessions",
				"/api/k8s/containers/*/files",
				"/api/k8s/containers/*/files/download",
				"/api/k8s/containers/*/logs",
				"/api/k8s/containers/*/logs/stream",
				"/api/k8s/containers/*/logs/search",
				"/api/k8s/containers/*/logs/history",
			},
			"POST": {
				"/api/k8s/containers/*/exec",
				"/api/k8s/containers/*/exec/terminal",
				"/api/k8s/containers/*/files/upload",
				"/api/k8s/containers/*/logs/export",
			},
			"PUT": {
				"/api/k8s/containers/*/files/edit",
			},
			"DELETE": {
				"/api/k8s/containers/*/sessions/*",
				"/api/k8s/containers/*/files/delete",
			},
		},
		"operator": {
			"GET": {
				"/api/k8s/containers/*/exec/history",
				"/api/k8s/containers/*/sessions",
				"/api/k8s/containers/*/files",
				"/api/k8s/containers/*/files/download",
				"/api/k8s/containers/*/logs",
				"/api/k8s/containers/*/logs/stream",
				"/api/k8s/containers/*/logs/search",
			},
			"POST": {
				"/api/k8s/containers/*/exec",
				"/api/k8s/containers/*/exec/terminal",
				"/api/k8s/containers/*/logs/export",
			},
			"DELETE": {
				"/api/k8s/containers/*/sessions/*",
			},
		},
		"viewer": {
			"GET": {
				"/api/k8s/containers/*/exec/history",
				"/api/k8s/containers/*/sessions",
				"/api/k8s/containers/*/files",
				"/api/k8s/containers/*/files/download",
				"/api/k8s/containers/*/logs",
				"/api/k8s/containers/*/logs/search",
			},
		},
	}

	// 检查角色权限
	rolePerm, exists := permissions[role]
	if !exists {
		return false
	}

	// 检查方法权限
	methodPerm, exists := rolePerm[method]
	if !exists {
		return false
	}

	// 检查路径权限
	for _, pattern := range methodPerm {
		if p.matchPath(pattern, path) {
			return true
		}
	}

	return false
}

// matchPath 匹配路径模式
func (p *ContainerExecPermission) matchPath(pattern, path string) bool {
	// 简单的通配符匹配，支持 * 号
	patternParts := strings.Split(pattern, "/")
	pathParts := strings.Split(path, "/")

	if len(patternParts) != len(pathParts) {
		return false
	}

	for i, patternPart := range patternParts {
		if patternPart == "*" {
			continue
		}
		if patternPart != pathParts[i] {
			return false
		}
	}

	return true
}

// CheckClusterAccess 检查集群访问权限
func (p *ContainerExecPermission) CheckClusterAccess() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从请求中获取集群ID
		clusterIdStr := c.Query("cluster_id")
		if clusterIdStr == "" {
			// 尝试从请求体中获取
			var reqBody map[string]interface{}
			if err := c.ShouldBindJSON(&reqBody); err == nil {
				if clusterId, exists := reqBody["cluster_id"]; exists {
					clusterIdStr = fmt.Sprintf("%v", clusterId)
				}
			}
		}

		if clusterIdStr == "" {
			p.logger.Warn("集群ID缺失")
			c.JSON(http.StatusBadRequest, gin.H{"error": "集群ID缺失"})
			c.Abort()
			return
		}

		// 获取用户信息
		userInfo, exists := c.Get("user")
		if !exists {
			p.logger.Warn("用户未认证")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "用户未认证"})
			c.Abort()
			return
		}

		user, ok := userInfo.(map[string]interface{})
		if !ok {
			p.logger.Error("用户信息格式错误")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "用户信息格式错误"})
			c.Abort()
			return
		}

		// 检查用户是否有访问该集群的权限
		// 这里需要根据实际的权限系统实现
		// 示例：检查用户的集群权限列表
		if !p.hasClusterAccess(user, clusterIdStr) {
			p.logger.Warn("用户无权访问该集群",
				zap.String("cluster_id", clusterIdStr),
				zap.Any("user", user))
			c.JSON(http.StatusForbidden, gin.H{"error": "无权访问该集群"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// hasClusterAccess 检查用户是否有集群访问权限
func (p *ContainerExecPermission) hasClusterAccess(user map[string]interface{}, clusterId string) bool {
	// 管理员有所有集群的访问权限
	if role, exists := user["role"]; exists && role == "admin" {
		return true
	}

	// 检查用户的集群权限列表
	if clusterList, exists := user["cluster_access"]; exists {
		if clusters, ok := clusterList.([]interface{}); ok {
			for _, cluster := range clusters {
				if fmt.Sprintf("%v", cluster) == clusterId {
					return true
				}
			}
		}
	}

	return false
}

// LogOperation 记录操作日志
func (p *ContainerExecPermission) LogOperation() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取用户信息
		userInfo, exists := c.Get("user")
		if !exists {
			c.Next()
			return
		}

		user, ok := userInfo.(map[string]interface{})
		if !ok {
			c.Next()
			return
		}

		// 记录操作
		p.logger.Info("容器执行操作",
			zap.Any("user", user),
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.String("remote_addr", c.ClientIP()),
			zap.String("user_agent", c.Request.UserAgent()))

		c.Next()
	}
}