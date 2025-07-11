# Kubernetes 模块 API 参考文档 (更新版)

## 概述

本文档详细描述了 AI-CloudOps 项目中 Kubernetes 模块的 API 接口设计，包括用户明确提出的功能需求和最佳实践建议的 API 实现。

## API 设计原则

### 1. RESTful 设计
- 使用标准的 HTTP 方法 (GET, POST, PUT, DELETE)
- 统一的资源命名规范
- 一致的响应格式

### 2. 版本控制
- API 版本化支持
- 向后兼容性保证
- 版本迁移策略

### 3. 错误处理
- 统一的错误响应格式
- 详细的错误信息
- 适当的 HTTP 状态码

### 4. 安全设计
- 身份认证和授权
- 输入验证和过滤
- 敏感信息保护

## 通用响应格式

### 成功响应
```json
{
  "code": 200,
  "message": "success",
  "data": {
    // 具体数据
  },
  "timestamp": "2024-01-01T00:00:00Z"
}
```

### 错误响应
```json
{
  "code": 400,
  "message": "Bad Request",
  "error": "详细错误信息",
  "timestamp": "2024-01-01T00:00:00Z"
}
```

## 容器运维 API

### 1. 容器日志管理

#### 1.1 获取容器日志
```http
GET /api/k8s/containers/{id}/logs
```

**请求参数**:
- `id` (path): 容器 ID
- `tail` (query): 返回的日志行数，默认 100
- `since` (query): 开始时间 (RFC3339 格式)
- `until` (query): 结束时间 (RFC3339 格式)
- `level` (query): 日志级别 (info, warn, error, debug)
- `search` (query): 搜索关键词

**响应示例**:
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "logs": [
      {
        "timestamp": "2024-01-01T00:00:00Z",
        "level": "INFO",
        "message": "Application started",
        "container_name": "app-container",
        "pod_name": "app-pod",
        "namespace": "default"
      }
    ],
    "total": 100,
    "has_more": true
  }
}
```

#### 1.2 实时日志流
```http
GET /api/k8s/containers/{id}/logs/stream
```

**请求参数**:
- `id` (path): 容器 ID
- `follow` (query): 是否持续跟踪，默认 true
- `level` (query): 日志级别过滤

**响应**: Server-Sent Events 流

#### 1.3 搜索容器日志
```http
GET /api/k8s/containers/{id}/logs/search
```

**请求参数**:
- `id` (path): 容器 ID
- `query` (query): 搜索查询
- `start_time` (query): 开始时间
- `end_time` (query): 结束时间
- `limit` (query): 结果数量限制
- `offset` (query): 偏移量

#### 1.4 导出容器日志
```http
POST /api/k8s/containers/{id}/logs/export
```

**请求体**:
```json
{
  "format": "json", // json, csv, txt
  "start_time": "2024-01-01T00:00:00Z",
  "end_time": "2024-01-01T23:59:59Z",
  "filters": {
    "level": ["INFO", "ERROR"],
    "search": "error"
  }
}
```

**响应**: 文件下载

#### 1.5 获取日志历史记录
```http
GET /api/k8s/containers/{id}/logs/history
```

**响应示例**:
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "history": [
      {
        "id": 1,
        "export_time": "2024-01-01T00:00:00Z",
        "format": "json",
        "file_size": "1.2MB",
        "exported_by": "user1"
      }
    ]
  }
}
```

### 2. 容器 Exec 功能

#### 2.1 执行容器命令
```http
POST /api/k8s/containers/{id}/exec
```

**请求体**:
```json
{
  "command": ["ls", "-la"],
  "timeout": 30,
  "working_dir": "/app"
}
```

**响应示例**:
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "session_id": "exec-12345",
    "stdout": "total 8\ndrwxr-xr-x 2 root root 4096 Jan 1 00:00 .",
    "stderr": "",
    "exit_code": 0,
    "execution_time": 0.5
  }
}
```

#### 2.2 打开终端会话
```http
POST /api/k8s/containers/{id}/exec/terminal
```

**请求体**:
```json
{
  "tty": true,
  "stdin": true,
  "working_dir": "/app"
}
```

**响应示例**:
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "session_id": "terminal-12345",
    "websocket_url": "ws://localhost:8080/api/k8s/containers/123/exec/ws?session=terminal-12345"
  }
}
```

#### 2.3 WebSocket 终端连接
```http
WS /api/k8s/containers/{id}/exec/ws
```

**查询参数**:
- `session`: 会话 ID
- `tty`: 是否启用 TTY

#### 2.4 获取命令执行历史
```http
GET /api/k8s/containers/{id}/exec/history
```

**请求参数**:
- `id` (path): 容器 ID
- `limit` (query): 返回数量，默认 50
- `offset` (query): 偏移量
- `user_id` (query): 执行用户 ID

**响应示例**:
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "history": [
      {
        "id": 1,
        "command": "ls -la",
        "executed_at": "2024-01-01T00:00:00Z",
        "executed_by": "user1",
        "exit_code": 0,
        "execution_time": 0.5,
        "session_id": "exec-12345"
      }
    ],
    "total": 100
  }
}
```

### 3. 容器文件管理

#### 3.1 获取文件列表
```http
GET /api/k8s/containers/{id}/files
```

**请求参数**:
- `id` (path): 容器 ID
- `path` (query): 文件路径，默认 "/"
- `recursive` (query): 是否递归，默认 false

**响应示例**:
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "files": [
      {
        "name": "app.py",
        "path": "/app/app.py",
        "size": 1024,
        "type": "file",
        "permissions": "644",
        "modified_time": "2024-01-01T00:00:00Z"
      }
    ]
  }
}
```

#### 3.2 下载文件
```http
GET /api/k8s/containers/{id}/files/download
```

**请求参数**:
- `id` (path): 容器 ID
- `path` (query): 文件路径

**响应**: 文件下载

#### 3.3 上传文件
```http
POST /api/k8s/containers/{id}/files/upload
```

**请求体**: multipart/form-data
- `file`: 文件内容
- `path`: 目标路径
- `overwrite`: 是否覆盖，默认 false

#### 3.4 编辑文件
```http
PUT /api/k8s/containers/{id}/files/edit
```

**请求体**:
```json
{
  "path": "/app/config.json",
  "content": "{\"key\": \"value\"}",
  "backup": true
}
```

#### 3.5 删除文件
```http
DELETE /api/k8s/containers/{id}/files/delete
```

**请求体**:
```json
{
  "path": "/app/temp.log",
  "recursive": false
}
```

## YAML 版本管理 API

### 1. YAML 版本控制

#### 1.1 获取版本列表
```http
GET /api/k8s/yaml/versions/{resource_id}
```

**请求参数**:
- `resource_id` (path): 资源 ID
- `resource_type` (query): 资源类型 (deployment, service, configmap)
- `limit` (query): 返回数量，默认 20
- `offset` (query): 偏移量
- `branch` (query): 分支名称

**响应示例**:
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "versions": [
      {
        "id": 1,
        "version": "v1.0.0",
        "change_log": "Initial deployment",
        "created_by": "user1",
        "created_at": "2024-01-01T00:00:00Z",
        "tags": ["stable"],
        "is_current": true,
        "branch": "main"
      }
    ],
    "total": 10
  }
}
```

#### 1.2 查看版本差异
```http
GET /api/k8s/yaml/versions/{resource_id}/diff
```

**请求参数**:
- `resource_id` (path): 资源 ID
- `version1` (query): 版本1
- `version2` (query): 版本2

**响应示例**:
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "diff": [
      {
        "type": "added",
        "path": "spec.replicas",
        "old_value": null,
        "new_value": 3
      },
      {
        "type": "modified",
        "path": "spec.template.spec.containers[0].image",
        "old_value": "nginx:1.19",
        "new_value": "nginx:1.20"
      }
    ],
    "summary": {
      "added": 1,
      "modified": 1,
      "deleted": 0
    }
  }
}
```

#### 1.3 回滚到指定版本
```http
POST /api/k8s/yaml/versions/{resource_id}/rollback
```

**请求体**:
```json
{
  "version_id": 5,
  "reason": "Rollback due to performance issues",
  "notify_users": ["user1", "user2"]
}
```

#### 1.4 比较两个版本
```http
POST /api/k8s/yaml/versions/{resource_id}/compare
```

**请求体**:
```json
{
  "version1": "v1.0.0",
  "version2": "v1.1.0",
  "include_metadata": true
}
```

#### 1.5 添加版本标签
```http
POST /api/k8s/yaml/versions/{resource_id}/tag
```

**请求体**:
```json
{
  "version_id": 5,
  "tag": "production-ready",
  "description": "Version ready for production deployment"
}
```

### 2. YAML 备份管理

#### 2.1 创建备份
```http
POST /api/k8s/yaml/backup/create
```

**请求体**:
```json
{
  "resource_id": 123,
  "resource_type": "deployment",
  "backup_name": "pre-update-backup",
  "description": "Backup before major update",
  "encrypt": true
}
```

#### 2.2 获取备份列表
```http
GET /api/k8s/yaml/backup/list
```

**请求参数**:
- `resource_id` (query): 资源 ID
- `resource_type` (query): 资源类型
- `limit` (query): 返回数量
- `offset` (query): 偏移量

**响应示例**:
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "backups": [
      {
        "id": 1,
        "backup_name": "pre-update-backup",
        "resource_id": 123,
        "resource_type": "deployment",
        "created_at": "2024-01-01T00:00:00Z",
        "file_size": "2.5MB",
        "encrypted": true,
        "status": "completed"
      }
    ]
  }
}
```

#### 2.3 恢复备份
```http
POST /api/k8s/yaml/backup/restore
```

**请求体**:
```json
{
  "backup_id": 1,
  "restore_name": "restored-deployment",
  "overwrite": false
}
```

#### 2.4 删除备份
```http
DELETE /api/k8s/yaml/backup/delete/{backup_id}
```

#### 2.5 配置备份策略
```http
POST /api/k8s/yaml/backup/schedule
```

**请求体**:
```json
{
  "resource_id": 123,
  "resource_type": "deployment",
  "schedule": "0 2 * * *", // Cron 表达式
  "retention_days": 30,
  "encrypt": true
}
```

## 多云集群 API

### 1. 集群接入管理

#### 1.1 添加集群
```http
POST /api/k8s/clusters/add
```

**请求体**:
```json
{
  "name": "prod-cluster",
  "display_name": "生产环境集群",
  "kubeconfig": "base64-encoded-kubeconfig-content",
  "context": "prod-context",
  "provider": "aws", // aws, huawei, aliyun, tencent, other
  "region": "us-west-2",
  "description": "生产环境 Kubernetes 集群",
  "tags": ["production", "aws"]
}
```

**响应示例**:
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "id": 1,
    "name": "prod-cluster",
    "display_name": "生产环境集群",
    "provider": "aws",
    "region": "us-west-2",
    "version": "1.28",
    "status": "connected",
    "health_status": "healthy",
    "created_at": "2024-01-01T00:00:00Z",
    "resource_count": 150
  }
}
```

#### 1.2 获取集群列表
```http
GET /api/k8s/clusters/list
```

**请求参数**:
- `provider` (query): 云厂商过滤
- `status` (query): 状态过滤
- `tag` (query): 标签过滤
- `limit` (query): 返回数量
- `offset` (query): 偏移量

**响应示例**:
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "clusters": [
      {
        "id": 1,
        "name": "prod-cluster",
        "display_name": "生产环境集群",
        "provider": "aws",
        "region": "us-west-2",
        "version": "1.28",
        "status": "connected",
        "health_status": "healthy",
        "resource_count": 150,
        "last_sync_time": "2024-01-01T00:00:00Z",
        "tags": ["production", "aws"]
      }
    ],
    "total": 5
  }
}
```

#### 1.3 获取集群详情
```http
GET /api/k8s/clusters/{id}
```

**响应示例**:
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "id": 1,
    "name": "prod-cluster",
    "display_name": "生产环境集群",
    "provider": "aws",
    "region": "us-west-2",
    "version": "1.28",
    "status": "connected",
    "health_status": "healthy",
    "created_at": "2024-01-01T00:00:00Z",
    "last_sync_time": "2024-01-01T00:00:00Z",
    "resource_count": 150,
    "description": "生产环境 Kubernetes 集群",
    "tags": ["production", "aws"],
    "metrics": {
      "cpu_usage": 65.5,
      "memory_usage": 78.2,
      "disk_usage": 45.1,
      "pod_count": 50,
      "node_count": 10,
      "service_count": 25,
      "deployment_count": 30
    }
  }
}
```

#### 1.4 更新集群信息
```http
PUT /api/k8s/clusters/{id}
```

**请求体**:
```json
{
  "display_name": "更新后的集群名称",
  "description": "更新后的描述",
  "tags": ["production", "aws", "updated"]
}
```

#### 1.5 删除集群
```http
DELETE /api/k8s/clusters/{id}
```

#### 1.6 测试集群连接
```http
POST /api/k8s/clusters/{id}/test
```

**响应示例**:
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "connected": true,
    "response_time": 0.5,
    "version": "1.28",
    "api_server": "https://api.prod-cluster.com",
    "nodes_count": 10
  }
}
```

#### 1.7 同步集群状态
```http
POST /api/k8s/clusters/{id}/sync
```

**响应示例**:
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "sync_time": "2024-01-01T00:00:00Z",
    "resource_count": 150,
    "health_status": "healthy",
    "issues": []
  }
}
```

#### 1.8 获取集群资源统计
```http
GET /api/k8s/clusters/{id}/resources
```

**响应示例**:
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "cluster_id": 1,
    "timestamp": "2024-01-01T00:00:00Z",
    "resources": {
      "pods": 50,
      "deployments": 30,
      "services": 25,
      "configmaps": 15,
      "secrets": 10,
      "persistent_volumes": 5,
      "nodes": 10
    },
    "usage": {
      "cpu_usage": 65.5,
      "memory_usage": 78.2,
      "disk_usage": 45.1
    }
  }
}
```

### 2. 统一资源管理

#### 2.1 跨集群资源查询
```http
GET /api/k8s/unified/resources
```

**请求参数**:
- `cluster_ids` (query): 集群ID列表，逗号分隔
- `resource_type` (query): 资源类型 (pods, deployments, services, etc.)
- `namespace` (query): 命名空间
- `label_selector` (query): 标签选择器
- `limit` (query): 返回数量
- `offset` (query): 偏移量

**响应示例**:
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "resources": [
      {
        "cluster_id": 1,
        "cluster_name": "prod-cluster",
        "type": "deployment",
        "namespace": "default",
        "name": "nginx-deployment",
        "status": "running",
        "creation_time": "2024-01-01T00:00:00Z",
        "labels": {
          "app": "nginx"
        },
        "spec": {
          "replicas": 3,
          "selector": {
            "matchLabels": {
              "app": "nginx"
            }
          }
        },
        "status_info": {
          "available_replicas": 3,
          "ready_replicas": 3,
          "replicas": 3
        }
      }
    ],
    "total": 100
  }
}
```

#### 2.2 获取资源详情
```http
GET /api/k8s/unified/resources/{id}
```

#### 2.3 创建资源
```http
POST /api/k8s/unified/resources
```

**请求体**:
```json
{
  "cluster_id": 1,
  "resource_type": "deployment",
  "namespace": "default",
  "resource": {
    "apiVersion": "apps/v1",
    "kind": "Deployment",
    "metadata": {
      "name": "nginx-deployment",
      "labels": {
        "app": "nginx"
      }
    },
    "spec": {
      "replicas": 3,
      "selector": {
        "matchLabels": {
          "app": "nginx"
        }
      },
      "template": {
        "metadata": {
          "labels": {
            "app": "nginx"
          }
        },
        "spec": {
          "containers": [
            {
              "name": "nginx",
              "image": "nginx:1.19",
              "ports": [
                {
                  "containerPort": 80
                }
              ]
            }
          ]
        }
      }
    }
  }
}
```

#### 2.4 更新资源
```http
PUT /api/k8s/unified/resources/{id}
```

#### 2.5 删除资源
```http
DELETE /api/k8s/unified/resources/{id}
```

#### 2.6 批量操作
```http
POST /api/k8s/unified/batch
```

**请求体**:
```json
{
  "cluster_ids": [1, 2, 3],
  "operation": "scale",
  "resources": [
    {
      "type": "deployment",
      "namespace": "default",
      "name": "nginx-deployment",
      "parameters": {
        "replicas": 5
      }
    }
  ]
}
```

#### 2.7 集群间资源对比
```http
GET /api/k8s/unified/compare
```

**请求参数**:
- `cluster_ids` (query): 集群ID列表，逗号分隔
- `resource_type` (query): 资源类型
- `namespace` (query): 命名空间

**响应示例**:
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "comparison": [
      {
        "resource_name": "nginx-deployment",
        "cluster_1": {
          "cluster_id": 1,
          "cluster_name": "prod-cluster",
          "replicas": 3,
          "image": "nginx:1.19"
        },
        "cluster_2": {
          "cluster_id": 2,
          "cluster_name": "staging-cluster",
          "replicas": 2,
          "image": "nginx:1.18"
        },
        "differences": [
          {
            "field": "spec.replicas",
            "cluster_1_value": 3,
            "cluster_2_value": 2
          },
          {
            "field": "spec.template.spec.containers[0].image",
            "cluster_1_value": "nginx:1.19",
            "cluster_2_value": "nginx:1.18"
          }
        ]
      }
    ]
  }
}
```

### 3. 集群监控和健康检查

#### 3.1 获取集群指标
```http
GET /api/k8s/clusters/{id}/metrics
```

**请求参数**:
- `time_range` (query): 时间范围 (1h, 24h, 7d, 30d)

**响应示例**:
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "cluster_id": 1,
    "timestamp": "2024-01-01T00:00:00Z",
    "cpu_usage": 65.5,
    "memory_usage": 78.2,
    "disk_usage": 45.1,
    "network_usage": 12.3,
    "pod_count": 50,
    "node_count": 10,
    "service_count": 25,
    "deployment_count": 30,
    "health_score": 85.5,
    "issues": [
      {
        "type": "warning",
        "message": "High memory usage detected",
        "severity": "medium"
      }
    ]
  }
}
```

#### 3.2 获取集群健康状态
```http
GET /api/k8s/clusters/{id}/health
```

**响应示例**:
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "cluster_id": 1,
    "health_status": "healthy",
    "health_score": 85.5,
    "last_check": "2024-01-01T00:00:00Z",
    "components": [
      {
        "name": "api-server",
        "status": "healthy",
        "response_time": 0.1
      },
      {
        "name": "etcd",
        "status": "healthy",
        "response_time": 0.05
      },
      {
        "name": "controller-manager",
        "status": "healthy",
        "response_time": 0.08
      },
      {
        "name": "scheduler",
        "status": "healthy",
        "response_time": 0.12
      }
    ],
    "nodes": [
      {
        "name": "node-1",
        "status": "Ready",
        "cpu_usage": 60.5,
        "memory_usage": 75.2
      }
    ],
    "issues": []
  }
}
```

#### 3.3 获取所有集群指标汇总
```http
GET /api/k8s/clusters/metrics/summary
```

**响应示例**:
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "summary": {
      "total_clusters": 5,
      "healthy_clusters": 4,
      "warning_clusters": 1,
      "error_clusters": 0,
      "total_resources": 750,
      "avg_cpu_usage": 62.3,
      "avg_memory_usage": 71.8
    },
    "clusters": [
      {
        "id": 1,
        "name": "prod-cluster",
        "health_status": "healthy",
        "resource_count": 150,
        "cpu_usage": 65.5,
        "memory_usage": 78.2
      }
    ]
  }
}
```

#### 3.4 批量健康检查
```http
POST /api/k8s/clusters/health/check
```

**请求体**:
```json
{
  "cluster_ids": [1, 2, 3, 4, 5],
  "check_components": true,
  "check_nodes": true,
  "check_resources": true
}
```

**响应示例**:
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "check_time": "2024-01-01T00:00:00Z",
    "results": [
      {
        "cluster_id": 1,
        "cluster_name": "prod-cluster",
        "status": "healthy",
        "health_score": 85.5,
        "issues": []
      }
    ],
    "summary": {
      "total_checked": 5,
      "healthy": 4,
      "warning": 1,
      "error": 0
    }
  }
}
```

## MCP 集成 API

### 1. K8s MCP 工具

#### 1.1 集群健康扫描
```http
POST /api/k8s/mcp/cluster/scan
```

**请求体**:
```json
{
  "cluster_id": 1,
  "scan_type": "health",
  "options": {
    "include_nodes": true,
    "include_pods": true,
    "include_services": true
  }
}
```

**响应示例**:
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "scan_id": "scan-12345",
    "cluster_id": 1,
    "scan_type": "health",
    "health_status": "healthy",
    "issues": [
      {
        "type": "warning",
        "resource": "pod/app-pod",
        "message": "High memory usage detected",
        "severity": "medium"
      }
    ],
    "recommendations": [
      {
        "type": "optimization",
        "title": "Optimize memory usage",
        "description": "Consider increasing memory limits",
        "action": "update_resource_limits"
      }
    ],
    "scan_time": "2024-01-01T00:00:00Z",
    "duration": 2.5
  }
}
```

#### 1.2 资源使用监控
```http
POST /api/k8s/mcp/resources/monitor
```

**请求体**:
```json
{
  "cluster_id": 1,
  "resource_types": ["pods", "nodes", "services"],
  "time_range": "1h",
  "metrics": ["cpu", "memory", "disk"]
}
```

#### 1.3 配置合规性检查
```http
POST /api/k8s/mcp/config/validate
```

**请求体**:
```json
{
  "cluster_id": 1,
  "policies": [
    "security-context",
    "resource-limits",
    "network-policy"
  ],
  "resources": ["deployments", "services", "configmaps"]
}
```

#### 1.4 性能分析
```http
POST /api/k8s/mcp/performance/analyze
```

**请求体**:
```json
{
  "cluster_id": 1,
  "analysis_type": "bottleneck",
  "time_range": "24h",
  "include_recommendations": true
}
```


```http
POST /api/k8s/mcp/cost/analyze
```

**请求体**:
```json
{
  "cluster_id": 1,
  "time_range": "30d",
  "include_optimization": true,
  "include_forecast": true
}
```

### 2. MCP 工具管理

#### 2.1 获取可用工具列表
```http
GET /api/k8s/mcp/tools
```

**响应示例**:
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "tools": [
      {
        "name": "cluster_scanner",
        "description": "Scan cluster health status",
        "version": "1.0.0",
        "enabled": true,
        "parameters": [
          {
            "name": "scan_type",
            "type": "string",
            "required": true,
            "options": ["health", "security", "performance"]
          }
        ]
      }
    ]
  }
}
```

#### 2.2 执行 MCP 工具
```http
POST /api/k8s/mcp/tools/{tool_name}/execute
```

**请求体**:
```json
{
  "cluster_id": 1,
  "parameters": {
    "scan_type": "health",
    "include_nodes": true
  }
}
```

## CRD 资源 API

### 1. CRD 资源发现

#### 1.1 获取 CRD 列表
```http
GET /api/k8s/crds
```

**请求参数**:
- `cluster_id` (query): 集群 ID
- `group` (query): API 组
- `version` (query): API 版本
- `scope` (query): 作用域 (Namespaced, Cluster)

**响应示例**:
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "crds": [
      {
        "name": "customresources.example.com",
        "group": "example.com",
        "version": "v1",
        "kind": "CustomResource",
        "plural": "customresources",
        "singular": "customresource",
        "scope": "Namespaced",
        "schema": {
          "type": "object",
          "properties": {
            "spec": {
              "type": "object"
            }
          }
        },
        "status": "established"
      }
    ]
  }
}
```

#### 1.2 获取 CRD 详情
```http
GET /api/k8s/crds/{name}
```

#### 1.3 获取 CRD Schema
```http
GET /api/k8s/crds/{name}/schema
```

### 2. CRD 资源管理

#### 2.1 获取 CRD 资源列表
```http
GET /api/k8s/crds/{name}/resources
```

**请求参数**:
- `name` (path): CRD 名称
- `namespace` (query): 命名空间 (如果是 Namespaced)
- `label_selector` (query): 标签选择器
- `field_selector` (query): 字段选择器

#### 2.2 创建 CRD 资源
```http
POST /api/k8s/crds/{name}/resources
```

**请求体**:
```json
{
  "metadata": {
    "name": "my-custom-resource",
    "namespace": "default",
    "labels": {
      "app": "my-app"
    }
  },
  "spec": {
    "replicas": 3,
    "image": "nginx:latest"
  }
}
```

#### 2.3 更新 CRD 资源
```http
PUT /api/k8s/crds/{name}/resources/{resource_name}
```

#### 2.4 删除 CRD 资源
```http
DELETE /api/k8s/crds/{name}/resources/{resource_name}
```

## 资源配额管理 API

### 1. ResourceQuota 管理

#### 1.1 创建 ResourceQuota
```http
POST /api/k8s/resourcequota/create
```

**请求体**:
```json
{
  "name": "compute-quota",
  "namespace": "default",
  "spec": {
    "hard": {
      "requests.cpu": "4",
      "requests.memory": "8Gi",
      "limits.cpu": "8",
      "limits.memory": "16Gi",
      "pods": "10",
      "services": "5",
      "configmaps": "10",
      "persistentvolumeclaims": "4"
    },
    "scopes": ["BestEffort", "NotBestEffort"]
  },
  "description": "Compute resource quota for default namespace"
}
```

**响应示例**:
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "id": 1,
    "name": "compute-quota",
    "namespace": "default",
    "spec": {
      "hard": {
        "requests.cpu": "4",
        "requests.memory": "8Gi",
        "limits.cpu": "8",
        "limits.memory": "16Gi",
        "pods": "10",
        "services": "5",
        "configmaps": "10",
        "persistentvolumeclaims": "4"
      },
      "scopes": ["BestEffort", "NotBestEffort"]
    },
    "status": {
      "hard": {
        "requests.cpu": "4",
        "requests.memory": "8Gi",
        "limits.cpu": "8",
        "limits.memory": "16Gi",
        "pods": "10",
        "services": "5",
        "configmaps": "10",
        "persistentvolumeclaims": "4"
      },
      "used": {
        "requests.cpu": "2",
        "requests.memory": "4Gi",
        "limits.cpu": "4",
        "limits.memory": "8Gi",
        "pods": "5",
        "services": "2",
        "configmaps": "3",
        "persistentvolumeclaims": "1"
      }
    },
    "created_at": "2024-01-01T00:00:00Z"
  }
}
```

#### 1.2 获取 ResourceQuota 列表
```http
GET /api/k8s/resourcequota/list
```

**请求参数**:
- `namespace` (query): 命名空间
- `cluster_id` (query): 集群 ID
- `limit` (query): 返回数量
- `offset` (query): 偏移量

**响应示例**:
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "resourcequotas": [
      {
        "id": 1,
        "name": "compute-quota",
        "namespace": "default",
        "cluster_id": 1,
        "hard": {
          "requests.cpu": "4",
          "requests.memory": "8Gi",
          "pods": "10"
        },
        "used": {
          "requests.cpu": "2",
          "requests.memory": "4Gi",
          "pods": "5"
        },
        "usage_percentage": {
          "requests.cpu": 50.0,
          "requests.memory": 50.0,
          "pods": 50.0
        },
        "status": "active",
        "created_at": "2024-01-01T00:00:00Z"
      }
    ],
    "total": 5
  }
}
```

#### 1.3 获取 ResourceQuota 详情
```http
GET /api/k8s/resourcequota/{id}
```

**响应示例**:
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "id": 1,
    "name": "compute-quota",
    "namespace": "default",
    "cluster_id": 1,
    "spec": {
      "hard": {
        "requests.cpu": "4",
        "requests.memory": "8Gi",
        "limits.cpu": "8",
        "limits.memory": "16Gi",
        "pods": "10",
        "services": "5",
        "configmaps": "10",
        "persistentvolumeclaims": "4"
      },
      "scopes": ["BestEffort", "NotBestEffort"]
    },
    "status": {
      "hard": {
        "requests.cpu": "4",
        "requests.memory": "8Gi",
        "limits.cpu": "8",
        "limits.memory": "16Gi",
        "pods": "10",
        "services": "5",
        "configmaps": "10",
        "persistentvolumeclaims": "4"
      },
      "used": {
        "requests.cpu": "2",
        "requests.memory": "4Gi",
        "limits.cpu": "4",
        "limits.memory": "8Gi",
        "pods": "5",
        "services": "2",
        "configmaps": "3",
        "persistentvolumeclaims": "1"
      }
    },
    "usage_percentage": {
      "requests.cpu": 50.0,
      "requests.memory": 50.0,
      "limits.cpu": 50.0,
      "limits.memory": 50.0,
      "pods": 50.0,
      "services": 40.0,
      "configmaps": 30.0,
      "persistentvolumeclaims": 25.0
    },
    "alerts": [
      {
        "type": "warning",
        "resource": "requests.cpu",
        "usage": 50.0,
        "threshold": 80.0,
        "message": "CPU requests usage is approaching limit"
      }
    ],
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z"
  }
}
```

#### 1.4 更新 ResourceQuota
```http
PUT /api/k8s/resourcequota/{id}
```

**请求体**:
```json
{
  "spec": {
    "hard": {
      "requests.cpu": "6",
      "requests.memory": "12Gi",
      "limits.cpu": "12",
      "limits.memory": "24Gi",
      "pods": "15"
    }
  },
  "description": "Updated compute quota with increased limits"
}
```

#### 1.5 删除 ResourceQuota
```http
DELETE /api/k8s/resourcequota/{id}
```

#### 1.6 获取配额使用统计
```http
GET /api/k8s/resourcequota/{id}/usage
```

**请求参数**:
- `time_range` (query): 时间范围 (1h, 24h, 7d, 30d)

**响应示例**:
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "quota_id": 1,
    "time_range": "24h",
    "usage_history": [
      {
        "timestamp": "2024-01-01T00:00:00Z",
        "requests.cpu": 2.5,
        "requests.memory": "5Gi",
        "pods": 6
      }
    ],
    "trends": {
      "requests.cpu": {
        "trend": "increasing",
        "growth_rate": 5.2
      },
      "requests.memory": {
        "trend": "stable",
        "growth_rate": 0.0
      }
    }
  }
}
```

### 2. LimitRange 管理

#### 2.1 创建 LimitRange
```http
POST /api/k8s/limitrange/create
```

**请求体**:
```json
{
  "name": "default-limits",
  "namespace": "default",
  "spec": {
    "limits": [
      {
        "type": "Container",
        "default": {
          "cpu": "500m",
          "memory": "512Mi"
        },
        "defaultRequest": {
          "cpu": "250m",
          "memory": "256Mi"
        },
        "min": {
          "cpu": "100m",
          "memory": "128Mi"
        },
        "max": {
          "cpu": "2",
          "memory": "2Gi"
        }
      },
      {
        "type": "Pod",
        "max": {
          "cpu": "4",
          "memory": "4Gi"
        }
      }
    ]
  }
}
```

#### 2.2 获取 LimitRange 列表
```http
GET /api/k8s/limitrange/list
```

#### 2.3 获取 LimitRange 详情
```http
GET /api/k8s/limitrange/{id}
```

#### 2.4 更新 LimitRange
```http
PUT /api/k8s/limitrange/{id}
```

#### 2.5 删除 LimitRange
```http
DELETE /api/k8s/limitrange/{id}
```

## 标签与亲和性管理 API

### 1. 标签管理

#### 1.1 获取资源标签
```http
GET /api/k8s/labels/{resource_type}/{resource_id}
```

**请求参数**:
- `resource_type` (path): 资源类型 (pods, deployments, services, nodes)
- `resource_id` (path): 资源 ID
- `cluster_id` (query): 集群 ID
- `namespace` (query): 命名空间

**响应示例**:
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "resource_type": "deployment",
    "resource_id": "nginx-deployment",
    "namespace": "default",
    "cluster_id": 1,
    "labels": {
      "app": "nginx",
      "version": "1.19",
      "environment": "production",
      "team": "frontend"
    },
    "annotations": {
      "kubernetes.io/change-cause": "Update to nginx 1.19"
    }
  }
}
```

#### 1.2 添加/更新标签
```http
POST /api/k8s/labels/{resource_type}/{resource_id}/add
```

**请求体**:
```json
{
  "labels": {
    "environment": "production",
    "team": "frontend",
    "priority": "high"
  },
  "annotations": {
    "description": "Production nginx deployment"
  },
  "overwrite": true
}
```

#### 1.3 删除标签
```http
DELETE /api/k8s/labels/{resource_type}/{resource_id}/remove
```

**请求体**:
```json
{
  "labels": ["environment", "team"],
  "annotations": ["description"]
}
```

#### 1.4 批量标签操作
```http
POST /api/k8s/labels/batch
```

**请求体**:
```json
{
  "operation": "add", // add, remove, replace
  "resources": [
    {
      "type": "deployment",
      "name": "nginx-deployment",
      "namespace": "default"
    },
    {
      "type": "service",
      "name": "nginx-service",
      "namespace": "default"
    }
  ],
  "labels": {
    "environment": "production",
    "team": "frontend"
  }
}
```

#### 1.5 标签选择器查询
```http
GET /api/k8s/labels/select
```

**请求参数**:
- `cluster_id` (query): 集群 ID
- `resource_type` (query): 资源类型
- `namespace` (query): 命名空间
- `label_selector` (query): 标签选择器 (app=nginx,environment=production)
- `limit` (query): 返回数量
- `offset` (query): 偏移量

**响应示例**:
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "resources": [
      {
        "type": "deployment",
        "name": "nginx-deployment",
        "namespace": "default",
        "labels": {
          "app": "nginx",
          "environment": "production"
        },
        "created_at": "2024-01-01T00:00:00Z"
      }
    ],
    "total": 5
  }
}
```

#### 1.6 标签策略管理
```http
POST /api/k8s/labels/policies/create
```

**请求体**:
```json
{
  "name": "production-labels",
  "description": "Required labels for production resources",
  "rules": [
    {
      "resource_type": "deployment",
      "required_labels": ["environment", "team", "version"],
      "forbidden_labels": ["test", "dev"],
      "label_patterns": {
        "environment": "^production$|^staging$",
        "version": "^v\\d+\\.\\d+\\.\\d+$"
      }
    }
  ],
  "enabled": true
}
```

#### 1.7 标签合规性检查
```http
POST /api/k8s/labels/compliance/check
```

**请求体**:
```json
{
  "cluster_id": 1,
  "namespace": "default",
  "resource_types": ["deployments", "services"],
  "policy_id": 1
}
```

### 2. 节点亲和性管理

#### 2.1 获取节点亲和性配置
```http
GET /api/k8s/affinity/node/{resource_id}
```

**请求参数**:
- `resource_id` (path): 资源 ID
- `resource_type` (query): 资源类型 (deployment, pod)
- `cluster_id` (query): 集群 ID
- `namespace` (query): 命名空间

**响应示例**:
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "resource_id": "nginx-deployment",
    "resource_type": "deployment",
    "affinity": {
      "nodeAffinity": {
        "requiredDuringSchedulingIgnoredDuringExecution": {
          "nodeSelectorTerms": [
            {
              "matchExpressions": [
                {
                  "key": "kubernetes.io/os",
                  "operator": "In",
                  "values": ["linux"]
                },
                {
                  "key": "node-type",
                  "operator": "In",
                  "values": ["compute"]
                }
              ]
            }
          ]
        },
        "preferredDuringSchedulingIgnoredDuringExecution": [
          {
            "weight": 100,
            "preference": {
              "matchExpressions": [
                {
                  "key": "zone",
                  "operator": "In",
                  "values": ["us-west-2a"]
                }
              ]
            }
          }
        ]
      }
    }
  }
}
```

#### 2.2 设置节点亲和性
```http
POST /api/k8s/affinity/node/{resource_id}/set
```

**请求体**:
```json
{
  "affinity": {
    "nodeAffinity": {
      "requiredDuringSchedulingIgnoredDuringExecution": {
        "nodeSelectorTerms": [
          {
            "matchExpressions": [
              {
                "key": "kubernetes.io/os",
                "operator": "In",
                "values": ["linux"]
              },
              {
                "key": "node-type",
                "operator": "In",
                "values": ["compute"]
              }
            ]
          }
        ]
      },
      "preferredDuringSchedulingIgnoredDuringExecution": [
        {
          "weight": 100,
          "preference": {
            "matchExpressions": [
              {
                "key": "zone",
                "operator": "In",
                "values": ["us-west-2a"]
              }
            ]
          }
        }
      ]
    }
  }
}
```

#### 2.3 获取节点选择器建议
```http
GET /api/k8s/affinity/node/suggestions
```

**请求参数**:
- `cluster_id` (query): 集群 ID
- `resource_type` (query): 资源类型
- `requirements` (query): 资源需求 (cpu, memory, gpu)

**响应示例**:
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "suggestions": [
      {
        "node_selector": {
          "kubernetes.io/os": "linux",
          "node-type": "compute"
        },
        "matching_nodes": 5,
        "available_resources": {
          "cpu": "20",
          "memory": "40Gi"
        },
        "score": 85.5
      }
    ]
  }
}
```

### 3. Pod 亲和性管理

#### 3.1 获取 Pod 亲和性配置
```http
GET /api/k8s/affinity/pod/{resource_id}
```

**响应示例**:
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "resource_id": "app-deployment",
    "affinity": {
      "podAffinity": {
        "requiredDuringSchedulingIgnoredDuringExecution": [
          {
            "labelSelector": {
              "matchExpressions": [
                {
                  "key": "app",
                  "operator": "In",
                  "values": ["database"]
                }
              ]
            },
            "topologyKey": "kubernetes.io/hostname"
          }
        ]
      },
      "podAntiAffinity": {
        "preferredDuringSchedulingIgnoredDuringExecution": [
          {
            "weight": 100,
            "podAffinityTerm": {
              "labelSelector": {
                "matchExpressions": [
                  {
                    "key": "app",
                    "operator": "In",
                    "values": ["app"]
                  }
                ]
              },
              "topologyKey": "kubernetes.io/hostname"
            }
          }
        ]
      }
    }
  }
}
```

#### 3.2 设置 Pod 亲和性
```http
POST /api/k8s/affinity/pod/{resource_id}/set
```

**请求体**:
```json
{
  "affinity": {
    "podAffinity": {
      "requiredDuringSchedulingIgnoredDuringExecution": [
        {
          "labelSelector": {
            "matchExpressions": [
              {
                "key": "app",
                "operator": "In",
                "values": ["database"]
              }
            ]
          },
          "topologyKey": "kubernetes.io/hostname"
        }
      ]
    },
    "podAntiAffinity": {
      "preferredDuringSchedulingIgnoredDuringExecution": [
        {
          "weight": 100,
          "podAffinityTerm": {
            "labelSelector": {
              "matchExpressions": [
                {
                  "key": "app",
                  "operator": "In",
                  "values": ["app"]
                }
              ]
            },
            "topologyKey": "kubernetes.io/hostname"
          }
        }
      ]
    }
  }
}
```

#### 3.3 获取拓扑域信息
```http
GET /api/k8s/affinity/pod/topology
```

**请求参数**:
- `cluster_id` (query): 集群 ID
- `topology_key` (query): 拓扑键 (kubernetes.io/hostname, kubernetes.io/zone)

**响应示例**:
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "topology_key": "kubernetes.io/hostname",
    "domains": [
      {
        "name": "node-1",
        "resources": [
          {
            "type": "pod",
            "name": "app-pod-1",
            "namespace": "default"
          }
        ]
      }
    ]
  }
}
```

### 4. 污点容忍管理

#### 4.1 获取污点容忍配置
```http
GET /api/k8s/taints/tolerations/{resource_id}
```

**响应示例**:
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "resource_id": "app-deployment",
    "tolerations": [
      {
        "key": "dedicated",
        "operator": "Equal",
        "value": "app",
        "effect": "NoSchedule"
      },
      {
        "key": "CriticalAddonsOnly",
        "operator": "Exists",
        "effect": "NoExecute",
        "tolerationSeconds": 300
      }
    ]
  }
}
```

#### 4.2 设置污点容忍
```http
POST /api/k8s/taints/tolerations/{resource_id}/set
```

**请求体**:
```json
{
  "tolerations": [
    {
      "key": "dedicated",
      "operator": "Equal",
      "value": "app",
      "effect": "NoSchedule"
    },
    {
      "key": "CriticalAddonsOnly",
      "operator": "Exists",
      "effect": "NoExecute",
      "tolerationSeconds": 300
    }
  ]
}
```

#### 4.3 获取节点污点信息
```http
GET /api/k8s/taints/nodes/{node_name}
```

**响应示例**:
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "node_name": "node-1",
    "taints": [
      {
        "key": "dedicated",
        "value": "app",
        "effect": "NoSchedule"
      }
    ],
    "effect_summary": {
      "NoSchedule": 1,
      "PreferNoSchedule": 0,
      "NoExecute": 0
    }
  }
}
```

#### 4.4 添加节点污点
```http
POST /api/k8s/taints/nodes/{node_name}/add
```

**请求体**:
```json
{
  "taints": [
    {
      "key": "dedicated",
      "value": "app",
      "effect": "NoSchedule"
    }
  ]
}
```

#### 4.5 移除节点污点
```http
DELETE /api/k8s/taints/nodes/{node_name}/remove
```

**请求体**:
```json
{
  "taints": [
    {
      "key": "dedicated",
      "value": "app"
    }
  ]
}
```

### 5. 亲和性可视化

#### 5.1 获取亲和性关系图
```http
GET /api/k8s/affinity/visualization
```

**请求参数**:
- `cluster_id` (query): 集群 ID
- `namespace` (query): 命名空间
- `resource_type` (query): 资源类型
- `include_nodes` (query): 是否包含节点信息

**响应示例**:
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "nodes": [
      {
        "id": "node-1",
        "name": "node-1",
        "type": "node",
        "labels": {
          "zone": "us-west-2a",
          "node-type": "compute"
        }
      }
    ],
    "edges": [
      {
        "from": "app-deployment",
        "to": "node-1",
        "type": "nodeAffinity",
        "weight": 100
      }
    ],
    "groups": [
      {
        "name": "database-group",
        "resources": ["db-pod-1", "db-pod-2"],
        "affinity_type": "podAffinity"
      }
    ]
  }
}
```

## 智能运维 API

### 1. 自动扩缩容

#### 1.1 创建 HPA
```http
POST /api/k8s/autoscaling/hpa/create
```

**请求体**:
```json
{
  "name": "app-hpa",
  "namespace": "default",
  "target": {
    "api_version": "apps/v1",
    "kind": "Deployment",
    "name": "app-deployment"
  },
  "metrics": [
    {
      "type": "Resource",
      "resource": {
        "name": "cpu",
        "target": {
          "type": "Utilization",
          "average_utilization": 70
        }
      }
    }
  ],
  "min_replicas": 1,
  "max_replicas": 10
}
```

#### 1.2 获取 HPA 列表
```http
GET /api/k8s/autoscaling/hpa
```

#### 1.3 更新 HPA 配置
```http
PUT /api/k8s/autoscaling/hpa/{name}
```

### 2. 故障自愈

#### 2.1 创建自愈规则
```http
POST /api/k8s/self-healing/rules/create
```

**请求体**:
```json
{
  "name": "pod-restart-rule",
  "description": "Auto restart failed pods",
  "conditions": [
    {
      "type": "pod_status",
      "operator": "equals",
      "value": "CrashLoopBackOff"
    }
  ],
  "actions": [
    {
      "type": "restart_pod",
      "parameters": {
        "delay": "30s"
      }
    }
  ],
  "enabled": true
}
```

#### 2.2 获取自愈规则列表
```http
GET /api/k8s/self-healing/rules
```

#### 2.3 获取自愈历史
```http
GET /api/k8s/self-healing/history
```


### 2. 网络策略生成

#### 2.1 分析流量模式
```http
POST /api/k8s/security/network/analyze
```

**请求体**:
```json
{
  "namespace": "default",
  "time_range": "24h",
  "include_recommendations": true
}
```

#### 2.2 生成网络策略
```http
POST /api/k8s/security/network/generate-policy
```

**请求体**:
```json
{
  "namespace": "default",
  "pods": ["app-pod", "db-pod"],
  "policy_type": "deny-all",
  "exceptions": [
    {
      "from": "app-pod",
      "to": "db-pod",
      "ports": [3306]
    }
  ]
}
```



### 1. 资源成本分析

#### 1.1 获取成本报告
```http
GET /api/k8s/cost/report
```

**请求参数**:
- `cluster_id` (query): 集群 ID
- `time_range` (query): 时间范围
- `group_by` (query): 分组方式 (namespace, pod, node)

**响应示例**:
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "total_cost": 1250.50,
    "currency": "USD",
    "period": "2024-01-01 to 2024-01-31",
    "breakdown": [
      {
        "namespace": "default",
        "cost": 450.25,
        "percentage": 36.0,
        "resources": {
          "cpu": 200.10,
          "memory": 150.15,
          "storage": 100.00
        }
      }
    ],
    "trend": {
      "daily_average": 40.34,
      "growth_rate": 5.2
    }
  }
}
```

#### 1.2 获取成本优化建议
```http
GET /api/k8s/cost/optimization
```

**请求参数**:
- `cluster_id` (query): 集群 ID
- `optimization_type` (query): 优化类型 (resource, instance, storage)

## 合规性管理 API

### 1. 策略检查

#### 1.1 运行合规性检查
```http
POST /api/k8s/compliance/check
```

**请求体**:
```json
{
  "cluster_id": 1,
  "policies": [
    "security-context",
    "resource-limits",
    "network-policy"
  ],
  "resources": ["deployments", "services", "configmaps"],
  "auto_fix": false
}
```

**响应示例**:
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "check_id": "check-12345",
    "cluster_id": 1,
    "status": "completed",
    "results": [
      {
        "resource": "deployment/app-deployment",
        "policy": "security-context",
        "status": "failed",
        "violations": [
          {
            "field": "spec.template.spec.securityContext.runAsNonRoot",
            "expected": true,
            "actual": false,
            "message": "Containers should not run as root"
          }
        ],
        "fix_suggestion": {
          "action": "update_security_context",
          "patch": {
            "spec": {
              "template": {
                "spec": {
                  "securityContext": {
                    "runAsNonRoot": true,
                    "runAsUser": 1000
                  }
                }
              }
            }
          }
        }
      }
    ],
    "summary": {
      "total": 50,
      "passed": 45,
      "failed": 5,
      "compliance_score": 90.0
    }
  }
}
```

#### 1.2 获取合规性报告
```http
GET /api/k8s/compliance/report
```

**请求参数**:
- `cluster_id` (query): 集群 ID
- `time_range` (query): 时间范围
- `format` (query): 报告格式 (json, pdf, csv)

## 开发工具 API

### 1. CLI 工具

#### 1.1 获取 CLI 配置
```http
GET /api/k8s/cli/config
```

**响应示例**:
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "clusters": [
      {
        "id": 1,
        "name": "prod-cluster",
        "context": "prod",
        "endpoint": "https://api.prod-cluster.com"
      }
    ],
    "default_context": "prod",
    "plugins": [
      {
        "name": "kubectl",
        "version": "1.28.0",
        "enabled": true
      }
    ]
  }
}
```

#### 1.2 执行 CLI 命令
```http
POST /api/k8s/cli/execute
```

**请求体**:
```json
{
  "command": "kubectl get pods",
  "cluster_id": 1,
  "namespace": "default",
  "timeout": 30
}
```

## 错误码定义

### 通用错误码
- `200`: 成功
- `400`: 请求参数错误
- `401`: 未授权
- `403`: 禁止访问
- `404`: 资源不存在
- `409`: 资源冲突
- `422`: 请求格式正确但语义错误
- `500`: 服务器内部错误
- `502`: 网关错误
- `503`: 服务不可用

### 业务错误码
- `1001`: 集群连接失败
- `1002`: 资源不存在
- `1003`: 权限不足
- `1004`: 操作超时
- `1005`: 配置错误
- `1006`: 版本冲突
- `1007`: 备份失败
- `1008`: 恢复失败
- `1009`: 扫描失败
- `1010`: 工具执行失败

## 认证和授权

### 1. 认证方式
- JWT Token
- API Key
- OAuth 2.0

### 2. 权限控制
- 基于角色的访问控制 (RBAC)
- 资源级别的权限控制
- 操作级别的权限控制

### 3. 审计日志
所有 API 调用都会记录审计日志，包括：
- 用户信息
- 操作类型
- 资源信息
- 时间戳
- IP 地址
- 请求参数
- 响应状态

## 限流和配额

### 1. API 限流
- 基于用户 ID 的限流
- 基于 IP 地址的限流
- 基于 API 端点的限流

### 2. 资源配额
- 集群数量限制
- 资源创建频率限制
- 存储空间限制

## 监控和指标

### 1. API 指标
- 请求数量
- 响应时间
- 错误率
- 并发数

### 2. 业务指标
- 集群数量
- 资源使用情况
- 操作成功率
- 用户活跃度

## 总结

本 API 参考文档详细描述了 Kubernetes 模块的所有 API 接口，涵盖了用户明确提出的功能需求和最佳实践建议。API 设计遵循 RESTful 原则，提供统一的响应格式和完善的错误处理机制。

关键特点：
1. **完整性**: 覆盖所有核心功能模块
2. **一致性**: 统一的 API 设计风格
3. **安全性**: 完善的认证和授权机制
4. **可扩展性**: 支持版本控制和向后兼容
5. **易用性**: 详细的参数说明和示例 

## 网络管理 API

### 1. Endpoint 管理

#### 1.1 获取 Endpoint 列表
```http
GET /api/k8s/endpoints
```

**请求参数**:
- `cluster_id` (query): 集群 ID
- `namespace` (query): 命名空间
- `service_name` (query): 服务名称
- `label_selector` (query): 标签选择器
- `limit` (query): 返回数量
- `offset` (query): 偏移量

**响应示例**:
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "endpoints": [
      {
        "id": 1,
        "name": "nginx-service",
        "namespace": "default",
        "cluster_id": 1,
        "subsets": [
          {
            "addresses": [
              {
                "ip": "10.244.1.5",
                "hostname": "nginx-pod-1",
                "node_name": "node-1",
                "target_ref": {
                  "kind": "Pod",
                  "name": "nginx-pod-1",
                  "namespace": "default"
                }
              }
            ],
            "ports": [
              {
                "name": "http",
                "port": 80,
                "protocol": "TCP"
              }
            ]
          }
        ],
        "labels": {
          "app": "nginx"
        },
        "annotations": {
          "description": "Nginx service endpoints"
        },
        "created_at": "2024-01-01T00:00:00Z",
        "updated_at": "2024-01-01T00:00:00Z"
      }
    ],
    "total": 10
  }
}
```

#### 1.2 获取 Endpoint 详情
```http
GET /api/k8s/endpoints/{name}
```

**请求参数**:
- `name` (path): Endpoint 名称
- `namespace` (query): 命名空间
- `cluster_id` (query): 集群 ID

**响应示例**:
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "id": 1,
    "name": "nginx-service",
    "namespace": "default",
    "cluster_id": 1,
    "subsets": [
      {
        "addresses": [
          {
            "ip": "10.244.1.5",
            "hostname": "nginx-pod-1",
            "node_name": "node-1",
            "target_ref": {
              "kind": "Pod",
              "name": "nginx-pod-1",
              "namespace": "default"
            },
            "ready": true,
            "serving": true,
            "terminating": false
          }
        ],
        "not_ready_addresses": [],
        "ports": [
          {
            "name": "http",
            "port": 80,
            "protocol": "TCP"
          }
        ]
      }
    ],
    "labels": {
      "app": "nginx"
    },
    "annotations": {
      "description": "Nginx service endpoints"
    },
    "status": {
      "total_endpoints": 3,
      "ready_endpoints": 3,
      "not_ready_endpoints": 0,
      "health_score": 100.0
    },
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z"
  }
}
```

#### 1.3 创建 Endpoint
```http
POST /api/k8s/endpoints/create
```

**请求体**:
```json
{
  "name": "custom-endpoint",
  "namespace": "default",
  "cluster_id": 1,
  "subsets": [
    {
      "addresses": [
        {
          "ip": "192.168.1.100",
          "hostname": "external-service-1"
        }
      ],
      "ports": [
        {
          "name": "http",
          "port": 8080,
          "protocol": "TCP"
        }
      ]
    }
  ],
  "labels": {
    "app": "external-service",
    "type": "custom"
  },
  "annotations": {
    "description": "External service endpoint"
  }
}
```

#### 1.4 更新 Endpoint
```http
PUT /api/k8s/endpoints/{name}
```

**请求体**:
```json
{
  "subsets": [
    {
      "addresses": [
        {
          "ip": "192.168.1.101",
          "hostname": "external-service-2"
        }
      ],
      "ports": [
        {
          "name": "http",
          "port": 8080,
          "protocol": "TCP"
        }
      ]
    }
  ],
  "labels": {
    "app": "external-service",
    "type": "custom",
    "updated": "true"
  }
}
```

#### 1.5 删除 Endpoint
```http
DELETE /api/k8s/endpoints/{name}
```

**请求参数**:
- `name` (path): Endpoint 名称
- `namespace` (query): 命名空间
- `cluster_id` (query): 集群 ID

#### 1.6 获取 Endpoint 状态
```http
GET /api/k8s/endpoints/{name}/status
```

**响应示例**:
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "name": "nginx-service",
    "namespace": "default",
    "cluster_id": 1,
    "status": {
      "total_endpoints": 3,
      "ready_endpoints": 3,
      "not_ready_endpoints": 0,
      "health_score": 100.0,
      "last_check": "2024-01-01T00:00:00Z"
    },
    "endpoint_details": [
      {
        "ip": "10.244.1.5",
        "hostname": "nginx-pod-1",
        "node_name": "node-1",
        "ready": true,
        "serving": true,
        "terminating": false,
        "last_probe": "2024-01-01T00:00:00Z",
        "response_time": 0.05
      }
    ],
    "health_checks": [
      {
        "endpoint": "10.244.1.5:80",
        "status": "healthy",
        "response_time": 0.05,
        "last_check": "2024-01-01T00:00:00Z"
      }
    ]
  }
}
```

#### 1.7 执行端点健康检查
```http
POST /api/k8s/endpoints/{name}/health-check
```

**请求体**:
```json
{
  "timeout": 30,
  "retries": 3,
  "check_ports": [80, 443],
  "custom_checks": [
    {
      "port": 8080,
      "path": "/health",
      "method": "GET",
      "expected_status": 200
    }
  ]
}
```

**响应示例**:
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "check_id": "health-check-12345",
    "endpoint_name": "nginx-service",
    "check_time": "2024-01-01T00:00:00Z",
    "results": [
      {
        "endpoint": "10.244.1.5:80",
        "status": "healthy",
        "response_time": 0.05,
        "details": {
          "tcp_check": "passed",
          "http_check": "passed",
          "custom_check": "passed"
        }
      }
    ],
    "summary": {
      "total_checked": 3,
      "healthy": 3,
      "unhealthy": 0,
      "timeout": 0
    }
  }
}
```

#### 1.8 获取 Endpoint 与 Service 关联
```http
GET /api/k8s/endpoints/{name}/service
```

**响应示例**:
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "endpoint_name": "nginx-service",
    "service": {
      "name": "nginx-service",
      "namespace": "default",
      "type": "ClusterIP",
      "cluster_ip": "10.96.1.100",
      "external_ips": [],
      "ports": [
        {
          "name": "http",
          "port": 80,
          "target_port": 80,
          "protocol": "TCP"
        }
      ],
      "selector": {
        "app": "nginx"
      }
    },
    "endpoint_sync_status": "synced",
    "last_sync_time": "2024-01-01T00:00:00Z"
  }
}
```

#### 1.9 批量操作 Endpoint
```http
POST /api/k8s/endpoints/batch
```

**请求体**:
```json
{
  "operation": "update", // create, update, delete
  "endpoints": [
    {
      "name": "service-1",
      "namespace": "default",
      "subsets": [
        {
          "addresses": [
            {
              "ip": "192.168.1.100",
              "hostname": "service-1"
            }
          ],
          "ports": [
            {
              "name": "http",
              "port": 8080,
              "protocol": "TCP"
            }
          ]
        }
      ]
    }
  ]
}
```

#### 1.10 获取 Endpoint 监控指标
```http
GET /api/k8s/endpoints/{name}/metrics
```

**请求参数**:
- `name` (path): Endpoint 名称
- `namespace` (query): 命名空间
- `cluster_id` (query): 集群 ID
- `time_range` (query): 时间范围 (1h, 24h, 7d, 30d)
- `metrics` (query): 指标类型 (traffic, latency, errors)

**响应示例**:
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "endpoint_name": "nginx-service",
    "time_range": "24h",
    "metrics": {
      "traffic": {
        "total_requests": 1000000,
        "requests_per_second": 11.57,
        "bytes_transferred": "2.5GB"
      },
      "latency": {
        "average": 0.05,
        "p95": 0.12,
        "p99": 0.25
      },
      "errors": {
        "total_errors": 50,
        "error_rate": 0.005,
        "error_types": {
          "timeout": 30,
          "connection_refused": 20
        }
      }
    },
    "endpoint_performance": [
      {
        "timestamp": "2024-01-01T00:00:00Z",
        "requests_per_second": 12.5,
        "average_latency": 0.05,
        "error_rate": 0.005
      }
    ]
  }
}
```

### 2. Ingress 管理

#### 2.1 获取 Ingress 列表
```http
GET /api/k8s/ingress
```

**请求参数**:
- `cluster_id` (query): 集群 ID
- `namespace` (query): 命名空间
- `label_selector` (query): 标签选择器
- `limit` (query): 返回数量
- `offset` (query): 偏移量

**响应示例**:
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "ingresses": [
      {
        "id": 1,
        "name": "nginx-ingress",
        "namespace": "default",
        "cluster_id": 1,
        "class": "nginx",
        "rules": [
          {
            "host": "example.com",
            "http": {
              "paths": [
                {
                  "path": "/",
                  "path_type": "Prefix",
                  "backend": {
                    "service": {
                      "name": "nginx-service",
                      "port": {
                        "number": 80
                      }
                    }
                  }
                }
              ]
            }
          }
        ],
        "tls": [
          {
            "hosts": ["example.com"],
            "secret_name": "example-tls"
          }
        ],
        "status": {
          "load_balancer": {
            "ingress": [
              {
                "ip": "192.168.1.100"
              }
            ]
          }
        },
        "labels": {
          "app": "nginx"
        },
        "created_at": "2024-01-01T00:00:00Z"
      }
    ],
    "total": 5
  }
}
```

#### 2.2 创建 Ingress
```http
POST /api/k8s/ingress/create
```

**请求体**:
```json
{
  "name": "app-ingress",
  "namespace": "default",
  "cluster_id": 1,
  "class": "nginx",
  "rules": [
    {
      "host": "app.example.com",
      "http": {
        "paths": [
          {
            "path": "/api",
            "path_type": "Prefix",
            "backend": {
              "service": {
                "name": "api-service",
                "port": {
                  "number": 8080
                }
              }
            }
          },
          {
            "path": "/",
            "path_type": "Prefix",
            "backend": {
              "service": {
                "name": "web-service",
                "port": {
                  "number": 80
                }
              }
            }
          }
        ]
      }
    }
  ],
  "tls": [
    {
      "hosts": ["app.example.com"],
      "secret_name": "app-tls"
    }
  ],
  "annotations": {
    "nginx.ingress.kubernetes.io/rewrite-target": "/",
    "nginx.ingress.kubernetes.io/ssl-redirect": "true"
  }
}
```

#### 2.3 更新 Ingress
```http
PUT /api/k8s/ingress/{name}
```

#### 2.4 删除 Ingress
```http
DELETE /api/k8s/ingress/{name}
```

#### 2.5 获取 Ingress 状态
```http
GET /api/k8s/ingress/{name}/status
```

#### 2.6 SSL 证书管理
```http
POST /api/k8s/ingress/{name}/ssl
```

**请求体**:
```json
{
  "action": "update", // create, update, delete
  "tls_config": {
    "hosts": ["app.example.com"],
    "secret_name": "app-tls",
    "certificate": "-----BEGIN CERTIFICATE-----...",
    "private_key": "-----BEGIN PRIVATE KEY-----..."
  }
}
```

### 3. NetworkPolicy 管理

#### 3.1 获取 NetworkPolicy 列表
```http
GET /api/k8s/networkpolicies
```

#### 3.2 创建 NetworkPolicy
```http
POST /api/k8s/networkpolicies/create
```

**请求体**:
```json
{
  "name": "default-deny",
  "namespace": "default",
  "cluster_id": 1,
  "pod_selector": {
    "matchLabels": {
      "app": "web"
    }
  },
  "policy_types": ["Ingress", "Egress"],
  "ingress": [
    {
      "from": [
        {
          "pod_selector": {
            "matchLabels": {
              "app": "api"
            }
          }
        }
      ],
      "ports": [
        {
          "protocol": "TCP",
          "port": 80
        }
      ]
    }
  ],
  "egress": [
    {
      "to": [
        {
          "pod_selector": {
            "matchLabels": {
              "app": "database"
            }
          }
        }
      ],
      "ports": [
        {
          "protocol": "TCP",
          "port": 3306
        }
      ]
    }
  ]
}
```

#### 3.3 更新 NetworkPolicy
```http
PUT /api/k8s/networkpolicies/{name}
```

#### 3.4 删除 NetworkPolicy
```http
DELETE /api/k8s/networkpolicies/{name}
```

#### 3.5 获取 NetworkPolicy 状态
```http
GET /api/k8s/networkpolicies/{name}/status
```