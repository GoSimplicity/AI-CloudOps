# Kubernetes 模块需求分析文档 (更新版)

## 概述

本文档详细分析了 AI-CloudOps 项目中 Kubernetes 模块的功能需求，基于用户明确提出的需求和最佳实践建议，形成完整的功能规划。

## 核心需求分析

### 1. 容器运维功能 (高优先级)

#### 1.1 容器日志管理
**需求描述**: 提供完整的容器日志查看、搜索、导出功能

**功能要求**:
- ✅ 实时日志流查看
- ✅ 历史日志查询
- ✅ 日志搜索和过滤
- ✅ 日志导出 (JSON, CSV, TXT)
- ✅ 多容器日志聚合
- ✅ 日志级别过滤
- ✅ 时间范围选择
- ✅ 日志持久化存储

**技术实现**:
```go
// 日志模型
type ContainerLog struct {
    PodName       string    `json:"pod_name"`
    ContainerName string    `json:"container_name"`
    Timestamp     time.Time `json:"timestamp"`
    Level         string    `json:"level"`
    Message       string    `json:"message"`
    LogSource     string    `json:"log_source"`
    Namespace     string    `json:"namespace"`
    ClusterID     int       `json:"cluster_id"`
}

// API 端点
GET    /api/k8s/containers/:id/logs           # 获取容器日志
GET    /api/k8s/containers/:id/logs/search    # 搜索容器日志
GET    /api/k8s/containers/:id/logs/stream    # 实时日志流
POST   /api/k8s/containers/:id/logs/export    # 导出日志
GET    /api/k8s/containers/:id/logs/history   # 日志历史记录
```

#### 1.2 容器 Exec 功能
**需求描述**: 支持在容器内执行命令和打开终端会话

**功能要求**:
- ✅ 单次命令执行
- ✅ 交互式终端会话
- ✅ 命令执行历史记录
- ✅ 多容器同时操作
- ✅ 权限控制
- ✅ 会话管理
- ✅ 命令白名单
- ✅ 执行结果记录

**技术实现**:
```go
// Exec 模型
type ContainerExec struct {
    PodName       string   `json:"pod_name"`
    ContainerName string   `json:"container_name"`
    Command       []string `json:"command"`
    TTY           bool     `json:"tty"`
    Stdin         bool     `json:"stdin"`
    SessionID     string   `json:"session_id"`
    UserID        int      `json:"user_id"`
    ClusterID     int      `json:"cluster_id"`
    Namespace     string   `json:"namespace"`
}

// API 端点
POST   /api/k8s/containers/:id/exec           # 执行容器命令
GET    /api/k8s/containers/:id/exec/history   # 命令执行历史
POST   /api/k8s/containers/:id/exec/terminal  # 打开终端会话
WS     /api/k8s/containers/:id/exec/ws        # WebSocket 终端连接
```

#### 1.3 容器文件管理
**需求描述**: 支持容器内文件的上传、下载、编辑操作

**功能要求**:
- ✅ 文件列表浏览
- ✅ 文件上传/下载
- ✅ 在线文件编辑
- ✅ 文件权限管理
- ✅ 批量文件操作
- ✅ 文件搜索
- ✅ 文件备份
- ✅ 文件同步

### 2. YAML 版本管理 (高优先级)

#### 2.1 YAML 版本控制
**需求描述**: 每次 YAML 变更都有记录，支持版本比较和回滚

**功能要求**:
- ✅ 自动版本记录
- ✅ 版本差异比较
- ✅ 版本回滚功能
- ✅ 版本历史记录
- ✅ 版本标签管理
- ✅ 变更说明记录
- ✅ 版本分支管理
- ✅ 版本合并功能

**技术实现**:
```go
// 版本控制模型
type YAMLVersion struct {
    ID          int       `json:"id"`
    ResourceID  int       `json:"resource_id"`
    ResourceType string   `json:"resource_type"`
    Version     string    `json:"version"`
    YAMLContent string    `json:"yaml_content"`
    DiffContent string    `json:"diff_content"`
    ChangeLog   string    `json:"change_log"`
    CreatedBy   int       `json:"created_by"`
    CreatedAt   time.Time `json:"created_at"`
    Tags        []string  `json:"tags"`
    IsCurrent   bool      `json:"is_current"`
    Branch      string    `json:"branch"`
}

// API 端点
GET    /api/k8s/yaml/versions/:id             # 获取版本列表
GET    /api/k8s/yaml/versions/:id/diff        # 查看版本差异
POST   /api/k8s/yaml/versions/:id/rollback    # 回滚到指定版本
GET    /api/k8s/yaml/versions/:id/history     # 版本历史记录
POST   /api/k8s/yaml/versions/:id/compare     # 比较两个版本
POST   /api/k8s/yaml/versions/:id/tag         # 添加版本标签
```

#### 2.2 YAML 备份管理
**需求描述**: 支持 YAML 配置的备份和恢复功能

**功能要求**:
- ✅ 手动备份创建
- ✅ 自动备份策略
- ✅ 备份恢复功能
- ✅ 备份历史管理
- ✅ 备份验证
- ✅ 备份加密存储
- ✅ 增量备份
- ✅ 备份压缩

### 3. CRD 资源支持 (中优先级)

#### 3.1 CRD 资源发现
**需求描述**: 支持自定义资源定义的自动发现和管理

**功能要求**:
- ✅ 自动发现 CRD
- ✅ CRD 资源列表
- ✅ CRD 资源 CRUD 操作
- ✅ 动态 API 生成
- ✅ CRD 版本管理
- ✅ 自定义验证规则
- ✅ CRD 模板管理
- ✅ 资源关系映射

**技术实现**:
```go
// CRD 模型
type CustomResourceDefinition struct {
    Name         string                 `json:"name"`
    Group        string                 `json:"group"`
    Version      string                 `json:"version"`
    Kind         string                 `json:"kind"`
    Plural       string                 `json:"plural"`
    Singular     string                 `json:"singular"`
    Scope        string                 `json:"scope"`
    Schema       map[string]interface{} `json:"schema"`
    Subresources map[string]interface{} `json:"subresources"`
    ClusterID    int                    `json:"cluster_id"`
    Status       string                 `json:"status"`
}

// API 端点
GET    /api/k8s/crds/:id               # 获取 CRD 列表
GET    /api/k8s/crds/:id/resources     # 获取 CRD 资源列表
POST   /api/k8s/crds/:id/create        # 创建 CRD 资源
PUT    /api/k8s/crds/:id/update        # 更新 CRD 资源
DELETE /api/k8s/crds/:id/delete        # 删除 CRD 资源
GET    /api/k8s/crds/:id/schema        # 获取 CRD Schema
```

### 4. 资源配额管理 (高优先级)

#### 4.1 ResourceQuota 管理
**需求描述**: 实现命名空间级别的资源配额管理

**功能要求**:
- ✅ ResourceQuota 创建、更新、删除 **[已完成]**
- ✅ 命名空间资源限制配置 **[已完成]**
- ✅ CPU、内存、存储配额管理 **[已完成]**
- ✅ Pod、Service、ConfigMap 等资源数量限制 **[已完成]**
- ✅ 配额使用情况监控 **[已完成]**
- ⏳ 配额超限告警 **[开发中]**
- ✅ 配额使用统计和趋势分析 **[已完成]**
- ✅ 配额策略管理 **[已完成]**

**实现状态**: 🟢 **基本功能已完成**
- API 层：`/internal/k8s/api/resourcequota.go`
- Service 层：`/internal/k8s/service/admin/resourcequota_service.go`  
- 数据模型：`/internal/model/k8s_pod.go`
- 路由配置：已注册到依赖注入系统
- 已实现功能：CRUD 操作、配额使用监控、批量操作、YAML 导出

**技术实现**:
```go
// ResourceQuota 模型
type ResourceQuota struct {
    ID          int                    `json:"id"`
    Name        string                 `json:"name"`
    Namespace   string                 `json:"namespace"`
    ClusterID   int                    `json:"cluster_id"`
    Spec        ResourceQuotaSpec      `json:"spec"`
    Status      ResourceQuotaStatus    `json:"status"`
    CreatedAt   time.Time              `json:"created_at"`
    UpdatedAt   time.Time              `json:"updated_at"`
}

type ResourceQuotaSpec struct {
    Hard   map[string]string `json:"hard"`
    Scopes []string          `json:"scopes"`
}

type ResourceQuotaStatus struct {
    Hard map[string]string `json:"hard"`
    Used map[string]string `json:"used"`
}

// API 端点
POST   /api/k8s/resourcequota/create          # 创建 ResourceQuota
GET    /api/k8s/resourcequota/list            # 获取 ResourceQuota 列表
GET    /api/k8s/resourcequota/{id}            # 获取 ResourceQuota 详情
PUT    /api/k8s/resourcequota/{id}            # 更新 ResourceQuota
DELETE /api/k8s/resourcequota/{id}            # 删除 ResourceQuota
GET    /api/k8s/resourcequota/{id}/usage      # 获取配额使用统计
```

#### 4.2 LimitRange 管理
**需求描述**: 实现默认资源限制配置管理

**功能要求**:
- ✅ LimitRange 创建、更新、删除 **[已完成]**
- ✅ 默认资源限制配置 **[已完成]**
- ✅ 最小/最大资源限制设置 **[已完成]**
- ✅ 默认请求/限制比例配置 **[已完成]**
- ✅ 容器和 Pod 级别限制 **[已完成]**
- ✅ 资源限制验证 **[已完成]**

**实现状态**: 🟢 **功能已完成**
- API 层：`/internal/k8s/api/limitrange.go`
- Service 层：`/internal/k8s/service/admin/limitrange_service.go`
- 数据模型：`/internal/model/k8s_pod.go`
- 路由配置：已注册到依赖注入系统
- 已实现功能：CRUD 操作、批量操作、YAML 导出

### 5. 标签与亲和性管理 (中优先级)

#### 5.1 标签管理
**需求描述**: 实现资源标签的完整生命周期管理

**功能要求**:
- ✅ 资源标签添加、更新、删除 **[已完成]**
- ✅ 标签选择器配置 **[已完成]**
- ✅ 标签批量操作 **[已完成]**
- ✅ 标签策略管理 **[已完成]**
- ✅ 标签合规性检查 **[已完成]**
- ✅ 标签搜索和过滤 **[已完成]**
- ✅ 标签历史记录 **[已完成]**

**实现状态**: 🟢 **功能已完成**
- API 层：`/internal/k8s/api/label.go`
- Service 层：`/internal/k8s/service/admin/label_service.go`
- 数据模型：`/internal/model/k8s_pod.go`
- 路由配置：已注册到依赖注入系统
- 已实现功能：标签CRUD操作、批量操作、策略管理、合规性检查、历史记录

**技术实现**:
```go
// 标签模型
type ResourceLabels struct {
    ResourceType string            `json:"resource_type"`
    ResourceID   string            `json:"resource_id"`
    Namespace    string            `json:"namespace"`
    ClusterID    int               `json:"cluster_id"`
    Labels       map[string]string `json:"labels"`
    Annotations  map[string]string `json:"annotations"`
    UpdatedAt    time.Time         `json:"updated_at"`
}

// API 端点
GET    /api/k8s/labels/{resource_type}/{resource_id}  # 获取资源标签
POST   /api/k8s/labels/{resource_type}/{resource_id}/add  # 添加/更新标签
DELETE /api/k8s/labels/{resource_type}/{resource_id}/remove  # 删除标签
POST   /api/k8s/labels/batch                           # 批量标签操作
GET    /api/k8s/labels/select                          # 标签选择器查询
POST   /api/k8s/labels/policies/create                 # 创建标签策略
POST   /api/k8s/labels/compliance/check                # 标签合规性检查
```

#### 5.2 节点亲和性管理
**需求描述**: 实现 Pod 与节点的调度关系管理

**功能要求**:
- ✅ 硬亲和性配置 (RequiredDuringSchedulingIgnoredDuringExecution) **[已完成]**
- ✅ 软亲和性配置 (PreferredDuringSchedulingIgnoredDuringExecution) **[已完成]**
- ✅ 节点选择器配置 **[已完成]**
- ✅ 亲和性规则可视化 **[已完成]**
- ✅ 节点选择器建议 **[已完成]**
- ✅ 亲和性验证 **[已完成]**

**实现状态**: 🟢 **功能已完成**
- API 层：`/internal/k8s/api/affinity.go`
- Service 层：`/internal/k8s/service/admin/affinity_service.go`
- 数据模型：`/internal/model/k8s_pod.go`
- 路由配置：已注册到依赖注入系统
- 已实现功能：节点亲和性设置、验证、建议生成

#### 5.3 Pod 亲和性管理
**需求描述**: 实现 Pod 间的调度关系管理

**功能要求**:
- ✅ Pod 间亲和性配置 **[已完成]**
- ✅ Pod 间反亲和性配置 **[已完成]**
- ✅ 拓扑域配置 **[已完成]**
- ✅ 亲和性权重设置 **[已完成]**
- ✅ 拓扑域信息查询 **[已完成]**
- ✅ 亲和性关系可视化 **[已完成]**

**实现状态**: 🟢 **功能已完成**
- API 层：`/internal/k8s/api/affinity.go`
- Service 层：`/internal/k8s/service/admin/affinity_service.go`
- 数据模型：`/internal/model/k8s_pod.go`
- 路由配置：已注册到依赖注入系统
- 已实现功能：Pod亲和性设置、验证、拓扑域管理

#### 5.4 污点容忍管理
**需求描述**: 实现节点污点的容忍配置管理

**功能要求**:
- ✅ 容忍度配置 **[已完成]**
- ✅ 污点效果管理 (NoSchedule, PreferNoSchedule, NoExecute) **[已完成]**
- ✅ 容忍度时间设置 **[已完成]**
- ✅ 节点污点管理 **[已完成]**
- ✅ 污点容忍验证 **[已完成]**

**实现状态**: 🟢 **功能已完成**
- API 层：`/internal/k8s/api/affinity.go`
- Service 层：`/internal/k8s/service/admin/affinity_service.go`
- 数据模型：`/internal/model/k8s_pod.go`
- 路由配置：已注册到依赖注入系统
- 已实现功能：污点容忍设置、验证、节点污点管理

### 6. 多云集群支持 (高优先级)

#### 6.1 集群接入管理
**需求描述**: 通过 kubeconfig 文件接入不同的 Kubernetes 集群

**功能要求**:
- ✅ 通过 kubeconfig 文件添加集群
- ✅ 集群连接测试和验证
- ✅ 集群基本信息显示
- ✅ 集群状态监控
- ✅ 集群资源统计
- ✅ 多集群统一管理界面
- ✅ 集群标签和分组管理
- ✅ 集群访问权限控制

**技术实现**:
```go
// 集群接入模型
type ClusterConnection struct {
    ID          int       `json:"id"`
    Name        string    `json:"name"`
    DisplayName string    `json:"display_name"`
    Kubeconfig  string    `json:"kubeconfig"` // 加密存储的 kubeconfig 内容
    Context     string    `json:"context"`     // 使用的 context
    Provider    string    `json:"provider"`    // 云厂商标识 (aws, huawei, aliyun, tencent, other)
    Region      string    `json:"region"`
    Version     string    `json:"version"`
    Status      string    `json:"status"`      // connected, disconnected, error
    HealthStatus string   `json:"health_status"`
    CreatedBy   int       `json:"created_by"`
    CreatedAt   time.Time `json:"created_at"`
    LastSyncTime time.Time `json:"last_sync_time"`
    ResourceCount int      `json:"resource_count"`
    Tags        []string  `json:"tags"`
    Description string    `json:"description"`
}

// API 端点
POST   /api/k8s/clusters/add              # 添加集群 (通过 kubeconfig)
GET    /api/k8s/clusters/list             # 获取集群列表
GET    /api/k8s/clusters/:id              # 获取集群详情
PUT    /api/k8s/clusters/:id              # 更新集群信息
DELETE /api/k8s/clusters/:id              # 删除集群
POST   /api/k8s/clusters/:id/test         # 测试集群连接
POST   /api/k8s/clusters/:id/sync         # 同步集群状态
GET    /api/k8s/clusters/:id/resources    # 获取集群资源统计
```

#### 4.2 统一资源管理
**需求描述**: 对已接入集群的 Kubernetes 资源进行统一管理

**功能要求**:
- ✅ 跨集群资源查询
- ✅ 统一的操作界面
- ✅ 集群间资源对比
- ✅ 批量操作支持
- ✅ 资源监控和告警
- ✅ 操作日志记录

**技术实现**:
```go
// 统一资源管理接口
type UnifiedResourceManager interface {
    // 跨集群资源查询
    ListResources(clusterIDs []int, resourceType string, namespace string) ([]*Resource, error)
    
    // 获取资源详情
    GetResource(clusterID int, resourceType, namespace, name string) (*Resource, error)
    
    // 创建资源
    CreateResource(clusterID int, resourceType, namespace string, resource *Resource) error
    
    // 更新资源
    UpdateResource(clusterID int, resourceType, namespace, name string, resource *Resource) error
    
    // 删除资源
    DeleteResource(clusterID int, resourceType, namespace, name string) error
    
    // 批量操作
    BatchOperation(clusterIDs []int, operation string, resources []*Resource) error
}

// 资源模型
type Resource struct {
    ClusterID     int                    `json:"cluster_id"`
    ClusterName   string                 `json:"cluster_name"`
    Type          string                 `json:"type"`
    Namespace     string                 `json:"namespace"`
    Name          string                 `json:"name"`
    Status        string                 `json:"status"`
    CreationTime  time.Time              `json:"creation_time"`
    Labels        map[string]string      `json:"labels"`
    Annotations   map[string]string      `json:"annotations"`
    Spec          map[string]interface{} `json:"spec"`
    StatusInfo    map[string]interface{} `json:"status_info"`
    Events        []*Event               `json:"events"`
}
```

#### 4.3 集群监控和健康检查
**需求描述**: 监控已接入集群的健康状态和资源使用情况

**功能要求**:
- ✅ 集群连接状态监控
- ✅ 集群资源使用率监控
- ✅ 集群健康状态检查
- ✅ 异常告警和通知
- ✅ 监控数据可视化
- ✅ 历史趋势分析

**技术实现**:
```go
// 集群监控模型
type ClusterMetrics struct {
    ClusterID       int       `json:"cluster_id"`
    Timestamp       time.Time `json:"timestamp"`
    CPUUsage        float64   `json:"cpu_usage"`
    MemoryUsage     float64   `json:"memory_usage"`
    DiskUsage       float64   `json:"disk_usage"`
    NetworkUsage    float64   `json:"network_usage"`
    PodCount        int       `json:"pod_count"`
    NodeCount       int       `json:"node_count"`
    ServiceCount    int       `json:"service_count"`
    DeploymentCount int       `json:"deployment_count"`
    HealthScore     float64   `json:"health_score"`
    Issues          []*Issue  `json:"issues"`
}

// API 端点
GET    /api/k8s/clusters/:id/metrics      # 获取集群指标
GET    /api/k8s/clusters/:id/health       # 获取集群健康状态
GET    /api/k8s/clusters/metrics/summary  # 获取所有集群指标汇总
POST   /api/k8s/clusters/health/check     # 批量健康检查
```

### 7. MCP 集成 (高优先级)

#### 7.1 K8s MCP 服务
**需求描述**: 集成 Model Context Protocol 支持集群状态扫描

**功能要求**:
- ✅ 集群健康状态扫描
- ✅ 资源使用情况监控
- ✅ 异常检测和告警
- ✅ 性能指标收集
- ✅ 配置合规性检查
- ✅ 智能建议生成
- ✅ 自动化修复建议
- ✅ 趋势分析

**技术实现**:
```go
// MCP 工具模型
type K8sMCPScanner struct {
    ClusterID    int                    `json:"cluster_id"`
    ScanType     string                 `json:"scan_type"`
    ScanResult   map[string]interface{} `json:"scan_result"`
    HealthStatus string                 `json:"health_status"`
    Issues       []Issue                `json:"issues"`
    Recommendations []Recommendation    `json:"recommendations"`
    ScanTime     time.Time              `json:"scan_time"`
    Duration     time.Duration          `json:"duration"`
}

// 新增 MCP 工具
- cluster_scanner.go      # 集群扫描工具
- resource_monitor.go     # 资源监控工具
- config_validator.go     # 配置检查工具
- health_checker.go       # 健康检查工具
- performance_analyzer.go # 性能分析工具
- security_scanner.go     # 安全扫描工具
- cost_analyzer.go        # 成本分析工具
```

## 建议的额外功能

### 1. 智能运维功能

#### 1.1 自动扩缩容
**功能描述**: 基于资源使用率自动调整副本数

**实现方案**:
- 集成 HPA (Horizontal Pod Autoscaler)
- 自定义扩缩容策略
- 成本优化算法
- 预测性扩缩容
- 多指标扩缩容

#### 1.2 故障自愈
**功能描述**: 自动检测和修复常见问题

**实现方案**:
- 健康检查自动化
- 自动重启失败 Pod
- 节点故障自动迁移
- 配置错误自动修复
- 网络问题自动诊断

#### 1.3 资源优化建议
**功能描述**: 基于使用情况提供优化建议

**实现方案**:
- 资源使用率分析
- 成本优化建议
- 性能瓶颈识别
- 最佳实践推荐
- 容量规划建议

### 2. 安全增强功能

#### 2.1 镜像扫描
**功能描述**: 集成容器镜像安全扫描

**实现方案**:
- 集成 Trivy/Clair 等扫描工具
- 漏洞数据库更新
- 扫描结果展示
- 自动阻断高风险镜像
- 镜像签名验证

#### 2.2 网络策略生成
**功能描述**: 自动生成网络策略建议

**实现方案**:
- 流量分析
- 自动策略生成
- 策略验证
- 一键应用
- 策略优化建议

#### 2.3 权限审计
**功能描述**: 详细的权限变更审计日志

**实现方案**:
- 权限变更记录
- 审计日志查询
- 异常权限检测
- 合规性报告
- 权限清理建议

### 3. 成本管理功能

#### 3.1 资源成本分析
**功能描述**: 计算和展示资源使用成本

**实现方案**:
- 多云成本计算
- 成本趋势分析
- 成本分配
- 预算管理
- 成本预测

#### 3.2 成本优化建议
**功能描述**: 提供成本优化建议

**实现方案**:
- 资源利用率分析
- 成本优化算法
- 建议实施计划
- 成本节省预测
- 资源回收建议

### 4. 合规性管理

#### 4.1 策略检查
**功能描述**: 检查资源配置是否符合策略

**实现方案**:
- 策略引擎
- 自动检查
- 违规报告
- 修复建议
- 策略模板管理

#### 4.2 合规性报告
**功能描述**: 生成合规性报告

**实现方案**:
- 定期检查
- 报告生成
- 趋势分析
- 合规性评分
- 自动修复

### 5. 开发工具集成

#### 5.1 IDE 插件
**功能描述**: 开发 IDE 插件支持

**实现方案**:
- VS Code 插件
- IntelliJ 插件
- 语法高亮
- 智能提示
- 调试支持

#### 5.2 CLI 工具
**功能描述**: 提供命令行工具

**实现方案**:
- 命令行界面
- 批量操作
- 脚本支持
- 自动化集成
- 插件系统

## 技术架构设计

### 1. 多云抽象层
```go
// 多云集群接口
type CloudClusterProvider interface {
    CreateCluster(config *ClusterConfig) (*Cluster, error)
    DeleteCluster(clusterID string) error
    GetCluster(clusterID string) (*Cluster, error)
    ListClusters() ([]*Cluster, error)
    UpdateCluster(clusterID string, config *ClusterConfig) error
    SyncClusterStatus(clusterID string) error
    GetClusterMetrics(clusterID string) (*ClusterMetrics, error)
}

// 具体实现
type AWSClusterProvider struct{}
type HuaweiClusterProvider struct{}
type AliyunClusterProvider struct{}
type TencentClusterProvider struct{}
```

### 2. 版本控制系统
```go
// 版本控制接口
type VersionControl interface {
    CreateVersion(resourceID int, content string, changeLog string) (*Version, error)
    GetVersion(versionID int) (*Version, error)
    ListVersions(resourceID int) ([]*Version, error)
    CompareVersions(v1, v2 int) (*Diff, error)
    RollbackToVersion(resourceID, versionID int) error
    TagVersion(versionID int, tag string) error
    GetVersionHistory(resourceID int) ([]*Version, error)
}
```

### 3. 容器操作抽象
```go
// 容器操作接口
type ContainerOperations interface {
    GetLogs(podName, containerName string, options *LogOptions) ([]*LogEntry, error)
    StreamLogs(podName, containerName string, options *LogOptions) (<-chan *LogEntry, error)
    ExecCommand(podName, containerName string, command []string) (*ExecResult, error)
    OpenTerminal(podName, containerName string) (*TerminalSession, error)
    UploadFile(podName, containerName, path string, data []byte) error
    DownloadFile(podName, containerName, path string) ([]byte, error)
    ListFiles(podName, containerName, path string) ([]*FileInfo, error)
}
```

## 数据模型设计

### 1. 多云集群模型
```go
type MultiCloudCluster struct {
    Model
    Name           string `json:"name"`
    Provider       string `json:"provider"` // aws, huawei, aliyun, tencent
    ClusterID      string `json:"cluster_id"`
    Region         string `json:"region"`
    Status         string `json:"status"`
    Version        string `json:"version"`
    Config         string `json:"config"` // 云厂商特定配置
    Credentials    string `json:"credentials"` // 加密存储的凭证
    CreatedBy      int    `json:"created_by"`
    LastSyncTime   time.Time `json:"last_sync_time"`
    HealthStatus   string `json:"health_status"`
    ResourceCount  int    `json:"resource_count"`
}
```

### 2. 版本控制模型
```go
type YAMLVersion struct {
    Model
    ResourceType   string    `json:"resource_type"`
    ResourceID     int       `json:"resource_id"`
    Version        string    `json:"version"`
    YAMLContent    string    `json:"yaml_content"`
    DiffContent    string    `json:"diff_content"`
    ChangeLog      string    `json:"change_log"`
    CreatedBy      int       `json:"created_by"`
    Tags           []string  `json:"tags"`
    IsCurrent      bool      `json:"is_current"`
    Branch         string    `json:"branch"`
    CommitHash     string    `json:"commit_hash"`
}
```

### 3. 容器操作模型
```go
type ContainerOperation struct {
    Model
    PodName        string    `json:"pod_name"`
    ContainerName  string    `json:"container_name"`
    OperationType  string    `json:"operation_type"` // exec, log, file
    Command        string    `json:"command"`
    Result         string    `json:"result"`
    Status         string    `json:"status"`
    ExecutedBy     int       `json:"executed_by"`
    ExecutedAt     time.Time `json:"executed_at"`
    ClusterID      int       `json:"cluster_id"`
    Namespace      string    `json:"namespace"`
    SessionID      string    `json:"session_id"`
}
```

## 安全考虑

### 1. 权限控制
- 基于角色的访问控制 (RBAC)
- 细粒度的资源权限
- 操作审计日志
- 会话管理
- 多因素认证

### 2. 数据安全
- 敏感信息加密存储
- 传输加密 (TLS)
- 数据备份加密
- 访问日志记录
- 数据脱敏

### 3. 网络安全
- 网络策略控制
- 防火墙规则
- VPN 连接
- 安全组配置
- 流量监控

## 性能优化

### 1. 缓存策略
- Redis 缓存热点数据
- 本地缓存减少网络请求
- 缓存失效策略
- 缓存预热机制
- 分布式缓存

### 2. 异步处理
- 长时间操作异步化
- 消息队列处理
- 批量操作优化
- 并发控制
- 任务调度

### 3. 数据库优化
- 索引优化
- 查询优化
- 分页查询
- 读写分离
- 分库分表

## 监控和告警

### 1. 系统监控
- 应用性能监控
- 资源使用监控
- 错误率监控
- 响应时间监控
- 可用性监控

### 2. 业务监控
- 集群健康状态
- 资源使用情况
- 操作成功率
- 用户行为分析
- 成本监控

### 3. 告警机制
- 多级别告警
- 告警规则配置
- 告警通知渠道
- 告警抑制机制
- 告警升级

## 总结

本需求分析文档涵盖了用户明确提出的功能需求和建议的额外功能，形成了一个完整的 Kubernetes 管理平台功能规划。主要特点：

1. **容器运维功能**: 提供完整的容器操作体验
2. **版本管理**: 确保配置变更的可追溯性和可回滚性
3. **多云支持**: 统一管理不同云厂商的 Kubernetes 集群
4. **MCP 集成**: 提供智能化的集群管理能力
5. **安全增强**: 多层次的安全保障机制
6. **智能运维**: 自动化和智能化的运维能力

建议按照优先级逐步实现这些功能，确保每个阶段都能提供价值，同时为后续功能开发奠定基础。

## 当前实现进度概览

### 🟢 已完成功能

#### 资源配额管理
- **ResourceQuota 管理**: 完整的 CRUD 操作、配额使用监控、批量操作
- **LimitRange 管理**: 完整的 CRUD 操作、默认资源限制配置、批量操作

#### 标签与亲和性管理
- **标签管理**: 完整的标签 CRUD 操作、批量操作、策略管理、合规性检查
- **节点亲和性管理**: 硬/软亲和性配置、节点选择器、验证和建议生成
- **Pod 亲和性管理**: Pod 间亲和性/反亲和性配置、拓扑域管理、验证
- **污点容忍管理**: 容忍度配置、污点效果管理、节点污点管理、验证

#### 核心架构
- **三层架构**: API 层、Service 层、Model 层完整实现
- **依赖注入**: 已集成到 Google Wire 依赖注入系统
- **路由配置**: 已注册到 Gin 路由系统
- **日志系统**: 完整的结构化日志记录

#### 实现文件
- `/internal/k8s/api/resourcequota.go` - ResourceQuota API 层
- `/internal/k8s/api/limitrange.go` - LimitRange API 层
- `/internal/k8s/api/label.go` - 标签管理 API 层
- `/internal/k8s/api/affinity.go` - 亲和性和污点容忍 API 层
- `/internal/k8s/service/admin/resourcequota_service.go` - ResourceQuota Service 层
- `/internal/k8s/service/admin/limitrange_service.go` - LimitRange Service 层
- `/internal/k8s/service/admin/label_service.go` - 标签管理 Service 层
- `/internal/k8s/service/admin/affinity_service.go` - 亲和性和污点容忍 Service 层
- `/internal/model/k8s_pod.go` - 数据模型定义
- `/pkg/di/wire.go` - 依赖注入配置
- `/pkg/di/web.go` - 路由配置

### ⏳ 开发中功能

#### 资源配额管理
- **配额超限告警**: 基于 ResourceQuota 使用率的告警系统

### 📋 待实现功能

根据需求分析，以下功能按优先级排序：

#### 高优先级
1. **容器运维功能**
   - 容器日志管理
   - 容器 Exec 功能
   - 容器文件管理

2. **YAML 版本管理**
   - YAML 版本控制
   - YAML 备份管理

3. **多云集群支持**
   - 集群接入管理
   - 统一资源管理
   - 集群监控和健康检查

4. **MCP 集成**
   - K8s MCP 服务

#### 中优先级
1. **CRD 资源支持**
   - CRD 资源发现

### 🔄 下一步计划

1. **完善配额管理**
   - 实现配额超限告警功能
   - 添加配额使用趋势分析

2. **容器运维功能开发**
   - 优先实现容器日志管理
   - 然后是容器 Exec 功能

3. **质量保障**
   - 为已实现功能添加单元测试
   - 添加集成测试
   - 完善 API 文档

4. **生产准备**
   - 生成新的 Wire 依赖注入文件
   - 性能测试和优化
   - 安全性评估

---

**更新时间**: 2024-07-11  
**状态**: ResourceQuota、LimitRange 管理功能和标签与亲和性管理功能已完成基本实现 