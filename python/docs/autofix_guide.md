# Kubernetes 故障自动修复模块使用指南

## 概述

Kubernetes 故障自动修复模块是 AI-CloudOps 系统中的核心组件，旨在自动检测、诊断和修复 Kubernetes 集群中的常见问题，减少人工干预，提高系统可靠性。该模块利用人工智能技术分析集群问题，并应用适当的修复策略。

## 主要特性

- **自动诊断**: 识别常见的 Kubernetes 问题，如健康检查配置不当、资源限制不合理等
- **智能修复**: 基于问题分析自动生成和应用修复策略
- **修复验证**: 执行修复操作后验证修复效果
- **工作流协作**: 多个智能 Agent 协作完成复杂修复任务
- **通知机制**: 在重要事件发生时通知运维人员

## 架构设计

自动修复系统由多个组件协同工作：

```
┌──────────────────┐      ┌──────────────────┐      ┌──────────────────┐
│                  │      │                  │      │                  │
│  SupervisorAgent │◄────►│   K8sFixerAgent  │◄────►│ KubernetesService│
│                  │      │                  │      │                  │
└────────┬─────────┘      └──────────────────┘      └──────────────────┘
         │
         │
         ▼
┌──────────────────┐
│                  │
│   NotifierAgent  │
│                  │
└──────────────────┘
```

- **SupervisorAgent**: 控制修复工作流程，协调各 Agent 的行动
- **K8sFixerAgent**: 执行具体的 Kubernetes 问题分析和修复操作
- **KubernetesService**: 提供与 Kubernetes API 交互的接口
- **NotifierAgent**: 处理修复过程中的通知发送

## 可检测和修复的问题

### Pod 健康检查问题

- ReadinessProbe 配置错误
- 探针路径不存在
- 探针频率过高/过低
- 失败阈值不合理

### 资源配置问题

- 资源请求过高，导致 Pod 无法被调度
- 内存/CPU 限制不合理，导致容器被终止
- 资源配置与应用实际需求不符

### 其他问题

- 镜像拉取失败
- 配置文件语法错误
- 存储卷配置问题
- 权限问题

## 使用方法

### 配置要求

在使用自动修复模块前，确保：

1. Kubernetes 配置文件正确设置
2. 环境变量已正确配置：

   ```bash
   export KUBECONFIG=/path/to/kubeconfig
   export K8S_CONFIG_PATH=/path/to/kubeconfig
   ```

3. 应用运行环境变量：
   ```bash
   export PYTHONPATH=$(pwd)
   ```

### API 接口说明

#### 1. 自动修复部署

**端点**: `POST /api/v1/autofix`

**请求体**:

```json
{
  "deployment": "nginx-deployment",
  "namespace": "default",
  "event": "Pod启动失败的问题描述",
  "force": true
}
```

**参数说明**:

- `deployment`: 要修复的部署名称
- `namespace`: 部署所在的命名空间
- `event`: 问题描述，用于 AI 分析
- `force`: 是否强制执行修复，即使风险较高

**响应**:

```json
{
  "status": "success",
  "result": "修复结果描述",
  "deployment": "nginx-deployment",
  "namespace": "default",
  "actions_taken": ["修改了readinessProbe配置", "调整了资源限制"],
  "timestamp": "2025-07-03T03:48:23.234063",
  "success": true
}
```

#### 2. 集群健康诊断

**端点**: `POST /api/v1/autofix/diagnose`

**请求体**:

```json
{
  "namespace": "default"
}
```

**参数说明**:

- `namespace`: 要诊断的命名空间，可选

**响应**:

```json
{
  "status": "success",
  "namespace": "default",
  "diagnosis": "诊断结果...",
  "timestamp": "2025-07-03T03:48:23.252909"
}
```

#### 3. 执行完整修复工作流

**端点**: `POST /api/v1/autofix/workflow`

**请求体**:

```json
{
  "problem_description": "详细的问题描述，用于工作流分析"
}
```

**参数说明**:

- `problem_description`: 问题的详细描述，越具体越好

**响应**:
包含工作流执行的详细步骤和结果。

### 实际使用示例

#### 示例 1: 修复健康检查问题

以下是一个自动修复 Nginx 部署健康检查问题的示例：

```bash
# 创建有问题的部署
kubectl apply -f data/sample/problematic-deployment.yaml

# 验证Pod无法启动
kubectl get pods -l app=nginx-problematic

# 调用自动修复API
curl -X POST http://localhost:8080/api/v1/autofix \
  -H "Content-Type: application/json" \
  -d '{
    "deployment": "nginx-problematic",
    "namespace": "default",
    "event": "Pod启动失败，原因是readinessProbe探针路径(/health)不存在，探针频率过高，且资源请求过高",
    "force": true
  }'

# 验证修复效果
kubectl get pods -l app=nginx-problematic
```

#### 示例 2: 集群健康诊断

```bash
# 执行集群诊断
curl -X POST http://localhost:8080/api/v1/autofix/diagnose \
  -H "Content-Type: application/json" \
  -d '{
    "namespace": "default"
  }'
```

## 测试

项目包含一个自动化测试脚本`tests/test-autofix.py`，用于验证自动修复功能。该脚本会测试各 API 端点，并验证修复效果。

### 运行测试

```bash
# 确保环境变量已设置
export KUBECONFIG=deploy/kubernetes/config
export PYTHONPATH=$(pwd)

# 启动应用
python app/main.py

# 在另一个终端运行测试
python tests/test-autofix.py
```

### 测试用例

测试脚本包含以下测试用例：

1. 健康检查 API 测试
2. 自动修复 API 健康状态测试
3. 集群诊断测试
4. 正常部署的自动修复测试
5. 问题部署的自动修复测试
6. 通知发送测试
7. 完整工作流测试

## 常见问题排查

### 1. 无法连接到 Kubernetes

**症状**:

- `KubernetesService未初始化`错误
- API 返回"无法连接到 Kubernetes 集群"

**解决方案**:

- 检查 kubeconfig 文件是否存在且有效
- 确认环境变量正确设置
- 验证集群是否可访问：`kubectl get nodes`

### 2. 修复不生效

**症状**:

- API 返回成功但 Pod 仍未就绪

**可能原因**:

- 集群资源不足
- 应用实际问题与分析不符
- 网络或权限问题

**解决方案**:

1. 检查 Pod 事件：`kubectl describe pod <pod-name>`
2. 查看应用日志：`kubectl logs <pod-name>`
3. 尝试手动减小资源请求：
   ```bash
   kubectl patch deployment <deployment-name> -p '{"spec":{"template":{"spec":{"containers":[{"name":"nginx","resources":{"requests":{"memory":"32Mi","cpu":"50m"},"limits":{"memory":"64Mi","cpu":"100m"}},"readinessProbe":{"httpGet":{"path":"/"},"periodSeconds":10,"failureThreshold":3}}]}}}}'
   ```

### 3. LLM 模型调用超时

**症状**:

- API 请求超时
- 修复操作未完成

**解决方案**:

- 增加 API 请求超时时间
- 检查 LLM 服务状态
- 检查网络连接

## 最佳实践

1. **提供详细的问题描述**：在 API 请求中提供尽可能详细的问题描述，帮助 AI 更准确分析
2. **逐步修复**：对于复杂问题，先尝试修复最关键的问题，再处理次要问题
3. **结合监控**：将自动修复系统与监控系统集成，实现问题的自动检测和修复
4. **人工验证**：对于关键系统，建议在自动修复后进行人工验证

## 限制和注意事项

- 自动修复系统不能解决所有 Kubernetes 问题，特别是应用逻辑错误
- 对于生产环境，建议启用通知功能，及时了解修复操作
- 复杂或高风险操作可能需要人工介入
- 确保有适当的备份和回滚机制

## 未来计划

1. **更多资源类型支持**：扩展到更多 Kubernetes 资源类型的诊断和修复
2. **历史数据分析**：基于历史修复数据改进修复策略
3. **预测性修复**：在问题严重影响系统前预测并修复潜在问题
4. **跨集群支持**：支持多集群环境的统一诊断和修复
