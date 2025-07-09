# Kubernetes 故障诊断与自动修复指南

## 概述

AI-CloudOps 的 Kubernetes 自动修复模块是一个智能化的故障诊断和修复系统，能够自动识别、分析和修复 Kubernetes 集群中的常见问题。

## 🔧 支持的修复场景

### 1. Pod 启动失败问题

#### CrashLoopBackOff
- **问题识别**: Pod 反复重启，处于 CrashLoopBackOff 状态
- **常见原因**: 
  - 健康检查配置错误
  - 资源限制设置过低
  - 探针路径不存在
  - 容器启动命令错误

#### ImagePullBackOff
- **问题识别**: 镜像拉取失败
- **常见原因**: 
  - 镜像不存在或标签错误
  - 私有仓库认证失败
  - 网络连接问题

#### Pending 状态
- **问题识别**: Pod 长时间处于 Pending 状态
- **常见原因**: 
  - 资源不足（CPU、内存）
  - 节点选择器不匹配
  - 污点和容忍度配置问题

### 2. 健康检查优化

#### Readiness Probe 配置
```yaml
readinessProbe:
  httpGet:
    path: /health
    port: 8080
  initialDelaySeconds: 30
  periodSeconds: 10
  timeoutSeconds: 5
  failureThreshold: 3
```

#### Liveness Probe 配置
```yaml
livenessProbe:
  httpGet:
    path: /health
    port: 8080
  initialDelaySeconds: 60
  periodSeconds: 30
  timeoutSeconds: 10
  failureThreshold: 3
```

### 3. 资源配置优化

#### CPU 和内存优化
```yaml
resources:
  requests:
    cpu: "100m"
    memory: "128Mi"
  limits:
    cpu: "500m"
    memory: "512Mi"
```

## 🤖 自动修复流程

### 1. 问题检测
- 监控 Pod 状态变化
- 分析 Kubernetes 事件
- 收集容器日志信息

### 2. 根因分析
- 基于 LLM 的智能分析
- 结合历史问题库
- 多维度关联分析

### 3. 修复方案生成
- 自动生成修复配置
- 计算修复置信度
- 提供多种修复选项

### 4. 修复执行
- 应用配置变更
- 监控修复效果
- 回滚机制保障

## 📚 典型修复案例

### 案例1: Nginx 部署健康检查修复

**问题描述**: nginx-deployment 的 Pod 无法启动，健康检查失败

**修复前配置**:
```yaml
livenessProbe:
  httpGet:
    path: /status  # 错误路径
    port: 80
  initialDelaySeconds: 10  # 延迟太短
  periodSeconds: 5         # 检查频率过高
  failureThreshold: 1      # 失败阈值太低
```

**修复后配置**:
```yaml
livenessProbe:
  httpGet:
    path: /
    port: 80
  initialDelaySeconds: 30
  periodSeconds: 10
  failureThreshold: 3
readinessProbe:
  httpGet:
    path: /
    port: 80
  initialDelaySeconds: 10
  periodSeconds: 5
  failureThreshold: 3
```

### 案例2: Spring Boot 应用资源优化

**问题描述**: Spring Boot 应用 Pod 频繁重启，内存不足

**修复前配置**:
```yaml
resources:
  requests:
    memory: "64Mi"    # 内存请求过低
    cpu: "50m"
  limits:
    memory: "128Mi"   # 内存限制过低
    cpu: "200m"
```

**修复后配置**:
```yaml
resources:
  requests:
    memory: "256Mi"
    cpu: "100m"
  limits:
    memory: "512Mi"
    cpu: "500m"
```

### 案例3: 多因素问题综合修复

**问题描述**: 应用同时存在健康检查和资源配置问题

**修复策略**:
1. 调整健康检查配置
2. 优化资源分配
3. 添加启动探针
4. 配置优雅关闭

## 🔍 诊断工具和命令

### 1. 基础诊断命令

```bash
# 查看 Pod 状态
kubectl get pods -n <namespace>

# 查看 Pod 详细信息
kubectl describe pod <pod-name> -n <namespace>

# 查看 Pod 日志
kubectl logs <pod-name> -n <namespace>

# 查看 Deployment 状态
kubectl get deployment <deployment-name> -n <namespace>

# 查看 Events
kubectl get events -n <namespace> --sort-by=.metadata.creationTimestamp
```

### 2. 高级诊断

```bash
# 查看资源使用情况
kubectl top pods -n <namespace>

# 查看节点资源
kubectl top nodes

# 查看网络策略
kubectl get networkpolicies -n <namespace>

# 查看服务端点
kubectl get endpoints -n <namespace>
```

### 3. 自动修复 API 调用

```bash
# 自动修复部署
curl -X POST http://localhost:8080/api/v1/autofix \
  -H "Content-Type: application/json" \
  -d '{
    "deployment": "nginx-deployment",
    "namespace": "default",
    "event": "Pod启动失败，CrashLoopBackOff状态",
    "force": true
  }'

# 诊断集群状态
curl -X POST http://localhost:8080/api/v1/autofix/diagnose \
  -H "Content-Type: application/json" \
  -d '{
    "namespace": "default"
  }'
```

## ⚙️ 配置和部署

### 1. 权限配置

创建 ServiceAccount 和 RBAC 权限：

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: aiops-autofix
  namespace: default
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: aiops-autofix
rules:
- apiGroups: [""]
  resources: ["pods", "services", "endpoints"]
  verbs: ["get", "list", "watch"]
- apiGroups: ["apps"]
  resources: ["deployments", "replicasets"]
  verbs: ["get", "list", "watch", "update", "patch"]
- apiGroups: [""]
  resources: ["events"]
  verbs: ["get", "list", "watch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: aiops-autofix
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: aiops-autofix
subjects:
- kind: ServiceAccount
  name: aiops-autofix
  namespace: default
```

### 2. 环境变量配置

```bash
# Kubernetes 配置
export KUBECONFIG=/path/to/kubeconfig
export K8S_CONFIG_PATH=/path/to/kubeconfig
export K8S_IN_CLUSTER=false

# 自动修复配置
export AUTOFIX_ENABLED=true
export AUTOFIX_DRY_RUN=false
export AUTOFIX_BACKUP_ENABLED=true
```

### 3. 安全配置

```yaml
security:
  autofix:
    enabled: true
    dry_run: false
    backup_enabled: true
    max_replicas: 50
    allowed_namespaces: ["default", "staging"]
    forbidden_namespaces: ["kube-system", "kube-public"]
```

## 📊 监控和告警

### 1. 关键指标

- **修复成功率**: 自动修复任务的成功率
- **修复时间**: 从问题检测到修复完成的时间
- **回滚次数**: 修复失败后的回滚操作次数
- **覆盖率**: 能够自动修复的问题类型覆盖率

### 2. 告警规则

```yaml
# Prometheus 告警规则
groups:
- name: aiops-autofix
  rules:
  - alert: AutoFixHighFailureRate
    expr: (autofix_failures_total / autofix_attempts_total) > 0.2
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "自动修复失败率过高"
      description: "过去5分钟内自动修复失败率超过20%"
```

### 3. 日志监控

```bash
# 查看自动修复日志
kubectl logs -l app=aiops-backend -n aiops-system | grep "autofix"

# 查看修复结果
kubectl get events --field-selector reason=AutoFixApplied
```

## 🚨 故障排除

### 1. 常见错误

#### 权限不足
```
Error: deployments.apps "nginx-deployment" is forbidden: User "system:serviceaccount:default:aiops-autofix" cannot patch resource "deployments" in API group "apps"
```

**解决方案**: 检查 RBAC 权限配置

#### 配置文件错误
```
Error: invalid configuration: no configuration has been provided
```

**解决方案**: 检查 KUBECONFIG 环境变量

### 2. 调试技巧

```bash
# 启用调试模式
export LOG_LEVEL=DEBUG

# 查看详细日志
kubectl logs -f aiops-backend --tail=100

# 测试 API 连接
curl -X GET http://localhost:8080/api/v1/autofix/health
```

### 3. 性能优化

- **并发控制**: 限制同时执行的修复任务数量
- **缓存机制**: 缓存 Pod 状态和配置信息
- **批量处理**: 批量处理同类型问题

## 🔄 最佳实践

### 1. 修复策略

- **渐进式修复**: 从低风险修复开始，逐步升级
- **备份机制**: 修复前自动备份原始配置
- **监控验证**: 修复后持续监控应用状态

### 2. 安全考虑

- **权限最小化**: 只给必要的 Kubernetes 权限
- **命名空间隔离**: 限制修复范围
- **审计日志**: 记录所有修复操作

### 3. 团队协作

- **通知机制**: 及时通知相关人员
- **知识共享**: 将修复经验加入知识库
- **持续改进**: 定期评估和优化修复规则

---

*本文档涵盖了 AI-CloudOps Kubernetes 自动修复的核心功能和使用方法，更多详细信息请参考 API 文档和源代码。*